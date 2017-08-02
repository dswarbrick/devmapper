/*
 * Devicemapper / LVM bindings for Go
 * Copyright 2017 Daniel Swarbrick

 * LVM2 tests
 */

// +build linux

package devmapper

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"syscall"
	"testing"
	"time"
)

const (
	// Defined in <linux/loop.h>
	LOOP_SET_FD       = 0x4c00
	LOOP_CLR_FD       = 0x4c01
	LOOP_CTL_GET_FREE = 0x4c82

	LOOP_SIZE = 100 * (1 << 20)
)

func randString(length int) string {
	rand.Seed(time.Now().UnixNano())

	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, length)

	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

// WIP: Create loop image, attach it to first available loop device
func TestLVM2(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "lvm2_")
	if err != nil {
		t.Fail()
	}

	// Truncating the new file to non-zero size should create a sparse file
	if syscall.Ftruncate(int(tmpfile.Fd()), LOOP_SIZE) != nil {
		t.Fail()
	}

	fd, err := syscall.Open("/dev/loop-control", syscall.O_RDWR, 0600)
	if err != nil {
		t.Fatal("Cannot open /dev/loop-control:", err)
	}

	// Get next available loop device
	loop_dev, _, _ := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), LOOP_CTL_GET_FREE, 0)
	t.Logf("Available loop dev: %d\n", loop_dev)
	syscall.Close(fd)

	loopDevName := fmt.Sprintf("/dev/loop%d", loop_dev)

	dev_fd, err := syscall.Open(loopDevName, syscall.O_RDWR, 0600)
	if err != nil {
		t.Fatal("LOOP_SET_FD failed: cannot open %s - %s", loopDevName, err)
	}

	// Associate the tmpfile with the available loop device
	r1, r2, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(dev_fd), LOOP_SET_FD, tmpfile.Fd())
	t.Logf("LOOP_SET_FD r1: %#v r2: %#v err: %#v", r1, r2, err)

	tmpfile.Close()

	lvm := InitLVM()
	defer lvm.Close()

	lvm.CreatePV(loopDevName, 0)

	vg := lvm.CreateVG(randString(16))
	vg.Extend(loopDevName)
	vg.Write()

	t.Logf("VG names: %#v\n", lvm.GetVgNames())

	lv := vg.CreateLvLinear("testvol1", 10*(1<<20))
	t.Logf("LV UUID: %s\n", lv.GetUuid())
	lv.Remove()

	vg.Remove()
	vg.Write()

	vg.Close()

	lvm.RemovePV(loopDevName)

	// Disassociate the loop dev_fd from any file descriptor
	r1, r2, err = syscall.Syscall(syscall.SYS_IOCTL, uintptr(dev_fd), LOOP_CLR_FD, 0)
	t.Logf("LOOP_CLR_FD r1: %#v r2: %#v err: %#v", r1, r2, err)

	syscall.Close(dev_fd)
	os.Remove(tmpfile.Name())
}
