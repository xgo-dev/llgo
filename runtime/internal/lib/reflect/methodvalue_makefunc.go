//go:build !llgo_methodvalue_noffi

package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

const methodvalueNoFFIRepresentation = false

type directMethodValueEnv struct {
	marker       *byte
	receiver     unsafe.Pointer
	receiverType *abi.Type
}

func directMethodValueEnvAt(unsafe.Pointer) (*directMethodValueEnv, bool) {
	return nil, false
}

func makeMethodValue(op string, v Value) Value {
	if v.flag&flagMethod == 0 {
		panic("reflect: internal error: invalid use of makeMethodValue")
	}

	fl := v.flag & (flagRO | flagAddr | flagIndir)
	fl |= flag(v.typ().Kind())
	rcvr := Value{v.typ(), v.ptr, fl}
	_, _, recoverTo := methodReceiver(op, rcvr, int(v.flag)>>flagMethodShift)
	method := v
	callOp := "Call"
	if method.Type().(*rtype).t.FuncType().Variadic() {
		callOp = "CallSlice"
	}
	ret := makeFunc(v.Type(), func(args []Value) []Value {
		return method.call(callOp, args)
	}, recoverTo)
	ret.flag |= v.flag & flagRO
	return ret
}
