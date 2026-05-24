/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/pthread"
	"github.com/goplus/llgo/runtime/internal/clite/setjmp"
)

// -----------------------------------------------------------------------------

// Defer presents defer statements in a function.
type Defer struct {
	Addr unsafe.Pointer // sigjmpbuf
	Bits uintptr
	Link *Defer
	Reth unsafe.Pointer // block address after Rethrow
	Rund unsafe.Pointer // block address after RunDefers
	Args unsafe.Pointer // defer func and args links
}

type panicNode struct {
	prev unsafe.Pointer
	arg  any
}

// Recover recovers a panic.
func Recover(token unsafe.Pointer) (ret any) {
	if token == nil || token != recoverFrameKey.Get() {
		return nil
	}
	ptr := panicKey.Get()
	if ptr != nil {
		node := (*panicNode)(ptr)
		panicKey.Set(node.prev)
		recoverFrameKey.Set(nil)
		ret = node.arg
		c.Free(unsafe.Pointer(node))
	}
	return
}

// StartRecoverFrame enables direct recover calls made by the deferred function
// currently being invoked from frame.
func StartRecoverFrame(frame unsafe.Pointer) unsafe.Pointer {
	old := recoverFrameKey.Get()
	recoverFrameKey.Set(frame)
	return old
}

// EndRecoverFrame restores direct recover permission after a deferred call.
func EndRecoverFrame(frame unsafe.Pointer) {
	recoverFrameKey.Set(frame)
}

// Panic panics with a value.
func Panic(v any) {
	ptr := (*panicNode)(c.Malloc(unsafe.Sizeof(panicNode{})))
	ptr.prev = panicKey.Get()
	ptr.arg = v
	panicKey.Set(unsafe.Pointer(ptr))

	Rethrow((*Defer)(c.GoDeferData()))
}

var (
	panicKey        pthread.Key
	recoverFrameKey pthread.Key
	goexitKey       pthread.Key
	mainThread      pthread.Thread
)

func Goexit() {
	goexitKey.Set(unsafe.Pointer(&goexitKey))
	Rethrow((*Defer)(c.GoDeferData()))
}

func init() {
	panicKey.Create(nil)
	recoverFrameKey.Create(nil)
	goexitKey.Create(nil)
	mainThread = pthread.Self()
}

// -----------------------------------------------------------------------------

// TracePanic prints panic message.
func TracePanic(v any) {
	print("panic: ")
	printany(v)
	println("\n")
}

/*
func stringTracef(fp c.FilePtr, format *c.Char, s String) {
	cs := c.Alloca(uintptr(s.len) + 1)
	c.Fprintf(fp, format, CStrCopy(cs, s))
}
*/

// -----------------------------------------------------------------------------

// New allocates memory and initializes it to zero.
func New(t *Type) unsafe.Pointer {
	return AllocZ(t.Size_)
}

// NewArray allocates memory for an array and initializes it to zero.
func NewArray(t *Type, n int) unsafe.Pointer {
	return AllocZ(uintptr(n) * t.Size_)
}

// -----------------------------------------------------------------------------

// TODO(xsw): check this
// must match declarations in runtime/map.go.
const MaxZero = 1024

var ZeroVal [MaxZero]byte

// -----------------------------------------------------------------------------

type SigjmpBuf struct {
	Unused [setjmp.SigjmpBufSize]byte
}

// -----------------------------------------------------------------------------
