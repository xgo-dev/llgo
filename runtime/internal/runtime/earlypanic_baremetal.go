//go:build llgo && baremetal && tinygo.riscv32 && esp && !llgo.idf && !nogc

package runtime

// Early panic diagnostics for the ESP RISC-V baremetal targets: the first
// thing every fatal path prints, through the _wrap/earlypanic_riscv32.c
// polled USB-Serial/JTAG FIFO writer, before any stdio, allocator, lock or
// interrupt machinery is touched. Panics during package initialization -
// before clite's Stdout/Stderr vars are even initialized - still produce
// output, which is exactly the window where the old path died silently
// (print → newlib fputc on a nil FILE → abort → semihosting exit).
//
// Every fatal path on these targets routes through here:
//
//   - Go panics: Rethrow's terminal branch calls earlyPanicReport before
//     the stdio-backed TracePanic, so the normal "panic: ..." output keeps
//     its golden format and the banner rides in front of it.
//   - C-level faults (wild pointer writes, illegal instructions): the
//     chained trap handler installed by init captures mcause/mepc/mtval/sp
//     and calls llgoEarlyFaultReport, then chains to the device file's
//     handler, so the semihosting exit path is untouched. Interrupts and
//     the exit EBREAK chain without reporting.
//
// The call chain is a filtered stack scan, not an FP walk: baremetal Go
// code is deliberately compiled without frame pointers
// (ssa.Program.NeedsFramePointer returns false for named targets - the
// attribute both costs text size and was observed to retain stale stack
// slots under the conservative collector on ESP32-C3), so no FP chain
// exists to walk. Each stack word is reported as a frame when it lies in
// the linked text range and the instruction immediately before it decodes
// as a call (jal, jalr, c.jr, c.jalr), which is the return-address
// fingerprint. The result is an ordered, symbolizable superset of the call
// chain: real frames dominate, and false positives (locals holding text
// pointers, e.g. setjmp buffers) still resolve to real code under the
// symbolizer.
//
// Output format (one line each):
//
//	llgo panic: <panic value, best effort without allocation>
//	at sp=0x...
//	goroutine <goid> [running]:
//	#N pc=0x...
//	llgo fault: mcause=0x... mepc=0x... mtval=0x... s0=0x... sp=0x...
//
// Host-side symbolization (the linked ELF keeps all symbols):
//
//	llvm-symbolizer --obj=app.elf 0x2f0049d2
//
// The report never allocates: strings pass through by header, numbers
// render into c.Alloca scratch, and the goroutine dump walks the scheduler
// lists with plain pointer reads. It also never trusts a single load:
// every stack read is bounded by the fiber's stackHi (the bootstrap stack's
// linker symbols before the scheduler runs), and the walk stops after
// maxEarlyFrames entries. A reentrancy guard keeps a fault inside the
// report from looping.

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// earlyPanicLLGoFiles links the early-console C helpers into every binary
// (via the runtime core's LLGoFiles chain), mirroring fiberLLGoFiles.
const earlyPanicLLGoFiles = "; _wrap/earlypanic_riscv32.c"

//go:linkname earlyWrite C.llgo_early_write
func earlyWrite(p unsafe.Pointer, n uint32)

//go:linkname earlyTrapInstall C.llgo_early_trap_install
func earlyTrapInstall()

// Text bounds from the ESP linker scripts (targets/esp32*.app.elf.ld):
// _start anchors the first byte of .text and _etext is its PROVIDE'd end.
// Read-only data lives above _etext, so the range test also rejects rodata
// pointers that happen to sit on the stack.
//
//go:linkname earlyTextLo C._start
var earlyTextLo [0]byte

//go:linkname earlyTextHi C._etext
var earlyTextHi [0]byte

func init() {
	// The device files install their trap in .init_array, which always runs
	// before Go code; installing here chains behind them while still being
	// ahead of every package init and the main fiber.
	earlyTrapInstall()
}

// maxEarlyFrames caps every stack scan so a corrupt or oversized stack
// cannot turn the report into an endless stream.
const maxEarlyFrames = 32

// earlyReporting guards against reentry: a fault raised while the report
// itself runs must not recurse into the report.
var earlyReporting bool

func earlyPuts(s string) {
	hdr := (*String)(unsafe.Pointer(&s))
	if hdr.len == 0 {
		return
	}
	earlyWrite(hdr.data, uint32(hdr.len))
}

func earlyPutUint(v uint64) {
	// uint64 needs at most 20 decimal digits, printHex-style stack scratch.
	buf := c.Alloca(20)
	i := uintptr(20)
	for {
		i--
		*(*byte)(unsafe.Add(buf, i)) = '0' + byte(v%10)
		v /= 10
		if v == 0 {
			break
		}
	}
	earlyWrite(unsafe.Add(buf, i), uint32(20-i))
}

func earlyPutHex(v uintptr) {
	buf := c.Alloca(10) // 8 hex digits and the 0x prefix
	i := uintptr(10)
	for {
		i--
		digit := byte(v & 15)
		if digit < 10 {
			digit += '0'
		} else {
			digit += 'a' - 10
		}
		*(*byte)(unsafe.Add(buf, i)) = digit
		v >>= 4
		if v == 0 {
			break
		}
	}
	i -= 2
	*(*byte)(unsafe.Add(buf, i)) = '0'
	*(*byte)(unsafe.Add(buf, i+1)) = 'x'
	earlyWrite(unsafe.Add(buf, i), uint32(10-i))
}

// earlyPanicValue renders the panic payload without running user code or
// allocating: string values (plain and named, e.g. the runtime errorString
// boxes) read the header directly, and the numeric cases follow
// printanycustomtype's boxed-value layout. Anything else - error values
// whose Error() builds a new string, structs, slices - prints nothing
// rather than risking a nested fault.
func earlyPanicValue(arg any) {
	if arg == nil {
		earlyPuts("nil")
		return
	}
	e := (*eface)(unsafe.Pointer(&arg))
	t := e._type
	if t == nil {
		earlyPuts("nil")
		return
	}
	switch t.Kind() {
	case abi.String:
		earlyPuts(*(*string)(e.data))
	case abi.Int, abi.Int8, abi.Int16, abi.Int32, abi.Int64:
		var v int64
		if isDirectIface(t) {
			v = int64(uintptr(e.data))
		} else {
			v = *(*int64)(e.data)
		}
		negative := v < 0
		u := uint64(v)
		if negative {
			u = 0 - u
			earlyPuts("-")
		}
		earlyPutUint(u)
	case abi.Uint, abi.Uint8, abi.Uint16, abi.Uint32, abi.Uint64, abi.Uintptr:
		var v uint64
		if isDirectIface(t) {
			v = uint64(uintptr(e.data))
		} else {
			v = *(*uint64)(e.data)
		}
		earlyPutUint(v)
	}
}

func earlyGStatusName(status uint32) string {
	switch status {
	case _Grunnable:
		return "runnable"
	case _Grunning:
		return "running"
	case _Gwaiting:
		return "waiting"
	case _Gdead:
		return "dead"
	}
	return "unknown"
}

func earlyGoroutineLine(gp *g) {
	earlyPuts("goroutine ")
	earlyPutUint(gp.goid)
	earlyPuts(" [")
	earlyPuts(earlyGStatusName(readgstatus(gp)))
	earlyPuts("]:\n")
}

// earlyStackHiFor returns the top of the stack a report should scan: the
// running G's fiber bound when one exists, else the bootstrap stack's
// linker symbol (the scheduler G before any fiber exists, or code running
// before the scheduler was configured).
func earlyStackHiFor(gp *g) uintptr {
	if gp != nil && gp.context != nil {
		if hi := gp.context.platform.context.stackHi; hi != 0 {
			return hi
		}
	}
	return uintptr(unsafe.Pointer(&baremetalStackTop))
}

// earlyIsCallSite reports whether pc sits right after a call: the 4-byte
// jal/jalr opcodes at pc-4 or a compressed c.jr/c.jalr halfword at pc-2.
// Text is always mapped, so the instruction reads cannot fault.
func earlyIsCallSite(pc uintptr) bool {
	w := *(*uint32)(unsafe.Pointer(pc - 4))
	if w&0x7f == 0x67 || w&0x7f == 0x6f {
		return true
	}
	h := *(*uint16)(unsafe.Pointer(pc - 2))
	return h&0xe003 == 0x8002
}

// earlyScanChain prints every return address on [sp, stackHi): the deepest
// call site sits at the lowest address, so #N grows toward main. Loads are
// bounded by stackHi, text-range filtered, and call-site verified.
func earlyScanChain(sp, stackHi uintptr) {
	if stackHi == 0 || sp == 0 || sp >= stackHi {
		return
	}
	textLo := uintptr(unsafe.Pointer(&earlyTextLo))
	textHi := uintptr(unsafe.Pointer(&earlyTextHi))
	frames := 0
	for w := sp &^ 3; w+4 <= stackHi && frames < maxEarlyFrames; w += 4 {
		pc := *(*uintptr)(unsafe.Pointer(w))
		if pc < textLo+4 || pc >= textHi {
			continue
		}
		if !earlyIsCallSite(pc) {
			continue
		}
		earlyPuts("#")
		earlyPutUint(uint64(frames))
		earlyPuts(" pc=")
		earlyPutHex(pc)
		earlyPuts("\n")
		frames++
	}
}

// earlyDumpParkedGs reports every other registered goroutine with the call
// chain preserved on its parked fiber: the fiber swap frame keeps the return
// address of the yielding function at [sp] and its frame pointer at [sp+4],
// so a scan from the saved sp reads the same shape as a live stack. Parked
// stacks cannot change while the report runs (single hart, cooperative
// scheduler, interrupts excluded from reporting).
func earlyDumpParkedGs(skip *g) {
	if !baremetalSched.started {
		return
	}
	for current := baremetalSched.all; current != nil; current = current.context.platform.allNext {
		if current == skip || current.context == nil {
			continue
		}
		fc := &current.context.platform.context
		if fc.sp == 0 || fc.stackLo == 0 || fc.sp <= fc.stackLo || fc.sp >= fc.stackHi {
			continue
		}
		earlyGoroutineLine(current)
		earlyScanChain(fc.sp, fc.stackHi)
	}
}

// earlyPanicReport is the front of the baremetal panic path: it prints the
// banner, the goroutine summary and the scanned chain before the stdio
// path gets a chance to fail. It returns so the usual TracePanic/fatal
// flow (and its exit semantics) runs unchanged afterwards.
func earlyPanicReport(arg any) {
	if earlyReporting {
		return
	}
	earlyReporting = true
	gp := currentG
	ep := uintptr(baremetalGetSP())
	earlyPuts("llgo panic: ")
	earlyPanicValue(arg)
	earlyPuts("\nat sp=")
	earlyPutHex(ep)
	earlyPuts("\n")
	if gp != nil {
		earlyGoroutineLine(gp)
	}
	earlyScanChain(ep, earlyStackHiFor(gp))
	earlyDumpParkedGs(gp)
	earlyReporting = false
}

// captureRecord mirrors struct llgo_early_fault_capture in
// _wrap/earlypanic_riscv32.c; the naked trap handler fills it.
type captureRecord struct {
	mepc   uint32
	mcause uint32
	mtval  uint32
	s0     uint32
	sp     uint32
}

//export llgoEarlyFaultReport
func llgoEarlyFaultReport(cap unsafe.Pointer) unsafe.Pointer {
	rec := (*captureRecord)(cap)
	if rec != nil && !earlyReporting {
		earlyReporting = true
		earlyPuts("llgo fault: mcause=")
		earlyPutHex(uintptr(rec.mcause))
		earlyPuts(" mepc=")
		earlyPutHex(uintptr(rec.mepc))
		earlyPuts(" mtval=")
		earlyPutHex(uintptr(rec.mtval))
		earlyPuts(" s0=")
		earlyPutHex(uintptr(rec.s0))
		earlyPuts(" sp=")
		earlyPutHex(uintptr(rec.sp))
		earlyPuts("\n")
		gp := currentG
		if gp != nil {
			earlyGoroutineLine(gp)
		}
		earlyScanChain(uintptr(rec.sp), earlyStackHiFor(gp))
		earlyDumpParkedGs(gp)
		earlyReporting = false
	}
	return nil
}
