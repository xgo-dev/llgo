//go:build !baremetal && !wasm && !windows

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	rtdebug "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"github.com/xgo-dev/llgo/runtime/internal/stacktrace"
	"github.com/xgo-dev/llgo/runtime/internal/traceback"
)

// Hardware-fault stacks: a SA_SIGINFO handler captures the interrupted
// context (the handler's own frame-pointer chain dead-ends at the signal
// trampoline), unwinds it — dynamically-resolved libunwind first, which
// survives C frames without frame pointers and (nongnu flavor) names
// static C symbols, then the FP chain — and converts the fault into the
// usual Go panic. The unrecovered traceback prints the fault-site chain:
// C frames down through the Go callers.

//go:linkname c_installFaultHandler C.llgo_install_fault_handler
func c_installFaultHandler(cb func(uintptr, uintptr, int32))

//go:linkname c_dynunwindPCCount C.llgo_dynunwind_pccount
func c_dynunwindPCCount() int32

//go:linkname c_dynunwindEndFP C.llgo_dynunwind_endfp
func c_dynunwindEndFP() uintptr

//go:linkname c_dynunwindName C.llgo_dynunwind_name
func c_dynunwindName(i int32) *c.Char

//go:linkname c_memReadable C.llgo_mem_readable
func c_memReadable(p unsafe.Pointer) int32

//go:linkname c_faultCaptureDone C.llgo_fault_capture_done
func c_faultCaptureDone()

func memReadable(addr uintptr) bool {
	return c_memReadable(unsafe.Pointer(addr)) != 0
}

// recoverMark records the recovering deferred frame so the panic snapshot
// stays spliceable while that frame is live. After siglongjmp the chain can
// reach stale storage, so panicSplicePCs probes each slot before reading it.
func recoverMark() {
	fp := callerFramePointer()
	if fp == 0 {
		return
	}
	rtdebug.MarkPanicRecoverFPs(fp, 0)
}

func panicSplicePCs() []uintptr {
	pcs := rtdebug.PanicPCs()
	if len(pcs) == 0 {
		return nil
	}
	if rtdebug.PanicActive() {
		return pcs
	}
	mark, _ := rtdebug.PanicRecoverFPs()
	if mark == 0 {
		return nil
	}
	fp := callerFramePointer()
	for i := 0; fp != 0 && i < maxPanicSpliceFrames; i++ {
		if !memReadable(fp) {
			break
		}
		prev := *(*uintptr)(unsafe.Pointer(fp))
		if fp <= mark && (prev > mark || prev == 0) {
			return pcs
		}
		if prev <= fp || prev-fp > maxFPStride || prev&(unsafe.Sizeof(uintptr(0))-1) != 0 {
			break
		}
		fp = prev
	}
	return nil
}

// fpCallers walks conventional frame records. Win64 has a separate SEH
// implementation because its frame register may be biased into the frame.
//
//go:noinline
func fpCallers(skip int, pc []uintptr) int {
	if len(pc) == 0 {
		return 0
	}
	initRuntimeFuncPCFrames()
	var low, high uintptr
	stacktrace.Bounds(&low, &high)
	fp := uintptr(c_framepointer())
	n, lastGo := 0, 0
	const maxFrames = maxPanicSpliceFrames
	for i := 0; fp != 0 && n < len(pc) && i < maxFrames; i++ {
		if high != 0 && (fp < low || fp > high-2*unsafe.Sizeof(uintptr(0))) {
			break
		}
		prev := *(*uintptr)(unsafe.Pointer(fp))
		ret := *(*uintptr)(unsafe.Pointer(fp + unsafe.Sizeof(uintptr(0))))
		inGo := prebuiltTextContains(ret - 1)
		if ret < minLegalPC || high == 0 && !inGo {
			break
		}
		if skip > 0 {
			skip--
		} else {
			pc[n] = ret
			n++
			if inGo {
				lastGo = n
			}
		}
		if prev <= fp || prev-fp > maxFPStride || prev&(unsafe.Sizeof(uintptr(0))-1) != 0 {
			break
		}
		fp = prev
	}
	// Preserve C library frames between Go activations, while dropping the
	// native thread/process startup tail after the outermost Go frame.
	if n == len(pc) {
		return n
	}
	return lastGo
}

func init() {
	c_installFaultHandler(onFault)
}

// Each native thread reserves its fault buffer before entering Go user code.
// Capture performs no Go allocation, including for a deep interrupted stack.
func onFault(pc, fp uintptr, sig int32) {
	var pcs []uintptr
	if buf := stacktrace.FaultBuffer(); buf != nil {
		pcs = unsafe.Slice(buf, stacktrace.MaxFrames)
	}
	n := 0
	if len(pcs) != 0 {
		if dn := int(c_dynunwindPCCount()); dn > 0 {
			if dn > len(pcs) {
				dn = len(pcs)
			}
			// dynunwind writes directly into this thread's reserved buffer.
			pcs[0]++
			n = dn
			if efp := c_dynunwindEndFP(); efp != 0 && n < len(pcs) {
				m := fpWalkFrom(efp, pcs[n:])
				if m > 0 && pcs[n] == pcs[n-1] {
					copy(pcs[n:], pcs[n+1:n+m])
					m--
				}
				n += m
			}
		} else {
			if pc != 0 {
				pcs[0] = pc + 1
				n = 1
			}
			n += fpWalkFrom(fp, pcs[n:])
		}
	}
	rtdebug.StoreFaultPCs(pcs[:n])
	c_faultCaptureDone()
	rtdebug.PanicSignal(int(sig))
}

func faultTracebackActive() bool {
	return rtdebug.PanicPCsAreFault() && rtdebug.PanicActive()
}

// fpWalkFrom walks the frame-pointer chain from an arbitrary frame pointer
// (a fault context's fp). Chain-discipline guards only: it runs in signal
// context where the first-use frame tables cannot be built, so the
// program-text bound is applied at print time.
func fpWalkFrom(fp uintptr, pc []uintptr) int {
	n := 0
	const maxFrames = maxTracebackFrames
	wordSize := unsafe.Sizeof(uintptr(0))
	for i := 0; fp != 0 && n < len(pc) && i < maxFrames; i++ {
		if fp&(wordSize-1) != 0 || !memReadable(fp) || !memReadable(fp+wordSize) {
			break
		}
		prev := *(*uintptr)(unsafe.Pointer(fp))
		ret := *(*uintptr)(unsafe.Pointer(fp + unsafe.Sizeof(uintptr(0))))
		if ret < minLegalPC {
			break
		}
		pc[n] = ret
		n++
		if prev <= fp || prev-fp > maxFPStride || prev&(unsafe.Sizeof(uintptr(0))-1) != 0 {
			break
		}
		fp = prev
	}
	return n
}

func stringContainsDot(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return true
		}
	}
	return false
}

// faultPCSymbol preserves leading C symbols, including static names supplied
// by libunwind that Go's nearest-function table cannot identify.
func faultPCSymbol(pc uintptr, index int) pcSymbol {
	sym := frameSymbol(pc - 1)
	if dyn := c.GoString(c_dynunwindName(int32(index))); dyn != "" &&
		(sym.function == "" || !stringContainsDot(dyn)) {
		sym.function, sym.file, sym.line, sym.entry = dyn, "", 0, 0
	}
	return sym
}

// faultTraceback prints the gc-style traceback for an unrecovered panic
// that originated in a hardware fault; other panics fall back to the
// clite dump (reports false).
func faultTraceback(skip int) bool {
	pcs := rtdebug.PanicPCs()
	if !rtdebug.PanicPCsAreFault() || len(pcs) == 0 {
		return false
	}
	// Missing external metadata must never cause file I/O in a fault path.
	// Raw PCs still identify the interrupted code for offline symbolization.
	ready := fpUnwindAvailable()
	if ready {
		initRuntimeFuncPCFrames()
	}
	system := traceback.Level(rtdebug.TracebackSetting()) > 1
	if ready {
		pcs = trimNativeTracebackTail(pcs)
	}
	count := 0
	for i, pc := range pcs {
		if ready {
			sym := faultPCSymbol(pc, i)
			if stringContainsDot(sym.function) && !visibleTracebackFrame(sym.function, system) {
				continue
			}
		}
		count++
	}
	print("goroutine ", goid(), " [running]:\n")
	shown := 0
	for i, pc := range pcs {
		var sym pcSymbol
		if ready {
			sym = faultPCSymbol(pc, i)
		}
		if ready && stringContainsDot(sym.function) && !visibleTracebackFrame(sym.function, system) {
			continue
		}
		if count > traceback.InnerFrames+traceback.OuterFrames &&
			shown >= traceback.InnerFrames && shown < count-traceback.OuterFrames {
			if shown == traceback.InnerFrames {
				print("...", count-traceback.InnerFrames-traceback.OuterFrames, " frames elided...\n")
			}
			shown++
			continue
		}
		shown++
		name := sym.function

		if name == "" {
			name = unknownFunctionName(pc)
		}
		print(name, "(...)\n\t")
		if sym.file == "" {
			print("pc=0x", string(appendHexUint(nil, pc-1)))
		} else {
			print(sym.file, ":", sym.line)
			if sym.entry != 0 && pc >= sym.entry {
				print(" +0x", string(appendHexUint(nil, pc-sym.entry)))
			}
		}
		print("\n")
	}
	print(string(appendCurrentCreatedBy(nil)))
	return shown != 0
}
