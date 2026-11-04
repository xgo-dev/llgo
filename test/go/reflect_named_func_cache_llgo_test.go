//go:build llgo

package gotest

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

// Start from *T rather than a function-valued TypeOf. The reverse lookup must
// preserve the compiler's canonical Func descriptor, even when many workers
// first request its pointer, allocation, and interface identities together.
func TestReflectNamedFuncReverseConcurrentPublication(t *testing.T) {
	for round := 0; round < 20; round++ {
		canonical := abi.TypeFor[reflectColdNamedFunc]()
		const workers = 32
		start := make(chan struct{})
		results := make(chan reflect.Type, workers)
		for i := 0; i < workers; i++ {
			go func() {
				<-start
				pointer := reflect.TypeOf((*reflectColdNamedFunc)(nil))
				function := pointer.Elem()
				value := reflect.New(function)
				data := (*[2]unsafe.Pointer)(unsafe.Pointer(&function))[1]
				if data != unsafe.Pointer(canonical) || function.Kind() != reflect.Func ||
					function.Size() != unsafe.Sizeof(uintptr(0)) ||
					value.Type() != pointer || value.Elem().Type() != function ||
					reflect.PointerTo(function) != pointer ||
					reflect.TypeOf(value.Elem().Interface()) != function {
					results <- nil
					return
				}
				if _, ok := value.Interface().(interface{ Marker() }); !ok {
					results <- nil
					return
				}
				results <- function
			}()
		}
		close(start)
		want := <-results
		if want == nil {
			t.Fatal("reverse lookup lost the canonical named function descriptor")
		}
		for i := 1; i < workers; i++ {
			if got := <-results; got != want {
				t.Fatalf("round %d: competing named function types: %v != %v", round, got, want)
			}
		}
	}
}
