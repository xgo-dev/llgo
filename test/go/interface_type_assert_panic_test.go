package gotest

import "testing"

type typeAssertInterface interface {
	Get() int
}

func typeAssertValue(v any) any {
	return v
}

func TestInterfaceAssertToInterfacePanicsWithRuntimeError(t *testing.T) {
	expectPanicContaining(t, "interface conversion", func() {
		_ = typeAssertValue(0).(typeAssertInterface)
	})
}

func TestInterfaceAssertToConcretePanicsWithRuntimeError(t *testing.T) {
	expectPanicContaining(t, "interface conversion", func() {
		_ = typeAssertValue(0).(string)
	})
}
