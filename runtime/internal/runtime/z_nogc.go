//go:build nogc
// +build nogc

/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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
)

// AllocU allocates uninitialized memory and returns a non-nil pointer or panics.
// A zero-byte request still allocates at least one byte.
func AllocU(size uintptr) unsafe.Pointer {
	n := size
	if n == 0 {
		n = 1
	}
	ret := c.Malloc(n)
	if ret == nil {
		panic("out of memory")
	}
	recordMemProfileAlloc(size)
	return ret
}

// AllocZ allocates zero-initialized memory.
func AllocZ(size uintptr) unsafe.Pointer {
	ret := AllocU(size)
	c.Memset(ret, 0, size)
	return ret
}

func AllocRoot(size uintptr) unsafe.Pointer {
	if size == 0 {
		size = 1
	}
	ret := c.Malloc(size)
	if ret == nil {
		panic("out of memory")
	}
	return ret
}

func FreeRoot(ptr unsafe.Pointer) {
	c.Free(ptr)
}

// AddCleanupPtr is not implemented when GC is disabled.
// Cleanup functions will never be called.
func AddCleanupPtr(ptr unsafe.Pointer, cleanup func()) (cancel func()) {
	return func() {} // no-op cancel
}

func AddCancelableCleanupPtr(ptr unsafe.Pointer, cleanup func()) uint64 {
	return 0
}

func StopCleanupPtr(id uint64) {}
