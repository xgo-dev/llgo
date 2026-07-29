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

import "github.com/goplus/llgo/runtime/internal/clite/sync/atomic"

// LLGo's atomic.Add returns the value before the addition. G and M reserve ID
// zero, while P IDs are zero-based like the Go runtime.
func nextGoid(gp *g) uint64 {
	return atomic.Add(&sched.goidgen, uint64(1)) + 1
}

func nextMid(mp *m) int64 {
	return atomic.Add(&sched.midgen, int64(1)) + 1
}

func nextPid(pp *p) int32 {
	return atomic.Add(&sched.pidgen, int32(1))
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
