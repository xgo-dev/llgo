//go:build llgo

package gotest

import (
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

// A public TypeOf already materializes the forward mapping. Exercise the
// reverse cache's cold construction path directly, without that precondition.
//
//go:linkname reflectCacheClosureOf reflect.closureOf
func reflectCacheClosureOf(*abi.FuncType) *abi.Type

//go:linkname reflectCacheToFuncType reflect.toFuncType
func reflectCacheToFuncType(*abi.StructType) *abi.FuncType

func TestReflectNamedFuncReverseConcurrentPublication(t *testing.T) {
	for round := 0; round < 20; round++ {
		ft := &abi.FuncType{Type: abi.Type{
			Kind_: uint8(abi.Func), TFlag: abi.TFlagNamed,
			Str_: "gotest.reverseCacheFunc", Size_: 2 * unsafe.Sizeof(uintptr(0)),
		}}
		const workers = 32
		start := make(chan struct{})
		results := make(chan *abi.Type, workers)
		for i := 0; i < workers; i++ {
			go func() {
				<-start
				ct := reflectCacheClosureOf(ft)
				if reflectCacheToFuncType(ct.StructType()) != ft {
					results <- nil
					return
				}
				results <- ct
			}()
		}
		close(start)
		want := <-results
		if want == nil {
			t.Fatal("reverse mapping was published without the canonical forward mapping")
		}
		for i := 1; i < workers; i++ {
			if got := <-results; got != want {
				t.Fatalf("round %d: competing named closure descriptors: %p != %p", round, got, want)
			}
		}
	}
}
