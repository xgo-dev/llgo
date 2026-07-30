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

// These G and P states intentionally keep the values used by the Go runtime.
// Only states reachable by the current backends are defined here.
const (
	_Grunnable = 1
	_Grunning  = 2
	_Gwaiting  = 4
	_Gdead     = 6
)

const (
	_Pidle    = 0
	_Prunning = 1
	_Pdead    = 4
)

// g holds state owned by one LLGo goroutine.
//
// A backend decides the M/P ownership model: pthread gives every G its own M/P,
// while the WebAssembly fiber scheduler shares one M/P across its Gs. Suspended
// execution state is held by the backend-specific runtimeContext.
type g struct {
	defer_ *Defer
	panic_ unsafe.Pointer
	m      *m

	atomicstatus uint32
	goid         uint64
	parentGoid   uint64

	startfn  goroutineFunc
	startarg unsafe.Pointer

	context *runtimeContext

	goexit       bool
	isMain       bool
	paniconfault bool
}

// m represents the host execution resource running Go code.
type m struct {
	curg *g
	p    *p
	id   int64
	os   mOS
}

// p represents the scheduling resources attached to an M.
type p struct {
	id     int32
	status uint32
	m      *m
}
