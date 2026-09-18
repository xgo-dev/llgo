//go:build llgo_methodvalue_noffi

package reflect

import (
	"unsafe"
)

const methodvalueNoFFIRepresentation = true

type directMethodValueEnv struct {
	receiver unsafe.Pointer
}

func directMethodValueEnvAt(unsafe.Pointer) (*directMethodValueEnv, bool) {
	return nil, false
}

// makeMethodValue builds a Go funcval whose code is an ABI-correct thunk.
// The thunk loads the receiver from the hidden closure environment and calls
// the real method with that receiver as its ordinary first argument.
func makeMethodValue(op string, v Value) Value {
	if v.flag&flagMethod == 0 {
		panic("reflect: internal error: invalid use of makeMethodValue")
	}
	fl := v.flag & (flagRO | flagAddr | flagIndir)
	fl |= flag(v.typ().Kind())
	rcvr := Value{v.typ(), v.ptr, fl}

	fn := methodValueThunk(op, rcvr, int(v.flag)>>flagMethodShift)
	var receiver unsafe.Pointer
	storeRcvr(v, unsafe.Pointer(&receiver))
	env := &directMethodValueEnv{receiver: receiver}
	fv := &struct {
		fn  unsafe.Pointer
		env unsafe.Pointer
	}{fn, unsafe.Pointer(env)}
	ftyp := (*funcType)(unsafe.Pointer(v.Type().(*rtype)))
	return Value{closureOf(ftyp), unsafe.Pointer(fv), v.flag&flagRO | flagIndir | flag(Func)}
}
