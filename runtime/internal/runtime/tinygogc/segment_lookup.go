/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package tinygogc

// Every arena owns its block-state bytes. A sentinel block separates adjacent
// arenas in the logical block index, so allocations never cross libc regions.
type heapSegment struct {
	start, end  uintptr
	metadata    uintptr // configure places metadata before end
	first, last uintptr // [first, last) are usable blocks; last is the sentinel
}

var heapSegmentCount int

// libc arenas need not be returned in address order. Keep a separate sorted
// index for conservative pointer lookup; logical block indices stay append-only.
var heapSegmentsByAddress [maxHeapSegments]*heapSegment

func segmentForAddress(address uintptr) *heapSegment {
	// Reject an empty heap or values below the first arena before searching,
	// and avoid a search for the single-arena heap.
	if heapSegmentCount == 0 || address < heapSegmentsByAddress[0].start {
		return nil
	}
	if maxHeapSegments == 1 || heapSegmentCount == 1 {
		segment := heapSegmentsByAddress[0]
		if address < segment.metadata {
			return segment
		}
		return nil
	}
	low, high := 0, heapSegmentCount
	for low < high {
		mid := low + (high-low)/2
		segment := heapSegmentsByAddress[mid]
		if address < segment.start {
			high = mid
		} else if address >= segment.end {
			low = mid + 1
		} else {
			if address < segment.metadata {
				return segment
			}
			return nil
		}
	}
	return nil
}
