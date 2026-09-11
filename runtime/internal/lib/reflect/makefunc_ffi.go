//go:build !llgo || !wasm || !wasip1

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
	ins := make([]Value, fd.nin)
	for i := 0; i < fd.nin; i++ {
		ins[i] = makeFuncArgValue(ffi.Index(args, uintptr(i)), fd.ftyp.In[i])
	}
	fd.call(ins)
}

func bind1(_ *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	fd := (*funcData)(userdata)
	ins := make([]Value, fd.nin)
	for i := 0; i < fd.nin; i++ {
		ins[i] = makeFuncArgValue(ffi.Index(args, uintptr(i)), fd.ftyp.In[i])
	}
	out := validateMakeFuncResults(fd.call(ins), fd.ftyp, fd.tout)
	storeMakeFuncResult(ret, out[0], fd.tout[0])
}

func bindn(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	fd := (*funcData)(userdata)
	ins := make([]Value, fd.nin)
	for i := 0; i < fd.nin; i++ {
		ins[i] = makeFuncArgValue(ffi.Index(args, uintptr(i)), fd.ftyp.In[i])
	}
	outs := validateMakeFuncResults(fd.call(ins), fd.ftyp, fd.tout)
	var offset uintptr
	alignment := uintptr(cif.RType.Alignment)
	for i, out := range outs {
		typ := fd.tout[i]
		storeMakeFuncResult(add(ret, offset, ""), out, typ)
		offset += (typ.Size_ + alignment - 1) &^ (alignment - 1)
	}
}
