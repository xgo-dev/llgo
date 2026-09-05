//go:build !nogc && !llgo_noffi

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

// Garbage collector: finalizers and block profiling.

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
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
	once       psync.Once
	registryMu psync.Mutex // protects m
	queueMu    psync.Mutex // protects queue state and callback argument publication
	m          map[uintptr]*finalizerEntry
	head       *finalizerEntry
	tail       *finalizerEntry
}

func initFinalizerState() {
	finalizerState.registryMu.Init(nil)
	finalizerState.queueMu.Init(nil)
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

	finalizerState.registryMu.Lock()
	if old := finalizerState.m[key]; old != nil {
		atomic.Store(&old.state, finalizerStopped)
		delete(finalizerState.m, key)
		restoreFinalizer(objPtr, old)
	}
	finalizerState.registryMu.Unlock()

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

	finalizerState.registryMu.Lock()
	finalizerState.m[key] = entry
	finalizerState.registryMu.Unlock()
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

	// GC_invoke_finalizers may run on several threads concurrently. BDWGC
	// releases its allocation lock before invoking client finalizers, including
	// from its default allocation-time path, so serialize queue publication
	// with concurrent drainers.
	// Do not acquire registryMu here: this callback may run during an allocation
	// made while registryMu is held.
	finalizerState.queueMu.Lock()
	// Keep the object alive until runFinalizers invokes the Go finalizer. The
	// mutex publishes these writes together with the queue entry.
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
	finalizerState.queueMu.Unlock()
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
		finalizerState.queueMu.Lock()
		entry := finalizerState.head
		if entry == nil {
			finalizerState.queueMu.Unlock()
			return
		}
		finalizerState.head = entry.next
		if finalizerState.head == nil {
			finalizerState.tail = nil
		}
		entry.next = nil
		finalizerState.queueMu.Unlock()

		// A later SetFinalizer may already have installed a replacement entry
		// for the same object key; do not delete that newer entry.
		finalizerState.registryMu.Lock()
		if finalizerState.m[entry.key] == entry {
			delete(finalizerState.m, entry.key)
		}
		finalizerState.registryMu.Unlock()

		state := atomic.Load(&entry.state)
		if state != finalizerStopped {
			callFinalizer(entry, state == finalizerInterfaceState)
		}
		// A finalizerEntry is registered and dequeued at most once, so no
		// callback can race this release of its retained argument.
		entry.arg = nil
	}
}

func hideFinalizerPtr(ptr unsafe.Pointer) uintptr {
	return ^uintptr(ptr)
}
