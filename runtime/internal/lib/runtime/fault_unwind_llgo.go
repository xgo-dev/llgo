//go:build !baremetal && !wasm

package runtime

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	rtdebug "github.com/goplus/llgo/runtime/internal/runtime"
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

//go:linkname c_dynunwindPCBuf C.llgo_dynunwind_pcbuf
func c_dynunwindPCBuf() unsafe.Pointer

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

func init() {
	c_installFaultHandler(onFault)
}

// Fault snapshot: written once in the fault handler, consumed by the
// panic traceback (the process either recovers — dropping the snapshot's
// relevance — or dies printing it; a concurrent fault on another thread
// is a lost race on a doomed process).
var (
	faultPCs [64]uintptr
	faultN   int32
	// index into the dynunwind name table per pc, -1 when the pc came
	// from the FP-chain resume.
	faultNameIdx [64]int32
)

func onFault(pc, fp uintptr, sig int32) {
	if fpUnwindAvailable() {
		n := 0
		if dn := int(c_dynunwindPCCount()); dn > 0 {
			buf := (*[64]uintptr)(c_dynunwindPCBuf())
			if dn > len(faultPCs) {
				dn = len(faultPCs)
			}
			for i := 0; i < dn; i++ {
				faultPCs[i] = buf[i]
				faultNameIdx[i] = int32(i)
			}
			// Frame 0 is the fault pc itself; +1 keeps the pc-1
			// return-address convention landing on the faulting
			// instruction (gc's sigpanic does the same).
			faultPCs[0]++
			n = dn
			// libunwind stops where unwind info runs out; resume with the
			// FP chain from its final cursor position.
			if efp := c_dynunwindEndFP(); efp != 0 && n < len(faultPCs) {
				m := fpWalkFrom(efp, faultPCs[n:])
				if m > 0 && faultPCs[n] == faultPCs[n-1] {
					copy(faultPCs[n:], faultPCs[n+1:n+m])
					m--
				}
				for i := 0; i < m; i++ {
					faultNameIdx[n+i] = -1
				}
				n += m
			}
		} else {
			if pc != 0 {
				faultPCs[0] = pc + 1
				faultNameIdx[0] = -1
				n = 1
			}
			m := fpWalkFrom(fp, faultPCs[n:])
			for i := 0; i < m; i++ {
				faultNameIdx[n+i] = -1
			}
			n += m
		}
		faultN = int32(n)
		rtdebug.StoreFaultPCs(faultPCs[:n])
	}
	// Capture done: re-arm the recursion guard before this fault turns
	// into an ordinary (recoverable) panic.
	c_faultCaptureDone()
	rtdebug.PanicSignal(int(sig))
}

// fpWalkFrom walks the frame-pointer chain from an arbitrary frame pointer
// (a fault context's fp). Chain-discipline guards only: it runs in signal
// context where the first-use frame tables cannot be built, so the
// program-text bound is applied at print time.
func fpWalkFrom(fp uintptr, pc []uintptr) int {
	n := 0
	const maxFrames = 4096
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

// faultTraceback prints the gc-style traceback for an unrecovered panic
// that originated in a hardware fault; other panics fall back to the
// clite dump (reports false).
func faultTraceback(skip int) bool {
	if faultN == 0 || !fpUnwindAvailable() {
		return false
	}
	initRuntimeFuncPCFrames()
	print("goroutine 1 [running]:\n")
	printed := 0
	printedInText := 0
	for i := 0; i < int(faultN); i++ {
		pc := faultPCs[i]
		inText := prebuiltTextContains(pc)
		sym := frameSymbol(pc - 1)
		name := sym.function
		if faultNameIdx[i] >= 0 {
			// libunwind's proc name: the nongnu flavor reads .symtab and
			// names static C symbols dladdr cannot see. A dot-less name is
			// a C symbol — prefer it over the table's nearest-below
			// attribution, which cannot see foreign functions linked
			// between Go functions and misnames them.
			if dyn := c.GoString(c_dynunwindName(faultNameIdx[i])); dyn != "" {
				if name == "" || !stringContainsDot(dyn) {
					name = dyn
					sym.file = ""
					sym.line = 0
					sym.entry = 0
				}
			}
		}
		if !inText {
			// Frames outside the program's own text: keep leading library
			// frames (the fault may sit inside libc/libssl), but once the
			// module's frames have been shown the rest is startup plumbing
			// below main (__libc_start_main, _start) — cut it. Unnamed
			// out-of-text slots are unidentifiable either way.
			if printedInText > 0 || name == "" {
				break
			}
		}
		if name == "" {
			name = unknownFunctionName(pc)
		}
		print(name, "(...)\n\t")
		if sym.file == "" {
			// No line info (C frame): print the raw pc — resolvable
			// offline (addr2line/atos) — instead of a ???:0 placeholder.
			print("pc=0x", string(appendHexUint(nil, pc-1)))
		} else {
			print(sym.file, ":", sym.line)
			if sym.entry != 0 && pc >= sym.entry {
				print(" +0x", string(appendHexUint(nil, pc-sym.entry)))
			}
		}
		print("\n")
		printed++
		if inText {
			printedInText++
		}
	}
	return printed > 0
}
