//go:build llgo && wasm

package wasmtest

import (
	"reflect"
	"testing"
	"unsafe"
)

//go:linkname pushCallerFrame github.com/xgo-dev/llgo/runtime/internal/runtime.PushCallerLocationFrameWasm
func pushCallerFrame(entry uintptr, nameData *byte, nameLen int, fileData *byte, fileLen, line int) int

//go:linkname recordCallerLocation github.com/xgo-dev/llgo/runtime/internal/runtime.RecordCallerLocationWasm
func recordCallerLocation(entry uintptr, nameData *byte, nameLen int, fileData *byte, fileLen, line int)

//go:linkname recordPanicLocation github.com/xgo-dev/llgo/runtime/internal/runtime.RecordPanicLocationWasm
func recordPanicLocation(entry uintptr, nameData *byte, nameLen int, fileData *byte, fileLen, line int)

//go:linkname popCallerFrame github.com/xgo-dev/llgo/runtime/internal/runtime.PopCallerLocationFrame
func popCallerFrame(mark int)

//go:linkname pushCallerFrameDirect github.com/xgo-dev/llgo/runtime/internal/runtime.PushCallerLocationFrame
func pushCallerFrameDirect(entry uintptr, name, file string, line int) int

//go:linkname recordCallerLocationDirect github.com/xgo-dev/llgo/runtime/internal/runtime.RecordCallerLocation
func recordCallerLocationDirect(entry uintptr, name, file string, line int)

//go:linkname recordPanicLocationDirect github.com/xgo-dev/llgo/runtime/internal/runtime.RecordPanicLocation
func recordPanicLocationDirect(entry uintptr, name, file string, line int)

//go:noinline
func callerAllocationFunction() {}

func TestCallerInstrumentationAllocations(t *testing.T) {
	const name, file = "wasmtest.callerAllocationFunction", "caller_alloc_test.go"
	entry := reflect.ValueOf(callerAllocationFunction).Pointer()
	nameData, fileData := unsafe.StringData(name), unsafe.StringData(file)
	// AllocsPerRun warms up the shadow stack and frame registry. The underlying
	// push still allocates a frame snapshot, so compare the same operations
	// without wrappers: reconstructing string values must add no allocations.
	want := testing.AllocsPerRun(100, func() {
		mark := pushCallerFrameDirect(entry, name, file, 1)
		recordCallerLocationDirect(entry, name, file, 2)
		recordPanicLocationDirect(entry, name, file, 3)
		popCallerFrame(mark)
	})
	if got := testing.AllocsPerRun(100, func() {
		mark := pushCallerFrame(entry, nameData, len(name), fileData, len(file), 1)
		recordCallerLocation(entry, nameData, len(name), fileData, len(file), 2)
		recordPanicLocation(entry, nameData, len(name), fileData, len(file), 3)
		popCallerFrame(mark)
	}); got != want {
		t.Fatalf("caller metadata wrappers allocate %v objects, underlying helpers allocate %v", got, want)
	}
}
