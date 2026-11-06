//go:build !llgo || !wasm || (!wasip1 && (!js || llgo.wasm.reflect.libffi))

package reflect

import (
	"sync"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

func makeProviderFunc(ftyp *funcType, fn func([]Value) []Value, recoverTo unsafe.Pointer) Value {
	if ftyp.Make_ != nil || needsCompiledMakeFunc(ftyp) {
		return makeCompiledFunc(ftyp, fn, recoverTo)
	}
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

// FuncOf may construct a signature containing a packed type for which there
// is no compiled entry. Such functions can still be called through reflection;
// libffi must not infer natural offsets for their unaligned fields.
func needsCompiledMakeFunc(ft *funcType) bool {
	var packed func(*abi.Type) bool
	packed = func(t *abi.Type) bool {
		switch t.Kind() {
		case abi.Array:
			return packed((*abi.ArrayType)(unsafe.Pointer(t)).Elem)
		case abi.Struct:
			st := (*abi.StructType)(unsafe.Pointer(t))
			for _, f := range st.Fields {
				a := uintptr(f.Typ.Align_)
				if a > uintptr(t.Align_) || f.Offset%a != 0 || packed(f.Typ) {
					return true
				}
			}
		}
		return false
	}
	for _, list := range [][]*abi.Type{ft.In, ft.Out} {
		for _, t := range list {
			if packed(t) {
				return true
			}
		}
	}
	return false
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

func bind1(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	fd := (*funcData)(userdata)
	ins := make([]Value, fd.nin)
	for i := 0; i < fd.nin; i++ {
		ins[i] = makeFuncArgValue(ffi.Index(args, uintptr(i)), fd.ftyp.In[i])
	}
	out := validateMakeFuncResults(fd.call(ins), fd.ftyp, fd.tout)
	storeMakeFuncFFIResult(ret, out[0], fd.tout[0], cif.RType)
}

func bindn(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	fd := (*funcData)(userdata)
	ins := make([]Value, fd.nin)
	for i := 0; i < fd.nin; i++ {
		ins[i] = makeFuncArgValue(ffi.Index(args, uintptr(i)), fd.ftyp.In[i])
	}
	outs := validateMakeFuncResults(fd.call(ins), fd.ftyp, fd.tout)
	var offset uintptr
	for i, out := range outs {
		typ := fd.tout[i]
		field, fieldOffset, next := ffiResultField(cif.RType, i, offset)
		storeMakeFuncFFIResult(add(ret, fieldOffset, ""), out, typ, field)
		offset = next
	}
}
