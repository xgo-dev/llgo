package wasmtest

import (
	"reflect"
	"runtime"
	"testing"
)

// Reflection's branch-dependent by-value wrappers copy this aggregate even
// though it is below the separate 64 KiB indirect-return ABI threshold.
type gcReflectionAggregate struct {
	pointer *int
	data    [8192]byte
}

//go:noinline
func (value gcReflectionAggregate) Check() int {
	collectGCAggregate()
	return *value.pointer + int(value.data[0]) + int(value.data[len(value.data)-1])
}

//go:noinline
func callGCReflectionDeferred(value gcReflectionAggregate, callback func(gcReflectionAggregate)) {
	// The indirect deferred call loads its aggregate argument from another
	// aggregate snapshot. Mutating the source must not change that argument.
	defer callback(value)
	value.data[0], value.data[len(value.data)-1] = 0, 0
	collectGCAggregate()
}

func TestLargeReflectionGCRoots(t *testing.T) {
	value := gcReflectionAggregate{pointer: new(int)}
	*value.pointer = 101
	value.data[0], value.data[len(value.data)-1] = 11, 97
	const want = 101 + 11 + 97
	for i := 0; i < 3; i++ {
		method := reflect.ValueOf(value).Method(0)
		function := reflect.ValueOf(gcReflectionAggregate.Check)
		runtime.GC()
		for _, got := range []int{
			value.Check(),
			int(method.Call(nil)[0].Int()),
			int(function.Call([]reflect.Value{reflect.ValueOf(value)})[0].Int()),
		} {
			if got != want {
				t.Fatalf("reflection snapshot = %d, want %d", got, want)
			}
		}
		called := false
		callGCReflectionDeferred(value, func(snapshot gcReflectionAggregate) {
			called = true
			if got := snapshot.Check(); got != want {
				t.Fatalf("deferred aggregate snapshot = %d, want %d", got, want)
			}
		})
		if !called {
			t.Fatal("deferred aggregate callback did not run")
		}
	}
}
