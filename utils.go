// Miscellaneous utility functions
// Copyright 2017 Daniel Swarbrick

package devmapper

import (
	"fmt"
	"syscall"
)

const (
	// Defined in <linux/loop.h>
	LOOP_SET_FD       = 0x4c00
	LOOP_CLR_FD       = 0x4c01
	LOOP_CTL_GET_FREE = 0x4c82
)

func attachLoopDev(loopDev int, filename string) error {
	loopDevName := fmt.Sprintf("/dev/loop%d", loopDev)

	loopFd, err := syscall.Open(loopDevName, syscall.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("Cannot open %s - %s", loopDevName, err)
	}

	defer syscall.Close(loopFd)

	backingFd, err := syscall.Open(filename, syscall.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("Cannot open %s - %s", filename, err)
	}

	defer syscall.Close(backingFd)

	r1, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(loopFd), LOOP_SET_FD, uintptr(backingFd))
	if int(r1) == -1 {
		return fmt.Errorf("LOOP_SET_FD ioctl failed: errno %d", errno)
	}

	return nil
}

func detachLoopDev(loopDev int) error {
	loopDevName := fmt.Sprintf("/dev/loop%d", loopDev)

	loopFd, err := syscall.Open(loopDevName, syscall.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("Cannot open %s - %s", loopDevName, err)
	}

	defer syscall.Close(loopFd)

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(loopFd), LOOP_CLR_FD, 0)
	if errno != 0 {
		return fmt.Errorf("LOOP_CLR_FD ioctl failed: errno %d", errno)
	}

	return nil
}

func getFreeLoopDev() (int, error) {
	ctlFd, err := syscall.Open("/dev/loop-control", syscall.O_RDWR, 0600)
	if err != nil {
		return -1, err
	}

	defer syscall.Close(ctlFd)

	devNr, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(ctlFd), LOOP_CTL_GET_FREE, 0)
	if int(devNr) == -1 {
		return -1, fmt.Errorf("LOOP_CTL_GET_FREE ioctl failed: errno %d", errno)
	} else {
		return int(devNr), nil
	}
}
