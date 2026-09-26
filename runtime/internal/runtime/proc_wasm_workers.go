//go:build llgo && js && wasm && llgo.wasm.workers

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

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/pollbudget"
	"github.com/xgo-dev/llgo/runtime/internal/runqueue"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
	"github.com/xgo-dev/llgo/runtime/internal/wasmcontext"
	"github.com/xgo-dev/llgo/runtime/internal/wasmsync"
	"github.com/xgo-dev/llgo/runtime/internal/wasmworkers"
)

const maxWasmWorkers = 16

func SchedulerMultiplexesGoroutinesForTesting() bool { return true }

// SpawnIndependentWasmG starts work that captures no syscall/js values in a
// worker chosen by the scheduler. Ordinary descendants inherit their
// parent's JS realm because Emscripten emval handles are worker-local.
func SpawnIndependentWasmG(fn func()) {
	gp := getg()
	if gp == nil {
		fatal("runtime: independent goroutine without caller")
		return
	}
	affine := gp.context.platform.jsRealm
	gp.context.platform.jsRealm = false
	go fn()
	gp.context.platform.jsRealm = affine
}

type runtimeContextPlatform struct {
	context    wasmcontext.Context
	gcRoot     wasmGCRootContext
	glsContext LocalContext
	runqNext   *g
	runqQueued bool
	owner      *wasmWorker
	jsRealm    bool
	// Keep runqQueued inside unsafe.Sizeof(runtimeContext{}) on wasm32. LLVM
	// aligns the preceding uint64 G fields more strictly than go/types does.
	layoutEnd [8]byte
}

type wasmWorker struct {
	m m
	p p

	lock wasmsync.Mutex
	runq runqueue.Queue[*g]
	wake uint32

	system          wasmcontext.Context
	systemReady     bool
	localContext    LocalContext
	pollingCallback bool
	jsEvents        []*wasmJSEvent
	syncEventDepth  int
	index           int
	safepointBudget pollbudget.Budget
	gc              wasmWorkerGCState
}

var wasmMultiSched struct {
	workers [maxWasmWorkers]wasmWorker
	count   int

	nextWorker uint32
	active     uint32
	started    bool

	mainReturned bool
	mainGoexit   bool
}

func runtimeContextAllocSize() uintptr {
	const alignment = uintptr(unsafe.Sizeof(uint64(0)))
	return (unsafe.Sizeof(runtimeContext{}) + alignment - 1) &^ (alignment - 1)
}

func initRuntimeContext(ctx *runtimeContext, callergp *g, status uint32) *g {
	gp := initG(ctx, callergp, status)
	if wasmGCRootEnabled {
		registerWasmGCRoot(&ctx.platform.gcRoot, status == _Grunning)
	}
	if status == _Grunning {
		initWasmScheduler(gp)
	}
	return gp
}

func initWasmScheduler(gp *g) {
	if wasmMultiSched.started {
		fatal("runtime: WebAssembly scheduler initialized twice")
		return
	}
	count := wasmworkers.Count()
	if count < 2 || count > maxWasmWorkers {
		fatal("runtime: invalid WebAssembly worker count")
		return
	}
	wasmMultiSched.count = count
	// atomic.Add returns the pre-increment value, so the first child starts
	// away from the main worker.
	wasmMultiSched.nextWorker = 1
	wasmMultiSched.active = 1

	for i := 0; i < count; i++ {
		worker := &wasmMultiSched.workers[i]
		worker.index = i
		worker.safepointBudget = pollbudget.New(wasmSafepointQuantum)
		worker.m.id = nextMid(&worker.m)
		worker.m.p = &worker.p
		worker.p.id = nextPid(&worker.p)
		worker.p.m = &worker.m
		setpstatus(&worker.p, _Prunning)
	}

	mainWorker := &wasmMultiSched.workers[0]
	setCurrentWasmWorker(mainWorker)
	bindWasmWorkerG(mainWorker, gp)
	gp.context.platform.owner = mainWorker
	wasmMultiSched.started = true

	for i := 1; i < count; i++ {
		worker := &wasmMultiSched.workers[i]
		if errno := wasmworkers.Start(wasmworkers.Entry(wasmWorkerStart), unsafe.Pointer(worker), 0); errno != 0 {
			fatal("runtime: failed to start WebAssembly worker")
			return
		}
	}
}

//go:linkname wasmMainTask __llgo_wasm_main
func wasmMainTask(unsafe.Pointer) unsafe.Pointer

func RunWasmMain() {
	gp := getg()
	worker := currentWasmWorker()
	if gp == nil || !gp.isMain || worker == nil || worker.index != 0 {
		fatal("runtime: invalid WebAssembly main goroutine")
		return
	}
	adoptWasmWorkerLocalContext(&worker.localContext)
	if !initWasmFiber(gp, wasmcontext.Entry(wasmMainStart), unsafe.Pointer(gp), wasmMainStackSize) {
		panic("runtime: failed to allocate WebAssembly main stack")
	}
	initWasmWorkerSystem(worker)
	releaseWasmWorkerG(worker, gp)
	casgstatus(gp, _Grunning, _Grunnable)
	enqueueWasmG(worker, gp)
	runWasmWorker(worker, true)
	if wasmMultiSched.mainReturned {
		c.Exit(0)
	}
	// The proxied main pthread must survive its initial entry returning. Later
	// idle periods are entered through llgoWasmWorkerResume and return normally
	// to the async-wait callback instead of repeatedly abandoning C frames.
	wasmworkers.Suspend()
}

func wasmMainStart(arg unsafe.Pointer) {
	gp := (*g)(arg)
	if gp == nil || getg() != gp {
		fatal("runtime: invalid WebAssembly main entry")
		return
	}
	wasmMainTask(nil)
	wasmMultiSched.mainReturned = true
	finishWasmG(gp)
}

func wasmWorkerStart(arg unsafe.Pointer) unsafe.Pointer {
	worker := (*wasmWorker)(arg)
	if worker == nil || worker.index == 0 {
		fatal("runtime: invalid WebAssembly worker entry")
		return nil
	}
	// Goroutines interleave on this native worker, so generated GLS/TLS access
	// needs one long-lived locality owner rather than a goroutine entry frame.
	// Keep it outside this entry stack: an idle pthread unwinds back to its JS
	// event loop and later resumes through llgo_wasm_worker_resume.
	adoptWasmWorkerLocalContext(&worker.localContext)
	setCurrentWasmWorker(worker)
	setg(nil)
	initWasmWorkerSystem(worker)
	runWasmWorker(worker, false)
	// Keep the pthread alive after its one native entry. Resume callbacks use
	// the ordinary return path, which lets Emscripten restore their C stack.
	wasmworkers.Suspend()
	return nil // unreachable
}

//export llgo_wasm_worker_resume
func llgoWasmWorkerResume(arg unsafe.Pointer) {
	worker := (*wasmWorker)(arg)
	if worker == nil || !worker.systemReady {
		fatal("runtime: invalid WebAssembly worker resume")
		return
	}
	setCurrentWasmWorker(worker)
	adoptWasmWorkerLocalContext(&worker.localContext)
	setg(nil)
	worker.system.ResetCurrent()
	resumeWasmWorkerGCSystem(worker)
	runWasmWorker(worker, worker.index == 0)
	if worker.index == 0 && wasmMultiSched.mainReturned {
		c.Exit(0)
	}
}

func initWasmWorkerSystem(worker *wasmWorker) {
	if worker.systemReady {
		return
	}
	if !worker.system.InitCurrent(AllocRoot) {
		panic("runtime: failed to allocate WebAssembly system context")
	}
	worker.systemReady = true
	initWasmWorkerGCSystem(worker)
}

func runWasmWorker(worker *wasmWorker, stopAtMain bool) {
	for {
		gp := waitWasmWorkerRunq(worker)
		if gp == nil {
			return
		}
		casgstatus(gp, _Grunnable, _Grunning)
		runWasmG(worker, gp)
		wasmWorkerStopForGC(worker)

		if readgstatus(gp) != _Gdead {
			continue
		}
		isMain := gp.isMain
		releaseWasmContext(gp)
		if isMain && stopAtMain {
			if wasmMultiSched.mainReturned {
				return
			}
			if wasmMultiSched.mainGoexit && atomic.Load(&wasmMultiSched.active) == 0 {
				fatal("no goroutines (main called runtime.Goexit) - deadlock!")
				return
			}
		}
	}
}

func runWasmG(worker *wasmWorker, gp *g) {
	for {
		bindWasmWorkerG(worker, gp)
		setg(gp)
		worker.system.Swap(
			&gp.context.platform.context,
			wasmGCRootPointer(&gp.context.platform.gcRoot),
		)
		setg(nil)
		releaseWasmWorkerG(worker, gp)
		if readgstatus(gp) != _Grunning {
			return
		}
		if !wasmWorkerStopForGC(worker) {
			fatal("runtime: running WebAssembly goroutine returned without a GC request")
			return
		}
	}
}

func bindWasmWorkerG(worker *wasmWorker, gp *g) {
	worker.m.curg = gp
	gp.m = &worker.m
}

func releaseWasmWorkerG(worker *wasmWorker, gp *g) {
	if gp != nil {
		gp.m = nil
	}
	worker.m.curg = nil
}

func newprocBackend(fn goroutineFunc, arg unsafe.Pointer, stackSize uintptr, callergp *g) {
	gp := newproc1(fn, arg, callergp)
	var worker *wasmWorker
	current := currentWasmWorker()
	if callergp == nil || current != nil &&
		(current.pollingCallback || len(current.jsEvents) != 0 || current.syncEventDepth != 0 ||
			callergp != nil && callergp.context.platform.jsRealm) {
		// Host callbacks carry thread-local JavaScript handles. Dispatch the G
		// from the same physical worker/JS realm that received the callback.
		worker = current
	}
	if worker == nil {
		worker = nextWasmWorker()
	}
	gp.context.platform.owner = worker
	if callergp != nil {
		gp.context.platform.jsRealm = callergp.context.platform.jsRealm
	}
	if !initWasmFiber(gp, wasmcontext.Entry(wasmGStart), unsafe.Pointer(gp), stackSize) {
		releaseG()
		releaseStartArg(gp)
		freeRuntimeContext(gp.context)
		panic("runtime: failed to allocate WebAssembly goroutine stack")
	}
	atomic.Add(&wasmMultiSched.active, uint32(1))
	enqueueWasmG(worker, gp)
}

func nextWasmWorker() *wasmWorker {
	index := atomic.Add(&wasmMultiSched.nextWorker, uint32(1))
	return &wasmMultiSched.workers[int(index%uint32(wasmMultiSched.count))]
}

func initWasmFiber(gp *g, entry wasmcontext.Entry, arg unsafe.Pointer, stackSize uintptr) bool {
	return gp.context.platform.context.Init(
		entry,
		arg,
		stackSize,
		AllocRoot,
		FreeRoot,
	)
}

func releaseWasmContext(gp *g) {
	if gp == nil || gp.context == nil {
		return
	}
	ctx := gp.context
	platform := &ctx.platform
	if wasmGCRootEnabled {
		unregisterWasmGCRoot(&platform.gcRoot)
	}
	releaseGoroutineLocalBlocks(&platform.glsContext)
	platform.context.Close(FreeRoot)
	freeRuntimeContext(ctx)
}

func wasmGStart(arg unsafe.Pointer) {
	gp := (*g)(arg)
	if gp == nil || getg() != gp {
		fatal("runtime: invalid WebAssembly goroutine entry")
		return
	}
	fn, fnarg := gp.startfn, gp.startarg
	gp.startfn = nil
	fn(fnarg)
	finishWasmG(gp)
}

func finishWasmG(gp *g) {
	releaseStartArg(gp)
	casgstatus(gp, _Grunning, _Gdead)
	atomic.Add(&wasmMultiSched.active, ^uint32(0))
	releaseGAndCheckDeadlock()
	wakeWasmEventWorker()
	worker := gp.context.platform.owner
	gp.context.platform.context.Swap(
		&worker.system,
		wasmWorkerSystemRootPointer(worker),
	)
	fatal("runtime: resumed dead WebAssembly goroutine")
}

func goschedBackend() {
	gp := getg()
	worker := currentWasmWorker()
	casgstatus(gp, _Grunning, _Grunnable)
	enqueueWasmG(worker, gp)
	gp.context.platform.context.Swap(
		&worker.system,
		wasmWorkerSystemRootPointer(worker),
	)
}

func gopark() {
	gp := getg()
	parkWasmG(gp)
}

func parkWasmG(gp *g) {
	casgstatus(gp, _Grunning, _Gwaiting)
	atomic.Add(&wasmMultiSched.active, ^uint32(0))
	wakeWasmEventWorker()
	worker := gp.context.platform.owner
	gp.context.platform.context.Swap(
		&worker.system,
		wasmWorkerSystemRootPointer(worker),
	)
}

func goready(gp *g) {
	if gp == nil {
		fatal("runtime: ready of nil goroutine")
		return
	}
	casgstatus(gp, _Gwaiting, _Grunnable)
	atomic.Add(&wasmMultiSched.active, uint32(1))
	enqueueWasmG(gp.context.platform.owner, gp)
}

func goexitBackend(gp *g) {
	if gp.isMain {
		wasmMultiSched.mainGoexit = true
		markMainExited()
	}
	finishWasmG(gp)
}

func enqueueWasmG(worker *wasmWorker, gp *g) {
	if worker == nil {
		fatal("runtime: enqueue on nil WebAssembly worker")
		return
	}
	worker.lock.Lock(CooperativeSafepoint)
	ok := worker.runq.Push(gp)
	worker.lock.Unlock()
	if !ok {
		fatal("runtime: invalid WebAssembly run queue insertion")
		return
	}
	wakeWasmWorker(worker)
}

func popWasmWorkerRunq(worker *wasmWorker) *g {
	worker.lock.Lock(CooperativeSafepoint)
	gp := worker.runq.Pop()
	worker.lock.Unlock()
	return gp
}

func wasmWorkerRunqLen(worker *wasmWorker) uintptr {
	worker.lock.Lock(CooperativeSafepoint)
	size := worker.runq.Len()
	worker.lock.Unlock()
	return size
}

func wakeWasmWorker(worker *wasmWorker) {
	atomic.Add(&worker.wake, uint32(1))
	wasmworkers.Wake(&worker.wake)
}

func wakeWasmEventWorker() {
	if wasmMultiSched.count != 0 {
		wakeWasmWorker(&wasmMultiSched.workers[0])
	}
}

func waitWasmWorkerRunq(worker *wasmWorker) *g {
	for {
		// Snapshot the wake sequence before checking either the GC request or
		// the run queue. A producer that races with those checks then changes
		// the sequence, so the futex wait returns instead of losing the wake.
		sequence := atomic.Load(&worker.wake)
		wasmWorkerStopForGC(worker)
		if gp := popWasmWorkerRunq(worker); gp != nil {
			return gp
		}
		hooks := loadWasmEventHooks()
		hooks.pollCallbackEvents(worker)
		if gp := popWasmWorkerRunq(worker); gp != nil {
			return gp
		}
		timeout := int64(-1)
		asyncWait := hooks.pollCallbacks != nil
		if worker.index == 0 {
			hooks.pollTimerEvents()
			if gp := popWasmWorkerRunq(worker); gp != nil {
				return gp
			}
			hasWaitSource := hooks.pollCallbacks != nil
			if hooks.timerWait != nil {
				if wait, active := hooks.timerWait(); active {
					hasWaitSource = true
					asyncWait = true
					if wait > uint64(^uint64(0)>>1) {
						timeout = int64(^uint64(0) >> 1)
					} else {
						timeout = int64(wait)
					}
				}
			}
			if !hasWaitSource && atomic.Load(&wasmMultiSched.active) == 0 {
				if wasmMultiSched.mainGoexit {
					fatal("no goroutines (main called runtime.Goexit) - deadlock!")
				} else {
					fatal("all goroutines are asleep - deadlock!")
				}
				return nil
			}
		}

		if gp := popWasmWorkerRunq(worker); gp != nil {
			return gp
		}
		// A synchronous JavaScript callback still owns the caller's JS stack.
		// Do not unwind the pthread entry while its Go handler is parked; a
		// cross-worker channel wake or a Go timer must resume it in place.
		if len(worker.jsEvents) != 0 || worker.syncEventDepth != 0 {
			asyncWait = false
		}
		if !asyncWait {
			// Pure Go work can block in the futex without starving a host event.
			// Keeping this path synchronous avoids one JavaScript callback per
			// channel handoff or mutex wake.
			wasmworkers.Wait(&worker.wake, sequence, timeout)
			continue
		}
		if wasmworkers.ArmWait(&worker.wake, sequence, timeout, unsafe.Pointer(worker)) {
			// A host callback can be queued after the poll above but before
			// ArmWait publishes this worker's JavaScript wake function. Such a
			// callback could not wake the worker when it was queued. Poll once
			// more after the wait is armed; enqueueing its dispatch G changes the
			// wake sequence and completes the async wait. Callbacks arriving after
			// this point observe the published wake function themselves.
			hooks.pollCallbackEvents(worker)
			// A pthread cannot synchronously block and still receive JavaScript
			// callbacks. Keep the async wait alive and discard roots for the
			// system stack being retired. The initial pthread entry unwinds once
			// to stay alive; later resume callbacks return through their normal C
			// epilogues when runWasmWorker observes this nil result.
			suspendWasmWorkerGCSystem(worker)
			return nil
		}
	}
}

// CurrentGForTesting returns an opaque handle suitable for ReadyForTesting.
func CurrentGForTesting() unsafe.Pointer {
	return unsafe.Pointer(getg())
}

func ParkForTesting() {
	gopark()
}

func ReadyForTesting(handle unsafe.Pointer) {
	goready((*g)(handle))
}

func SchedulerStateForTesting() (runq uintptr, mid int64, pid int32) {
	worker := currentWasmWorker()
	if worker == nil {
		return
	}
	for i := 0; i < wasmMultiSched.count; i++ {
		runq += wasmWorkerRunqLen(&wasmMultiSched.workers[i])
	}
	return runq, worker.m.id, worker.p.id
}

func GMPForTesting() (goid, parentGoid uint64, mid int64, pid int32, gstatus, pstatus uint32, linked bool) {
	gp := getg()
	worker := currentWasmWorker()
	if gp == nil || worker == nil || gp.m == nil || gp.m.p == nil {
		return
	}
	mp := gp.m
	pp := mp.p
	ctx := gp.context
	return gp.goid, gp.parentGoid, mp.id, pp.id, readgstatus(gp), readpstatus(pp),
		mp == &worker.m && pp == &worker.p && mp.curg == gp &&
			pp.m == mp && ctx != nil && &ctx.g == gp &&
			ctx.platform.owner == worker
}
