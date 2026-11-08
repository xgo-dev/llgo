package gotest

import (
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
)

type keepAliveStackHolder struct {
	object *[10]int
	unused int
}

//go:noinline
func useKeepAliveStackHolder(*keepAliveStackHolder) {}

// This is a behavioral smoke test: conservative GC may also retain stale
// pointers. cl/keepalive_test.go checks the optimized barrier deterministically.
func TestRuntimeKeepAliveStackObjectGraph(t *testing.T) {
	for i := 0; i < 8; i++ {
		var finalized atomic.Bool
		x := new([10]int)
		runtime.SetFinalizer(x, func(*[10]int) { finalized.Store(true) })
		holder := &keepAliveStackHolder{x, 0}
		useKeepAliveStackHolder(holder)
		runtime.GC()
		runtime.GC()
		runtime.GC()
		if finalized.Load() {
			t.Fatal("finalizer ran while reachable through a live stack object")
		}
		runtime.KeepAlive(holder)
	}
}

func TestRuntimeKeepAliveNilAndScalars(t *testing.T) {
	for _, value := range []any{nil, (*int)(nil), 42, "alive", [2]int{1, 2}} {
		runtime.KeepAlive(value)
	}
}

//go:noinline
func callKeepAliveValue(keep func(any), value any) {
	keep(value)
}

func TestRuntimeKeepAliveFunctionValue(t *testing.T) {
	keep := runtime.KeepAlive
	value := new([10]int)
	callKeepAliveValue(keep, value)
	function := reflect.ValueOf(keep)
	if function.Pointer() == 0 {
		t.Fatal("KeepAlive function value has no code address")
	}
	function.Call([]reflect.Value{reflect.ValueOf(value)})
}
