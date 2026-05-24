package gotest

import (
	"runtime"
	"strings"
	"testing"
)

type nilPanicRuntimeInterface interface {
	M()
}

type nilPanicRuntimeLargeStruct struct {
	pad [2 << 20]byte
	i   int
}

func TestNilPanicValuesAreRuntimeErrors(t *testing.T) {
	tests := []struct {
		name string
		f    func()
	}{
		{
			name: "generic nil interface method",
			f: func() {
				nilPanicRuntimeCall[nilPanicRuntimeInterface](nil)
			},
		},
		{
			name: "nil interface method expression",
			f: func() {
				nilPanicRuntimeMethodExpression[int]()
			},
		},
		{
			name: "array pointer slice",
			f: func() {
				var p *[4]int
				_ = p[:]
			},
		},
		{
			name: "array element pointer load",
			f: func() {
				var m [2]*int
				_ = *m[1]
			},
		},
		{
			name: "large field from call",
			f: func() {
				_ = nilPanicRuntimeLargeStructPtr().i
			},
		},
		{
			name: "large field from parameter",
			f: func() {
				_ = nilPanicRuntimeLargeField(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := expectRuntimeErrorPanic(t, tt.f)
			if !strings.Contains(err.Error(), "nil pointer dereference") {
				t.Fatalf("panic = %q, want nil pointer dereference", err.Error())
			}
		})
	}
}

func nilPanicRuntimeCall[P nilPanicRuntimeInterface](p P) {
	p.M()
}

func nilPanicRuntimeMethodExpression[T any]() {
	interface{ M() T }.M(nil)
}

func nilPanicRuntimeLargeStructPtr() *nilPanicRuntimeLargeStruct {
	return nil
}

func nilPanicRuntimeLargeField(p *nilPanicRuntimeLargeStruct) int {
	return p.i
}

func expectRuntimeErrorPanic(t *testing.T, f func()) runtime.Error {
	t.Helper()
	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()
		f()
	}()
	if recovered == nil {
		t.Fatal("expected panic")
	}
	err, ok := recovered.(runtime.Error)
	if !ok {
		t.Fatalf("panic type = %T, want runtime.Error", recovered)
	}
	return err
}
