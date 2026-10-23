/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package build

import (
	"runtime"
	"testing"
)

func TestShouldReleaseGoHeapForLLVM(t *testing.T) {
	const threshold = uint64(256 << 20)
	const released = uint64(64 << 20)
	const maximum = ^uint64(0)

	for _, test := range []struct {
		name           string
		idle, released uint64
		want           bool
	}{
		{"zero", 0, 0, false},
		{"all-released", maximum, maximum, false},
		{"underflow", 1, maximum, false},
		{"below-threshold", threshold - 1, 0, false},
		{"at-threshold", threshold, 0, true},
		{"above-threshold", threshold + 1, 0, true},
		{"subtract-released-below", released + threshold - 1, released, false},
		{"subtract-released-at", released + threshold, released, true},
		{"subtract-released-above", released + threshold + 1, released, true},
		{"maximum-unreleased", maximum, 0, true},
		{"maximum-idle-below", maximum, maximum - threshold + 1, false},
		{"maximum-idle-at", maximum, maximum - threshold, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			stats := runtime.MemStats{HeapIdle: test.idle, HeapReleased: test.released}
			if got := shouldReleaseGoHeapForLLVM(stats); got != test.want {
				t.Fatalf("idle=%d released=%d: release=%t, want %t",
					test.idle, test.released, got, test.want)
			}
		})
	}
}

func TestLLVMGoHeapReleaseBudgetTriggers(t *testing.T) {
	const threshold = uint64(256 << 20)
	const maximum = ^uint64(0)

	for _, test := range []struct {
		name                string
		previous, allocated uint64
		idle, released      uint64
		want                bool
	}{
		{"zero", 0, 0, 0, 0, false},
		{"allocation-below", 0, threshold - 1, 0, 0, false},
		{"allocation-at", 0, threshold, 0, 0, true},
		{"allocation-above", 0, threshold + 1, 0, 0, true},
		{"delta-below", 17, 17 + threshold - 1, 0, 0, false},
		{"delta-at", 17, 17 + threshold, 0, 0, true},
		{"delta-above", 17, 17 + threshold + 1, 0, 0, true},
		{"allocation-underflow", maximum, 1, 0, 0, false},
		{"idle-despite-allocation-underflow", maximum, 1, threshold, 0, true},
		{"idle-despite-low-allocation", 3, 17, threshold, 0, true},
		{"all-idle-released", 17, 17, maximum, maximum, false},
		{"allocation-despite-idle-underflow", 0, threshold, 0, maximum, true},
		{"maximum-new-allocation", 0, maximum, 0, 0, true},
		{"maximum-already-accounted", maximum, maximum, 0, 0, false},
		{"maximum-delta-below", maximum - threshold + 1, maximum, 0, 0, false},
		{"maximum-delta-at", maximum - threshold, maximum, 0, 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			budget := llvmGoHeapReleaseBudget{lastReleaseTotalAlloc: test.previous}
			stats := runtime.MemStats{
				TotalAlloc: test.allocated, HeapIdle: test.idle, HeapReleased: test.released,
			}
			calls := 0
			got := budget.release(stats, func() {
				calls++
				if budget.lastReleaseTotalAlloc != test.previous {
					t.Fatal("allocation baseline advanced before reclamation completed")
				}
			})
			wantCalls, wantBaseline := 0, test.previous
			if test.want {
				wantCalls, wantBaseline = 1, test.allocated
			}
			if got != test.want || calls != wantCalls || budget.lastReleaseTotalAlloc != wantBaseline {
				t.Fatalf("released=%t calls=%d baseline=%d; want %t/%d/%d",
					got, calls, budget.lastReleaseTotalAlloc, test.want, wantCalls, wantBaseline)
			}
		})
	}
}

func TestLLVMGoHeapReleaseBudgetAccumulatesSmallPackages(t *testing.T) {
	var budget llvmGoHeapReleaseBudget
	calls := 0
	for index, allocatedMiB := range []uint64{64, 128, 192, 255, 256, 256, 320, 384, 448, 512} {
		// Ordinary GC activity must not consume the explicit reclamation budget.
		stats := runtime.MemStats{TotalAlloc: allocatedMiB << 20, NumGC: uint32(index + 1)}
		budget.release(stats, func() { calls++ })
		wantCalls := int(allocatedMiB / 256)
		wantBaseline := uint64(wantCalls) * (256 << 20)
		if calls != wantCalls || budget.lastReleaseTotalAlloc != wantBaseline {
			t.Fatalf("after package %d: calls=%d baseline=%d; want %d/%d",
				index, calls, budget.lastReleaseTotalAlloc, wantCalls, wantBaseline)
		}
	}
}

func TestLLVMGoHeapReleaseBudgetIdleReclamationAdvancesBaseline(t *testing.T) {
	var budget llvmGoHeapReleaseBudget
	calls := 0
	for _, step := range []struct {
		allocatedMiB, idleMiB uint64
		wantCalls             int
		wantBaselineMiB       uint64
	}{
		{100, 256, 1, 100},
		{256, 0, 1, 100},
		{355, 0, 1, 100},
		{356, 0, 2, 356},
	} {
		stats := runtime.MemStats{TotalAlloc: step.allocatedMiB << 20, HeapIdle: step.idleMiB << 20}
		budget.release(stats, func() { calls++ })
		if calls != step.wantCalls || budget.lastReleaseTotalAlloc != step.wantBaselineMiB<<20 {
			t.Fatalf("allocated=%d MiB: calls=%d baseline=%d; want %d/%d MiB",
				step.allocatedMiB, calls, budget.lastReleaseTotalAlloc,
				step.wantCalls, step.wantBaselineMiB)
		}
	}
}
