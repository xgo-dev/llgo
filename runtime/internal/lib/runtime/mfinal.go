// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Garbage collector: finalizers and block profiling.

package runtime

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
	"github.com/goplus/llgo/runtime/internal/clite/bdwgc"
	psync "github.com/goplus/llgo/runtime/internal/clite/pthread/sync"
	"github.com/goplus/llgo/runtime/internal/clite/sync/atomic"
)

type finalizerClosure struct {
	fn  unsafe.Pointer
	env unsafe.Pointer
}

type finalizerEntry struct {
	fn     any
	obj    unsafe.Pointer
	key    uintptr
	next   *finalizerEntry
	prevFn bdwgc.FinalizerFunc
	prevCb unsafe.Pointer
	stop   int32
}

var finalizerState struct {
	once psync.Once
	mu   psync.Mutex
	m    map[uintptr]*finalizerEntry
	head *finalizerEntry
	tail *finalizerEntry
}

func initFinalizerState() {
	finalizerState.mu.Init(nil)
	finalizerState.m = make(map[uintptr]*finalizerEntry)
}

func SetFinalizer(obj any, finalizer any) {
	objFace := (*eface)(unsafe.Pointer(&obj))
	if objFace._type == nil {
		throw("runtime.SetFinalizer: first argument is nil")
	}
	if objFace._type.Kind() != abi.Pointer {
		throw("runtime.SetFinalizer: first argument is " + objFace._type.String() + ", not pointer")
	}
	objPtr := ifacePointerData(objFace)
	if objPtr == nil {
		throw("runtime.SetFinalizer: first argument is nil")
	}

	finalizerState.once.Do(initFinalizerState)
	key := hideFinalizerPtr(objPtr)

	finalizerState.mu.Lock()
	if old := finalizerState.m[key]; old != nil {
		atomic.Store(&old.stop, 1)
		delete(finalizerState.m, key)
		restoreFinalizer(objPtr, old)
	}
	finalizerState.mu.Unlock()

	finalizerFace := (*eface)(unsafe.Pointer(&finalizer))
	if finalizerFace._type == nil {
		return
	}
	ft := finalizerFuncType(finalizerFace._type)
	if ft == nil {
		throw("runtime.SetFinalizer: second argument is " + finalizerFace._type.String() + ", not a function")
	}
	if len(ft.In) != 1 || ft.In[0] != objFace._type {
		throw("runtime.SetFinalizer: cannot pass " + objFace._type.String() + " to finalizer " + finalizerFace._type.String())
	}
	entry := &finalizerEntry{fn: finalizer, key: key}
	var oldFn bdwgc.FinalizerFunc
	var oldCb unsafe.Pointer
	bdwgc.RegisterFinalizer(objPtr, setFinalizerCallback, unsafe.Pointer(entry), &oldFn, &oldCb)
	entry.prevFn = oldFn
	entry.prevCb = oldCb

	finalizerState.mu.Lock()
	finalizerState.m[key] = entry
	finalizerState.mu.Unlock()
}

func ifacePointerData(e *eface) unsafe.Pointer {
	if e._type.IsDirectIface() {
		return e.data
	}
	return *(*unsafe.Pointer)(e.data)
}

func finalizerFuncType(t *abi.Type) *abi.FuncType {
	if t.IsClosure() {
		st := t.StructType()
		if st == nil || len(st.Fields) == 0 {
			return nil
		}
		return st.Fields[0].Typ.FuncType()
	}
	return t.FuncType()
}

func callFinalizer(fn any, ptr unsafe.Pointer) {
	c := (*finalizerClosure)((*eface)(unsafe.Pointer(&fn)).data)
	f := *(*func(unsafe.Pointer))(unsafe.Pointer(c))
	f(ptr)
}

func setFinalizerCallback(ptr unsafe.Pointer, cb unsafe.Pointer) {
	entry := (*finalizerEntry)(cb)
	if entry.prevFn != nil {
		entry.prevFn(ptr, entry.prevCb)
	}
	if atomic.Load(&entry.stop) == 1 {
		return
	}

	// Keep the object alive until runFinalizers invokes the Go finalizer.
	// Do not allocate or lock here; BDWGC calls this while collecting.
	entry.obj = ptr
	entry.next = nil
	if finalizerState.tail == nil {
		finalizerState.head = entry
		finalizerState.tail = entry
	} else {
		finalizerState.tail.next = entry
		finalizerState.tail = entry
	}
}

func restoreFinalizer(ptr unsafe.Pointer, entry *finalizerEntry) {
	var oldFn bdwgc.FinalizerFunc
	var oldCb unsafe.Pointer
	if entry.prevFn != nil {
		bdwgc.RegisterFinalizer(ptr, entry.prevFn, entry.prevCb, &oldFn, &oldCb)
		return
	}
	bdwgc.RegisterFinalizer(ptr, nil, nil, &oldFn, &oldCb)
}

func runFinalizers() {
	finalizerState.once.Do(initFinalizerState)
	for {
		entry := finalizerState.head
		if entry == nil {
			return
		}
		finalizerState.head = entry.next
		if finalizerState.head == nil {
			finalizerState.tail = nil
		}
		entry.next = nil
		finalizerState.mu.Lock()
		if finalizerState.m[entry.key] == entry {
			delete(finalizerState.m, entry.key)
		}
		finalizerState.mu.Unlock()

		if atomic.Load(&entry.stop) != 1 {
			callFinalizer(entry.fn, entry.obj)
		}
		entry.obj = nil
	}
}

func hideFinalizerPtr(ptr unsafe.Pointer) uintptr {
	return ^uintptr(ptr)
}
