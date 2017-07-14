/*
 * Pure Go devicemapper library
 * Copyright 2017 Daniel Swarbrick
 *
 * This package contains some alternatives to functions in
 * https://github.com/docker/docker/tree/master/pkg/devicemapper, which uses cgo and requires the
 * actual libdevmapper to build.
 */

package devmapper

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"syscall"
	"unsafe"
)

const (
	DM_MAX_TYPE_NAME = 16
	DM_NAME_LEN      = 128
	DM_UUID_LEN      = 129

	DM_IOCTL = 0xfd
)

// Declared in Linux header <uapi/linux/dm-ioctl.h>
const (
	DM_BUFFER_FULL_FLAG = 1 << 8
)

// Declared in Linux header <uapi/linux/dm-ioctl.h>
const (
	// Top level commands
	DM_VERSION_CMD = iota
	DM_REMOVE_ALL_CMD
	DM_LIST_DEVICES_CMD

	// Device level commands
	DM_DEV_CREATE_CMD
	DM_DEV_REMOVE_CMD
	DM_DEV_RENAME_CMD
	DM_DEV_SUSPEND_CMD
	DM_DEV_STATUS_CMD
	DM_DEV_WAIT_CMD

	// Table level commands
	DM_TABLE_LOAD_CMD
	DM_TABLE_CLEAR_CMD
	DM_TABLE_DEPS_CMD
	DM_TABLE_STATUS_CMD
)

var (
	sizeofDmIoctl = uintptr(binary.Size(dmIoctl{}))

	DM_VERSION      = _iowr(DM_IOCTL, DM_VERSION_CMD, sizeofDmIoctl)
	DM_LIST_DEVICES = _iowr(DM_IOCTL, DM_LIST_DEVICES_CMD, sizeofDmIoctl)
	DM_TABLE_STATUS = _iowr(DM_IOCTL, DM_TABLE_STATUS_CMD, sizeofDmIoctl)

	nativeEndian binary.ByteOrder
)

type dmVersion [3]uint32

func (v dmVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2])
}

// Devmapper ioctl struct, defined in Linux header <uapi/linux/dm-ioctl.h>
type dmIoctl struct {
	Version     dmVersion
	DataSize    uint32
	DataStart   uint32
	TargetCount uint32
	OpenCount   int32
	Flags       uint32
	EventNr     uint32
	_           uint32 // padding
	Dev         uint64
	Name        [DM_NAME_LEN]byte
	Uuid        [DM_UUID_LEN]byte
	Data        [7]byte // padding or data
} // 312 bytes

// Used to specify tables. These structures appear after the dmIoctl at location specified by
// DataStart. Defined in Linux header <uapi/linux/dm-ioctl.h>
type dmTargetSpec struct {
	SectorStart uint64
	Length      uint64
	Status      int32
	Next        uint32
	TargetType  [DM_MAX_TYPE_NAME]byte
	// Parameter string starts immediately after this object.
} // 40 bytes

type dmTarget struct {
	Start  uint64
	Length uint64
	Type   string
	Params string
}

type dmDevice struct {
	Dev  uint64
	Name string
}

type devMapper struct {
	fd      int
	version dmVersion
}

// Determine native endianness of system
func init() {
	i := uint32(1)
	b := (*[4]byte)(unsafe.Pointer(&i))
	if b[0] == 1 {
		nativeEndian = binary.LittleEndian
	} else {
		nativeEndian = binary.BigEndian
	}
}

func (ioc *dmIoctl) packedBytes() []byte {
	b := new(bytes.Buffer)
	binary.Write(b, nativeEndian, ioc)
	return b.Bytes()
}

func (dm *devMapper) Close() {
	syscall.Close(dm.fd)
}

func (dm *devMapper) ioctl(cmd uintptr, dmi *dmIoctl) ([]byte, error) {
	var x uint32

	for x = 1024; x <= (64 * 1024); x *= 2 {
		dmi.DataSize = x

		// Encode dmIoctl struct into byte array and enlarge it for the response
		buf := dmi.packedBytes()
		buf = append(buf, make([]byte, dmi.DataSize)...)

		err := ioctl(uintptr(dm.fd), cmd, uintptr(unsafe.Pointer(&buf[0])))
		if err != nil {
			return nil, err
		}

		// Read ioctl response buffer back into dmi struct
		binary.Read(bytes.NewReader(buf), nativeEndian, dmi)

		if dmi.Flags&DM_BUFFER_FULL_FLAG == 0 {
			return buf, nil
		}
	}

	// If we made it this far without success, the buffer was too small
	return nil, fmt.Errorf("ioctl buffer full")
}

func NewDevMapper() (devMapper, error) {
	var err error

	dm := devMapper{}
	dm.fd, err = syscall.Open("/dev/mapper/control", syscall.O_RDWR, 0600)

	if err != nil {
		return dm, err
	}

	// Query and store the version number
	dmi := dmIoctl{Version: dmVersion{4, 0, 0}}
	dm.ioctl(DM_VERSION, &dmi)
	dm.version = dmi.Version

	return dm, nil
}

// ListDevices returns a list of devmapper devices
func (dm *devMapper) ListDevices() ([]dmDevice, error) {
	var (
		dmName struct {
			Dev  uint64
			Next uint32
		}
		devices []dmDevice
	)

	// TODO: Move command version numbers to central location, like the C libdevmapper does
	dmi := dmIoctl{Version: dmVersion{4, 0, 0}}

	buf, err := dm.ioctl(DM_LIST_DEVICES, &dmi)
	if err != nil {
		return nil, err
	}

	// Reader spanning the dm ioctl reponse payload
	r := bytes.NewReader(buf[dmi.DataStart : dmi.DataStart+dmi.DataSize])

	for {
		var name []byte

		binary.Read(r, nativeEndian, &dmName)

		if dmName.Next != 0 {
			// Make byte array large enough to hold the bytes up until next struct head
			name = make([]byte, int(dmName.Next)-binary.Size(dmName))
		} else {
			// Last device in list - consume all remaining bytes
			name = make([]byte, r.Len())
		}

		r.Read(name)

		devices = append(devices, dmDevice{Dev: dmName.Dev, Name: string(bytes.TrimRight(name, "\x00"))})

		if dmName.Next == 0 {
			break
		}
	}

	return devices, nil
}

func (dm *devMapper) TableStatus(dev uint64) ([]dmTarget, error) {
	var tgt dmTargetSpec

	// TODO: Move command version numbers to central location, like the C libdevmapper does
	dmi := dmIoctl{Version: dmVersion{4, 0, 0}, Dev: dev}

	buf, err := dm.ioctl(DM_TABLE_STATUS, &dmi)
	if err != nil {
		return nil, err
	}

	// Reader spanning the dm ioctl reponse payload
	r := bytes.NewReader(buf[dmi.DataStart : dmi.DataStart+dmi.DataSize])
	targets := make([]dmTarget, dmi.TargetCount)

	// Iterate over targets, reading into dmTargetSpec struct
	for x := 0; x < int(dmi.TargetCount); x++ {
		binary.Read(r, nativeEndian, &tgt)

		// Get reader position
		pos, _ := r.Seek(0, io.SeekCurrent)

		params := make([]byte, int64(tgt.Next)-pos)
		r.Read(params)

		targets[x] = dmTarget{
			Start:  tgt.SectorStart,
			Length: tgt.Length,
			Type:   string(bytes.TrimRight(tgt.TargetType[:], "\x00")),
			Params: string(bytes.TrimRight(params, "\x00")),
		}
	}

	return targets, nil
}

// Version returns a string containing the x.y.z version number of the kernel devmapper
func (dm *devMapper) Version() dmVersion {
	return dm.version
}
