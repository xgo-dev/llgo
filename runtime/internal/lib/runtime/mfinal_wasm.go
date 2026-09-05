//go:build wasm && llgo.wasm.gc.linear

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

type wasmFinalizerInterfaceArg struct {
	typeOrItab unsafe.Pointer
	data       unsafe.Pointer
}

type wasmFinalizerClosure struct {
	function    unsafe.Pointer
	environment unsafe.Pointer
}

type wasmFinalizerEntry struct {
	fn              any
	interfaceType   unsafe.Pointer
	result          wasmFinalizerResult
	resultSize      uintptr
	cancelCollector func()
}

type wasmFinalizerResult uint8

const (
	wasmFinalizerVoid wasmFinalizerResult = iota
	wasmFinalizerI32
	wasmFinalizerI64
	wasmFinalizerF32
	wasmFinalizerF64
	wasmFinalizerSRet
)

var wasmFinalizers struct {
	once psync.Once
	mu   psync.Mutex
	m    map[uintptr]*wasmFinalizerEntry
}

func initWasmFinalizers() {
	wasmFinalizers.mu.Init(nil)
	wasmFinalizers.m = make(map[uintptr]*wasmFinalizerEntry)
}

// SetFinalizer implements the Go finalizer contract for the linear-memory
// WebAssembly collector. The registry stores only an encoded object address;
// the collector publishes a real pointer after preserving an unreachable
// object for the duration of its callback.
func SetFinalizer(obj any, finalizer any) {
	objFace := (*eface)(unsafe.Pointer(&obj))
	if objFace._type == nil {
		throw("runtime.SetFinalizer: first argument is nil")
	}
	if objFace._type.Kind() != abi.Pointer {
		throw("runtime.SetFinalizer: first argument is " + objFace._type.String() + ", not pointer")
	}
	objPtr := wasmFinalizerObjectPointer(objFace)
	if objPtr == nil {
		throw("runtime.SetFinalizer: first argument is nil")
	}

	wasmFinalizers.once.Do(initWasmFinalizers)
	key := ^uintptr(objPtr)
	wasmFinalizers.mu.Lock()
	if old := wasmFinalizers.m[key]; old != nil {
		delete(wasmFinalizers.m, key)
		old.cancelCollector()
	}
	wasmFinalizers.mu.Unlock()

	finalizerFace := (*eface)(unsafe.Pointer(&finalizer))
	if finalizerFace._type == nil {
		return
	}
	fnType := wasmFinalizerFuncType(finalizerFace._type)
	if fnType == nil {
		throw("runtime.SetFinalizer: second argument is " + finalizerFace._type.String() + ", not a function")
	}
	if fnType.Variadic() {
		throw("runtime.SetFinalizer: cannot pass " + objFace._type.String() + " to finalizer " + finalizerFace._type.String() + " because dotdotdot")
	}
	if len(fnType.In) != 1 {
		throw("runtime.SetFinalizer: cannot pass " + objFace._type.String() + " to finalizer " + finalizerFace._type.String())
	}
	interfaceType, ok := prepareWasmFinalizerArgument(objFace._type, fnType.In[0])
	if !ok {
		throw("runtime.SetFinalizer: cannot pass " + objFace._type.String() + " to finalizer " + finalizerFace._type.String())
	}
	result, resultSize := classifyWasmFinalizerResult(fnType.Out)
	entry := &wasmFinalizerEntry{
		fn:            finalizer,
		interfaceType: interfaceType,
		result:        result,
		resultSize:    resultSize,
	}
	cancel, registered := tinygogc.AddFinalizer(objPtr, func(ptr unsafe.Pointer) {
		wasmFinalizers.mu.Lock()
		if wasmFinalizers.m[key] != entry {
			wasmFinalizers.mu.Unlock()
			return
		}
		delete(wasmFinalizers.m, key)
		wasmFinalizers.mu.Unlock()
		callWasmFinalizer(entry, ptr)
	})
	if !registered {
		throw("runtime.SetFinalizer: pointer not in allocated block")
	}
	entry.cancelCollector = cancel

	wasmFinalizers.mu.Lock()
	wasmFinalizers.m[key] = entry
	wasmFinalizers.mu.Unlock()
	KeepAlive(obj)
}

func wasmFinalizerObjectPointer(e *eface) unsafe.Pointer {
	if e._type.IsDirectIface() {
		return e.data
	}
	return *(*unsafe.Pointer)(e.data)
}

func wasmFinalizerFuncType(t *abi.Type) *abi.FuncType {
	if !t.IsClosure() {
		return nil
	}
	st := t.StructType()
	if st == nil || len(st.Fields) == 0 {
		return nil
	}
	return st.Fields[0].Typ.FuncType()
}

// prepareWasmFinalizerArgument follows the same assignability subset used by
// the native runtime implementation: identical or unnamed-compatible pointer
// types, and interfaces implemented by the object pointer type.
func prepareWasmFinalizerArgument(objType, argType *abi.Type) (interfaceType unsafe.Pointer, ok bool) {
	if argType == objType {
		return nil, true
	}
	switch argType.Kind() {
	case abi.Pointer:
		if (argType.Uncommon() == nil || objType.Uncommon() == nil) && argType.Elem() == objType.Elem() {
			return nil, true
		}
	case abi.Interface:
		if llruntime.Implements(argType, objType) {
			interfaceType := argType.InterfaceType()
			typeOrItab := unsafe.Pointer(objType)
			if len(interfaceType.Methods) != 0 {
				typeOrItab = unsafe.Pointer(llruntime.NewItab(interfaceType, objType))
			}
			return typeOrItab, true
		}
	}
	return nil, false
}

func classifyWasmFinalizerResult(results []*abi.Type) (wasmFinalizerResult, uintptr) {
	if len(results) == 0 {
		return wasmFinalizerVoid, 0
	}
	resultSize := wasmFinalizerResultsSize(results)
	if resultSize == 0 {
		return wasmFinalizerVoid, 0
	}
	if len(results) != 1 {
		return wasmFinalizerSRet, resultSize
	}
	t := results[0]
	switch t.Kind() {
	case abi.Bool, abi.Int8, abi.Int16, abi.Int32, abi.Uint8, abi.Uint16, abi.Uint32:
		return wasmFinalizerI32, 0
	case abi.Int, abi.Uint, abi.Uintptr:
		if t.Size_ <= 4 {
			return wasmFinalizerI32, 0
		}
		return wasmFinalizerI64, 0
	case abi.Int64, abi.Uint64:
		return wasmFinalizerI64, 0
	case abi.Float32:
		return wasmFinalizerF32, 0
	case abi.Float64:
		return wasmFinalizerF64, 0
	case abi.Chan, abi.Map, abi.Pointer, abi.UnsafePointer:
		if t.Size_ <= 4 {
			return wasmFinalizerI32, 0
		}
		return wasmFinalizerI64, 0
	default:
		return wasmFinalizerSRet, t.Size_
	}
}

func wasmFinalizerResultsSize(results []*abi.Type) uintptr {
	var size, align uintptr = 0, 1
	for _, result := range results {
		fieldAlign := uintptr(result.Align_)
		if fieldAlign == 0 {
			fieldAlign = 1
		}
		size = (size + fieldAlign - 1) &^ (fieldAlign - 1)
		size += result.Size_
		if fieldAlign > align {
			align = fieldAlign
		}
	}
	return (size + align - 1) &^ (align - 1)
}

func callWasmFinalizer(entry *wasmFinalizerEntry, ptr unsafe.Pointer) {
	face := (*eface)(unsafe.Pointer(&entry.fn))
	if entry.interfaceType == nil {
		callWasmPointerFinalizer(face.data, ptr, entry.result, entry.resultSize)
	} else {
		arg := wasmFinalizerInterfaceArg{typeOrItab: entry.interfaceType, data: ptr}
		callWasmInterfaceFinalizer(face.data, arg, entry.result, entry.resultSize)
	}
	KeepAlive(entry.fn)
}

func callWasmPointerFinalizer(fn unsafe.Pointer, ptr unsafe.Pointer, result wasmFinalizerResult, size uintptr) {
	switch result {
	case wasmFinalizerVoid:
		(*(*func(unsafe.Pointer))(fn))(ptr)
	case wasmFinalizerI32:
		_ = (*(*func(unsafe.Pointer) uint32)(fn))(ptr)
	case wasmFinalizerI64:
		_ = (*(*func(unsafe.Pointer) uint64)(fn))(ptr)
	case wasmFinalizerF32:
		_ = (*(*func(unsafe.Pointer) float32)(fn))(ptr)
	case wasmFinalizerF64:
		_ = (*(*func(unsafe.Pointer) float64)(fn))(ptr)
	case wasmFinalizerSRet:
		ret := llruntime.AllocU(size)
		closure := (*wasmFinalizerClosure)(fn)
		callWasmFinalizerSRet(closure.function, ret, closure.environment, ptr)
	}
}

func callWasmInterfaceFinalizer(fn unsafe.Pointer, arg wasmFinalizerInterfaceArg, result wasmFinalizerResult, size uintptr) {
	switch result {
	case wasmFinalizerVoid:
		(*(*func(wasmFinalizerInterfaceArg))(fn))(arg)
	case wasmFinalizerI32:
		_ = (*(*func(wasmFinalizerInterfaceArg) uint32)(fn))(arg)
	case wasmFinalizerI64:
		_ = (*(*func(wasmFinalizerInterfaceArg) uint64)(fn))(arg)
	case wasmFinalizerF32:
		_ = (*(*func(wasmFinalizerInterfaceArg) float32)(fn))(arg)
	case wasmFinalizerF64:
		_ = (*(*func(wasmFinalizerInterfaceArg) float64)(fn))(arg)
	case wasmFinalizerSRet:
		ret := llruntime.AllocU(size)
		closure := (*wasmFinalizerClosure)(fn)
		callWasmInterfaceFinalizerSRet(
			closure.function, ret, closure.environment, arg.typeOrItab, arg.data,
		)
	}
}

//go:linkname callWasmFinalizerSRet C.llgo_wasm_call_sret
func callWasmFinalizerSRet(function, result, environment, argument unsafe.Pointer)

//go:linkname callWasmInterfaceFinalizerSRet C.llgo_wasm_call_interface_sret
func callWasmInterfaceFinalizerSRet(function, result, environment, typeOrItab, data unsafe.Pointer)
