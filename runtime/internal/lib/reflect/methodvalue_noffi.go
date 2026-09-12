//go:build llgo_methodvalue_noffi

package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

const methodvalueNoFFIRepresentation = true

type directMethodValueEnv struct {
	marker       *byte
	receiver     unsafe.Pointer
	receiverType *abi.Type
}

var directMethodValueMarker byte

func directMethodValueEnvAt(p unsafe.Pointer) (*directMethodValueEnv, bool) {
	if p == nil {
		return nil, false
	}
	env := (*directMethodValueEnv)(p)
	if env.marker != &directMethodValueMarker {
		return nil, false
	}
	return env, true
}

// makeMethodValue builds the direct closure representation for llgo_noffi.
func makeMethodValue(op string, v Value) Value {
	if v.flag&flagMethod == 0 {
		panic("reflect: internal error: invalid use of makeMethodValue")
	}
	fl := v.flag & (flagRO | flagAddr | flagIndir)
	fl |= flag(v.typ().Kind())
	rcvr := Value{v.typ(), v.ptr, fl}

	_, _, fn := methodReceiver(op, rcvr, int(v.flag)>>flagMethodShift)
	var receiver unsafe.Pointer
	storeRcvr(v, unsafe.Pointer(&receiver))
	env := &directMethodValueEnv{
		marker: &directMethodValueMarker, receiver: receiver, receiverType: rcvr.typ(),
	}
	fv := &struct {
		fn  unsafe.Pointer
		env unsafe.Pointer
	}{fn, unsafe.Pointer(env)}
	ftyp := (*funcType)(unsafe.Pointer(v.Type().(*rtype)))
	return Value{closureOf(ftyp), unsafe.Pointer(fv), v.flag&flagRO | flagIndir | flag(Func)}
}
