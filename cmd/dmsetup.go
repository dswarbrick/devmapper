// +build linux

// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/dswarbrick/devmapper"
)

func lvmDemo() {
	// Get LVM2 handle
	lvm, _ := devmapper.InitLVM()
	defer lvm.Close()

	fmt.Printf("LVM2 handle: %#v\n", lvm)
	fmt.Printf("VG UUIDs: %v\n", lvm.GetVGUUIDs())
	fmt.Printf("VG Names: %v\n", lvm.GetVGNames())

	fmt.Println("VG Name       Size       Free   PE size  PE count  PE free count    Usage")

	for _, name := range lvm.GetVGNames() {
		vg, err := lvm.OpenVG(name, devmapper.LVM_VG_READ_ONLY)
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Printf("%-8s %9d  %9d %9d %9d      %9d  %6.2f%%\n",
			name, vg.GetSize(), vg.GetFreeSize(), vg.GetExtentSize(), vg.GetExtentCount(),
			vg.GetFreeExtentCount(), 100*(1-(float64(vg.GetFreeSize())/float64(vg.GetSize()))))
		vg.Close()
	}
}

func main() {
	lvmDemo()

	fmt.Println()

	// devmapper demo
	if devices, err := devmapper.GetDeviceList(); err == nil {
		fmt.Printf("%#v\n", devices)

		for _, device := range devices {
			if targets, err := devmapper.GetDeviceTable(device.Name); err == nil {
				fmt.Printf("%#v\n", targets)
			}
		}
	}
}
