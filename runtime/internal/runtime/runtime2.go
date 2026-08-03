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
// Only states reachable by the current 1:1 backend are defined here.
const (
	_Grunnable = 1
	_Grunning  = 2
	_Gdead     = 6
)

const (
	_Pidle    = 0
	_Prunning = 1
	_Pdead    = 4
)

// g holds state owned by one LLGo goroutine.
//
// The current pthread backend gives every G its own M and P. Fields that only
// make sense once LLGo can suspend and resume a G (saved registers, wait state,
// and stack roots) belong here when those facilities are added.
type g struct {
	defer_       *Defer
	panic_       unsafe.Pointer
	panicPCs     panicPCStore
	recoverFrame unsafe.Pointer
	m            *m

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

// m represents the host execution resource running Go code. The platform
// thread handle is deliberately confined to mOS so other backends do not leak
// pthread types into the scheduler core.
type m struct {
	curg *g
	p    *p
	id   int64
	os   mOS
}

// p represents the scheduling resources attached to an M. The pthread backend
// currently binds one P to one M; a later M:N scheduler can retain this object
// while replacing that fixed binding with a P pool and run queues.
type p struct {
	id     int32
	status uint32
	m      *m
}
