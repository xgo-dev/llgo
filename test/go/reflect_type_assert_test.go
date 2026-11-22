package gotest

import (
	"reflect"
	"testing"
)

type reflectAssertNumber int

func (n reflectAssertNumber) Number() int { return int(n) }

type reflectAssertNumberer interface{ Number() int }

func TestReflectTypeAssert(t *testing.T) {
	t.Run("primitive", func(t *testing.T) {
		n := 42
		for _, v := range []reflect.Value{reflect.ValueOf(n), reflect.ValueOf(&n).Elem()} {
			if got, ok := reflect.TypeAssert[int](v); !ok || got != n {
				t.Fatalf("TypeAssert = (%d, %v), want (%d, true)", got, ok, n)
			}
		}
		if got, ok := reflect.TypeAssert[*int](reflect.ValueOf(&n)); !ok || got != &n {
			t.Fatal("TypeAssert lost pointer identity")
		}
	})
	t.Run("function value", func(t *testing.T) {
		assert := reflect.TypeAssert[int]
		var typed func(reflect.Value) (int, bool) = reflect.TypeAssert[int]
		for _, fn := range []func(reflect.Value) (int, bool){assert, typed} {
			if got, ok := fn(reflect.ValueOf(42)); !ok || got != 42 {
				t.Fatalf("TypeAssert function = (%d, %v), want (42, true)", got, ok)
			}
		}
	})
	t.Run("closure", func(t *testing.T) {
		offset := 7
		closure := func(n int) int { return n + offset }
		fn, ok := reflect.TypeAssert[func(int) int](reflect.ValueOf(closure))
		if !ok || fn == nil || fn(5) != 12 {
			t.Fatal("TypeAssert lost closure state")
		}
		var nilFunc func(int) int
		if fn, ok := reflect.TypeAssert[func(int) int](reflect.ValueOf(nilFunc)); !ok || fn != nil {
			t.Fatal("TypeAssert did not preserve a typed nil function")
		}
	})
	t.Run("interface", func(t *testing.T) {
		n := reflectAssertNumber(9)
		var boxed any = n
		for _, v := range []reflect.Value{reflect.ValueOf(n), reflect.ValueOf(&boxed).Elem()} {
			if got, ok := reflect.TypeAssert[reflectAssertNumber](v); !ok || got != n {
				t.Fatal("TypeAssert did not preserve the concrete type")
			}
			if got, ok := reflect.TypeAssert[reflectAssertNumberer](v); !ok || got == nil || got.Number() != 9 {
				t.Fatal("TypeAssert did not construct the interface method table")
			}
			if got, ok := reflect.TypeAssert[any](v); !ok || got != boxed {
				t.Fatal("TypeAssert did not preserve the dynamic value")
			}
		}
	})
	t.Run("nil interface", func(t *testing.T) {
		var boxed any
		if got, ok := reflect.TypeAssert[any](reflect.ValueOf(&boxed).Elem()); ok || got != nil {
			t.Fatal("asserting a nil interface must fail")
		}
		var numberer reflectAssertNumberer
		if got, ok := reflect.TypeAssert[reflectAssertNumberer](reflect.ValueOf(&numberer).Elem()); ok || got != nil {
			t.Fatal("asserting a nil non-empty interface must fail")
		}
		boxed = (*int)(nil)
		if got, ok := reflect.TypeAssert[*int](reflect.ValueOf(&boxed).Elem()); !ok || got != nil {
			t.Fatal("asserting an interface containing a typed nil must succeed")
		}
	})
	t.Run("method", func(t *testing.T) {
		method := reflect.ValueOf(reflectAssertNumber(9)).MethodByName("Number")
		for _, v := range []reflect.Value{method, reflect.ValueOf(method.Interface())} {
			fn, ok := reflect.TypeAssert[func() int](v)
			if !ok || fn == nil || fn() != 9 {
				t.Fatal("TypeAssert lost the bound method receiver")
			}
		}
	})
	t.Run("mismatch", func(t *testing.T) {
		v := reflect.ValueOf(reflectAssertNumber(9))
		if got, ok := reflect.TypeAssert[int](v); ok || got != 0 {
			t.Fatal("TypeAssert accepted a different named type")
		}
		if got, ok := reflect.TypeAssert[error](v); ok || got != nil {
			t.Fatal("TypeAssert accepted an unimplemented interface")
		}
	})
	t.Run("invalid", func(t *testing.T) {
		defer func() {
			err, ok := recover().(*reflect.ValueError)
			if !ok || err.Method != "reflect.TypeAssert" || err.Kind != reflect.Invalid {
				t.Fatalf("invalid Value panic = %#v, want reflect.TypeAssert ValueError", err)
			}
		}()
		reflect.TypeAssert[int](reflect.Value{})
	})
}
