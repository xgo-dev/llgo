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

var llvmGoHeapReleaseMu sync.Mutex

func shouldReleaseGoHeapForLLVM(stats runtime.MemStats) bool {
	return stats.HeapIdle >= stats.HeapReleased &&
		stats.HeapIdle-stats.HeapReleased >= llvmGoHeapReleaseThreshold
}

// LLVM's native allocations are outside the Go heap. Return a substantial
// amount of already-unused Go heap memory before its optimizer needs space,
// without forcing a full GC for every package or discarding live source data.
func releaseGoHeapForLLVM() {
	// The heap is process-wide. Sample under the same lock as collection so
	// parallel backends cannot repeatedly collect from one stale observation.
	llvmGoHeapReleaseMu.Lock()
	defer llvmGoHeapReleaseMu.Unlock()

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	if shouldReleaseGoHeapForLLVM(stats) {
		debug.FreeOSMemory()
	}
}
