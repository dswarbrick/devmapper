/*
 * Devicemapper / LVM bindings for Go
 * Copyright 2017 Daniel Swarbrick

 * dm-cache Statistics Parser
 * See dm-cache documentation at: https://www.kernel.org/doc/Documentation/device-mapper/cache.txt
 */

package devmapper

import "fmt"

type dmCacheStatus struct {
	mdataBlockSize   int // Fixed block size for each metadata block in sectors
	mdataUsedBlocks  int // Number of metadata blocks used
	mdataTotalBlocks int // Total number of metadata blocks
	cacheBlockSize   int // Configurable block size for the cache device in sectors
	cacheUsedBlocks  int // Number of blocks resident in the cache
	cacheTotalBlocks int // Total number of cache blocks
	readHits         int // Number of times a READ bio has been mapped to the cache
	readMisses       int // Number of times a READ bio has been mapped to the origin
	writeHits        int // Number of times a WRITE bio has been mapped to the cache
	writeMisses      int // Number of times a WRITE bio has been mapped to the origin
	demotions        int // Number of times a block has been removed from the cache
	promotions       int // Number of times a block has been moved to the cache
	dirty            int // Number of blocks in the cache that differ from the origin
}

// cacheUsedPerc returns the percentage of cache blocks used
func (d *dmCacheStatus) cacheUsedPerc() float64 {
	return float64(d.cacheUsedBlocks) / float64(d.cacheTotalBlocks) * 100
}

// mdataUsedPerc returns the percentage of metadata blocks used
func (d *dmCacheStatus) mdataUsedPerc() float64 {
	return float64(d.mdataUsedBlocks) / float64(d.mdataTotalBlocks) * 100
}

// readHitRatio returns the cache read hit ratio (0.0 - 1.0)
func (d *dmCacheStatus) readHitRatio() float64 {
	if d.readHits > 0 {
		return float64(d.readHits) / float64(d.readHits+d.readMisses)
	} else {
		return 0
	}
}

// writeHitRatio returns the cache write hit ratio (0.0 - 1.0)
func (d *dmCacheStatus) writeHitRatio() float64 {
	if d.writeHits > 0 {
		return float64(d.writeHits) / float64(d.writeHits+d.writeMisses)
	} else {
		return 0
	}
}

func unmarshallParams(params string) dmCacheStatus {
	var s dmCacheStatus

	fmt.Sscanf(params, "%d %d/%d %d %d/%d %d %d %d %d %d %d %d",
		&s.mdataBlockSize, &s.mdataUsedBlocks, &s.mdataTotalBlocks,
		&s.cacheBlockSize, &s.cacheUsedBlocks, &s.cacheTotalBlocks,
		&s.readHits, &s.readMisses,
		&s.writeHits, &s.writeMisses,
		&s.demotions, &s.promotions, &s.dirty)

	// Remainder of table data needs to be handled token by token, e.g.:
	// tokens := strings.Fields(params)[11:]

	return s
}
