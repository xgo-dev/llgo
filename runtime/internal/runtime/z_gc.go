//go:build !nogc && !baremetal

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

	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/bdwgc"
	"github.com/goplus/llgo/runtime/internal/clite/sync/atomic"
)

// AllocU allocates uninitialized memory.
func AllocU(size uintptr) unsafe.Pointer {
	ret := bdwgc.Malloc(size)
	recordMemProfileAlloc(size)
	return ret
}

// AllocZ allocates zero-initialized memory.
func AllocZ(size uintptr) unsafe.Pointer {
	ret := bdwgc.Malloc(size)
	recordMemProfileAlloc(size)
	return c.Memset(ret, 0, size)
}

func AllocRoot(size uintptr) unsafe.Pointer {
	return bdwgc.MallocUncollectable(size)
}

func FreeRoot(ptr unsafe.Pointer) {
	bdwgc.Free(ptr)
}

type entry struct {
	fn   func()         // cleanup func
	prev unsafe.Pointer // prev cleanup func ptr
	stop int32
}

func finalizer(ptr unsafe.Pointer, cb unsafe.Pointer) {
	e := (*entry)(cb)
	if ptr := atomic.Load(&e.prev); ptr != nil {
		(*(*func())(ptr))()
	}
	if atomic.Load(&e.stop) != 1 {
		e.fn()
	}
}

// AddCleanupPtr attaches a cleanup function to ptr. Some time after ptr is no longer
// reachable, the runtime will call cleanup().
func AddCleanupPtr(ptr unsafe.Pointer, cleanup func()) (cancel func()) {
	e := &entry{fn: cleanup}
	var oldFn bdwgc.FinalizerFunc
	var oldCb unsafe.Pointer
	bdwgc.RegisterFinalizer(ptr, finalizer, unsafe.Pointer(e), &oldFn, &oldCb)
	if oldCb != nil {
		n := uintptr(ptr) ^ 0xffff // hides the pointer from escape analysis
		fn := func() {
			oldFn((unsafe.Pointer)(n^0xffff), oldCb)
		}
		atomic.Store(&e.prev, unsafe.Pointer(&fn))
	}
	return func() {
		atomic.Store(&e.stop, 1)
	}
}
