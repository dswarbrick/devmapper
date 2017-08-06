// +build linux

// Copyright 2017 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Tests for Go LVM bindings.

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

func randString(length int) string {
	rand.Seed(time.Now().UnixNano())

	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]byte, length)

	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

// WIP: Create loop image, attach it to first available loop device
// TODO: Break this up into subtests and fail fast if a preceding step fails
// TODO: Reassess `defer` statements - certain setup actions must be torn down in a specific order,
//       and may not actually be possible if a preceding step failed.
func TestLVM2(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "lvm2_")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	// Create 100 MiB sparse file
	if err := syscall.Ftruncate(int(tmpfile.Fd()), 100*(1<<20)); err != nil {
		t.Fatal(err)
	}

	tmpfile.Close()

	// Get next available loop device
	loop_dev, err := getFreeLoopDev()
	if err != nil {
		t.Fatal("Cannot determine next available loop device:", err)
	}

	loopDevName := fmt.Sprintf("/dev/loop%d", loop_dev)

	if err := attachLoopDev(loop_dev, tmpfile.Name()); err != nil {
		t.Fatal("Cannot attach loop device: %s", err)
	}

	lvm, err := InitLVM()
	if err != nil {
		t.Fatal(err)
	}
	defer lvm.Close()

	// Size of zero will use entire device
	if err := lvm.CreatePV(loopDevName, 0); err != nil {
		t.Fatal(err)
	}

	// Create an empty VG object
	vg, err := lvm.CreateVG(randString(16))
	if err != nil {
		t.Fatal(err)
	}

	// Add PV to VG; requires calling vg.Write() to commit changes.
	if err := vg.Extend(loopDevName); err != nil {
		t.Fatal(err)
	}

	// Commit changes to VG
	if err := vg.Write(); err != nil {
		t.Fatal(err)
	}

	_, err = vg.PVFromName(loopDevName)
	if err != nil {
		t.Fatal(err)
	}

	vgNames := lvm.GetVGNames()
	t.Logf("VG names: %v\n", vgNames)

	lv, err := vg.CreateLVLinear("testvol1", 50*(1<<20))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("LV UUID: %s\n", lv.GetUUID())
	t.Logf("LV name: %s  size: %d  active: %v\n", lv.GetName(), lv.GetSize(), lv.IsActive())
	t.Logf("LV attrs: %s\n", lv.GetAttrs())
	t.Logf("Deactivating LV...")

	if err := lv.Deactivate(); err != nil {
		t.Fatal(err)
	}

	t.Logf("LV attrs: %s\n", lv.GetAttrs())
	t.Logf("Activating LV...")

	if err := lv.Activate(); err != nil {
		t.Fatal(err)
	}

	t.Logf("LV attrs: %s\n", lv.GetAttrs())

	if err := lv.Remove(); err != nil {
		t.Fatal(err)
	}

	t.Logf("VG sequence no.: %d\n", vg.GetSequenceNum())

	// Remove VG object; requires calling vg.Write() to commit changes.
	if err := vg.Remove(); err != nil {
		t.Fatal(err)
	}

	if err := vg.Write(); err != nil {
		t.Fatal(err)
	}

	if err := vg.Close(); err != nil {
		t.Fatal(err)
	}

	if err := lvm.RemovePV(loopDevName); err != nil {
		t.Fatal(err)
	}

	detachLoopDev(loop_dev)
}
