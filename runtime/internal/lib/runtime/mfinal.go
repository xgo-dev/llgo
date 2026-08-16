//go:build !nogc

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
	"github.com/goplus/llgo/runtime/internal/ffi"
	llruntime "github.com/goplus/llgo/runtime/internal/runtime"
)

type finalizerClosure struct {
	fn  unsafe.Pointer
	env unsafe.Pointer
}

type finalizerInterfaceArg struct {
	typeOrItab unsafe.Pointer
	data       unsafe.Pointer
}

const (
	finalizerStopped        int32 = 1
	finalizerInterfaceState       = 2
)

type finalizerEntry struct {
	fn          any
	sig         *ffi.Signature // retains the conservatively allocated return-type graph
	argTypes    []*ffi.Type    // owns the backing storage referenced by sig.ArgTypes
	retSize     uintptr
	arg         unsafe.Pointer // object pointer or preallocated interface header
	key         uintptr
	next        *finalizerEntry
	prevFn      bdwgc.FinalizerFunc
	prevCb      unsafe.Pointer
	state       int32
	explicitEnv bool
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
		atomic.Store(&old.state, finalizerStopped)
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
	if ft.Variadic() {
		throw("runtime.SetFinalizer: cannot pass " + objFace._type.String() + " to finalizer " + finalizerFace._type.String() + " because dotdotdot")
	}
	if len(ft.In) != 1 {
		throw("runtime.SetFinalizer: cannot pass " + objFace._type.String() + " to finalizer " + finalizerFace._type.String())
	}
	argFFIType, interfaceTypeOrItab, ok := prepareFinalizerArgument(objFace._type, ft.In[0])
	if !ok {
		throw("runtime.SetFinalizer: cannot pass " + objFace._type.String() + " to finalizer " + finalizerFace._type.String())
	}
	c := (*finalizerClosure)(finalizerFace.data)
	explicitEnv := c.env != nil && ffi.ClosureEnvExplicit
	sig, argTypes, retSize := newFinalizerFFISignature(ft, explicitEnv, argFFIType)
	entry := &finalizerEntry{
		fn: finalizer, sig: sig, argTypes: argTypes, explicitEnv: explicitEnv,
		retSize: retSize, key: key,
	}
	if interfaceTypeOrItab != nil {
		entry.state = finalizerInterfaceState
		entry.arg = unsafe.Pointer(&finalizerInterfaceArg{typeOrItab: interfaceTypeOrItab})
	}
	var oldFn bdwgc.FinalizerFunc
	var oldCb unsafe.Pointer
	bdwgc.RegisterFinalizer(objPtr, setFinalizerCallback, unsafe.Pointer(entry), &oldFn, &oldCb)
	entry.prevFn = oldFn
	entry.prevCb = oldCb

	finalizerState.mu.Lock()
	finalizerState.m[key] = entry
	finalizerState.mu.Unlock()
}

func prepareFinalizerArgument(objType, argType *abi.Type) (*ffi.Type, unsafe.Pointer, bool) {
	if argType == objType {
		return ffi.TypePointer, nil, true
	}
	switch argType.Kind() {
	case abi.Pointer:
		if (argType.Uncommon() == nil || objType.Uncommon() == nil) && argType.Elem() == objType.Elem() {
			return ffi.TypePointer, nil, true
		}
	case abi.Interface:
		if llruntime.Implements(argType, objType) {
			interfaceType := argType.InterfaceType()
			typeOrItab := unsafe.Pointer(objType)
			if len(interfaceType.Methods) != 0 {
				typeOrItab = unsafe.Pointer(llruntime.NewItab(interfaceType, objType))
			}
			return ffi.TypeInterface, typeOrItab, true
		}
	}
	return nil, nil, false
}

func ifacePointerData(e *eface) unsafe.Pointer {
	if e._type.IsDirectIface() {
		return e.data
	}
	return *(*unsafe.Pointer)(e.data)
}

func finalizerFuncType(t *abi.Type) *abi.FuncType {
	if !t.IsClosure() {
		return nil
	}
	st := t.StructType()
	if st == nil || len(st.Fields) == 0 {
		return nil
	}
	return st.Fields[0].Typ.FuncType()
}

func callFinalizer(entry *finalizerEntry, hasInterfaceArg bool) {
	face := (*eface)(unsafe.Pointer(&entry.fn))
	c := (*finalizerClosure)(face.data)
	var ret unsafe.Pointer
	if entry.retSize != 0 {
		ret = llruntime.AllocU(entry.retSize)
	}
	arg := entry.arg
	if !hasInterfaceArg {
		ptr := entry.arg
		arg = unsafe.Pointer(&ptr)
	}
	if entry.explicitEnv {
		ffi.CallWithEnv(entry.sig, c.fn, c.env, ret, unsafe.Pointer(&c.env), arg)
	} else {
		ffi.CallWithEnv(entry.sig, c.fn, c.env, ret, arg)
	}
	KeepAlive(entry.fn)
	KeepAlive(entry.sig)
	KeepAlive(entry.argTypes)
}

func setFinalizerCallback(ptr unsafe.Pointer, cb unsafe.Pointer) {
	entry := (*finalizerEntry)(cb)
	if entry.prevFn != nil {
		entry.prevFn(ptr, entry.prevCb)
	}
	state := atomic.Load(&entry.state)
	if state == finalizerStopped {
		return
	}

	// Keep the object alive until runFinalizers invokes the Go finalizer.
	// Do not allocate or lock here; BDWGC calls this while collecting.
	if state == finalizerInterfaceState {
		(*finalizerInterfaceArg)(entry.arg).data = ptr
	} else {
		entry.arg = ptr
	}
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

		state := atomic.Load(&entry.state)
		if state != finalizerStopped {
			callFinalizer(entry, state == finalizerInterfaceState)
		}
		entry.arg = nil
	}
}

func hideFinalizerPtr(ptr unsafe.Pointer) uintptr {
	return ^uintptr(ptr)
}
