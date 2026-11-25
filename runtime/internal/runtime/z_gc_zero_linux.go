//go:build linux && !nogc && !baremetal && !wasm

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

// Fixed 256 KiB cutoff sized for Linux (64 x 4 KiB pages, or 4 x 64 KiB pages):
// avoids mincore/madvise overhead on smaller blocks while letting large
// fresh/sparse pointer-free allocations skip eager page faults via
// MADV_DONTNEED.
const linuxLazyAtomicZeroMinSize = 256 << 10

func zeroAtomicStorage(p unsafe.Pointer, size uintptr) {
	if size >= linuxLazyAtomicZeroMinSize {
		bdwgc.ZeroAtomicLarge(p, size)
		return
	}
	c.Memset(p, 0, size)
}
