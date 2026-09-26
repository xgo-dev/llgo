//go:build !windows && (!llgo || !wasm || (wasip1 && llgo.wasi_threads))

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
	"github.com/xgo-dev/llgo/runtime/internal/thread"
)

// Detached pthreads do not leave a native handle owned by the M.
type mOS struct{}

// newosproc provides the current host-thread backend for newm.
func newosproc(mp *m, stackSize uintptr) int {
	return int(thread.CreateDetached(
		pthreadStackSize(stackSize),
		thread.RoutineFunc(wasiGCThreadStart),
		c.Pointer(unsafe.Pointer(mp)),
	))
}

func goexitBackend(gp *g) {
	if gp.isMain {
		parkInitialWasiThread(gp)
	}
	leaveCurrentLocalContext()
	mp := gp.m
	mexit(mp)
	thread.Exit()
}
