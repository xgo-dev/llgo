//go:build !baremetal && !wasm

package runtime

import (
	"unsafe"

	latomic "github.com/goplus/llgo/runtime/internal/lib/sync/atomic"
	rtdebug "github.com/goplus/llgo/runtime/internal/runtime"
)

//go:linkname c_framepointer C.llgo_framepointer
func c_framepointer() unsafe.Pointer

func init() {
	rtdebug.PanicTraceback = panicTraceback
	rtdebug.MemProfileStackCapture = captureMemProfileStack
	rtdebug.MemProfileRatePtr = &MemProfileRate
}

// captureMemProfileStack walks the physical stack at a sampled allocation.
// The leading allocator plumbing (AllocZ/AllocU, this capture path) is
// trimmed at read time by the MemProfile wrapper, where symbolization is
// safe and cached.
func captureMemProfileStack(pcs []uintptr) int {
	if !fpUnwindAvailable() {
		return 0
	}
	// The frame-table init itself allocates; when such an allocation is
	// sampled, walking from here would wait on the init latch this very
	// thread holds (fpCallers -> init busy -> usleep forever — observed
	// hanging whole net/rpc and net/http test binaries at the first
	// testing.callerName). Drop the sample instead of waiting. Uninit is
	// fine to enter: memProfileInSample is set for the whole sample, so
	// the init's own allocations cannot re-sample.
	if latomic.LoadUint32(&runtimeFuncPCInitState) == runtimeFuncInfoInitBusy {
		return 0
	}
	return fpCallers(0, pcs)
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// panicTraceback prints a Go-style stack trace for an unrecovered panic:
// one "function(...)" line plus an indented file:line per physical frame,
// matching the shape of runtime.Stack and gc's panic output. Reports false
// (caller falls back to the clite dladdr dump) when the FP walk or the
// tables are unavailable.
func panicTraceback(skip int) bool {
	if !fpUnwindAvailable() {
		return false
	}
	var pcs [64]uintptr
	n := fpCallers(skip, pcs[:])
	if n <= 0 {
		return false
	}
	print("goroutine 1 [running]:\n")
	frames := CallersFrames(pcs[:n])
	skippingPlumbing := true
	for {
		frame, more := frames.Next()
		name := frame.Function
		if name == "" {
			name = unknownFunctionName(frame.PC)
		}
		// The frames between the hook and the panic site are runtime
		// plumbing (Rethrow, Panic, ...); their depth varies by panic
		// path, so filter by package rather than a fixed skip.
		if skippingPlumbing {
			if hasPrefix(name, "github.com/goplus/llgo/runtime/internal/") {
				if more {
					continue
				}
				break
			}
			skippingPlumbing = false
		}
		print(name, "(...)\n\t")
		if frame.File == "" {
			print("???")
		} else {
			print(frame.File)
		}
		print(":", frame.Line)
		// gc appends the frame pc's offset from the function entry; the
		// value is codegen-specific, only the format matches.
		if frame.Entry != 0 && frame.PC >= frame.Entry {
			print(" +0x", string(appendHexUint(nil, uintptr(frame.PC-frame.Entry))))
		}
		print("\n")
		if !more {
			break
		}
	}
	return true
}

// maxFPStride bounds how far up the stack one frame may sit from the next.
// A slot whose decoded parent is further away than any plausible frame is a
// corrupt chain, not a giant frame; stop rather than walk off the stack.
const maxFPStride = 1 << 20

// fpCallers walks the frame-pointer chain and fills pc with return
// addresses, Go-style: pc[0] is the return address in the frame `skip`
// levels above the caller of fpCallers. Every LLGo-compiled function keeps
// x29/rbp chained ("frame-pointer"="non-leaf" is set on all Go functions),
// so unlike the shadow stack this sees every physical frame; the walk stops
// at the first frame that breaks the chain discipline (e.g. foreign C code
// compiled without frame pointers).
//
// The clite walker (runtime/internal/clite/debug/_wrap/debug.c
// llgo_stacktrace) implements the same chain discipline and guards for the
// pre-table paths (unrecovered-panic dump, last-resort Callers fallback);
// keep the two in sync when changing the walk rules.
//
//go:noinline
func fpCallers(skip int, pc []uintptr) int {
	if len(pc) == 0 {
		return 0
	}
	// The walk bound needs the frame table's text range; make sure it is
	// built (no-op when the prebuilt table was adopted at startup).
	initRuntimeFuncPCFrames()
	fp := uintptr(c_framepointer())
	n := 0
	// The helper's saved chain starts at our own frame; skip fpCallers
	// itself so skip counting matches the caller's view.
	skip++
	const maxFrames = 4096
	for i := 0; fp != 0 && n < len(pc) && i < maxFrames; i++ {
		prev := *(*uintptr)(unsafe.Pointer(fp))
		ret := *(*uintptr)(unsafe.Pointer(fp + unsafe.Sizeof(uintptr(0))))
		if ret < minLegalPC {
			break
		}
		// Beyond main the chain runs into libc frames without FP
		// discipline; their slots decode as wild pcs that nearest-below
		// symbolization would map to arbitrary functions. Bound the walk
		// to the program's own text (Go tracebacks stop at runtime.main
		// for the same reason).
		if !prebuiltTextContains(ret) {
			break
		}
		if skip > 0 {
			skip--
		} else {
			pc[n] = ret
			n++
		}
		// Stacks grow down, so the chain must strictly increase; bound the
		// stride so a corrupt slot cannot walk off the stack.
		if prev <= fp || prev-fp > maxFPStride || prev&(unsafe.Sizeof(uintptr(0))-1) != 0 {
			break
		}
		fp = prev
	}
	return n
}

// runtimeFPChain is emitted next to the funcinfo table (one per binary,
// internal/build emitFuncInfoTable) and records whether this binary's Go
// functions were compiled with the frame-pointer attribute
// (ssa.Program.NeedsFramePointer).
//
//go:linkname runtimeFPChain __llgo_fp_chain
var runtimeFPChain uint8

// fpUnwindAvailable reports whether the physical walk can be used for the
// public stack APIs: the compiler declared the FP chain intact for this
// binary, and the funcinfo tables are present (without them symbolization
// would fall back to dlsym anyway).
func fpUnwindAvailable() bool {
	return runtimeFPChain != 0 && runtimeFuncInfoTable != nil && runtimeFuncInfoCount > 0
}
