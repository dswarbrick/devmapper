// Copyright 2017 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Tests for Go devmapper bindings.

package devmapper

import "testing"

func TestUnmarshallParams(t *testing.T) {
	params := `8 13/5120 512 322/3212 193 63 423 0 0 322 0 1 writeback 2 migration_threshold 2048 smq 0 rw -`

	d1 := dmCacheStatus{
		mdataBlockSize:   8,
		mdataUsedBlocks:  13,
		mdataTotalBlocks: 5120,
		cacheBlockSize:   512,
		cacheUsedBlocks:  322,
		cacheTotalBlocks: 3212,
		readHits:         193,
		readMisses:       63,
		writeHits:        423,
		writeMisses:      0,
		demotions:        0,
		promotions:       322,
		dirty:            0,
	}

	d2 := unmarshallParams(params)

	if d1 != d2 {
		t.Fail()
	}
}
