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

// Package wasmworkers contains the Emscripten host boundary for the bounded
// WebAssembly M/P worker pool.
package wasmworkers

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/pthread"
)

const LLGoFiles = "_wrap/workers.c"

//llgo:type C
type Entry func(unsafe.Pointer) unsafe.Pointer

func Count() int {
	return int(workerCount())
}

func Current() unsafe.Pointer {
	return workerCurrent()
}

func SetCurrent(worker unsafe.Pointer) {
	workerSetCurrent(worker)
}

func Start(entry Entry, arg unsafe.Pointer, stackSize uintptr) int {
	var attr pthread.Attr
	if ret := attr.Init(); ret != 0 {
		return int(ret)
	}
	if ret := attr.SetDetached(pthread.CreateDetached); ret != 0 {
		_ = attr.Destroy()
		return int(ret)
	}
	if stackSize != 0 {
		if ret := attr.SetStackSize(stackSize); ret != 0 {
			_ = attr.Destroy()
			return int(ret)
		}
	}
	var thread pthread.Thread
	ret := pthread.Create(
		&thread,
		&attr,
		pthread.RoutineFunc(entry),
		c.Pointer(arg),
	)
	_ = attr.Destroy()
	return int(ret)
}

func Wait(addr *uint32, expected uint32, timeoutNanoseconds int64) {
	workerWait(addr, expected, timeoutNanoseconds)
}

func Wake(addr *uint32) {
	workerWake(addr)
}

//go:linkname workerCount C.llgo_wasm_worker_count
func workerCount() c.Int

//go:linkname workerCurrent C.llgo_wasm_worker_current
func workerCurrent() unsafe.Pointer

//go:linkname workerSetCurrent C.llgo_wasm_worker_set_current
func workerSetCurrent(unsafe.Pointer)

//go:linkname workerWait C.llgo_wasm_worker_wait
func workerWait(addr *uint32, expected uint32, timeoutNanoseconds int64) c.Int

//go:linkname workerWake C.llgo_wasm_worker_wake
func workerWake(addr *uint32) c.Int
