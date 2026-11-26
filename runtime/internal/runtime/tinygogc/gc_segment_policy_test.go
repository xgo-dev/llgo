/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package tinygogc

import "testing"

// Block geometry of the rv32 targets the external heap is enabled on.
const (
	testBytesPerBlock      = 16
	testBlocksPerStateByte = 4
)

const mib = 1 << 20

// fakeExtHeap hands out consecutive ranges and records every request size.
type fakeExtHeap struct {
	next      uintptr
	sizes     []uintptr
	failAbove uintptr // requests of this size or larger fail; 0 never fails
}

func (f *fakeExtHeap) malloc(size uintptr) uintptr {
	f.sizes = append(f.sizes, size)
	if f.failAbove != 0 && size >= f.failAbove {
		return 0
	}
	addr := f.next
	f.next += size + 64
	return addr
}

func newTestExtHeap(free, limit uintptr) *extHeap {
	h := &extHeap{bytesPerBlock: testBytesPerBlock, blocksPerStateByte: testBlocksPerStateByte}
	h.probe(free, limit)
	return h
}

// dataCapacity returns how many data bytes a segment of the given size offers
// once addHeapSegment has carved its block-state bytes out of it.
func dataCapacity(size uintptr) uintptr {
	metadata := (size + testBlocksPerStateByte*testBytesPerBlock) / (1 + testBlocksPerStateByte*testBytesPerBlock)
	return (size - metadata) / testBytesPerBlock * testBytesPerBlock
}

func checkSegment(t *testing.T, start, end, minimum uintptr) {
	t.Helper()
	if start%testBytesPerBlock != 0 || end%testBytesPerBlock != 0 {
		t.Fatalf("segment [%#x, %#x) is not aligned to %d bytes", start, end, testBytesPerBlock)
	}
	if got := dataCapacity(end - start); got < minimum {
		t.Fatalf("segment [%#x, %#x) holds %d data bytes, want at least %d", start, end, got, minimum)
	}
}

func TestExtHeapWithoutExternalMemory(t *testing.T) {
	for _, free := range []uintptr{0, extHeapMinFree - 1} {
		h := newTestExtHeap(free, ^uintptr(0))
		f := &fakeExtHeap{next: 0x3c000008}
		if start, end := h.allocate(64, extHeapOps{malloc: f.malloc}); start != 0 || end != 0 {
			t.Fatalf("free=%d: got segment [%#x, %#x), want none", free, start, end)
		}
		// The verdict is final: the budget must not be probed again, and no
		// request must reach the allocator.
		if !h.probed || !h.disabled || len(f.sizes) != 0 {
			t.Fatalf("free=%d: probed=%v disabled=%v requests=%v", free, h.probed, h.disabled, f.sizes)
		}
	}
}

func TestExtHeapZeroLimitDisables(t *testing.T) {
	h := newTestExtHeap(16*mib, 0)
	f := &fakeExtHeap{next: 0x3c000000}
	if start, end := h.allocate(64, extHeapOps{malloc: f.malloc}); start != 0 || end != 0 || len(f.sizes) != 0 {
		t.Fatalf("limit=0: got segment [%#x, %#x) with requests %v, want none", start, end, f.sizes)
	}
}

func TestExtHeapSegmentSizes(t *testing.T) {
	h := newTestExtHeap(16*mib, 64*mib)
	f := &fakeExtHeap{next: 0x3c010000}
	var want = []uintptr{1 * mib, 2 * mib, 4 * mib, 4 * mib, 4 * mib}
	for i, size := range want {
		start, end := h.allocate(64, extHeapOps{malloc: f.malloc})
		if start == 0 || end-start != size {
			t.Fatalf("segment %d: got [%#x, %#x), want %d bytes", i, start, end, size)
		}
		checkSegment(t, start, end, 64)
	}
	if h.segments != len(want) || h.reserved != 15*mib {
		t.Fatalf("got %d segments and %d reserved bytes, want %d and %d", h.segments, h.reserved, len(want), 15*mib)
	}
}

func TestExtHeapLargeMinimum(t *testing.T) {
	h := newTestExtHeap(16*mib, ^uintptr(0))
	f := &fakeExtHeap{next: 0x3c000000}
	minimum := uintptr(5 * mib)
	start, end := h.allocate(minimum, extHeapOps{malloc: f.malloc})
	if start == 0 {
		t.Fatal("got no segment, want one sized for the minimum")
	}
	// 4 MiB is the default for the third segment on; the minimum wins here.
	if size := end - start; size <= 4*mib || size < minimum {
		t.Fatalf("got %d bytes, want more than max(4 MiB, minimum)", size)
	}
	checkSegment(t, start, end, minimum)
}

func TestExtHeapBudget(t *testing.T) {
	t.Run("explicit limit", func(t *testing.T) {
		h := newTestExtHeap(16*mib, 5*mib)
		f := &fakeExtHeap{next: 0x3c000000}
		want := []uintptr{1 * mib, 2 * mib, 2 * mib} // the last one is cut to the rest of the budget
		for i, size := range want {
			start, end := h.allocate(64, extHeapOps{malloc: f.malloc})
			if start == 0 || end-start != size {
				t.Fatalf("segment %d: got [%#x, %#x), want %d bytes", i, start, end, size)
			}
		}
		if start, end := h.allocate(64, extHeapOps{malloc: f.malloc}); start != 0 || end != 0 {
			t.Fatalf("budget spent, got [%#x, %#x)", start, end)
		}
		if h.reserved != 5*mib || len(f.sizes) != len(want) {
			t.Fatalf("reserved %d bytes in %d requests, want %d in %d", h.reserved, len(f.sizes), 5*mib, len(want))
		}
	})
	t.Run("automatic limit keeps a quarter", func(t *testing.T) {
		h := newTestExtHeap(16*mib, ^uintptr(0))
		if h.capacity != 12*mib {
			t.Fatalf("got budget %d, want %d", h.capacity, 12*mib)
		}
	})
	t.Run("rest too small for the minimum", func(t *testing.T) {
		h := newTestExtHeap(16*mib, 3*mib+100<<10)
		f := &fakeExtHeap{next: 0x3c000000}
		for i := 0; i < 2; i++ {
			if start, _ := h.allocate(64, extHeapOps{malloc: f.malloc}); start == 0 {
				t.Fatalf("segment %d: got none", i)
			}
		}
		// 100 KiB are left, less than the 1 MiB asked for: no request at all.
		if start, end := h.allocate(1*mib, extHeapOps{malloc: f.malloc}); start != 0 || end != 0 || len(f.sizes) != 2 {
			t.Fatalf("got [%#x, %#x) after %d requests, want none after 2", start, end, len(f.sizes))
		}
	})
}

func TestExtHeapRequestHalving(t *testing.T) {
	t.Run("gives up at 64 KiB", func(t *testing.T) {
		h := newTestExtHeap(16*mib, ^uintptr(0))
		f := &fakeExtHeap{failAbove: 1}
		if start, end := h.allocate(64, extHeapOps{malloc: f.malloc}); start != 0 || end != 0 {
			t.Fatalf("got [%#x, %#x), want none", start, end)
		}
		want := []uintptr{1 * mib, 512 << 10, 256 << 10, 128 << 10, 64 << 10}
		if len(f.sizes) != len(want) {
			t.Fatalf("requests %v, want %v", f.sizes, want)
		}
		for i, size := range want {
			if f.sizes[i] != size {
				t.Fatalf("requests %v, want %v", f.sizes, want)
			}
		}
		if h.reserved != 0 || h.segments != 0 {
			t.Fatalf("failed requests were counted: reserved=%d segments=%d", h.reserved, h.segments)
		}
	})
	t.Run("succeeds with a smaller segment", func(t *testing.T) {
		h := newTestExtHeap(16*mib, ^uintptr(0))
		f := &fakeExtHeap{next: 0x3c000000, failAbove: 256 << 10}
		start, end := h.allocate(64, extHeapOps{malloc: f.malloc})
		if start == 0 || end-start != 128<<10 || h.reserved != 128<<10 || h.segments != 1 {
			t.Fatalf("got [%#x, %#x) reserved=%d segments=%d, want 128 KiB counted once", start, end, h.reserved, h.segments)
		}
	})
	t.Run("never below the minimum", func(t *testing.T) {
		h := newTestExtHeap(16*mib, ^uintptr(0))
		f := &fakeExtHeap{failAbove: 1}
		if start, end := h.allocate(512<<10, extHeapOps{malloc: f.malloc}); start != 0 || end != 0 {
			t.Fatalf("got [%#x, %#x), want none", start, end)
		}
		// Half of 1 MiB is 8 KiB short of the minimum with its metadata.
		if len(f.sizes) != 1 {
			t.Fatalf("requests %v, want only the first one", f.sizes)
		}
	})
}

func TestExtHeapAlignment(t *testing.T) {
	for _, offset := range []uintptr{0, 1, 4, 8, 15} {
		h := newTestExtHeap(16*mib, ^uintptr(0))
		f := &fakeExtHeap{next: 0x3c000000 + offset}
		minimum := uintptr(300 << 10)
		start, end := h.allocate(minimum, extHeapOps{malloc: f.malloc})
		if start == 0 {
			t.Fatalf("offset %d: got no segment", offset)
		}
		// The request is [0x3c000000+offset, 0x3c010000+offset): both ends are
		// rounded inwards, so an unaligned allocation loses one block at the start.
		var wantStart uintptr = 0x3c000000
		if offset != 0 {
			wantStart += 16
		}
		if start != wantStart || end != 0x3c100000 {
			t.Fatalf("offset %d: got [%#x, %#x), want [%#x, 0x3c100000)", offset, start, end, wantStart)
		}
		checkSegment(t, start, end, minimum)
	}
}

func TestExtHeapRequestOverflow(t *testing.T) {
	h := newTestExtHeap(16*mib, ^uintptr(0))
	if size, required := h.request(^uintptr(0) - 8); size != 0 || required != 0 {
		t.Fatalf("got size=%d required=%d, want none for an unrepresentable minimum", size, required)
	}
}
