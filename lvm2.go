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

const (
	LVM_VG_READ_ONLY  = "r"
	LVM_VG_READ_WRITE = "w"
)

// Alias the LVM2 C struct so that we can attach our own methods to them
type CLvm C.struct_lvm

type VolumeGroup struct {
	lvm_t *CLvm
	vg    C.vg_t // Pointer to volume_group C struct
}

type LogicalVolume struct {
	vg *VolumeGroup // Parent volume group
	lv C.lv_t       // Pointer to logical_volume C struct
}

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

func (lvm *CLvm) RemovePV(name string) error {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))

	if C.lvm_pv_remove((*C.struct_lvm)(lvm), Cname) != 0 {
		return lvm.lastError()
	}

	return nil
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

func (lvm *CLvm) CreateVG(name string) (*VolumeGroup, error) {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))

	vg := C.lvm_vg_create((*C.struct_lvm)(lvm), Cname)
	if vg == nil {
		return nil, lvm.lastError()
	}

	return &VolumeGroup{lvm, vg}, nil
}

func (lvm *CLvm) OpenVg(name, mode string) (*VolumeGroup, error) {
	Cname := C.CString(name)
	Cmode := C.CString(mode)

	defer C.free(unsafe.Pointer(Cname))
	defer C.free(unsafe.Pointer(Cmode))

	vg := C.lvm_vg_open((*C.struct_lvm)(lvm), Cname, Cmode, 0)
	if vg == nil {
		return nil, lvm.lastError()
	}

	return &VolumeGroup{lvm, vg}, nil
}

func (vg *VolumeGroup) Close() error {
	if C.lvm_vg_close(vg.vg) != 0 {
		return vg.lvm_t.lastError()
	}

	return nil
}

func (vg *VolumeGroup) Extend(device string) error {
	Cdevice := C.CString(device)
	defer C.free(unsafe.Pointer(Cdevice))

	if C.lvm_vg_extend(vg.vg, Cdevice) != 0 {
		return vg.lvm_t.lastError()
	}

	return nil
}

func (vg *VolumeGroup) Remove() error {
	if C.lvm_vg_remove(vg.vg) != 0 {
		return vg.lvm_t.lastError()
	}

	return nil
}

func (vg *VolumeGroup) Write() error {
	if C.lvm_vg_write(vg.vg) != 0 {
		return vg.lvm_t.lastError()
	}

	return nil
}

// TEST ME
func (vg *VolumeGroup) GetSize() uint64 {
	return uint64(C.lvm_vg_get_size(vg.vg))
}

// TEST ME
func (vg *VolumeGroup) GetFreeSize() uint64 {
	return uint64(C.lvm_vg_get_free_size(vg.vg))
}

// TEST ME
func (vg *VolumeGroup) GetExtentSize() uint64 {
	return uint64(C.lvm_vg_get_extent_size(vg.vg))
}

// TEST ME
func (vg *VolumeGroup) GetExtentCount() uint64 {
	return uint64(C.lvm_vg_get_extent_count(vg.vg))
}

// TEST ME
func (vg *VolumeGroup) GetFreeExtentCount() uint64 {
	return uint64(C.lvm_vg_get_free_extent_count(vg.vg))
}

func (vg *VolumeGroup) CreateLvLinear(name string, size uint64) (*LogicalVolume, error) {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))

	lv := C.lvm_vg_create_lv_linear(vg.vg, Cname, C.uint64_t(size))
	if lv == nil {
		return nil, vg.lvm_t.lastError()
	}

	return &LogicalVolume{vg, lv}, nil
}

func (lv *LogicalVolume) Activate() error {
	if C.lvm_lv_activate(lv.lv) != 0 {
		return lv.vg.lvm_t.lastError()
	}

	return nil
}

func (lv *LogicalVolume) Deactivate() error {
	if C.lvm_lv_deactivate(lv.lv) != 0 {
		return lv.vg.lvm_t.lastError()
	}

	return nil
}

func (lv *LogicalVolume) GetAttrs() []byte {
	// Return byte slice of logical volume attrs, e.g.: -wi-a-----
	return []byte(C.GoString(C.lvm_lv_get_attr(lv.lv)))
}

func (lv *LogicalVolume) GetName() string {
	return C.GoString(C.lvm_lv_get_name(lv.lv))
}

func (lv *LogicalVolume) GetSize() uint64 {
	return uint64(C.lvm_lv_get_size(lv.lv))
}

func (lv *LogicalVolume) GetUuid() string {
	return C.GoString(C.lvm_lv_get_uuid(lv.lv))
}

func (lv *LogicalVolume) IsActive() bool {
	return C.lvm_lv_is_active(lv.lv) == 1
}

func (lv *LogicalVolume) Remove() error {
	if C.lvm_vg_remove_lv(lv.lv) != 0 {
		return lv.vg.lvm_t.lastError()
	}

	return nil
}
