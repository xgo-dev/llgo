//go:build !baremetal

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

import "github.com/xgo-dev/llgo/runtime/internal/clite/sync/atomic"

const (
	mainExitedBit = uint64(1) << 63
	gCountMask    = mainExitedBit - 1
)

func nextGoid(gp *g) uint64 {
	return atomic.Add(&sched.goidgen, uint64(1))
}

func nextMid(mp *m) int64 {
	return atomic.Add(&sched.midgen, int64(1))
}

func nextPid(pp *p) int32 {
	return atomic.Add(&sched.pidgen, int32(1)) - 1
}

func retainG() {
	atomic.Add(&sched.gstate, uint64(1))
}

// releaseG drops one registered context and returns the remaining count and
// main-exited state from the same atomic result.
func releaseG() (remaining uint64, mainExited bool) {
	// llgo.atomicSub follows LLVM atomicrmw and returns the value before the
	// subtraction, unlike sync/atomic.Add. Convert it to the post-release
	// state before deciding whether this was the last context.
	state := atomic.Sub(&sched.gstate, uint64(1)) - 1
	return state & gCountMask, state&mainExitedBit != 0
}

func markMainExited() {
	atomic.Or(&sched.gstate, mainExitedBit)
}

func gStateForTesting() (count uint64, mainExited bool) {
	state := atomic.Load(&sched.gstate)
	return state & gCountMask, state&mainExitedBit != 0
}

func readgstatus(gp *g) uint32 {
	return atomic.Load(&gp.atomicstatus)
}

func casgstatus(gp *g, oldval, newval uint32) {
	if _, ok := atomic.CompareAndExchange(&gp.atomicstatus, oldval, newval); !ok {
		fatal("runtime: invalid goroutine status transition")
	}
}

func readpstatus(pp *p) uint32 {
	return atomic.Load(&pp.status)
}

func setpstatus(pp *p, status uint32) {
	atomic.Store(&pp.status, status)
}
