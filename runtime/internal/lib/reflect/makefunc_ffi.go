//go:build !llgo || !wasm || (!wasip1 && (!js || llgo.wasm.reflect.libffi))

package reflect

import (
	"sync"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

func makeProviderFunc(ftyp *funcType, fn func([]Value) []Value, recoverTo unsafe.Pointer) Value {
	sig, err := toFFISig(ftyp.In, ftyp.Out)
	if err != nil {
		panic(err)
	}
	ffiClosure := ffi.NewClosure()
	userdata := &funcData{
		ftyp:        ftyp,
		fn:          fn,
		nin:         len(ftyp.In),
		tout:        toRuntimeTypes(ftyp.Out),
		recoverFrom: ffiClosure.Fn,
		recoverTo:   recoverTo,
	}

	if err := ffiClosure.Bind(sig, makeFuncCallback(len(ftyp.Out)), unsafe.Pointer(userdata)); err != nil {
		panic("libffi error: " + err.Error())
	}
	// Keep the executable closure, signature graph, and callback state alive.
	keepMutex.Lock()
	keepAlive = append(keepAlive, ffiClosure, sig, userdata)
	keepMutex.Unlock()

	fv := &closure{fn: ffiClosure.Fn}
	return Value{closureOf(ftyp), unsafe.Pointer(fv), flagIndir | flag(Func)}
}

var (
	keepMutex sync.Mutex
	keepAlive []any
)

func bind0(_ *ffi.Signature, _ unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	fd := (*funcData)(userdata)
	fd.call(makeFuncFFIArgs(fd, args))
}

func bind1(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	fd := (*funcData)(userdata)
	out := validateMakeFuncResults(fd.call(makeFuncFFIArgs(fd, args)), fd.ftyp, fd.tout)
	storeMakeFuncFFIResult(ret, out[0], fd.tout[0], cif.ReturnType())
}

func bindn(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	fd := (*funcData)(userdata)
	outs := validateMakeFuncResults(fd.call(makeFuncFFIArgs(fd, args)), fd.ftyp, fd.tout)
	var offset uintptr
	for i, out := range outs {
		typ := fd.tout[i]
		field, fieldOffset, next := ffiResultField(cif.ReturnType(), i, offset)
		storeMakeFuncFFIResult(add(ret, fieldOffset, ""), out, typ, field)
		offset = next
	}
}

func makeFuncFFIArgs(fd *funcData, args *unsafe.Pointer) []Value {
	ins := make([]Value, fd.nin)
	var physical uintptr
	for i, typ := range fd.ftyp.In {
		if omitFFIArg(typ) {
			// The logical argument still exists even when the callable ABI
			// omits it. Do not read a libffi argument slot for an empty value.
			ins[i] = Zero(toType(typ))
			continue
		}
		ins[i] = makeFuncArgValue(ffi.Index(args, physical), typ)
		physical++
	}
	return ins
}
