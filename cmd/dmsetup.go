package main

// #cgo LDFLAGS: -ldevmapper -llvm2app
// #include <stdlib.h>
// #include <libdevmapper.h>
// #include <lvm2app.h>
import "C"

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/dswarbrick/devmapper"
)

// Alias the LVM2 C structs so that we can attach our own methods to them
type (
	CLvm         C.struct_lvm
	CVolumeGroup C.struct_volume_group
)

func initLVM() *CLvm {
	// lvm_init returns an lvm_t, which is a pointer to a lvm struct. Since method receivers cannot
	// receive pointer types, we need to cast it to a *C.struct_lvm before we return it.
	lvm := C.lvm_init(nil)

	return (*CLvm)(unsafe.Pointer(lvm))
}

func (lvm *CLvm) close() {
	C.lvm_quit((*C.struct_lvm)(lvm))
}

// getVgNames returns a slice of strings containing volume group names
func (lvm *CLvm) getVgNames() (names []string) {
	vg_names := C.lvm_list_vg_names((*C.struct_lvm)(lvm))

	for item := vg_names.n; item != vg_names; item = item.n {
		names = append(names, C.GoString((*C.lvm_str_list_t)(unsafe.Pointer(item)).str))
	}

	return
}

// getVgNames returns a slice of strings containing volume group UUIDs
func (lvm *CLvm) getVgUuids() (uuids []string) {
	vg_uuids := C.lvm_list_vg_uuids((*C.struct_lvm)(lvm))

	for item := vg_uuids.n; item != vg_uuids; item = item.n {
		uuids = append(uuids, C.GoString((*C.lvm_str_list_t)(unsafe.Pointer(item)).str))
	}

	return
}

func (lvm *CLvm) openVg(name string) *CVolumeGroup {
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

func (vg *CVolumeGroup) close() {
	C.lvm_vg_close((*C.struct_volume_group)(vg))
}

func (vg *CVolumeGroup) getSize() uint64 {
	return uint64(C.lvm_vg_get_size((*C.struct_volume_group)(vg)))
}

func (vg *CVolumeGroup) getFreeSize() uint64 {
	return uint64(C.lvm_vg_get_free_size((*C.struct_volume_group)(vg)))
}

func (vg *CVolumeGroup) getExtentSize() uint64 {
	return uint64(C.lvm_vg_get_extent_size((*C.struct_volume_group)(vg)))
}

func (vg *CVolumeGroup) getExtentCount() uint64 {
	return uint64(C.lvm_vg_get_extent_count((*C.struct_volume_group)(vg)))
}

func (vg *CVolumeGroup) getFreeExtentCount() uint64 {
	return uint64(C.lvm_vg_get_free_extent_count((*C.struct_volume_group)(vg)))
}

func lvmDemo() {
	// Get LVM2 handle
	lvm := initLVM()
	defer lvm.close()

	fmt.Printf("\nLVM2 handle: %#v\n", lvm)
	fmt.Printf("VG UUIDs: %#v\n", lvm.getVgUuids())
	fmt.Printf("VG Names: %#v\n", lvm.getVgNames())

	fmt.Println("VG Name       Size       Free   PE size  PE count  PE free count    Usage")

	for _, name := range lvm.getVgNames() {
		vg := lvm.openVg(name)
		fmt.Printf("%-8s %9d  %9d %9d %9d      %9d  %6.2f%%\n",
			name, vg.getSize(), vg.getFreeSize(), vg.getExtentSize(), vg.getExtentCount(),
			vg.getFreeExtentCount(), 100*(1-(float64(vg.getFreeSize())/float64(vg.getSize()))))
		vg.close()
	}
}

func main() {
	dm, err := devmapper.NewDevMapper()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer dm.Close()

	fmt.Printf("Kernel devmapper version: %s\n", dm.Version())

	devices, err := dm.ListDevices()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, device := range devices {
		targets, err := dm.TableStatus(device.Dev)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("%#v %#v\n", device, targets)
	}

	//lvmDemo()

	// Get libdevmapper library version
	buf := make([]C.char, 64)
	C.dm_get_library_version(&buf[0], 64)
	fmt.Printf("libdevmapper version: %s\n", C.GoString(&buf[0]))

	// FIXME: Do more thorough error checking
	task := C.dm_task_create(C.DM_DEVICE_LIST)
	fmt.Printf("dm_task: %#v\n", task)

	res := C.dm_task_run(task)
	fmt.Printf("res: %#v\n", res)

	var info C.struct_dm_info
	res = C.dm_task_get_info(task, &info)
	fmt.Printf("res: %#v\n", res)
	fmt.Printf("info: %#v\n", info)

	// Get devmapper device names
	names := C.dm_task_get_names(task)

	if names.dev != 0 {
		/*
			dm_names is a "variable length" struct which is tricky to process due to Go's disdain
			for pointer arithmetic.

			struct dm_names {
				uint64_t dev;
				uint32_t next;      // Offset to next struct from start of this struct
				char name[0];
			};
		*/
		for dm_dev := names; ; dm_dev = (*C.struct_dm_names)(unsafe.Pointer(uintptr(unsafe.Pointer(dm_dev)) + uintptr(dm_dev.next))) {
			dev_name := (*C.char)(unsafe.Pointer(&dm_dev.name))
			fmt.Printf("dm_dev: %#v (%s)\n", dm_dev, C.GoString(dev_name))

			if dm_dev.next == 0 {
				break
			}
		}
	}
}
