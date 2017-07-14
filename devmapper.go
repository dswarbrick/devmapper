package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	DM_NAME_LEN = 128
	DM_UUID_LEN = 129

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
)

var (
	sizeofDmIoctl = uintptr(binary.Size(dmIoctl{}))

	DM_VERSION = _iowr(DM_IOCTL, DM_VERSION_CMD, sizeofDmIoctl)
)

// Devmapper ioctl struct, defined in Linux header <uapi/linux/dm-ioctl.h>
type dmIoctl struct {
	Version     [3]uint32
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
}

func (dm *devMapper) Close() {
	syscall.Close(dm.fd)
}

func (ioc *dmIoctl) packedBytes() []byte {
	b := new(bytes.Buffer)
	binary.Write(b, binary.LittleEndian, ioc)
	return b.Bytes()
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
		binary.Read(bytes.NewReader(buf), binary.LittleEndian, dmi)

		if dmi.Flags&DM_BUFFER_FULL_FLAG == 0 {
			return buf, nil
		}
	}

	// If we made it this far without success, the buffer was too small
	return nil, fmt.Errorf("ioctl buffer full")
}

type devMapper struct {
	fd      int
	version [3]uint32
}

func newDevMapper() (devMapper, error) {
	var err error

	dm := devMapper{}
	dm.fd, err = syscall.Open("/dev/mapper/control", syscall.O_RDWR, 0600)

	if err != nil {
		return dm, err
	}

	// Query and store the version number
	dmi := dmIoctl{Version: [3]uint32{4, 0, 0}}
	dm.ioctl(DM_VERSION, &dmi)
	dm.version = dmi.Version

	return dm, nil
}

func main() {
	dm, err := newDevMapper()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer dm.Close()

	fmt.Printf("%#v\n", dm)
}
