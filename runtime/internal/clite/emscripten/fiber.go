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

// Package emscripten exposes the small host ABI needed by the WebAssembly
// execution-context backend.
package emscripten

import c "github.com/goplus/llgo/runtime/internal/clite"

// Fiber is the opaque emscripten_fiber_t storage. The C layout consists of
// eight pointer-sized fields on wasm32.
type Fiber struct {
	_ [8]uintptr
}

//llgo:type C
type FiberEntry func(c.Pointer)

// llgo:link (*Fiber).Init C.emscripten_fiber_init
func (fiber *Fiber) Init(entry FiberEntry, arg, stack c.Pointer, stackSize uintptr, asyncifyStack c.Pointer, asyncifyStackSize uintptr) {
}

// llgo:link (*Fiber).InitCurrent C.emscripten_fiber_init_from_current_context
func (fiber *Fiber) InitCurrent(asyncifyStack c.Pointer, asyncifyStackSize uintptr) {
}

// llgo:link (*Fiber).Swap C.emscripten_fiber_swap
func (fiber *Fiber) Swap(next *Fiber) {
}
