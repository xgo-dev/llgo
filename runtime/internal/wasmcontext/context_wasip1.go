//go:build llgo && wasip1 && wasm && !llgo.wasi_threads

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

package wasmcontext

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
)

//llgo:type C
type Entry func(unsafe.Pointer) unsafe.Pointer

// Context is the state consumed by Binaryen Asyncify. The first five fields
// have fixed wasm32 offsets shared with context_wasm.S.
type Context struct {
	entry         unsafe.Pointer
	arg           unsafe.Pointer
	asyncifyStack unsafe.Pointer
	asyncifyEnd   unsafe.Pointer
	stackPointer  unsafe.Pointer
	launched      bool
}

func (ctx *Context) Init(entry Entry, arg, stack unsafe.Pointer, stackSize uintptr, asyncifyStack unsafe.Pointer, asyncifyStackSize uintptr) {
	ctx.entry = c.Func(entry)
	ctx.arg = arg
	ctx.asyncifyStack = asyncifyStack
	ctx.asyncifyEnd = unsafe.Add(asyncifyStack, asyncifyStackSize)
	ctx.stackPointer = unsafe.Add(stack, stackSize)
	ctx.launched = false
}

//go:linkname contextLaunch C.__llgo_wasm_context_launch
func contextLaunch(*Context)

//go:linkname contextRewind C.__llgo_wasm_context_rewind
func contextRewind(*Context)

//go:linkname contextUnwind C.__llgo_wasm_context_unwind
func contextUnwind(*Context)
