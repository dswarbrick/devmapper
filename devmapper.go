// +build linux

// Copyright 2017 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Package devicemapper is a collection of wrappers around libdevmapper / liblvm2.
//
package devmapper

// #cgo LDFLAGS: -ldevmapper
// #include <stdlib.h>
// #include <libdevmapper.h>
import "C"

import (
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

func GetDeviceTable(name string) (targets []dmTarget, err error) {
	var (
		info C.struct_dm_info
		next uintptr
	)

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

	for x := 0; x < int(info.target_count); x++ {
		var (
			Cstart, Clength      C.uint64_t
			CtargetType, Cparams *C.char
		)

		nextp := C.dm_get_next_target(dmt, unsafe.Pointer(next), &Cstart, &Clength, &CtargetType, &Cparams)
		targets = append(targets, dmTarget{
			uint64(Cstart),
			uint64(Clength),
			C.GoString(CtargetType),
			C.GoString(Cparams),
		})

		if nextp == nil {
			break
		}
	}

	// TODO: loop over possible multiple targets
	return
}
