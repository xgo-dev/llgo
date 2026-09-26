//go:build windows && (amd64 || arm64)

package runtime

import (
	"unsafe"

	rtdebug "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// Win64 frame registers are addressing bases, not conventional linked frame
// records: UNWIND_INFO may establish RBP/X29 at a non-zero offset into the
// frame. Use the platform unwinder, which consumes the same .pdata/.xdata
// records that the compiler emits for exception handling.

//go:linkname c_windowsCaptureContext C.llgo_windows_capture_context
func c_windowsCaptureContext(context *windowsFaultContext, pcOffset uintptr) unsafe.Pointer

//go:linkname c_windowsLookupFunctionEntry C.llgo_windows_lookup_function_entry
func c_windowsLookupFunctionEntry(pc uintptr, imageBase *uintptr) unsafe.Pointer

//go:linkname c_windowsVirtualUnwind C.llgo_windows_virtual_unwind
func c_windowsVirtualUnwind(imageBase, pc uintptr, functionEntry unsafe.Pointer, context *windowsFaultContext, establisherFrame *uintptr) unsafe.Pointer

const windowsFaultContextAlignment = 16

// Go only guarantees 8-byte alignment for the fields in windowsFaultContext,
// while Windows declares CONTEXT with 16-byte alignment. Reserve enough space
// to select a suitably aligned address before calling the platform unwinder.
type windowsFaultContextStorage [windowsFaultContextSize + windowsFaultContextAlignment - 1]byte

func (storage *windowsFaultContextStorage) context() *windowsFaultContext {
	base := unsafe.Pointer(&storage[0])
	offset := (-uintptr(base)) & (windowsFaultContextAlignment - 1)
	return (*windowsFaultContext)(unsafe.Add(base, offset))
}

func windowsUnwindOne(context *windowsFaultContext) bool {
	pc := context.pc()
	sp := context.sp()
	if pc < minLegalPC {
		return false
	}
	var imageBase uintptr
	entry := c_windowsLookupFunctionEntry(pc, &imageBase)
	if entry != nil {
		var frame uintptr
		c_windowsVirtualUnwind(imageBase, pc, entry, context, &frame)
		// Recursive calls can return to the same instruction in adjacent
		// frames. Advancing the stack still proves that the walk progressed.
		return context.pc() >= minLegalPC && (context.pc() != pc || context.sp() != sp)
	}

	// Leaf functions have no RUNTIME_FUNCTION entry. Simulate their return:
	// arm64 uses LR; amd64 takes the return pc from the current stack pointer.
	if lr := context.lr(); lr >= minLegalPC {
		context.setPC(lr)
		context.setLR(0)
		return true
	}
	wordSize := unsafe.Sizeof(uintptr(0))
	if sp&(wordSize-1) != 0 || !memReadable(sp) {
		return false
	}
	ret := *(*uintptr)(unsafe.Pointer(sp))
	if ret < minLegalPC {
		return false
	}
	context.setSP(sp + wordSize)
	context.setPC(ret)
	return true
}

func windowsContextCallers(context *windowsFaultContext, skip int, pc []uintptr, boundToGoText bool) int {
	n := 0
	var callee pcSymbol
	if current := context.pc(); current >= minLegalPC {
		callee = frameSymbol(current - 1)
	}
	for i := 0; n < len(pc) && i < maxPanicSpliceFrames; i++ {
		if !windowsUnwindOne(context) {
			break
		}
		ret := context.pc()
		if boundToGoText && !prebuiltTextContains(ret) {
			break
		}
		caller := frameSymbol(ret - 1)
		if pcSymbolIsWrapper(caller) && elideWrapperCalling(callee.function) {
			callee = caller
			continue
		}
		if skip > 0 {
			skip--
			callee = caller
			continue
		}
		pc[n] = ret
		n++
		// RtlVirtualUnwind can continue through the Windows CRT after the C
		// process entry. The entry is deliberately published as the logical
		// runtime.goexit frame, which is also the end of a Go caller chain.
		// Stop there instead of letting a nearby COFF function entry make the
		// CRT return address look like another Go frame.
		if boundToGoText && caller.function == "runtime.goexit" {
			break
		}
		callee = caller
	}
	return n
}

//go:noinline
func fpCallers(skip int, pc []uintptr) int {
	if len(pc) == 0 {
		return 0
	}
	initRuntimeFuncPCFrames()
	var storage windowsFaultContextStorage
	context := storage.context()
	if c_windowsCaptureContext(context, windowsFaultContextPCOffset) == nil {
		return 0
	}
	// The capture wrapper already unwound itself. The next unwind reaches
	// fpCallers' caller, matching runtime.Callers' public skip contract.
	return windowsContextCallers(context, skip, pc, true)
}

// Heap sampling runs from the allocator. Symbolizing each frame here can
// allocate or reenter the runtime, and is much slower than the OS unwind on
// Windows. Preserve raw PCs; MemProfile symbolizes them outside this path.
//
//go:noinline
func fpProfileCallers(pc []uintptr) int {
	if len(pc) == 0 {
		return 0
	}
	var storage windowsFaultContextStorage
	context := storage.context()
	if c_windowsCaptureContext(context, windowsFaultContextPCOffset) == nil {
		return 0
	}
	n := 0
	for i := 0; n < len(pc) && i < maxPanicSpliceFrames; i++ {
		if !windowsUnwindOne(context) {
			break
		}
		ret := context.pc()
		if !prebuiltTextContains(ret) {
			break
		}
		pc[n] = ret
		n++
	}
	return n
}

func platformFaultCallers(raw unsafe.Pointer, _ uintptr, pc []uintptr) int {
	// Keep the OS-owned exception record intact. Windows still owns it while
	// the vectored handler is active, even though LLGo leaves through its
	// non-local panic path rather than resuming the faulting instruction.
	var storage windowsFaultContextStorage
	context := storage.context()
	*context = *(*windowsFaultContext)(raw)
	if context.pc() == 0 {
		originPC := windowsFaultCallerPC(unsafe.Pointer(context))
		if originPC == 0 {
			return 0
		}
		// The snapshot already records originPC as its first frame. Resume
		// unwinding at the caller's call site so this walk begins with the
		// next frame and preserves the complete panic/recover caller chain.
		context.prepareNilCallUnwind(originPC)
	}
	// A fault can be the first operation that needs caller information. The
	// Go PC table is deliberately not initialized from the exception handler,
	// so do not use it to bound this OS-backed walk. RtlVirtualUnwind supplies
	// the structural bound; symbolization and the Go/C tail cut happen later
	// from an ordinary, allocation-safe context.
	return windowsContextCallers(context, 0, pc, false)
}

//go:noinline
func recoverFrameMarks() (uintptr, uintptr) {
	var storage windowsFaultContextStorage
	context := storage.context()
	if c_windowsCaptureContext(context, windowsFaultContextPCOffset) == nil {
		return 0, 0
	}
	// The capture wrapper already unwound itself. Walk recoverFrameMarks ->
	// recoverMark -> Recover -> the deferred function, then record both its
	// stack identity and function entry.
	for i := 0; i < 3; i++ {
		if !windowsUnwindOne(context) {
			return 0, 0
		}
	}
	entry := frameSymbol(context.pc() - 1).entry
	if entry == 0 {
		return 0, 0
	}
	return context.sp(), entry
}

// recoverFrameMarks relies on this function remaining a distinct frame in the
// Recover call chain.
//
//go:noinline
func recoverMark() {
	mark1, mark2 := recoverFrameMarks()
	if mark1 == 0 && mark2 == 0 {
		return
	}
	rtdebug.MarkPanicRecoverFPs(mark1, mark2)
}

//go:noinline
func recoverFrameLive(stack, entry uintptr) bool {
	if stack == 0 || entry == 0 {
		return false
	}
	var storage windowsFaultContextStorage
	context := storage.context()
	if c_windowsCaptureContext(context, windowsFaultContextPCOffset) == nil {
		return false
	}
	for i := 0; i < maxPanicSpliceFrames; i++ {
		if !windowsUnwindOne(context) {
			break
		}
		if context.sp() == stack && frameSymbol(context.pc()-1).entry == entry {
			return true
		}
	}
	return false
}

func panicSplicePCs() []uintptr {
	pcs := rtdebug.PanicPCs()
	if len(pcs) == 0 {
		return nil
	}
	if rtdebug.PanicActive() {
		return trimWindowsFaultPCs(pcs)
	}
	stack, entry := rtdebug.PanicRecoverFPs()
	if stack == 0 || entry == 0 || !recoverFrameLive(stack, entry) {
		return nil
	}
	return trimWindowsFaultPCs(pcs)
}
