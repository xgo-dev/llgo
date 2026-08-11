//go:build !llgo || !js || !wasm

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

import "unsafe"

// The pthread backend keeps its one-to-one M/P pair in the G context without
// exposing those fields to other execution-context backends.
type runtimeContextPlatform struct {
	m m
	p p
}

func newprocBackend(fn goroutineFunc, arg unsafe.Pointer, stackSize uintptr, callergp *g) {
	gp := newproc1(fn, arg, callergp)
	if errno := newm(gp.m, stackSize); errno != 0 {
		releaseG()
		FreeRoot(arg)
		freeRuntimeContext(gp.context)
		panic("runtime: failed to create new OS thread")
	}
}

func initRuntimeContext(ctx *runtimeContext, callergp *g, status uint32) *g {
	gp := initG(ctx, callergp, status)
	mp := &ctx.platform.m
	pp := &ctx.platform.p

	gp.m = mp
	mp.curg = gp
	mp.p = pp
	mp.id = nextMid(mp)

	pp.id = nextPid(pp)
	pstatus := uint32(_Pidle)
	if status == _Grunning {
		pstatus = _Prunning
	}
	setpstatus(pp, pstatus)
	pp.m = mp
	return gp
}

func newm(mp *m, stackSize uintptr) int {
	return newosproc(mp, stackSize)
}

func mstart(arg unsafe.Pointer) unsafe.Pointer {
	mp := (*m)(arg)
	if mp == nil || mp.curg == nil || mp.p == nil {
		fatal("runtime: invalid mstart context")
		return nil
	}
	gp := mp.curg
	pp := mp.p

	setg(gp)
	casgstatus(gp, _Grunnable, _Grunning)
	setpstatus(pp, _Prunning)

	fn, arg := gp.startfn, gp.startarg
	gp.startfn = nil
	gp.startarg = nil
	ret := fn(arg)
	mexit(mp)
	return ret
}

func mexit(mp *m) {
	if mp == nil || mp.curg == nil || mp.p == nil {
		fatal("runtime: invalid mexit context")
		return
	}
	gp := mp.curg
	pp := mp.p
	ctx := gp.context
	ownedByLifecycle := currentGUsesLifecycle()
	if !ownedByLifecycle {
		releaseGAndCheckDeadlock()
	}

	casgstatus(gp, _Grunning, _Gdead)
	setpstatus(pp, _Pdead)

	pp.m = nil
	mp.p = nil
	mp.curg = nil
	gp.m = nil

	setg(nil)
	if !ownedByLifecycle {
		freeRuntimeContext(ctx)
	}
}

func goschedBackend() {
}

// GMPForTesting reports the current runtime ownership graph.
func GMPForTesting() (goid, parentGoid uint64, mid int64, pid int32, gstatus, pstatus uint32, linked bool) {
	gp := getg()
	if gp == nil || gp.m == nil || gp.m.p == nil {
		return
	}
	mp := gp.m
	pp := mp.p
	ctx := gp.context
	return gp.goid, gp.parentGoid, mp.id, pp.id, readgstatus(gp), readpstatus(pp),
		mp.curg == gp && pp.m == mp && ctx != nil &&
			&ctx.g == gp && &ctx.platform.m == mp && &ctx.platform.p == pp
}
