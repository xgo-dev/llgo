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
	ins := makeFuncFFIArgs(fd, args)
	fd.call(ins)
}

func bind1(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	fd := (*funcData)(userdata)
	ins := makeFuncFFIArgs(fd, args)
	out := validateMakeFuncResults(fd.call(ins), fd.ftyp, fd.tout)
	storeMakeFuncFFIResult(ret, out[0], fd.tout[0], cif.RType)
}

func bindn(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	fd := (*funcData)(userdata)
	ins := makeFuncFFIArgs(fd, args)
	outs := validateMakeFuncResults(fd.call(ins), fd.ftyp, fd.tout)
	var offset uintptr
	for i, out := range outs {
		typ := fd.tout[i]
		field, fieldOffset, next := ffiResultField(cif.RType, i, offset)
		storeMakeFuncFFIResult(add(ret, fieldOffset, ""), out, typ, field)
		offset = next
	}
}
