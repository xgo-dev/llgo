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

// Recover recovers a panic.
func Recover() (ret any) {
	ptr := excepKey.Get()
	if ptr != nil {
		excepKey.Set(nil)
		ret = *(*any)(ptr)
		c.Free(ptr)
	}
	return
}

// Panic panics with a value.
func Panic(v any) {
	SavePanicCallerFrames()
	ptr := c.Malloc(unsafe.Sizeof(v))
	*(*any)(ptr) = v
	excepKey.Set(ptr)

	Rethrow((*Defer)(c.GoDeferData()))
}

var (
	excepKey   pthread.Key
	goexitKey  pthread.Key
	mainThread pthread.Thread
)

func Goexit() {
	goexitKey.Set(unsafe.Pointer(&goexitKey))
	Rethrow((*Defer)(c.GoDeferData()))
}

func init() {
	excepKey.Create(nil)
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
