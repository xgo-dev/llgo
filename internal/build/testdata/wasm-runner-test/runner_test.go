package wasmrunner

import (
	"os"
	"reflect"
	"runtime"
	"testing"
)

func TestRawWasmRunner(t *testing.T) {
	if runtime.GOARCH != "wasm" {
		t.Fatalf("GOARCH = %q, want wasm", runtime.GOARCH)
	}
	// Match Go's go_wasip1_wasm_exec contract: the default Wasmtime adapter
	// inherits PWD and PATH without exposing every host environment variable.
	if got := os.Getenv("PWD"); got == "" {
		t.Fatal("WASI runner did not inherit PWD")
	}
	done := make(chan struct{})
	go func() { close(done) }()
	<-done
}

func TestRawWasmReflectionBridge(t *testing.T) {
	type signature func(int64, func(int64) int64) int64
	value := reflect.MakeFunc(reflect.TypeOf(signature(nil)), func(in []reflect.Value) []reflect.Value {
		runtime.GC()
		result := in[1].Interface().(func(int64) int64)(in[0].Int())
		return []reflect.Value{reflect.ValueOf(result)}
	})
	callback := func(value int64) int64 { return value + 22 }
	if got := value.Interface().(signature)(20, callback); got != 42 {
		t.Fatalf("MakeFunc result = %d, want 42", got)
	}
	got := reflect.ValueOf(func(value int64) int64 { return value + 1 }).Call([]reflect.Value{reflect.ValueOf(int64(41))})
	if len(got) != 1 || got[0].Int() != 42 {
		t.Fatalf("Call result = %v, want 42", got)
	}
}
