/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package tinygogc

import "testing"

func TestSegmentForAddress(t *testing.T) {
	oldCount, oldIndex := heapSegmentCount, heapSegmentsByAddress
	t.Cleanup(func() {
		heapSegmentCount, heapSegmentsByAddress = oldCount, oldIndex
	})
	// Arenas can be allocated in a different order from their addresses. Only
	// the lookup index is sorted; the returned pointer must identify the arena.
	arenas := []heapSegment{
		{start: 0x4000, metadata: 0x4f00, end: 0x5000},
		{start: 0x1000, metadata: 0x1f00, end: 0x2000},
		{start: 0x2000, metadata: 0x2f00, end: 0x3000},
	}
	sorted := []*heapSegment{&arenas[1], &arenas[2], &arenas[0]}
	copy(heapSegmentsByAddress[:], sorted)
	for count := 0; count <= len(sorted) && count <= len(heapSegmentsByAddress); count++ {
		heapSegmentCount = count
		for address := uintptr(0); address <= 0x5100; address++ {
			var want *heapSegment
			for _, arena := range sorted[:count] {
				if address >= arena.start && address < arena.metadata {
					want = arena
				}
			}
			if got := segmentForAddress(address); got != want {
				t.Fatalf("%d arenas, address %#x: got %p, want %p", count, address, got, want)
			}
		}
		if got := segmentForAddress(^uintptr(0)); got != nil {
			t.Fatalf("%d arenas: non-pointer word matched %p", count, got)
		}
	}
}
