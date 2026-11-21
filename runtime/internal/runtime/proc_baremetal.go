//go:build baremetal

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

// Shared single-core G/M/P state helpers for baremetal backends. Object
// addresses are sufficient as live identifiers for these backends and avoid
// adding libatomic solely for diagnostics. Live-G counting lives in the
// backend files: proc_baremetal_count.go for the single-context backends and
// proc_fiber_baremetal.go for the cooperative fiber scheduler.
func nextGoid(gp *g) uint64 {
	return uint64(uintptr(unsafe.Pointer(gp)))
}

func nextMid(mp *m) int64 {
	return int64(uintptr(unsafe.Pointer(mp)))
}

func nextPid(pp *p) int32 {
	return int32((uintptr(unsafe.Pointer(pp)) >> 2) & 0x7fffffff)
}

// Each bare-metal G owns its status transitions in the current backend.
func readgstatus(gp *g) uint32 {
	return gp.atomicstatus
}

func casgstatus(gp *g, oldval, newval uint32) {
	if gp.atomicstatus != oldval {
		fatal("runtime: invalid goroutine status transition")
		return
	}
	gp.atomicstatus = newval
}

func readpstatus(pp *p) uint32 {
	return pp.status
}

func setpstatus(pp *p, status uint32) {
	pp.status = status
}
