/*
 * Pure Go devicemapper library
 * Copyright 2017 Daniel Swarbrick
 *
 * Implementation of Linux kernel ioctl macros (<uapi/asm-generic/ioctl.h>)
 * See https://www.kernel.org/doc/Documentation/ioctl/ioctl-number.txt
 */

package devmapper

import "testing"

func TestMajor(t *testing.T) {
	rdev := uint64(2162142)
	if major(rdev) != 253 {
		t.Fail()
	}
}

func TestMinor(t *testing.T) {
	rdev := uint64(2162142)
	if minor(rdev) != 734 {
		t.Fail()
	}
}
