//go:build llgo && js && wasm

package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

const LLGoFiles = "_wrap/makefunc_wasm_js.c"

type jsMakeFuncResult struct {
	out []Value
}

func makeFuncCallback(nout int) func(*ffi.Signature, unsafe.Pointer, *unsafe.Pointer, unsafe.Pointer) {
	switch nout {
	case 0:
		return bind0JS
	case 1:
		return bind1JS
	default:
		return bindnJS
	}
}

// The C entry points are excluded from Asyncify and therefore contain no Go
// compiler root frame. libffi's JavaScript trampoline calls them again while
// rewinding with freshly allocated ret and args buffers. invokeJSMakeFunc is a
// separately instrumented Go function: it resumes the original invocation and
// returns a heap-owned result for the C entry to write to the current buffer.

//go:linkname bind0JS C.llgo_reflect_bind0_js
func bind0JS(*ffi.Signature, unsafe.Pointer, *unsafe.Pointer, unsafe.Pointer)

//go:linkname bind1JS C.llgo_reflect_bind1_js
func bind1JS(*ffi.Signature, unsafe.Pointer, *unsafe.Pointer, unsafe.Pointer)

//go:linkname bindnJS C.llgo_reflect_bindn_js
func bindnJS(*ffi.Signature, unsafe.Pointer, *unsafe.Pointer, unsafe.Pointer)

//go:noinline
//export llgo_reflect_invoke_js
func invokeJSMakeFunc(args *unsafe.Pointer, userdata unsafe.Pointer) unsafe.Pointer {
	fd := (*funcData)(userdata)
	ins := make([]Value, fd.nin)
	for i := 0; i < fd.nin; i++ {
		ins[i] = makeFuncArgValue(ffi.Index(args, uintptr(i)), fd.ftyp.In[i])
	}
	out := fd.call(ins)
	if len(fd.tout) != 0 {
		out = validateMakeFuncResults(out, fd.ftyp, fd.tout)
	}
	return unsafe.Pointer(&jsMakeFuncResult{out: out})
}

//export llgo_reflect_store1_js
func storeJSMakeFuncResult1(ret, userdata, result unsafe.Pointer) {
	fd := (*funcData)(userdata)
	out := (*jsMakeFuncResult)(result).out
	storeMakeFuncResult(ret, out[0], fd.tout[0])
}

//export llgo_reflect_storen_js
func storeJSMakeFuncResultN(cif *ffi.Signature, ret, userdata, result unsafe.Pointer) {
	fd := (*funcData)(userdata)
	outs := (*jsMakeFuncResult)(result).out
	var offset uintptr
	alignment := uintptr(cif.RType.Alignment)
	for i, out := range outs {
		typ := fd.tout[i]
		storeMakeFuncResult(add(ret, offset, ""), out, typ)
		offset += (typ.Size_ + alignment - 1) &^ (alignment - 1)
	}
}
