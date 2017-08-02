/*
 * Devicemapper / LVM bindings for Go
 * Copyright 2017 Daniel Swarbrick
 *
 * This package contains some alternatives to functions in
 * https://github.com/docker/docker/tree/master/pkg/devicemapper
 *
 * TODO: Add error checking and return to all methods
 */

// +build linux

package devmapper

// #cgo LDFLAGS: -llvm2app
// #include <stdlib.h>
// #include <lvm2app.h>
import "C"

import (
	"fmt"
	"unsafe"
)

// Alias the LVM2 C structs so that we can attach our own methods to them
type (
	CLvm           C.struct_lvm
	CVolumeGroup   C.struct_volume_group
	CLogicalVolume C.struct_logical_volume
)

type lvmError struct {
	errno  int
	errmsg string
}

func (e *lvmError) Error() string {
	return fmt.Sprintf("LVM error %d: %s", e.errno, e.errmsg)
}

func InitLVM() *CLvm {
	// lvm_init returns an lvm_t, which is a pointer to a lvm struct. Since method receivers cannot
	// receive pointer types, we need to cast it to a *C.struct_lvm before we return it.
	lvm := C.lvm_init(nil)

	return (*CLvm)(unsafe.Pointer(lvm))
}

func (lvm *CLvm) lastError() error {
	err := &lvmError{
		errno:  int(C.lvm_errno((*C.struct_lvm)(lvm))),
		errmsg: C.GoString(C.lvm_errmsg((*C.struct_lvm)(lvm))),
	}

	return err
}

func (lvm *CLvm) Close() {
	C.lvm_quit((*C.struct_lvm)(lvm))
}

// CreatePV creates a physical volume on the specified absolute device name (e.g., /dev/sda1), with
// size `size` bytes. Size should be a multiple of 512 bytes. A size of zero bytes will use the
// entire device.
func (lvm *CLvm) CreatePV(device string, size uint64) error {
	Cdevice := C.CString(device)
	defer C.free(unsafe.Pointer(Cdevice))

	if C.lvm_pv_create((*C.struct_lvm)(lvm), Cdevice, C.uint64_t(size)) != 0 {
		return lvm.lastError()
	}

	return nil
}

func (lvm *CLvm) RemovePV(name string) {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))

	C.lvm_pv_remove((*C.struct_lvm)(lvm), Cname)
}

// GetVgNames returns a slice of strings containing volume group names
func (lvm *CLvm) GetVgNames() (names []string) {
	vg_names := C.lvm_list_vg_names((*C.struct_lvm)(lvm))

	for item := vg_names.n; item != vg_names; item = item.n {
		names = append(names, C.GoString((*C.lvm_str_list_t)(unsafe.Pointer(item)).str))
	}

	return
}

// GetVgNames returns a slice of strings containing volume group UUIDs
func (lvm *CLvm) GetVgUuids() (uuids []string) {
	vg_uuids := C.lvm_list_vg_uuids((*C.struct_lvm)(lvm))

	for item := vg_uuids.n; item != vg_uuids; item = item.n {
		uuids = append(uuids, C.GoString((*C.lvm_str_list_t)(unsafe.Pointer(item)).str))
	}

	return
}

func (lvm *CLvm) CreateVG(name string) *CVolumeGroup {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))

	// lvm_vg_create returns a vg_t, which is a pointer to a volume_group struct. Since method
	// receivers cannot receive pointer types, we need to cast it to a *C.struct_volume_group
	// before we return it.
	vg := C.lvm_vg_create((*C.struct_lvm)(lvm), Cname)

	return (*CVolumeGroup)(unsafe.Pointer(vg))
}

func (lvm *CLvm) OpenVg(name string) *CVolumeGroup {
	Cname := C.CString(name)
	Cmode := C.CString("r")

	defer C.free(unsafe.Pointer(Cname))
	defer C.free(unsafe.Pointer(Cmode))

	// lvm_vg_open returns a vg_t, which is a pointer to a volume_group struct. Since method
	// receivers cannot receive pointer types, we need to cast it to a *C.struct_volume_group
	// before we return it.
	vg := C.lvm_vg_open((*C.struct_lvm)(lvm), Cname, Cmode, 0)

	return (*CVolumeGroup)(unsafe.Pointer(vg))
}

func (vg *CVolumeGroup) Close() {
	C.lvm_vg_close((*C.struct_volume_group)(vg))
}

func (vg *CVolumeGroup) Extend(device string) {
	Cdevice := C.CString(device)
	defer C.free(unsafe.Pointer(Cdevice))

	C.lvm_vg_extend((*C.struct_volume_group)(vg), Cdevice)
}

func (vg *CVolumeGroup) Remove() {
	C.lvm_vg_remove((*C.struct_volume_group)(vg))
}

func (vg *CVolumeGroup) Write() {
	C.lvm_vg_write((*C.struct_volume_group)(vg))
}

func (vg *CVolumeGroup) GetSize() uint64 {
	return uint64(C.lvm_vg_get_size((*C.struct_volume_group)(vg)))
}

func (vg *CVolumeGroup) GetFreeSize() uint64 {
	return uint64(C.lvm_vg_get_free_size((*C.struct_volume_group)(vg)))
}

func (vg *CVolumeGroup) GetExtentSize() uint64 {
	return uint64(C.lvm_vg_get_extent_size((*C.struct_volume_group)(vg)))
}

func (vg *CVolumeGroup) GetExtentCount() uint64 {
	return uint64(C.lvm_vg_get_extent_count((*C.struct_volume_group)(vg)))
}

func (vg *CVolumeGroup) GetFreeExtentCount() uint64 {
	return uint64(C.lvm_vg_get_free_extent_count((*C.struct_volume_group)(vg)))
}

func (vg *CVolumeGroup) CreateLvLinear(name string, size uint64) *CLogicalVolume {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))

	// lvm_vg_create_lv_linear returns an lv_t, which is a pointer to a logical_volume struct.
	// Since method receivers cannot receive pointer types, we need to cast it to a
	// *C.struct_logical_volume before we return it.
	lv := C.lvm_vg_create_lv_linear((*C.struct_volume_group)(vg), Cname, C.uint64_t(size))

	return (*CLogicalVolume)(unsafe.Pointer(lv))
}

func (lv *CLogicalVolume) Activate() {
	C.lvm_lv_activate((*C.struct_logical_volume)(lv))
}

func (lv *CLogicalVolume) Deactivate() {
	C.lvm_lv_deactivate((*C.struct_logical_volume)(lv))
}

func (lv *CLogicalVolume) GetAttrs() []byte {
	// Return byte slice of logical volume attrs, e.g.: -wi-a-----
	return []byte(C.GoString(C.lvm_lv_get_attr((*C.struct_logical_volume)(lv))))
}

func (lv *CLogicalVolume) GetName() string {
	return C.GoString(C.lvm_lv_get_name((*C.struct_logical_volume)(lv)))
}

func (lv *CLogicalVolume) GetSize() uint64 {
	return uint64(C.lvm_lv_get_size((*C.struct_logical_volume)(lv)))
}

func (lv *CLogicalVolume) GetUuid() string {
	return C.GoString(C.lvm_lv_get_uuid((*C.struct_logical_volume)(lv)))
}

func (lv *CLogicalVolume) IsActive() bool {
	return C.lvm_lv_is_active((*C.struct_logical_volume)(lv)) == 1
}

func (lv *CLogicalVolume) Remove() {
	C.lvm_vg_remove_lv((*C.struct_logical_volume)(lv))
}
