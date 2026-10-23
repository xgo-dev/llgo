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
	"runtime/debug"
	"sync"
)

const llvmGoHeapReleaseThreshold = 256 << 20

var (
	llvmGoHeapReleaseMu   sync.Mutex
	llvmGoHeapReclamation llvmGoHeapReleaseBudget
)

type llvmGoHeapReleaseBudget struct {
	lastReleaseTotalAlloc uint64
}

func shouldReleaseGoHeapForLLVM(stats runtime.MemStats) bool {
	return stats.HeapIdle >= stats.HeapReleased &&
		stats.HeapIdle-stats.HeapReleased >= llvmGoHeapReleaseThreshold
}

func (budget *llvmGoHeapReleaseBudget) release(stats runtime.MemStats, reclaim func()) bool {
	// Allocation traffic is a work budget, not a measure of garbage. It also
	// covers dead objects that have not yet become idle spans after a GC.
	allocated := stats.TotalAlloc >= budget.lastReleaseTotalAlloc &&
		stats.TotalAlloc-budget.lastReleaseTotalAlloc >= llvmGoHeapReleaseThreshold
	if !shouldReleaseGoHeapForLLVM(stats) && !allocated {
		return false
	}
	reclaim()
	// Keep allocations made during reclamation in the next interval. Ordinary
	// GC and sub-threshold package boundaries do not reset this budget.
	budget.lastReleaseTotalAlloc = stats.TotalAlloc
	return true
}

// LLVM's native allocations are outside the Go heap. Reclaim Go pages before
// its optimizer needs space after substantial idle memory or allocation work,
// without forcing a full GC for every package or discarding live source data.
func releaseGoHeapForLLVM() {
	// The heap is process-wide. Sample under the same lock as collection so
	// parallel backends cannot repeatedly collect from one stale observation.
	llvmGoHeapReleaseMu.Lock()
	defer llvmGoHeapReleaseMu.Unlock()

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	llvmGoHeapReclamation.release(stats, debug.FreeOSMemory)
}
