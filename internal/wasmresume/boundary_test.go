package wasmresume

import "testing"

func TestRuntimeBoundaries(t *testing.T) {
	for _, name := range []string{
		runtimeResumePrefix + "Context.Run",
		runtimeResumePrefix + "Context.AllocateFrame",
	} {
		if !IsRuntimeABIImplementation(name) || !IsNonSuspendingBoundary(name) {
			t.Fatalf("%q is not a non-suspending ABI implementation", name)
		}
	}
	if !IsRuntimeABIImplementation(SuspendSymbol) {
		t.Fatal("SuspendCurrent is not recognized as an ABI implementation")
	}
	if IsNonSuspendingBoundary(SuspendSymbol) {
		t.Fatal("SuspendCurrent was classified as non-suspending")
	}
	for _, name := range []string{
		runtimeAllocRoot,
		runtimeFreeRoot,
		runtimeRunWasmMain,
		runtimeFrameAlloc,
		runtimeDynamicAlloc,
		runtimeFrameFree,
		runtimeFrameClose,
		"github.com/goplus/llgo/runtime/internal/runtime.GetThreadDefer",
		"github.com/goplus/llgo/runtime/internal/runtime.SetThreadDefer",
		"github.com/goplus/llgo/runtime/internal/runtime.Panic",
		"github.com/goplus/llgo/runtime/internal/runtime.Recover",
		"github.com/goplus/llgo/runtime/internal/runtime.Rethrow",
		"github.com/goplus/llgo/runtime/internal/runtime.Goexit",
		"runtime.Goexit",
	} {
		if !IsNonSuspendingBoundary(name) {
			t.Fatalf("%q is not a non-suspending boundary", name)
		}
	}
	if IsRuntimeABIImplementation("example.com/p.Run") ||
		IsNonSuspendingBoundary("example.com/p.Run") {
		t.Fatal("ordinary Go function was classified as a runtime boundary")
	}
}
