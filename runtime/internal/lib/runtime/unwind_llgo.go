//go:build !baremetal && !wasm

package runtime

import (
	"unsafe"

	rtdebug "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// c_framepointer returns its caller's frame pointer while the C helper frame
// is still alive.
//
//go:linkname c_framepointer C.llgo_framepointer
func c_framepointer() unsafe.Pointer

func init() {
	rtdebug.PanicTraceback = panicTraceback
	rtdebug.PanicRecovered = clearFaultTraceback
	rtdebug.PanicPCSnapshot = capturePanicPCs
	rtdebug.RecoverMark = recoverMark
}

// capturePanicPCs runs at panic time, before any longjmp unwinding, and
// stores the physical pc chain for later splicing (see spliceCallers).
func capturePanicPCs(v any) {
	if !rtdebug.SavePanicCallerFrames(v) || !fpUnwindAvailable() {
		return
	}
	var pcs [64]uintptr
	n := fpCallers(0, pcs[:])
	rtdebug.StorePanicPCs(pcs[:n])
}

const maxPanicSpliceFrames = 4096

// callerFramePointer returns the frame of its Go caller. llgo_framepointer
// returns this helper's frame; consume its saved link immediately, before
// another call can reuse the slot.
//
//go:noinline
func callerFramePointer() uintptr {
	fp := uintptr(c_framepointer())
	if fp == 0 {
		return 0
	}
	return *(*uintptr)(unsafe.Pointer(fp))
}

// trimPlumbingPCs drops leading pcs attributed to the LLGo runtime core
// (panic machinery, the capture path) and cuts the tail at the first pc
// outside the program text — fault snapshots are captured without the
// text bound (see fpWalkFrom).
func trimPlumbingPCs(pcs []uintptr) []uintptr {
	initRuntimeFuncPCFrames()
	head := 0
	for head < len(pcs) {
		sym := frameSymbol(pcs[head] - 1)
		if sym.function != "" && (hasPrefix(sym.function, "github.com/xgo-dev/llgo/runtime/internal/") ||
			sym.function == "runtime.capturePanicPCs" || sym.function == "runtime.onFault" ||
			sym.function == "runtime.fpWalkFrom") {
			head++
			continue
		}
		break
	}
	// Keep the innermost panic-machinery frame: gc's logical stack has
	// runtime.gopanic between the deferred function and the panic site,
	// and fixed Caller depths (issue5856's Caller(2)) count it. Fault
	// snapshots start at the fault pc and trim nothing — gc's walkers
	// skip runtime frames by name there, not by depth.
	if head > 0 {
		head--
	}
	pcs = pcs[head:]
	if rtdebug.PanicPCsAreFault() {
		// Fault snapshots come from a genuine interrupted context and the
		// chain-discipline guards already bounded the walk; keep unnamed
		// frames (linux dladdr cannot name non-dynamic C symbols — they
		// display as raw pcs, like gc does for unknown frames).
		return pcs
	}
	for i := 0; i < len(pcs); i++ {
		if prebuiltTextContains(pcs[i]) {
			continue
		}
		// Outside the Go text range: C frames in this binary (and its
		// libraries) still resolve to a symbol via dladdr — keep those;
		// cut at the first pc nothing can name (wild slots past the last
		// FP-disciplined frame).
		if frameSymbol(pcs[i]-1).function == "" {
			return pcs[:i]
		}
	}
	return pcs
}

// spliceCallers rebuilds the caller view a deferred function should see
// during (or right after recovering) a panic: its own live frames, then the
// panic-site chain from the snapshot. The junction is the first live frame
// whose function also appears in the snapshot — the defer owner; the
// snapshot side wins there because the live copy\'s pc points at the
// longjmp resume site, not at the call that panicked.
func spliceCallers(cur []uintptr) []uintptr {
	snap := panicSplicePCs()
	if len(snap) == 0 {
		return cur
	}
	snap = trimPlumbingPCs(snap)
	if len(snap) == 0 {
		return cur
	}
	// The junction is the first live frame whose function also appears in
	// the snapshot — the defer owner (or the panicking function itself when
	// defer and panic share a frame). Everything from there down is
	// replaced by the whole snapshot: it already contains the owner and its
	// callers, with the owner's pc on the panic path instead of the longjmp
	// resume site.
	for i := 0; i < len(cur); i++ {
		entry := frameSymbol(cur[i] - 1).entry
		if entry == 0 {
			continue
		}
		for j := 0; j < len(snap); j++ {
			if frameSymbol(snap[j]-1).entry == entry {
				out := make([]uintptr, 0, i+len(snap))
				out = append(out, cur[:i]...)
				out = append(out, snap...)
				return out
			}
		}
	}
	return cur
}

// callersWithPanicSplice is Callers with panic-frame splicing. With no
// snapshot stored (the overwhelmingly common case) it degrades to the
// plain walk at the cost of one TLS load. Otherwise the raw walk runs
// unskipped so splicing sees the junction frame, then the requested skip
// applies to the spliced view (matching gc, whose skip counts the logical
// panic-inclusive stack).
//
//go:noinline
func callersWithPanicSplice(skip int, pc []uintptr) int {
	if len(pc) == 0 {
		return 0
	}
	if len(rtdebug.PanicPCs()) == 0 {
		// One frame deeper than the extern.go call sites used to be.
		return fpCallers(skip+1, pc)
	}
	var raw [128]uintptr
	n := fpCallers(1, raw[:])
	if n <= 0 {
		return 0
	}
	view := spliceCallers(raw[:n])
	if skip < 0 {
		skip = 0
	}
	if skip >= len(view) {
		return 0
	}
	return copy(pc, view[skip:])
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
	// Hardware-fault panics carry a fault-site pc snapshot; print that
	// chain (fault pc through the Go callers) instead of walking the
	// live stack, whose walk would start inside the fault plumbing.
	if faultTraceback(skip) {
		return true
	}
	if faultTracebackActive() {
		// The sidecar was not already available in signal context. Preserve
		// the async-signal failure policy through the resulting fatal panic:
		// use the clite/raw-PC fallback and never initiate filesystem I/O.
		return false
	}
	// Normal panic traceback is an I/O-safe first-use point. Hardware fault
	// traceback above deliberately never initiates sidecar loading.
	ensureRuntimePCLN()
	if !fpUnwindAvailable() {
		return false
	}
	var pcs [64]uintptr
	n := fpCallers(skip, pcs[:])
	if n <= 0 {
		return false
	}
	// A stored panic snapshot (Go panic or hardware fault, including
	// faults inside C code) carries the frames the longjmp unwinding
	// already removed; splice them in like Callers does.
	view := spliceCallers(pcs[:n])
	print("goroutine 1 [running]:\n")
	frames := CallersFrames(view)
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
			if hasPrefix(name, "github.com/xgo-dev/llgo/runtime/internal/") {
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
	return runtimePCLNReady() && runtimeFPChain != 0 && runtimeFuncInfoTable != nil && runtimeFuncInfoCount > 0
}
