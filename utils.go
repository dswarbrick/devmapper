// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Miscellaneous utility functions.

package devmapper

import (
	"fmt"

	"golang.org/x/sys/unix"
)

const (
	// Defined in <linux/loop.h>
	LOOP_SET_FD       = 0x4c00
	LOOP_CLR_FD       = 0x4c01
	LOOP_CTL_GET_FREE = 0x4c82
)

func attachLoopDev(loopDev int, filename string) error {
	loopDevName := fmt.Sprintf("/dev/loop%d", loopDev)

	loopFd, err := unix.Open(loopDevName, unix.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("Cannot open %s - %s", loopDevName, err)
	}

	defer unix.Close(loopFd)

	backingFd, err := unix.Open(filename, unix.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("Cannot open %s - %s", filename, err)
	}

	defer unix.Close(backingFd)

	r1, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(loopFd), LOOP_SET_FD, uintptr(backingFd))
	if int(r1) == -1 {
		return fmt.Errorf("LOOP_SET_FD ioctl failed: errno %d", errno)
	}

	return nil
}

func detachLoopDev(loopDev int) error {
	loopDevName := fmt.Sprintf("/dev/loop%d", loopDev)

	loopFd, err := unix.Open(loopDevName, unix.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("Cannot open %s - %s", loopDevName, err)
	}

	defer unix.Close(loopFd)

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(loopFd), LOOP_CLR_FD, 0)
	if errno != 0 {
		return fmt.Errorf("LOOP_CLR_FD ioctl failed: errno %d", errno)
	}

	return nil
}

func getFreeLoopDev() (int, error) {
	ctlFd, err := unix.Open("/dev/loop-control", unix.O_RDWR, 0600)
	if err != nil {
		return -1, err
	}

	defer unix.Close(ctlFd)

	devNr, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(ctlFd), LOOP_CTL_GET_FREE, 0)
	if int(devNr) == -1 {
		return -1, fmt.Errorf("LOOP_CTL_GET_FREE ioctl failed: errno %d", errno)
	} else {
		return int(devNr), nil
	}
}
