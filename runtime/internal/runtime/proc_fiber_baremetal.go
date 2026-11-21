//go:build llgo && baremetal && tinygo.riscv32 && !nogc

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

// The baremetal fiber backend multiplexes every G onto one hart with a
// cooperative scheduler, mirroring proc_wasm.go: NewProc appends a fresh G to
// the run queue, parking swaps the current G out on its own fiber stack, and
// the scheduler loop in RunMain resumes queued Gs until only deadlocked
// waiters remain. time.Sleep parks through the sleeper list in
// sleep_fiber_baremetal.go; the scheduler busy-polls its deadlines when the
// run queue runs dry. There is no preemption, timer-interrupt delivery or
// SMP support; goroutines yield through channel operations, runtime.Gosched,
// sleep expiry or exit.
//
// Goroutine-local storage degenerates on this backend the same way it does on
// single-worker wasm: local_context_baremetal.go keeps one bootstrap context
// global, and fibers share it while swapped in. No generated GLS package
// block exists for baremetal, so there is nothing per-G to swap or release.

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/runqueue"
	"github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc"
)

// baremetalFiberContext is one fiber's suspended execution state. sp must
// stay the first field: llgoFiberSwap publishes the suspended frame address
// through it, and the collector scans a parked stack from sp to stackHi.
// stackLo/stackHi double as the stack-overflow guard's bounds: the swap's
// asm reads stackLo at offset 4, checks the canary word at the stack floor and
// compares the live sp against stackLo before pushing its own frame. A zero
// stackLo (no configured bounds) skips the guard.
type baremetalFiberContext struct {
	sp      uintptr
	stackLo uintptr
	stackHi uintptr
}

// Fiber swap-frame layout shared with _wrap/context_riscv32.c. The frame
// keeps ra plus all twelve RV32I callee-saved integer registers (s0-s11 =
// x8,x9,x18-x27; the ilp32 soft-float ABI affects only the f registers) with
// the psABI's 16-byte stack alignment.
const (
	fiberFrameSize   = 64
	fiberFrameRAOff  = 0
	fiberFrameArgOff = 52
)

// Fiber stacks for Gs that no configured size covers (see SetPthreadStackSize
// and the target's default-stack-size): the build side always injects a
// non-zero floor for fiber targets, so this fallback only covers targets that
// resolve without one. Fiber stacks are guarded at every cooperative swap:
// llgoFiberSwap fatals with "goroutine stack overflow" when the outgoing
// fiber's sp crossed its stack floor or the floor canary is smashed. The
// guard is checkpoint-granular (see _wrap/context_riscv32.c), so keep frame
// sizes conservative or pass an explicit stack size regardless.
const (
	defaultBaremetalGoroutineStack = 8 << 10
	defaultBaremetalMainStack      = 16 << 10
)

// fiberStackCanary marks the floor word of every guarded fiber stack (the
// word at stackLo). tinygogc zeroes fresh stacks, so any other value proves
// that a frame wrote at or below the floor since the last swap - including
// excursions whose sp recovered before the next yield, which the sp bound
// alone would miss. Value chosen nonzero and away from zero-fill; the C
// mirror in _wrap/context_riscv32.c must stay in sync.
const fiberStackCanary = uintptr(0x5afec0de)

// fiberMinStackSize floors explicit stack sizes so the swap guard's math
// always holds. The guard margin (68: 64-byte swap frame + 4-byte canary
// word, see llgoFiberGuardMargin in _wrap/context_riscv32.c) plus the
// initial swap frame (64) must fit above the floor even after the 16-byte
// alignment round-down of the top shaves up to 15 bytes: 68+64+15 = 147 is
// the bare correctness minimum. The floor keeps 61 bytes of entry-stub
// working room beyond that, so a fresh fiber can at least enter its entry
// stub before any further use is judged on its merits. A stack this small
// still fatals at its first real yield - every usable goroutine stack comes
// from the target's default-stack-size floor (2048); this only guards
// against tiny explicit -goroutine-stack-size values.
const fiberMinStackSize = 208

// TODO: the swap-point guard above is checkpoint-granular like TinyGo's
// task.Pause canary; catching overflow at every function entry instead would
// need a compiler pass in cl/ (llgo.stackSave gives the SP, and
// prepareCooperativeSafepoints shows where per-function instrumentation
// hooks in) plus a per-G bounds fast path. Left undone: it touches every
// function's hot path and roughly doubles the prologue cost for a guard the
// swap checkpoint already provides.

// The packed live-G state mirrors proc_atomic.go. Baremetal transitions are
// plain reads and writes: cooperative scheduling keeps every G state change
// on one hart, and interrupt handlers never touch scheduler state.
const (
	mainExitedBit = uint64(1) << 63
	gCountMask    = mainExitedBit - 1
)

// runtimeContextPlatform keeps each G's fiber plus the intrusive scheduler
// links. The scheduler state lives in globals, so the conservative collector
// reaches every registered G's context and parked stack through it.
type runtimeContextPlatform struct {
	context    baremetalFiberContext
	runqNext   *g
	runqQueued bool
	allNext    *g
	allQueued  bool
}

// Native ABIs agree with Go's runtimeContext layout, so no wasm allocator
// padding is added to baremetal goroutines.
func runtimeContextAllocSize() uintptr {
	return unsafe.Sizeof(runtimeContext{})
}

type mOS struct{}

// baremetalSched is the single-owner scheduler state, mirroring wasmSched.
var baremetalSched struct {
	m       m
	p       p
	systemG g
	system  baremetalFiberContext
	runq    runqueue.Queue[*g]
	all     *g
	started bool
}

//go:linkname fiberSwap C.llgoFiberSwap
func fiberSwap(from, to *baremetalFiberContext)

//go:linkname fiberGStartAddr C.llgoFiberGStartAddr
func fiberGStartAddr() unsafe.Pointer

//go:linkname fiberMainStartAddr C.llgoFiberMainStartAddr
func fiberMainStartAddr() unsafe.Pointer

// baremetalMainTask is the compiler-generated __llgo_baremetal_main task. It
// performs package initialization and calls main.main on the main G's fiber.
//
//go:linkname baremetalMainTask C.__llgo_baremetal_main
func baremetalMainTask(unsafe.Pointer) unsafe.Pointer

//go:linkname baremetalStackTop C._stack_top
var baremetalStackTop [0]byte

//go:linkname baremetalStackEnd C._stack_end
var baremetalStackEnd [0]byte

//go:linkname baremetalGetSP llgo.stackSave
func baremetalGetSP() unsafe.Pointer

// The baremetal fiber scheduler requires the conservative collector's stack
// hook, so the C fiber helpers link through the LLGoFiles chain in z_rt.go
// only for this backend.
const fiberLLGoFiles = "; _wrap/context_riscv32.c"

func initRuntimeContext(ctx *runtimeContext, callergp *g, status uint32) *g {
	gp := initG(ctx, callergp, status)
	if status == _Grunning {
		initBaremetalScheduler(gp)
	}
	return gp
}

// initBaremetalScheduler runs once, when the bootstrap context allocates the
// main G ahead of any package init. It installs the collector's stack hook so
// parked fibers keep their objects alive across collections.
func initBaremetalScheduler(gp *g) {
	if baremetalSched.started {
		fatal("runtime: baremetal scheduler initialized twice")
		return
	}
	tinygogc.StackRootsHook = scanBaremetalStackRoots
	mp := &baremetalSched.m
	pp := &baremetalSched.p
	mp.curg = gp
	mp.p = pp
	mp.id = nextMid(mp)
	pp.id = nextPid(pp)
	setpstatus(pp, _Prunning)
	pp.m = mp
	gp.m = mp
	// Guard the bootstrap stack the scheduler runs on: publish its
	// linker-provided bounds in the system context and mark its floor word,
	// so the swap guard covers scheduler code the same way it covers G
	// fibers. crt0 has long left the bootstrap stack by the time Go code
	// runs, so the floor word is free to hold the canary.
	baremetalSched.system.stackLo = uintptr(unsafe.Pointer(&baremetalStackEnd))
	baremetalSched.system.stackHi = uintptr(unsafe.Pointer(&baremetalStackTop))
	*(*uintptr)(unsafe.Pointer(&baremetalStackEnd)) = fiberStackCanary
	systemG := &baremetalSched.systemG
	systemG.atomicstatus = _Grunning
	systemG.m = mp
	baremetalSched.started = true
}

// RunMain keeps the bootstrap context as the scheduler and runs package
// initialization plus main.main as the first logical G on its own fiber, the
// native counterpart of RunWasmMain. The generated entry calls it instead of
// running inits and main.main on the C startup stack.
func RunMain() {
	gp := getg()
	if gp == nil || !gp.isMain {
		fatal("runtime: invalid baremetal main goroutine")
		return
	}
	if !baremetalSched.started {
		fatal("runtime: baremetal scheduler was not initialized")
		return
	}
	addBaremetalAll(gp)
	// The main fiber hosts package initialization too, so it scales the
	// goroutine stack size up (4x) with the same 16 KiB floor.
	mainStack := goroutineStackSize * 4
	if mainStack < defaultBaremetalMainStack {
		mainStack = defaultBaremetalMainStack
	}
	if !initBaremetalFiber(gp, fiberMainStartAddr(), unsafe.Pointer(gp), mainStack) {
		panic("runtime: failed to allocate baremetal main stack")
	}
	swapToBaremetalG(gp)
	for {
		if readgstatus(gp) == _Gdead {
			isMain := gp.isMain
			freeBaremetalFiber(gp)
			if isMain {
				if _, mainExited := gState(); !mainExited {
					// main.main returned: Go exits the program even when
					// other goroutines remain.
					return
				}
			}
		}
		// Wake due sleepers and dispatch due timers before picking the next
		// G: a runnable G that immediately requeues itself (runtime.Gosched)
		// must not starve an expired sleeper parked behind it in the FIFO run
		// queue, and a due timer's delivery goroutine joins the same queue
		// ahead of that pick.
		baremetalWakeSleepers()
		baremetalWakeTimers()
		gp = baremetalSched.runq.Pop()
		if gp == nil && baremetalSleepWait() {
			// The wait woke at least one sleeper, and goready queued it
			// before any other G could run: the queue is non-empty again.
			gp = baremetalSched.runq.Pop()
		}
		if gp == nil {
			_, mainExited := gState()
			if mainExited {
				fatal("no goroutines (main called runtime.Goexit) - deadlock!")
			} else {
				fatal("all goroutines are asleep - deadlock!")
			}
			c.Exit(2)
		}
		swapToBaremetalG(gp)
	}
}

// swapToBaremetalG resumes gp's fiber from the bootstrap scheduler context.
func swapToBaremetalG(gp *g) {
	if readgstatus(gp) == _Grunnable {
		casgstatus(gp, _Grunnable, _Grunning)
	}
	mp := &baremetalSched.m
	pp := &baremetalSched.p
	mp.curg = gp
	pp.m = mp
	gp.m = mp
	setg(gp)
	fiberSwap(&baremetalSched.system, &gp.context.platform.context)
	// The scheduler owns the bootstrap stack again. Do not leave currentG
	// pointing at the suspended G: a collector safepoint while no G runs
	// must not treat that G as active.
	setg(&baremetalSched.systemG)
	baremetalSched.m.curg = &baremetalSched.systemG
}

// initBaremetalFiber allocates gp's fiber stack and writes the initial swap
// frame so the first fiberSwap enters entry with arg in a0.
func initBaremetalFiber(gp *g, entry unsafe.Pointer, arg unsafe.Pointer, stackSize uintptr) bool {
	if stackSize == 0 {
		stackSize = defaultBaremetalGoroutineStack
	}
	// Keep the stack large enough for the guard's math (see
	// fiberMinStackSize): top-64 must stay above stackLo+68 even in the
	// worst alignment case, or the first swap out would false-fatal and
	// park the initial frame on the canary word.
	if stackSize < fiberMinStackSize {
		stackSize = fiberMinStackSize
	}
	stack := AllocRoot(stackSize)
	if stack == nil {
		return false
	}
	ctx := &gp.context.platform.context
	ctx.stackLo = uintptr(unsafe.Pointer(stack))
	// Keep the RISC-V psABI 16-byte stack alignment; llgoFiberSwap's frames
	// preserve it for every later switch.
	top := (ctx.stackLo + stackSize) &^ 15
	ctx.stackHi = top
	// Mark the stack floor for the swap guard (see fiberStackCanary).
	*(*uintptr)(stack) = fiberStackCanary
	frame := unsafe.Pointer(top - fiberFrameSize)
	*(*uintptr)(unsafe.Add(frame, fiberFrameRAOff)) = uintptr(entry)
	*(*uintptr)(unsafe.Add(frame, fiberFrameArgOff)) = uintptr(arg)
	ctx.sp = uintptr(frame)
	return true
}

// freeBaremetalFiber releases one dead G's fiber stack and context. It must
// run on the bootstrap stack, because a fiber cannot free the stack it is
// executing on.
func freeBaremetalFiber(gp *g) {
	ctx := gp.context
	if ctx == nil {
		return
	}
	fc := &ctx.platform.context
	if fc.stackLo == 0 {
		return
	}
	baremetalRemoveAll(gp)
	gp.m = nil
	baremetalSched.m.curg = &baremetalSched.systemG
	stack := unsafe.Pointer(fc.stackLo)
	fc.sp, fc.stackLo, fc.stackHi = 0, 0, 0
	FreeRoot(stack)
	freeRuntimeContext(ctx)
}

//export llgoFiberGoStart
func baremetalGStart(arg unsafe.Pointer) unsafe.Pointer {
	gp := (*g)(arg)
	if gp == nil || getg() != gp {
		fatal("runtime: invalid baremetal goroutine entry")
		return nil
	}
	fn, fnarg := gp.startfn, gp.startarg
	gp.startfn = nil
	fn(fnarg)
	goexitBackend(gp)
	return nil
}

//export llgoFiberMainStart
func baremetalMainStart(arg unsafe.Pointer) unsafe.Pointer {
	_ = arg
	gp := getg()
	if gp == nil || !gp.isMain {
		fatal("runtime: invalid baremetal main fiber entry")
		return nil
	}
	baremetalMainTask(nil)
	casgstatus(gp, _Grunning, _Gdead)
	releaseG()
	fiberSwap(&gp.context.platform.context, &baremetalSched.system)
	fatal("runtime: resumed completed baremetal main goroutine")
	return nil
}

func newprocBackend(fn goroutineFunc, arg unsafe.Pointer, stackSize uintptr, callergp *g) {
	gp := newproc1(fn, arg, callergp)
	if !initBaremetalFiber(gp, fiberGStartAddr(), unsafe.Pointer(gp), stackSize) {
		releaseG()
		releaseStartArg(gp)
		freeRuntimeContext(gp.context)
		panic("runtime: failed to allocate baremetal goroutine stack")
	}
	addBaremetalAll(gp)
	if !baremetalSched.runq.Push(gp) {
		fatal("runtime: duplicate new goroutine in baremetal run queue")
	}
}

func goschedBackend() {
	gp := getg()
	casgstatus(gp, _Grunning, _Grunnable)
	if !baremetalSched.runq.Push(gp) {
		fatal("runtime: current goroutine is already in baremetal run queue")
		return
	}
	fiberSwap(&gp.context.platform.context, &baremetalSched.system)
}

func gopark() {
	gp := getg()
	casgstatus(gp, _Grunning, _Gwaiting)
	fiberSwap(&gp.context.platform.context, &baremetalSched.system)
}

func goready(gp *g) {
	if gp == nil {
		fatal("runtime: ready of nil goroutine")
		return
	}
	casgstatus(gp, _Gwaiting, _Grunnable)
	if !baremetalSched.runq.Push(gp) {
		fatal("runtime: readied goroutine is already in baremetal run queue")
	}
}

func goexitBackend(gp *g) {
	releaseStartArg(gp)
	casgstatus(gp, _Grunning, _Gdead)
	releaseGAndCheckDeadlock()
	fiberSwap(&gp.context.platform.context, &baremetalSched.system)
	fatal("runtime: resumed dead baremetal goroutine")
}

// goexitBaremetal terminates a G whose deferred functions have all run after
// runtime.Goexit. The main G marks its exit so the scheduler can keep serving
// the remaining goroutines.
func goexitBaremetal(gp *g) {
	if gp.isMain {
		markMainExited()
	}
	goexitBackend(gp)
}

// Baremetal has no atomic counter. Cooperative scheduling keeps every G state
// transition on one hart, so plain reads and writes are exact.

func retainG() {
	sched.gstate++
}

func releaseG() (remaining uint64, mainExited bool) {
	sched.gstate--
	return sched.gstate & gCountMask, sched.gstate&mainExitedBit != 0
}

func markMainExited() {
	sched.gstate |= mainExitedBit
}

func gState() (count uint64, mainExited bool) {
	return sched.gstate & gCountMask, sched.gstate&mainExitedBit != 0
}

func addBaremetalAll(gp *g) {
	if gp.context.platform.allQueued {
		return
	}
	gp.context.platform.allNext = baremetalSched.all
	gp.context.platform.allQueued = true
	baremetalSched.all = gp
}

func baremetalRemoveAll(gp *g) {
	if gp.context == nil || !gp.context.platform.allQueued {
		return
	}
	gp.context.platform.allQueued = false
	var previous *g
	for current := baremetalSched.all; current != nil; current = current.context.platform.allNext {
		if current == gp {
			if previous == nil {
				baremetalSched.all = gp.context.platform.allNext
			} else {
				previous.context.platform.allNext = gp.context.platform.allNext
			}
			gp.context.platform.allNext = nil
			return
		}
		previous = current
	}
}

func (gp *g) RunqueueNext() *g {
	return gp.context.platform.runqNext
}

func (gp *g) SetRunqueueNext(next *g) {
	gp.context.platform.runqNext = next
}

func (gp *g) RunqueueQueued() bool {
	return gp.context.platform.runqQueued
}

func (gp *g) SetRunqueueQueued(queued bool) {
	gp.context.platform.runqQueued = queued
}

// scanBaremetalStackRoots reports every mapped execution range to the
// collector. The bootstrap stack is mapped regardless of current use, the
// running G scans from the live stack pointer, and each parked G scans from
// its saved frame to its stack top so its locals stay reachable.
func scanBaremetalStackRoots(mark func(start, end uintptr)) {
	lo := uintptr(unsafe.Pointer(&baremetalStackEnd))
	hi := uintptr(unsafe.Pointer(&baremetalStackTop))
	if lo < hi {
		mark(lo, hi)
	}
	running := currentG
	if running != nil && running != &baremetalSched.systemG && running.context != nil {
		ctx := &running.context.platform.context
		if sp := uintptr(baremetalGetSP()); ctx.stackLo < sp && sp < ctx.stackHi {
			mark(sp, ctx.stackHi)
		}
	}
	for current := baremetalSched.all; current != nil; current = current.context.platform.allNext {
		if current == running {
			continue
		}
		ctx := &current.context.platform.context
		if ctx.sp != 0 && ctx.stackLo < ctx.sp && ctx.sp < ctx.stackHi {
			mark(ctx.sp, ctx.stackHi)
		}
	}
}

// SchedulerMultiplexesGoroutinesForTesting reports fiber multiplexing.
func SchedulerMultiplexesGoroutinesForTesting() bool { return true }
