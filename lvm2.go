// +build linux

// LVM bindings for Go
// Copyright 2017 Daniel Swarbrick

package devmapper

// #cgo LDFLAGS: -llvm2app
// #include <stdlib.h>
// #include <lvm2app.h>
import "C"

import (
	"fmt"
	"unsafe"
)

const (
	LVM_VG_READ_ONLY  = "r"
	LVM_VG_READ_WRITE = "w"
)

type LvmHandle struct {
	lvm C.lvm_t // Pointer to lvm C struct
}

// A PhysicalVolume represents an LVM physical volume object.
type PhysicalVolume struct {
	pv C.pv_t // Pointer to physical_volume C struct
}

// A VolumeGroup represents an LVM volume group object, can contain zero or more logical volumes,
// and is comprised of one or more physical volumes.
type VolumeGroup struct {
	lvm *LvmHandle // Global LVM handle
	vg  C.vg_t     // Pointer to volume_group C struct
}

// A LogicalVolume represents an LVM logical volume object, and belongs to a parent volume group.
type LogicalVolume struct {
	vg *VolumeGroup // Parent volume group
	lv C.lv_t       // Pointer to logical_volume C struct
}

// LVMError represents an error from liblvm2.
type LVMError struct {
	errno  int
	errmsg string
}

func (e *LVMError) Error() string {
	return fmt.Sprintf("LVM error %d: %s", e.errno, e.errmsg)
}

func InitLVM() *LvmHandle {
	// lvm_init returns an lvm_t, which is a pointer to a lvm struct. Since method receivers cannot
	// receive pointer types, we need to cast it to a *C.struct_lvm before we return it.
	lvm := C.lvm_init(nil)

	return &LvmHandle{lvm}
}

func (lvm *LvmHandle) lastError() error {
	err := &LVMError{
		errno:  int(C.lvm_errno(lvm.lvm)),
		errmsg: C.GoString(C.lvm_errmsg(lvm.lvm)),
	}

	return err
}

func (lvm *LvmHandle) Close() {
	C.lvm_quit(lvm.lvm)
}

// CreatePV creates a physical volume on the specified absolute device name (e.g., /dev/sda1), with
// size `size` bytes. Size should be a multiple of 512 bytes. A size of zero bytes will use the
// entire device.
func (lvm *LvmHandle) CreatePV(device string, size uint64) error {
	Cdevice := C.CString(device)
	defer C.free(unsafe.Pointer(Cdevice))

	if C.lvm_pv_create(lvm.lvm, Cdevice, C.uint64_t(size)) != 0 {
		return lvm.lastError()
	}

	return nil
}

func (lvm *LvmHandle) RemovePV(name string) error {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))

	if C.lvm_pv_remove(lvm.lvm, Cname) != 0 {
		return lvm.lastError()
	}

	return nil
}

// GetVgNames returns a slice of strings containing volume group names
func (lvm *LvmHandle) GetVgNames() (names []string) {
	vg_names := C.lvm_list_vg_names(lvm.lvm)

	for item := vg_names.n; item != vg_names; item = item.n {
		names = append(names, C.GoString((*C.lvm_str_list_t)(unsafe.Pointer(item)).str))
	}

	return
}

// GetVgNames returns a slice of strings containing volume group UUIDs
func (lvm *LvmHandle) GetVgUuids() (uuids []string) {
	vg_uuids := C.lvm_list_vg_uuids(lvm.lvm)

	for item := vg_uuids.n; item != vg_uuids; item = item.n {
		uuids = append(uuids, C.GoString((*C.lvm_str_list_t)(unsafe.Pointer(item)).str))
	}

	return
}

func (lvm *LvmHandle) CreateVG(name string) (*VolumeGroup, error) {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))

	vg := C.lvm_vg_create(lvm.lvm, Cname)
	if vg == nil {
		return nil, lvm.lastError()
	}

	return &VolumeGroup{lvm, vg}, nil
}

func (lvm *LvmHandle) OpenVg(name, mode string) (*VolumeGroup, error) {
	Cname := C.CString(name)
	Cmode := C.CString(mode)

	defer C.free(unsafe.Pointer(Cname))
	defer C.free(unsafe.Pointer(Cmode))

	vg := C.lvm_vg_open(lvm.lvm, Cname, Cmode, 0)
	if vg == nil {
		return nil, lvm.lastError()
	}

	return &VolumeGroup{lvm, vg}, nil
}

func (pv *PhysicalVolume) GetDevSize() uint64 {
	return uint64(C.lvm_pv_get_dev_size(pv.pv))
}

func (pv *PhysicalVolume) GetFree() uint64 {
	return uint64(C.lvm_pv_get_free(pv.pv))
}

func (pv *PhysicalVolume) GetMdaCount() uint64 {
	return uint64(C.lvm_pv_get_mda_count(pv.pv))
}

func (pv *PhysicalVolume) GetName() string {
	return C.GoString(C.lvm_pv_get_name(pv.pv))
}

func (pv *PhysicalVolume) GetSize() uint64 {
	return uint64(C.lvm_pv_get_size(pv.pv))
}

func (pv *PhysicalVolume) GetUuid() string {
	return C.GoString(C.lvm_pv_get_uuid(pv.pv))
}

func (vg *VolumeGroup) Close() error {
	if C.lvm_vg_close(vg.vg) != 0 {
		return vg.lvm.lastError()
	}

	return nil
}

// Size must be at least one sector (512 bytes), and will be rounded up to the nearest extent
func (vg *VolumeGroup) CreateLvLinear(name string, size uint64) (*LogicalVolume, error) {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))

	lv := C.lvm_vg_create_lv_linear(vg.vg, Cname, C.uint64_t(size))
	if lv == nil {
		return nil, vg.lvm.lastError()
	}

	return &LogicalVolume{vg, lv}, nil
}

func (vg *VolumeGroup) Extend(device string) error {
	Cdevice := C.CString(device)
	defer C.free(unsafe.Pointer(Cdevice))

	if C.lvm_vg_extend(vg.vg, Cdevice) != 0 {
		return vg.lvm.lastError()
	}

	return nil
}

// TEST ME
func (vg *VolumeGroup) GetExtentCount() uint64 {
	return uint64(C.lvm_vg_get_extent_count(vg.vg))
}

// GetExtentSize returns the current extent size of a volume group in bytes.
func (vg *VolumeGroup) GetExtentSize() uint64 {
	return uint64(C.lvm_vg_get_extent_size(vg.vg))
}

// GetFreeExtentCount returns the current number of free extents in a volume group.
func (vg *VolumeGroup) GetFreeExtentCount() uint64 {
	return uint64(C.lvm_vg_get_free_extent_count(vg.vg))
}

// GetFreeSize returns the current unallocated space of a volume group in bytes.
func (vg *VolumeGroup) GetFreeSize() uint64 {
	return uint64(C.lvm_vg_get_free_size(vg.vg))
}

// GetMaxLV returns the maximum number of logical volumes allowed in a volume group.
func (vg *VolumeGroup) GetMaxLV() uint64 {
	return uint64(C.lvm_vg_get_max_lv(vg.vg))
}

// GetName returns the current name of a volume group.
func (vg *VolumeGroup) GetName() string {
	return C.GoString(C.lvm_vg_get_name(vg.vg))
}

// GetPVCount returns the current number of physical volumes of a volume group.
func (vg *VolumeGroup) GetPVCount() uint64 {
	return uint64(C.lvm_vg_get_pv_count(vg.vg))
}

// GetSequenceNum returns the current metadata sequence number of a volume group. The metadata
// sequence number is incrented for each metadata change. Applications may use the sequence number
// to determine if any LVM objects have changed from a prior query.
func (vg *VolumeGroup) GetSequenceNum() uint64 {
	return uint64(C.lvm_vg_get_seqno(vg.vg))
}

// GetSize returns the current size of a volume group in bytes.
func (vg *VolumeGroup) GetSize() uint64 {
	return uint64(C.lvm_vg_get_size(vg.vg))
}

// GetUUID returns the current LVM UUID of a volume group.
func (vg *VolumeGroup) GetUUID() string {
	return C.GoString(C.lvm_vg_get_uuid(vg.vg))
}

// PVFromName returns an object representing the physical volume specified by name.
func (vg *VolumeGroup) PVFromName(device string) (*PhysicalVolume, error) {
	Cdevice := C.CString(device)
	defer C.free(unsafe.Pointer(Cdevice))

	pv := C.lvm_pv_from_name(vg.vg, Cdevice)
	if pv == nil {
		return nil, vg.lvm.lastError()
	}

	return &PhysicalVolume{pv}, nil
}

// PVFromUUID returns an object representing the physical volume specified by UUID.
func (vg *VolumeGroup) PVFromUUID(uuid string) (*PhysicalVolume, error) {
	Cuuid := C.CString(uuid)
	defer C.free(unsafe.Pointer(Cuuid))

	pv := C.lvm_pv_from_uuid(vg.vg, Cuuid)
	if pv == nil {
		return nil, vg.lvm.lastError()
	}

	return &PhysicalVolume{pv}, nil
}

// Remove removes an underlying LVM handle to a volume group in memory, and requires calling
// Write() to commit the removal to disk.
func (vg *VolumeGroup) Remove() error {
	if C.lvm_vg_remove(vg.vg) != 0 {
		return vg.lvm.lastError()
	}

	return nil
}

// Write commits a volume group to disk. Upon error, retry the operation and / or release the VG
// handle with Close().
func (vg *VolumeGroup) Write() error {
	if C.lvm_vg_write(vg.vg) != 0 {
		return vg.lvm.lastError()
	}

	return nil
}

// Activate activates a logical volume, and is equivalent to the lvm command "lvchange -ay".
func (lv *LogicalVolume) Activate() error {
	if C.lvm_lv_activate(lv.lv) != 0 {
		return lv.vg.lvm.lastError()
	}

	return nil
}

// Deactivate deactivates a logical volume, and is equivalent to the lvm command "lvchange -an".
func (lv *LogicalVolume) Deactivate() error {
	if C.lvm_lv_deactivate(lv.lv) != 0 {
		return lv.vg.lvm.lastError()
	}

	return nil
}

// GetAttrs returns the current attributes of a logical volume, e.g.: "-wi-a-----".
func (lv *LogicalVolume) GetAttrs() []byte {
	return []byte(C.GoString(C.lvm_lv_get_attr(lv.lv)))
}

// GetName returns the current name of a logical volume.
func (lv *LogicalVolume) GetName() string {
	return C.GoString(C.lvm_lv_get_name(lv.lv))
}

// GetSize returns the current size of a logical volume in bytes.
func (lv *LogicalVolume) GetSize() uint64 {
	return uint64(C.lvm_lv_get_size(lv.lv))
}

// GetUUID returns the current LVM UUID of a logical volume.
func (lv *LogicalVolume) GetUUID() string {
	return C.GoString(C.lvm_lv_get_uuid(lv.lv))
}

// IsActive returns the current activation state of a logical volume.
func (lv *LogicalVolume) IsActive() bool {
	return C.lvm_lv_is_active(lv.lv) == 1
}

// Remove removes a logical volume from its volume group. This function commits the change to disk
// and does not require calling Write().
func (lv *LogicalVolume) Remove() error {
	if C.lvm_vg_remove_lv(lv.lv) != 0 {
		return lv.vg.lvm.lastError()
	}

	return nil
}
