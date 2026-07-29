//go:build llgo && js && wasm && llgo.wasm_workers

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

	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/pthread/sync"
	"github.com/goplus/llgo/runtime/internal/clite/sync/atomic"
	"github.com/goplus/llgo/runtime/internal/pollbudget"
	"github.com/goplus/llgo/runtime/internal/runqueue"
	"github.com/goplus/llgo/runtime/internal/wasmcontext"
	"github.com/goplus/llgo/runtime/internal/wasmevent"
	"github.com/goplus/llgo/runtime/internal/wasmworkers"
)

const (
	defaultWasmGStackSize        = 64 << 10
	defaultWasmAsyncifyStackSize = 64 << 10
	maxWasmWorkers               = 16
)

type runtimeContextPlatform struct {
	context       wasmcontext.Context
	gcRoot        wasmGCRootContext
	stack         unsafe.Pointer
	asyncifyStack unsafe.Pointer
	owner         *wasmWorker
}

type wasmWorker struct {
	m m
	p p

	lock sync.Mutex
	runq runqueue.Queue[*g]
	wake uint32

	system              wasmcontext.Context
	systemAsyncifyStack unsafe.Pointer
	index               int
	safepointBudget     pollbudget.Budget
	gc                  wasmWorkerGCState
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
		if worker.lock.Init(nil) != 0 {
			fatal("runtime: failed to initialize WebAssembly worker queue")
			return
		}
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
	wasmevent.InstallWake(wakeWasmEventWorker)
	wasmevent.InstallCooperativeSafepoint(CooperativeSafepoint)

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
	initWasmFiber(gp, wasmcontext.Entry(wasmMainStart), unsafe.Pointer(gp), 0)
	initWasmWorkerSystem(worker)
	releaseWasmWorkerG(worker, gp)
	casgstatus(gp, _Grunning, _Grunnable)
	enqueueWasmG(worker, gp)
	runWasmWorker(worker, true)
	c.Exit(0)
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
	setCurrentWasmWorker(worker)
	setg(nil)
	initWasmWorkerSystem(worker)
	runWasmWorker(worker, false)
	return nil
}

func initWasmWorkerSystem(worker *wasmWorker) {
	if worker.systemAsyncifyStack != nil {
		return
	}
	initWasmWorkerGCSystem(worker)
	worker.systemAsyncifyStack = allocWasmStack(defaultWasmAsyncifyStackSize)
	worker.system.InitCurrent(worker.systemAsyncifyStack, defaultWasmAsyncifyStackSize)
}

func runWasmWorker(worker *wasmWorker, stopAtMain bool) {
	for {
		gp := waitWasmWorkerRunq(worker)
		if gp == nil {
			continue
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
	worker := nextWasmWorker()
	gp.context.platform.owner = worker
	initWasmFiber(gp, wasmcontext.Entry(wasmGStart), unsafe.Pointer(gp), stackSize)
	atomic.Add(&wasmMultiSched.active, uint32(1))
	enqueueWasmG(worker, gp)
}

func nextWasmWorker() *wasmWorker {
	index := atomic.Add(&wasmMultiSched.nextWorker, uint32(1))
	return &wasmMultiSched.workers[int(index%uint32(wasmMultiSched.count))]
}

func initWasmFiber(gp *g, entry wasmcontext.Entry, arg unsafe.Pointer, stackSize uintptr) {
	if stackSize == 0 {
		stackSize = defaultWasmGStackSize
	}
	stackSize = alignWasmStackSize(stackSize)
	asyncifySize := uintptr(defaultWasmAsyncifyStackSize)
	if stackSize > asyncifySize {
		asyncifySize = stackSize
	}

	platform := &gp.context.platform
	platform.stack = allocWasmStack(stackSize)
	platform.asyncifyStack = allocWasmStack(asyncifySize)
	platform.context.Init(
		entry,
		arg,
		platform.stack,
		stackSize,
		platform.asyncifyStack,
		asyncifySize,
	)
}

func alignWasmStackSize(size uintptr) uintptr {
	const alignment = uintptr(16)
	return (size + alignment - 1) &^ (alignment - 1)
}

func allocWasmStack(size uintptr) unsafe.Pointer {
	stack := AllocRoot(size)
	if stack == nil {
		panic("runtime: failed to allocate WebAssembly goroutine stack")
	}
	return stack
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
	if platform.stack != nil {
		FreeRoot(platform.stack)
		platform.stack = nil
	}
	if platform.asyncifyStack != nil {
		FreeRoot(platform.asyncifyStack)
		platform.asyncifyStack = nil
	}
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
	gp.startarg = nil
	fn(fnarg)
	finishWasmG(gp)
}

func finishWasmG(gp *g) {
	casgstatus(gp, _Grunning, _Gdead)
	atomic.Add(&wasmMultiSched.active, ^uint32(0))
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
	}
	finishWasmG(gp)
}

func enqueueWasmG(worker *wasmWorker, gp *g) {
	if worker == nil {
		fatal("runtime: enqueue on nil WebAssembly worker")
		return
	}
	worker.lock.Lock()
	ok := worker.runq.Push(gp)
	worker.lock.Unlock()
	if !ok {
		fatal("runtime: invalid WebAssembly run queue insertion")
		return
	}
	wakeWasmWorker(worker)
}

func popWasmWorkerRunq(worker *wasmWorker) *g {
	worker.lock.Lock()
	gp := worker.runq.Pop()
	worker.lock.Unlock()
	return gp
}

func wasmWorkerRunqLen(worker *wasmWorker) uintptr {
	worker.lock.Lock()
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
		wasmWorkerStopForGC(worker)
		if gp := popWasmWorkerRunq(worker); gp != nil {
			return gp
		}
		timeout := int64(-1)
		if worker.index == 0 {
			wasmevent.Poll()
			if gp := popWasmWorkerRunq(worker); gp != nil {
				return gp
			}
			now := wasmevent.Now()
			if deadline, ok := wasmevent.NextDeadline(); ok {
				timeout = deadline - now
				if timeout < 0 {
					timeout = 0
				}
			} else if atomic.Load(&wasmMultiSched.active) == 0 {
				if wasmMultiSched.mainGoexit {
					fatal("no goroutines (main called runtime.Goexit) - deadlock!")
				} else {
					fatal("all goroutines are asleep - deadlock!")
				}
				return nil
			}
		}

		sequence := atomic.Load(&worker.wake)
		if gp := popWasmWorkerRunq(worker); gp != nil {
			return gp
		}
		wasmworkers.Wait(&worker.wake, sequence, timeout)
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
