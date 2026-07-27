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
	fiber emscripten.Fiber
}

func (ctx *Context) Init(entry Entry, arg, stack unsafe.Pointer, stackSize uintptr, asyncifyStack unsafe.Pointer, asyncifyStackSize uintptr) {
	ctx.fiber.Init(
		entry,
		arg,
		stack,
		stackSize,
		asyncifyStack,
		asyncifyStackSize,
	)
}

func (ctx *Context) InitCurrent(asyncifyStack unsafe.Pointer, asyncifyStackSize uintptr) {
	ctx.fiber.InitCurrent(asyncifyStack, asyncifyStackSize)
}

func (ctx *Context) Swap(next *Context) {
	ctx.fiber.Swap(&next.fiber)
}
