//go:build llgo && goexperiment.simd && (amd64 || arm64)

package simd_test

import (
	"fmt"
	"reflect"
	"simd/archsimd"
	"strings"
	"testing"
)

type reflectZeroVectorTail struct {
	X   uint64
	Pad [0]archsimd.Float32x4
}

func TestSIMDNativeReflectBoundary(t *testing.T) {
	cases := []struct {
		name  string
		fn    any
		args  []reflect.Value
		slice bool
	}{
		{"vector-arg", func(archsimd.Float32x4) {}, []reflect.Value{reflect.ValueOf(archsimd.Float32x4{})}, false},
		{"zero-tail-arg", func(reflectZeroVectorTail) {}, []reflect.Value{reflect.ValueOf(reflectZeroVectorTail{X: 12345})}, false},
		{"nested-arg", func([1]reflectZeroVectorTail) {}, []reflect.Value{reflect.ValueOf([1]reflectZeroVectorTail{})}, false},
		{"vector-result", func() archsimd.Float32x4 { return archsimd.Float32x4{} }, nil, false},
		{"zero-tail-result", func() reflectZeroVectorTail { return reflectZeroVectorTail{} }, nil, false},
		{"multi-result", func() (int, reflectZeroVectorTail) { return 1, reflectZeroVectorTail{} }, nil, false},
		{"call-slice", func(reflectZeroVectorTail, ...int) {}, []reflect.Value{reflect.ValueOf(reflectZeroVectorTail{}), reflect.ValueOf([]int{1})}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertRejected := func(f func()) {
				t.Helper()
				defer func() {
					if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "reflect: libffi calls with vector storage are not supported") {
						t.Errorf("unsupported vector ABI diagnostic: %v", r)
					}
				}()
				f()
			}
			fn := reflect.ValueOf(tc.fn)
			assertRejected(func() {
				if tc.slice {
					fn.CallSlice(tc.args)
				} else {
					fn.Call(tc.args)
				}
			})
			assertRejected(func() {
				reflect.MakeFunc(fn.Type(), func([]reflect.Value) []reflect.Value { t.Fatal("callback reached"); return nil })
			})
		})
	}
}

func TestSIMDNativeReflectScalarABI(t *testing.T) {
	v := archsimd.Float32x4{}
	fn := reflect.ValueOf(func(p *archsimd.Float32x4, s []archsimd.Float32x4) int {
		if p == nil {
			return -1
		}
		return len(s)
	})
	args := []reflect.Value{reflect.ValueOf(&v), reflect.ValueOf([]archsimd.Float32x4{v})}
	for _, f := range []reflect.Value{fn, reflect.MakeFunc(fn.Type(), func(in []reflect.Value) []reflect.Value { return []reflect.Value{reflect.ValueOf(in[1].Len())} })} {
		if got := f.Call(args)[0].Int(); got != 1 {
			t.Fatal(got)
		}
	}
	type scalarTail struct {
		X   uint64
		Pad [0]byte
	}
	scalar := reflect.ValueOf(func(x scalarTail) uint64 { return x.X })
	if got := scalar.Call([]reflect.Value{reflect.ValueOf(scalarTail{X: 12345})})[0].Uint(); got != 12345 {
		t.Fatal(got)
	}
	variadic := reflect.ValueOf(func(v ...archsimd.Float32x4) int { return len(v) })
	if got := variadic.CallSlice([]reflect.Value{args[1]})[0].Int(); got != 1 {
		t.Fatal(got)
	}
}
