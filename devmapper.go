/*
 * Devicemapper / LVM bindings for Go
 * Copyright 2017 Daniel Swarbrick
 *
 * This package contains some alternatives to functions in
 * https://github.com/docker/docker/tree/master/pkg/devicemapper
 */

package devmapper

// #cgo LDFLAGS: -ldevmapper -llvm2app
// #include <stdlib.h>
// #include <libdevmapper.h>
// #include <lvm2app.h>
import "C"

import (
	"fmt"
	"unsafe"
)

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

// GetDeviceList returns a list of devmapper devices, including device number and name
func GetDeviceList() (devices []dmDevice, err error) {
	var info C.struct_dm_info

	dmt, err := C.dm_task_create(C.DM_DEVICE_LIST)
	if err != nil {
		return
	}

	defer C.dm_task_destroy(dmt)

	_, err = C.dm_task_run(dmt)
	if err != nil {
		return
	}

	// FIXME: dmsetup.c also checks value of info.exists
	_, err = C.dm_task_get_info(dmt, &info)
	if err != nil {
		return
	}

	dm_names, err := C.dm_task_get_names(dmt)
	if err != nil {
		return
	}

	if dm_names.dev != 0 {
		/*
			dm_names is a "variable length" struct which is tricky to process due to Go's disdain
			for pointer arithmetic.

			struct dm_names {
				uint64_t dev;
				uint32_t next;	// Offset to next struct from start of this struct
				char name[0];
			};
		*/
		for dm_dev := dm_names; ; dm_dev = (*C.struct_dm_names)(unsafe.Pointer(uintptr(unsafe.Pointer(dm_dev)) + uintptr(dm_dev.next))) {
			devices = append(devices, dmDevice{
				uint64(dm_dev.dev),
				C.GoString((*C.char)(unsafe.Pointer(&dm_dev.name))),
			})

			if dm_dev.next == 0 {
				break
			}
		}
	}

	return
}

func GetDeviceTable(name string) (err error) {
	var info C.struct_dm_info

	dmt, err := C.dm_task_create(C.DM_DEVICE_TABLE)
	if err != nil {
		return
	}

	defer C.dm_task_destroy(dmt)

	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))

	_, err = C.dm_task_set_name(dmt, Cname)
	if err != nil {
		return
	}

	_, err = C.dm_task_run(dmt)
	if err != nil {
		return
	}

	// FIXME: dmsetup.c also checks value of info.exists
	_, err = C.dm_task_get_info(dmt, &info)
	if err != nil {
		return
	}

	fmt.Printf("info: %#v\n", info)

	var (
		Cstart, Clength      C.uint64_t
		CtargetType, Cparams *C.char
		next                 uintptr
	)

	// TODO: loop over possible multiple targets
	nextp := C.dm_get_next_target(dmt, unsafe.Pointer(next), &Cstart, &Clength, &CtargetType, &Cparams)
	fmt.Printf("nextp: %#v\n", nextp)

	tgt := dmTarget{uint64(Cstart), uint64(Clength), C.GoString(CtargetType), C.GoString(Cparams)}
	fmt.Printf("target: %#v\n", tgt)

	return
}
