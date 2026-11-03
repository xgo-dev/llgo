//go:build llgo && wasm && (wasip1 || (js && !llgo.wasm.reflect.libffi))

package gotest

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

//llgo:cdecl
type reflectNativeMakeFunc func(uintptr) uintptr

func TestReflectWasmNativeMakeFuncRequiresFFI(t *testing.T) {
	defer func() {
		message := fmt.Sprint(recover())
		if !strings.Contains(message, "native function") || !strings.Contains(message, "provider") {
			t.Fatalf("native MakeFunc panic = %q, want explicit unsupported provider diagnostic", message)
		}
	}()
	_ = reflect.MakeFunc(reflect.TypeOf(reflectNativeMakeFunc(nil)), func([]reflect.Value) []reflect.Value {
		return nil
	})
	t.Fatal("non-FFI Wasm provider accepted a native MakeFunc without a native entry bridge")
}
