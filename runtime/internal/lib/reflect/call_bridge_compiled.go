package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func isCompiledReflectOnly(fn unsafe.Pointer) bool { return fn == c.Func(wasmReflectOnly) }

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
		results[i] = runtime.AllocZAligned(typ.Size_, uintptr(typ.Align_))
	}
	if fn == c.Func(wasmReflectOnly) && !method {
		wasmMakeFuncInvoke(env, unsafe.SliceData(args), unsafe.SliceData(results))
	} else {
		code := ft.Call_
		if code == nil {
			panic("reflect: missing compiler call bridge for " + stringFor(&ft.Type))
		}
		entry := closure{fn: code}
		call := *(*func(unsafe.Pointer, unsafe.Pointer, uint8, *unsafe.Pointer, *unsafe.Pointer))(unsafe.Pointer(&entry))
		context := uint8(0)
		if method {
			context = 2
		} else if env != nil {
			context = 1
		}
		call(fn, env, context, unsafe.SliceData(args), unsafe.SliceData(results))
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

func makeCompiledFunc(ft *abi.FuncType, fn func([]Value) []Value, recoverTo unsafe.Pointer) Value {
	if ft.TFlag&abi.TFlagNative != 0 {
		panic("reflect: MakeFunc for a native callback requires a native closure trampoline")
	}
	code := ft.Make_
	if code == nil {
		// A FuncOf signature with dynamically created parameter types cannot
		// appear at a compiled call site. It remains callable via reflection.
		code = c.Func(wasmReflectOnly)
	}
	data := &wasmMakeFuncData{
		callback: c.Func(wasmMakeFuncInvoke),
		data: funcData{
			ftyp: ft, tout: toRuntimeTypes(ft.Out), fn: fn, nin: len(ft.In),
			recoverFrom: code, recoverTo: recoverTo,
		},
	}
	fv := &closure{fn: code, env: unsafe.Pointer(data)}
	return Value{closureOf(ft), unsafe.Pointer(fv), flagIndir | flag(Func)}
}

func wasmReflectOnly() {
	panic("reflect: dynamic function has no compiled entry")
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
