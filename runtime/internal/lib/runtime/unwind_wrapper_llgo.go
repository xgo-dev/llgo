//go:build !baremetal && !wasm

package runtime

func pcSymbolIsWrapper(sym pcSymbol) bool {
	return uint32(sym.startLine)&runtimeFuncInfoLineWrapper != 0
}

// elideWrapperCalling mirrors Go's runtime.elideWrapperCalling. A generated
// wrapper normally has no logical stack frame, but must remain visible when
// its own receiver/conversion check called a panic helper instead of the
// wrapped method.
func elideWrapperCalling(callee string) bool {
	switch callee {
	case "runtime.gopanic", "runtime.sigpanic", "runtime.panicwrap",
		"github.com/xgo-dev/llgo/runtime/internal/runtime.PanicWrapNilPointer":
		return false
	}
	return true
}

// The live FP walk needs only one flag per physical frame. Resolve a name
// only for the callee of an actual wrapper (the panic-helper exception).
func elideWrapperPC(pc, callee uintptr) bool {
	i := runtimeFuncPCFrameIndex(pc)
	if i < 0 {
		return false
	}
	var index uint32
	if runtimeFuncPCFramesPrebuilt {
		index = prebuiltFrame(i).funcIndex
	} else {
		index = runtimeFuncPCFrames[i].funcIndex
	}
	if index == 0 || uintptr(index) > runtimeFuncInfoCount || funcInfoAt(uintptr(index)-1).line&runtimeFuncInfoLineWrapper == 0 {
		return false
	}
	return elideWrapperCalling(frameSymbol(callee).function)
}

// Leave the interrupted instruction visible. Only callers of inner frames can
// represent forwarding wrappers. FaultPCs retain their original machine order
// for native symbolization, so skip wrappers when consuming that snapshot.
func elideFaultWrapper(pcs []uintptr, i int) bool {
	return i > 0 && prebuiltTextContains(pcs[i]-1) && elideWrapperPC(pcs[i]-1, pcs[i-1]-1)
}
