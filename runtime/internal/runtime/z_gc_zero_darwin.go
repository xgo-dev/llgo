//go:build darwin && !nogc && !baremetal && !wasm

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

// Fixed 128 KiB cutoff sized for Darwin (8 x 16 KiB pages on darwin/arm64,
// 32 x 4 KiB pages on darwin/amd64): keeps smaller allocations on direct
// inlineable memset, where a VM syscall is slower than clearing hot resident
// memory, while the C helper also enforces its own page-based minimum.
const darwinLazyAtomicZeroMinSize = 128 << 10

func zeroAtomicStorage(p unsafe.Pointer, size uintptr) {
	if size >= darwinLazyAtomicZeroMinSize {
		bdwgc.ZeroAtomicLarge(p, size)
		return
	}
	c.Memset(p, 0, size)
}
