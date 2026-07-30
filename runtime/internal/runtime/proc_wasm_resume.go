//go:build llgo && wasm && llgo.wasm_resume && (js || wasip1) && !(wasip1 && llgo.wasi_threads)

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
	"github.com/goplus/llgo/runtime/internal/runqueue"
	"github.com/goplus/llgo/runtime/internal/wasmresume"
)

type runtimeContextPlatform struct {
	context    wasmresume.Context
	runqNext   *g
	runqQueued bool
	unwind     unsafe.Pointer
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

//go:linkname wasmMainStart C.__llgo_wasm_start.__llgo_wasm_main
func wasmMainStart(*wasmresume.Context, unsafe.Pointer) *wasmresume.Frame

// RunWasmMain runs package initialization and main.main through the resumable
// ABI while the host entry remains on its original stack.
func RunWasmMain() {
	gp := getg()
	if gp == nil || !gp.isMain {
		fatal("runtime: invalid WebAssembly main goroutine")
		return
	}
	gp.context.platform.context.Start(
		wasmMainStart(&gp.context.platform.context, nil),
	)

	for {
		action := runWasmResumeContext(gp)
		status := readgstatus(gp)
		if action == wasmresume.Return {
			if status != _Grunning {
				fatal("runtime: invalid completed WebAssembly goroutine")
				return
			}
			casgstatus(gp, _Grunning, _Gdead)
			status = _Gdead
			if gp.isMain {
				releaseWasmContext(gp)
				return
			}
		} else if action != wasmresume.Suspend || status == _Grunning {
			fatal("runtime: invalid WebAssembly resume action")
			return
		}

		releaseWasmOwnership(gp)
		if status == _Gdead {
			releaseWasmContext(gp)
		}

		gp = wasmSched.runq.Pop()
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

func runWasmResumeContext(gp *g) wasmresume.Action {
	if readgstatus(gp) == _Grunnable {
		casgstatus(gp, _Grunnable, _Grunning)
	}
	mp := &wasmSched.m
	pp := &wasmSched.p
	mp.curg = gp
	pp.m = mp
	gp.m = mp
	setg(gp)

	platform := &gp.context.platform
	unwind := c.AllocaSigjmpBuf()
	previous := platform.unwind
	platform.unwind = unwind
	if c.Sigsetjmp(unwind, 0) != 0 {
		if !platform.context.Unwind(unsafe.Pointer(gp.defer_), FreeRoot) {
			platform.unwind = previous
			if gp.goexit {
				casgstatus(gp, _Grunning, _Gdead)
				if gp.isMain {
					wasmSched.mainExited = true
				}
				return wasmresume.Suspend
			}
			Rethrow(nil)
			return wasmresume.Return
		}
	}
	action := platform.context.Run()
	platform.unwind = previous
	return action
}

func releaseWasmOwnership(gp *g) {
	if gp != nil {
		gp.m = nil
	}
	wasmSched.m.curg = nil
}

func newprocBackend(fn goroutineFunc, arg unsafe.Pointer, _ uintptr, callergp *g) {
	gp := newproc1(fn, arg, callergp)
	gp.startfn = nil
	gp.startarg = nil
	gp.context.platform.context.Start(fn(&gp.context.platform.context, arg))
	if !wasmSched.runq.Push(gp) {
		fatal("runtime: invalid run queue insertion")
	}
}

func releaseWasmContext(gp *g) {
	if gp == nil || gp.context == nil {
		return
	}
	ctx := gp.context
	ctx.platform.context.Close(FreeRoot)
	freeRuntimeContext(ctx)
}

func goschedBackend() {
	gp := getg()
	casgstatus(gp, _Grunning, _Grunnable)
	if !wasmSched.runq.Push(gp) {
		fatal("runtime: invalid run queue insertion")
		return
	}
	wasmresume.SuspendCurrent()
}

func gopark() {
	gp := getg()
	casgstatus(gp, _Grunning, _Gwaiting)
	wasmresume.SuspendCurrent()
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
	wasmresume.SuspendCurrent()
	fatal("runtime: resumed dead WebAssembly goroutine")
}

//go:linkname wasmResumeAlloc __llgo_wasm_resume_alloc
func wasmResumeAlloc(ctx *wasmresume.Context, size, align uintptr) unsafe.Pointer {
	return ctx.AllocateFrame(size, align, AllocRoot)
}

//go:linkname wasmResumeAllocDynamic __llgo_wasm_resume_alloc_dynamic
func wasmResumeAllocDynamic(ctx *wasmresume.Context, size, align uintptr) unsafe.Pointer {
	return ctx.AllocateFrame(size, align, AllocRoot)
}

//go:linkname wasmResumeFree __llgo_wasm_resume_free
func wasmResumeFree(ctx *wasmresume.Context, frame *wasmresume.Frame) {
	ctx.ReleaseFrame(frame, FreeRoot)
}

//go:linkname wasmResumeClose __llgo_wasm_resume_close
func wasmResumeClose(ctx *wasmresume.Context) {
	ctx.Close(FreeRoot)
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
