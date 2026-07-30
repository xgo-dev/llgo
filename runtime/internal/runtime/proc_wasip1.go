//go:build llgo && wasip1 && wasm && !llgo.wasi_threads

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

	"github.com/goplus/llgo/runtime/internal/runqueue"
	"github.com/goplus/llgo/runtime/internal/wasmcontext"
)

const (
	defaultWasmGStackSize        = 64 << 10
	defaultWasmAsyncifyStackSize = 64 << 10
)

type runtimeContextPlatform struct {
	context       wasmcontext.Context
	stack         unsafe.Pointer
	asyncifyStack unsafe.Pointer
}

var wasmSched struct {
	m          m
	p          p
	runq       runqueue.Queue[*g]
	started    bool
	mainExited bool
}

func initRuntimeContext(ctx *runtimeContext, callergp *g, status uint32) *g {
	gp := initG(ctx, callergp, status)
	if status == _Grunning {
		initWasmScheduler(gp)
	}
	return gp
}

func initWasmScheduler(gp *g) {
	if wasmSched.started {
		fatal("runtime: WebAssembly scheduler initialized twice")
		return
	}
	wasmSched.started = true
	mp := &wasmSched.m
	pp := &wasmSched.p
	mp.curg = gp
	mp.p = pp
	mp.id = nextMid(mp)
	pp.id = nextPid(pp)
	setpstatus(pp, _Prunning)
	pp.m = mp
	gp.m = mp
}

//go:linkname wasmMainTask __llgo_wasm_main
func wasmMainTask(unsafe.Pointer) unsafe.Pointer

// RunWasmMain runs package initialization and main.main as the first
// Asyncify task. It remains on the system stack and dispatches one G at a time.
func RunWasmMain() {
	gp := getg()
	if gp == nil || !gp.isMain {
		fatal("runtime: invalid WebAssembly main goroutine")
		return
	}
	initWasmContext(gp, wasmcontext.Entry(wasmMainTask), nil, 0)

	for {
		runWasmContext(gp)
		status := readgstatus(gp)
		if gp.isMain && status == _Grunning {
			casgstatus(gp, _Grunning, _Gdead)
			releaseWasmContext(gp)
			return
		}
		releaseWasmOwnership(gp)
		if status == _Gdead {
			releaseWasmContext(gp)
		}

		gp = waitWasmRunq()
		if gp == nil {
			if wasmSched.mainExited {
				fatal("no goroutines (main called runtime.Goexit) - deadlock!")
			} else {
				fatal("all goroutines are asleep - deadlock!")
			}
			return
		}
	}
}

func runWasmContext(gp *g) {
	if readgstatus(gp) == _Grunnable {
		casgstatus(gp, _Grunnable, _Grunning)
	}
	mp := &wasmSched.m
	pp := &wasmSched.p
	mp.curg = gp
	pp.m = mp
	gp.m = mp
	setg(gp)
	gp.context.platform.context.Resume()
}

func releaseWasmOwnership(gp *g) {
	if gp != nil {
		gp.m = nil
	}
	wasmSched.m.curg = nil
}

func newprocBackend(fn goroutineFunc, arg unsafe.Pointer, stackSize uintptr, callergp *g) {
	gp := newproc1(fn, arg, callergp)
	initWasmContext(gp, wasmcontext.Entry(wasmGStart), unsafe.Pointer(gp), stackSize)
	if !wasmSched.runq.Push(gp) {
		fatal("runtime: invalid run queue insertion")
	}
}

func initWasmContext(gp *g, entry wasmcontext.Entry, arg unsafe.Pointer, stackSize uintptr) {
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

func wasmGStart(arg unsafe.Pointer) unsafe.Pointer {
	gp := (*g)(arg)
	if gp == nil || getg() != gp {
		fatal("runtime: invalid WebAssembly goroutine entry")
		return nil
	}
	fn, fnarg := gp.startfn, gp.startarg
	gp.startfn = nil
	gp.startarg = nil
	ret := fn(fnarg)
	goexitBackend(gp)
	return ret
}

func goschedBackend() {
	gp := getg()
	casgstatus(gp, _Grunning, _Grunnable)
	if !wasmSched.runq.Push(gp) {
		fatal("runtime: invalid run queue insertion")
		return
	}
	gp.context.platform.context.Suspend()
}

func gopark() {
	gp := getg()
	casgstatus(gp, _Grunning, _Gwaiting)
	gp.context.platform.context.Suspend()
}

func goready(gp *g) {
	if gp == nil {
		fatal("runtime: ready of nil goroutine")
		return
	}
	casgstatus(gp, _Gwaiting, _Grunnable)
	if !wasmSched.runq.Push(gp) {
		fatal("runtime: invalid run queue insertion")
	}
}

func goexitBackend(gp *g) {
	casgstatus(gp, _Grunning, _Gdead)
	if gp.isMain {
		wasmSched.mainExited = true
	}
	gp.context.platform.context.Suspend()
	fatal("runtime: resumed dead WebAssembly goroutine")
}

// CurrentGForTesting returns an opaque handle suitable for ReadyForTesting.
func CurrentGForTesting() unsafe.Pointer {
	return unsafe.Pointer(getg())
}

// ParkForTesting parks the current G until another G marks it runnable.
func ParkForTesting() {
	gopark()
}

// ReadyForTesting makes a G previously parked by ParkForTesting runnable.
func ReadyForTesting(handle unsafe.Pointer) {
	goready((*g)(handle))
}

// SchedulerStateForTesting reports single-worker queue and ownership state.
func SchedulerStateForTesting() (runq uintptr, mid int64, pid int32) {
	return wasmSched.runq.Len(), wasmSched.m.id, wasmSched.p.id
}

func GMPForTesting() (goid, parentGoid uint64, mid int64, pid int32, gstatus, pstatus uint32, linked bool) {
	gp := getg()
	if gp == nil || gp.m == nil || gp.m.p == nil {
		return
	}
	mp := gp.m
	pp := mp.p
	ctx := gp.context
	return gp.goid, gp.parentGoid, mp.id, pp.id, readgstatus(gp), readpstatus(pp),
		mp == &wasmSched.m && pp == &wasmSched.p &&
			mp.curg == gp && pp.m == mp && ctx != nil && &ctx.g == gp
}
