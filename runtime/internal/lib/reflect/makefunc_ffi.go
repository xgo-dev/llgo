//go:build (!llgo || !wasm || (!wasip1 && (!js || llgo.wasm.reflect.libffi))) && !(baremetal && llgo.baremetal.stdio && !wasm)

package reflect

import (
	"sync"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

func makeProviderFunc(ftyp *funcType, fn func([]Value) []Value, recoverTo unsafe.Pointer, origin *Value) Value {
	if baremetalReflection {
		panic("reflect: dynamic calls require a function call provider")
	}
	sig, err := toFFISig(ftyp.In(), ftyp.Out())
	if err != nil {
		panic(err)
	}
	ffiClosure := ffi.NewClosure()
	userdata := &funcData{
		ftyp:        ftyp,
		fn:          fn,
		nin:         ftyp.InCount,
		tout:        toRuntimeTypes(ftyp.Out()),
		recoverFrom: ffiClosure.Fn,
		recoverTo:   recoverTo,
	}

	if err := ffiClosure.Bind(sig, makeFuncCallback(ftyp.OutCount), unsafe.Pointer(userdata)); err != nil {
		panic("libffi error: " + err.Error())
	}
	// Keep the executable closure, signature graph, and callback state alive.
	ffiFuncs.Lock()
	if ffiFuncs.byEntry == nil {
		ffiFuncs.byEntry = make(map[unsafe.Pointer]ffiFunc)
	}
	ffiFuncs.byEntry[ffiClosure.Fn] = ffiFunc{ffiClosure, sig, userdata, origin}
	ffiFuncs.Unlock()

	fv := &closure{fn: ffiClosure.Fn}
	return Value{closureOf(ftyp), unsafe.Pointer(fv), flagIndir | flag(Func)}
}

type ffiFunc struct {
	closure *ffi.Closure
	sig     *ffi.Signature
	data    *funcData
	origin  *Value
}

var ffiFuncs struct {
	sync.RWMutex
	byEntry map[unsafe.Pointer]ffiFunc
}

func providerNativeSource(_ *funcType, entry *closure) *Value {
	if baremetalReflection || entry.env != nil {
		return nil
	}
	ffiFuncs.RLock()
	origin := ffiFuncs.byEntry[entry.fn].origin
	ffiFuncs.RUnlock()
	return origin
}

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
	for i, typ := range fd.ftyp.In() {
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

func storeMakeFuncFFIResult(ret unsafe.Pointer, v Value, typ *abi.Type, ffiType *ffi.Type) {
	size := typ.Size_
	if ffiType.ByteSize() < size {
		size = ffiType.ByteSize()
	}
	if size != 0 {
		c.Memmove(ret, toFFIArg(v, typ), size)
	}
}

func ffiResultField(aggregate *ffi.Type, index int, offset uintptr) (field *ffi.Type, fieldOffset, next uintptr) {
	field = ffi.TypeElement(aggregate, uintptr(index))
	if field == nil {
		panic("reflect: missing libffi result field")
	}
	fieldOffset = offset
	if alignment := uintptr(field.Alignment); alignment > 1 {
		fieldOffset = align(fieldOffset, alignment)
	}
	return field, fieldOffset, fieldOffset + field.ByteSize()
}
