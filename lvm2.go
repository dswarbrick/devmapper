/*
 * Devicemapper / LVM bindings for Go
 * Copyright 2017 Daniel Swarbrick
 *
 * This package contains some alternatives to functions in
 * https://github.com/docker/docker/tree/master/pkg/devicemapper
 */

// +build linux

package devmapper

// #cgo LDFLAGS: -llvm2app
// #include <stdlib.h>
// #include <lvm2app.h>
import "C"

import "unsafe"

// Alias the LVM2 C structs so that we can attach our own methods to them
type (
	CLvm         C.struct_lvm
	CVolumeGroup C.struct_volume_group
)

func InitLVM() *CLvm {
	// lvm_init returns an lvm_t, which is a pointer to a lvm struct. Since method receivers cannot
	// receive pointer types, we need to cast it to a *C.struct_lvm before we return it.
	lvm := C.lvm_init(nil)

	return (*CLvm)(unsafe.Pointer(lvm))
}

func (lvm *CLvm) Close() {
	C.lvm_quit((*C.struct_lvm)(lvm))
}

// getVgNames returns a slice of strings containing volume group names
func (lvm *CLvm) GetVgNames() (names []string) {
	vg_names := C.lvm_list_vg_names((*C.struct_lvm)(lvm))

	for item := vg_names.n; item != vg_names; item = item.n {
		names = append(names, C.GoString((*C.lvm_str_list_t)(unsafe.Pointer(item)).str))
	}

	return
}

// getVgNames returns a slice of strings containing volume group UUIDs
func (lvm *CLvm) GetVgUuids() (uuids []string) {
	vg_uuids := C.lvm_list_vg_uuids((*C.struct_lvm)(lvm))

	for item := vg_uuids.n; item != vg_uuids; item = item.n {
		uuids = append(uuids, C.GoString((*C.lvm_str_list_t)(unsafe.Pointer(item)).str))
	}

	return
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
