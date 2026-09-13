//go:build llgo && wasm && wasip1

package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/runtime"
)

const useWasmReflectBridges = true

var (
	wasmReflectOnlyPointer    unsafe.Pointer
	wasmMakeFuncInvokePointer unsafe.Pointer
)

func init() {
	wasmReflectOnlyPointer = ValueOf(wasmReflectOnly).UnsafePointer()
	wasmMakeFuncInvokePointer = ValueOf(wasmMakeFuncInvoke).UnsafePointer()
}

func callWasmBridge(ft *abi.FuncType, fn, env unsafe.Pointer, method bool, prefix []unsafe.Pointer, in []Value) []Value {
	if method {
		env = *(*unsafe.Pointer)(prefix[0])
	}
	args := make([]unsafe.Pointer, len(in))
	for i, arg := range in {
		args[i] = toFFIArg(arg, ft.In[i])
	}
	tout := toRuntimeTypes(ft.Out)
	results := make([]unsafe.Pointer, len(tout))
	out := make([]Value, len(tout))
	for i, typ := range tout {
		results[i] = runtime.AllocZ(typ.Size_)
	}
	if fn == wasmReflectOnlyPointer && !method {
		wasmMakeFuncInvoke(env, unsafe.SliceData(args), unsafe.SliceData(results))
	} else {
		code := ft.Call_
		if code == nil {
			panic("reflect: missing WebAssembly call bridge for " + stringFor(&ft.Type))
		}
		entry := closure{fn: code}
		call := *(*func(unsafe.Pointer, unsafe.Pointer, bool, *unsafe.Pointer, *unsafe.Pointer))(unsafe.Pointer(&entry))
		call(fn, env, method || env != nil, unsafe.SliceData(args), unsafe.SliceData(results))
	}
	for i, typ := range tout {
		out[i] = NewAt(toType(typ), results[i]).Elem()
		resolveIndirectValue(&out[i], typ)
	}
	return out
}

func resetWasmFuncBridge(ft *abi.FuncType) {
	ft.Call_ = nil
	ft.Make_ = nil
}

func copyWasmFuncBridge(dst, src *abi.FuncType) {
	dst.Call_ = src.Call_
	dst.Make_ = src.Make_
}

// The first word is the callback entry consumed by compiler-generated stubs.
// Keeping the callback in the environment avoids retaining reflect's runtime
// implementation just because an otherwise unused function type exists.
type wasmMakeFuncData struct {
	callback unsafe.Pointer
	data     funcData
}

func makeProviderFunc(ft *abi.FuncType, fn func([]Value) []Value, recoverTo unsafe.Pointer) Value {
	code := ft.Make_
	if code == nil {
		// A FuncOf signature with dynamically created parameter types cannot
		// appear at a compiled call site. It remains callable via reflection.
		code = wasmReflectOnlyPointer
	}
	data := &wasmMakeFuncData{
		callback: wasmMakeFuncInvokePointer,
		data: funcData{
			ftyp: ft, tout: toRuntimeTypes(ft.Out), fn: fn, nin: len(ft.In),
			recoverFrom: code, recoverTo: recoverTo,
		},
	}
	fv := &closure{fn: code, env: unsafe.Pointer(data)}
	return Value{closureOf(ft), unsafe.Pointer(fv), flagIndir | flag(Func)}
}

func wasmReflectOnly() {
	panic("reflect: dynamic function has no compiled WebAssembly entry")
}

func wasmMakeFuncInvoke(env unsafe.Pointer, args, results *unsafe.Pointer) {
	fd := &(*wasmMakeFuncData)(env).data
	argv := unsafe.Slice(args, fd.nin)
	ins := make([]Value, fd.nin)
	for i, typ := range fd.ftyp.In {
		ins[i] = makeFuncArgValue(argv[i], typ)
	}
	out := validateMakeFuncResults(fd.call(ins), fd.ftyp, fd.tout)
	retv := unsafe.Slice(results, len(out))
	for i, value := range out {
		storeMakeFuncResult(retv[i], value, fd.tout[i])
	}
}
