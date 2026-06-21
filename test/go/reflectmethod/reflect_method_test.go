package reflectmethod

import (
	"reflect"
	"testing"
)

type target int

func (m target) Alpha(delta int) int {
	return int(m) + delta
}

func (m target) Beta() string {
	return "beta"
}

type pointerTarget struct {
	n int
}

func (m *pointerTarget) Bump(delta int) int {
	m.n += delta
	return m.n
}

type methodIndexLookup interface {
	Method(int) reflect.Method
}

type methodNameLookup interface {
	MethodByName(string) (reflect.Method, bool)
}

func TestReflectTypeMethodFuncInvocation(t *testing.T) {
	typ := reflect.TypeOf(target(10))
	if got, want := typ.NumMethod(), 2; got != want {
		t.Fatalf("NumMethod() = %d, want %d", got, want)
	}

	method := typ.Method(0)
	if got, want := method.Name, "Alpha"; got != want {
		t.Fatalf("Method(0).Name = %q, want %q", got, want)
	}
	fn := method.Func.Interface().(func(target, int) int)
	if got, want := fn(10, 5), 15; got != want {
		t.Fatalf("Method(0).Func call = %d, want %d", got, want)
	}

	methodByName, ok := typ.MethodByName("Beta")
	if !ok {
		t.Fatal("MethodByName(Beta) returned ok=false")
	}
	beta := methodByName.Func.Interface().(func(target) string)
	if got, want := beta(10), "beta"; got != want {
		t.Fatalf("MethodByName(Beta).Func call = %q, want %q", got, want)
	}
}

func TestReflectTypeMethodExpressionRetention(t *testing.T) {
	typ := reflect.TypeOf(target(7))

	method := reflect.Type.Method(typ, 0)
	alpha := method.Func.Interface().(func(target, int) int)
	if got, want := alpha(7, 3), 10; got != want {
		t.Fatalf("reflect.Type.Method Func call = %d, want %d", got, want)
	}

	methodByName := reflect.Type.MethodByName
	betaMethod, ok := methodByName(typ, "Beta")
	if !ok {
		t.Fatal("reflect.Type.MethodByName returned ok=false")
	}
	beta := betaMethod.Func.Interface().(func(target) string)
	if got, want := beta(7), "beta"; got != want {
		t.Fatalf("reflect.Type.MethodByName Func call = %q, want %q", got, want)
	}
}

func TestReflectTypeMethodFuncInterfaceDispatch(t *testing.T) {
	typ := reflect.TypeOf(target(11))

	var byIndex methodIndexLookup = typ
	method := byIndex.Method(0)
	alpha := method.Func.Interface().(func(target, int) int)
	if got, want := alpha(11, 4), 15; got != want {
		t.Fatalf("interface Method Func call = %d, want %d", got, want)
	}

	var byName methodNameLookup = typ
	betaMethod, ok := byName.MethodByName("Beta")
	if !ok {
		t.Fatal("interface MethodByName returned ok=false")
	}
	beta := betaMethod.Func.Interface().(func(target) string)
	if got, want := beta(11), "beta"; got != want {
		t.Fatalf("interface MethodByName Func call = %q, want %q", got, want)
	}
}

func TestReflectPointerReceiverMethodFuncInvocation(t *testing.T) {
	target := &pointerTarget{n: 4}
	method, ok := reflect.TypeOf(target).MethodByName("Bump")
	if !ok {
		t.Fatal("MethodByName(Bump) returned ok=false")
	}
	bump := method.Func.Interface().(func(*pointerTarget, int) int)
	if got, want := bump(target, 6), 10; got != want {
		t.Fatalf("pointer receiver Method.Func call = %d, want %d", got, want)
	}
	if got, want := target.n, 10; got != want {
		t.Fatalf("pointer receiver side effect = %d, want %d", got, want)
	}
}
