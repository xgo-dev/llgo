package gotest

import (
	"reflect"
	"testing"
)

type reflectMethodIdentityPtr struct{}

func (*reflectMethodIdentityPtr) Ptr() int { return 7 }

type reflectMethodIdentityValue int

func (reflectMethodIdentityValue) Value() int { return 9 }

func TestReflectTypeMethodFuncInterfaceTypeIdentity(t *testing.T) {
	ptrMethod := reflect.TypeOf(&reflectMethodIdentityPtr{}).Method(0)
	ptrFn, ok := ptrMethod.Func.Interface().(func(*reflectMethodIdentityPtr) int)
	if !ok {
		t.Fatalf("Method.Func.Interface() has type %T, want func(*reflectMethodIdentityPtr) int", ptrMethod.Func.Interface())
	}
	if got := ptrFn(&reflectMethodIdentityPtr{}); got != 7 {
		t.Fatalf("pointer method func returned %d, want 7", got)
	}

	valueMethod, ok := reflect.TypeOf(reflectMethodIdentityValue(0)).MethodByName("Value")
	if !ok {
		t.Fatal("MethodByName did not find Value")
	}
	valueFn, ok := valueMethod.Func.Interface().(func(reflectMethodIdentityValue) int)
	if !ok {
		t.Fatalf("MethodByName.Func.Interface() has type %T, want func(reflectMethodIdentityValue) int", valueMethod.Func.Interface())
	}
	if got := valueFn(reflectMethodIdentityValue(0)); got != 9 {
		t.Fatalf("value method func returned %d, want 9", got)
	}
}
