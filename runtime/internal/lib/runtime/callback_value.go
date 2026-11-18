package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

// callbackValue is a decoded callable, not the storage of a native function
// value. Native values contain only code; Go closures also retain an environment.
type callbackValue struct {
	code unsafe.Pointer
	env  unsafe.Pointer
}

func callbackFuncType(t *abi.Type) *abi.FuncType {
	if t == nil {
		return nil
	}
	if t.HasNativeLayout() {
		return t.FuncType()
	}
	if !t.IsClosure() {
		return nil
	}
	st := t.StructType()
	if st == nil || len(st.Fields) == 0 {
		return nil
	}
	return st.Fields[0].Typ.FuncType()
}

func decodeCallback(face *eface) callbackValue {
	if face._type.HasNativeLayout() {
		if face._type.IsDirectIface() {
			return callbackValue{code: face.data}
		}
		if face.data != nil {
			return callbackValue{code: *(*unsafe.Pointer)(face.data)}
		}
	} else if face.data != nil {
		return *(*callbackValue)(face.data)
	}
	return callbackValue{}
}
