//go:build llgo && js && wasm

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

	"github.com/goplus/llgo/runtime/internal/clite/emscripten"
)

type Entry = emscripten.FiberEntry

// Context wraps the Emscripten Fiber ABI used by JavaScript hosts.
type Context struct {
	fiber         emscripten.Fiber
	stack         unsafe.Pointer
	asyncifyStack unsafe.Pointer
}

func (ctx *Context) Init(entry Entry, arg unsafe.Pointer, stackSize uintptr, alloc func(uintptr) unsafe.Pointer, free func(unsafe.Pointer)) bool {
	stack, stackSize, asyncifyStack, asyncifySize, ok := allocStorage(stackSize, alloc, free)
	if !ok {
		return false
	}
	ctx.stack = stack
	ctx.asyncifyStack = asyncifyStack
	emscripten.FiberInit(
		&ctx.fiber,
		entry,
		arg,
		stack,
		stackSize,
		asyncifyStack,
		asyncifySize,
	)
	return true
}

func (ctx *Context) InitCurrent(alloc func(uintptr) unsafe.Pointer) bool {
	asyncifyStack := alloc(defaultAsyncifyStackSize)
	if asyncifyStack == nil {
		return false
	}
	ctx.asyncifyStack = asyncifyStack
	emscripten.FiberInitCurrent(&ctx.fiber, asyncifyStack, defaultAsyncifyStackSize)
	return true
}

func (ctx *Context) Ready() bool {
	return ctx.asyncifyStack != nil
}

func (ctx *Context) Close(free func(unsafe.Pointer)) {
	freeStorage(ctx.stack, ctx.asyncifyStack, free)
	*ctx = Context{}
}

func (ctx *Context) Swap(next *Context) {
	emscripten.FiberSwap(&ctx.fiber, &next.fiber)
}
