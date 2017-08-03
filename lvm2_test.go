// +build linux

// Tests for Go LVM bindings

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
// TODO: Break this up into subtests and fail fast if a preceding step fails
// TODO: Reassess `defer` statements - certain setup actions must be torn down in a specific order,
//       and may not actually be possible if a preceding step failed.
func TestLVM2(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "lvm2_")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

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

	// Disassociate the loop dev_fd from any file descriptor
	r1, r2, err = syscall.Syscall(syscall.SYS_IOCTL, uintptr(dev_fd), LOOP_CLR_FD, 0)
	t.Logf("LOOP_CLR_FD r1: %#v r2: %#v err: %#v", r1, r2, err)

	syscall.Close(dev_fd)
}
