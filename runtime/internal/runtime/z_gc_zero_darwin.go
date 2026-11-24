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

// Keep smaller allocations on direct inlineable memset; use the Darwin
// lazy-zero path once the request spans at least four 16 KiB pages.
const darwinLazyAtomicZeroMinSize = 64 << 10

func zeroAtomicStorage(p unsafe.Pointer, size uintptr) {
	if size >= darwinLazyAtomicZeroMinSize {
		bdwgc.ZeroAtomicLarge(p, size)
		return
	}
	c.Memset(p, 0, size)
}
