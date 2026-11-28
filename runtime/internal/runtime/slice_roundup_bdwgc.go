//go:build llgo && !nogc && !baremetal && !wasm

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

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
)

// sliceAllocUsableSize returns the usable byte capacity of an already-allocated
// BDWGC block starting at p (p must be the exact allocation base, not an
// interior pointer from Memalign).
//
// Standard BDWGC builds define ALL_INTERIOR_POINTERS with EXTRA_BYTES == 1 so a
// one-past-the-end slice pointer remains inside the object's extent. Reserving
// 1 trailing byte avoids over-reporting capacity while reclaiming the actual
// size-class slop BDWGC already allocated without ever bumping a request into
// the next size class or heap block.
func sliceAllocUsableSize(p unsafe.Pointer, requested uintptr) uintptr {
	actual := bdwgc.Size(c.Pointer(p))
	if actual <= requested+1 {
		return requested
	}
	usable := actual - 1
	if usable > maxAlloc {
		return maxAlloc
	}
	return usable
}
