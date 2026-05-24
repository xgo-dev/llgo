package gotest

import "testing"

type typeAssertInterface interface {
	Get() int
}

type typeAssertScopedT struct{}

var typeAssertScopedValue any

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

func TestInterfaceAssertRejectsSameNameTypesFromDifferentScopes(t *testing.T) {
	typeAssertAssignLocalT()
	typeAssertLocalToLocalT(t)
	typeAssertLocalToPackageT(t)

	typeAssertScopedValue = typeAssertScopedT{}
	typeAssertLocalToLocalT(t)
}

func typeAssertAssignLocalT() {
	type typeAssertScopedT struct{}
	typeAssertScopedValue = typeAssertScopedT{}
}

func typeAssertLocalToLocalT(t *testing.T) {
	type typeAssertScopedT struct{}
	expectPanicContaining(t, "different scopes", func() {
		_ = typeAssertScopedValue.(typeAssertScopedT)
	})
}

func typeAssertLocalToPackageT(t *testing.T) {
	expectPanicContaining(t, "different scopes", func() {
		_ = typeAssertScopedValue.(typeAssertScopedT)
	})
}
