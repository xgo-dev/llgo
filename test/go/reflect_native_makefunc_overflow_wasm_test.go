//go:build llgo && js && wasm && llgo.wasm.reflect.libffi && !llgo.wasm.emscripten.memory64

package gotest

import (
	"reflect"
	"testing"
)

func TestReflectNativeMakeFuncWordOverflow(t *testing.T) {
	for _, tt := range []struct {
		name string
		call func()
	}{
		{"argument", func() {
			newReflectNativeMakeFunc().Call([]reflect.Value{reflect.ValueOf(uintptr(1) << 32)})
		}},
		{"result", func() {
			fn := reflect.MakeFunc(reflect.TypeOf(reflectNativeMakeFunc(nil)), func([]reflect.Value) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(uintptr(1) << 32)}
			}).Interface().(reflectNativeMakeFunc)
			fn(0)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("out-of-range native uintptr conversion did not panic")
				}
			}()
			tt.call()
		})
	}
}
