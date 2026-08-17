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

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/setjmp"
)

// -----------------------------------------------------------------------------

// Defer presents defer statements in a function.
type Defer struct {
	Addr unsafe.Pointer // sigjmpbuf
	Bits uintptr
	Link *Defer
	Reth unsafe.Pointer // native block address or wasm continuation selector
	Rund unsafe.Pointer // native block address or wasm continuation selector
	Args unsafe.Pointer // defer func and args links
}

// panicNode is LLGo's longjmp-backed counterpart of the standard runtime's
// _panic record. prev is the _panic.link equivalent; defer_ records the
// explicit defer frame currently owning the unwind.
type panicNode struct {
	prev   unsafe.Pointer
	arg    any
	defer_ *Defer
}

type recoverState struct {
	frame  unsafe.Pointer
	panic_ unsafe.Pointer
}

// movePanicToDefer advances a panic and the goroutine's unwind cursor to the
// next defer frame. A longjmp can reach a parent's rethrow block before the
// child's normal defer cleanup restores gp.defer_, so both cursors are updated
// here. A panic already unwinding that frame has been replaced by this newer
// panic and must not resume if the newer panic is recovered farther down the
// defer chain.
func (gp *g) movePanicToDefer(node *panicNode, link *Defer) {
	for node.prev != nil {
		prev := (*panicNode)(node.prev)
		if prev.defer_ != link {
			break
		}
		node.prev = prev.prev
		if gp.recoverPanic == unsafe.Pointer(prev) {
			gp.recoverPanic = nil
		}
		c.Free(unsafe.Pointer(prev))
	}
	node.defer_ = link
	gp.defer_ = link
}

// Recover recovers a panic.
func Recover(token unsafe.Pointer) (ret any) {
	gp := getg()
	if token == nil || token != gp.recoverFrame {
		return nil
	}
	ptr := gp.recoverPanic
	if ptr != nil && ptr == gp.panic_ {
		node := (*panicNode)(ptr)
		gp.panic_ = node.prev
		gp.recoverFrame = nil
		gp.recoverPanic = nil
		ret = node.arg
		c.Free(unsafe.Pointer(node))
		if PanicRecovered != nil {
			PanicRecovered()
		}
		// The deferred function that recovers keeps observing the panic
		// stack until it returns (gc runs defers on top of it). The public
		// runtime marks its frame so the pc snapshot stays spliceable that
		// long; the mark reads the frame-pointer chain, which after
		// siglongjmp can reach a stale/unmapped slot, so the guarded read
		// lives in the package that has a page probe (RecoverMark). Nil
		// when lib/runtime is not linked — no snapshot machinery, nothing
		// to mark.
		if RecoverMark != nil {
			RecoverMark()
		}
	}
	return
}

// StartRecoverFrame enables direct recover calls made by the deferred function
// currently being invoked from frame. The eligible panic is captured here,
// while the runtime is still executing the defer frame that selected it. A
// nested defer frame can then distinguish its own panic from an outer panic
// suspended in the direct deferred activation. This is LLGo's explicit-defer
// counterpart to gorecover locating the matching _panic through stack frames.
func StartRecoverFrame(frame unsafe.Pointer) recoverState {
	gp := getg()
	old := recoverState{frame: gp.recoverFrame, panic_: gp.recoverPanic}
	gp.recoverFrame = frame
	gp.recoverPanic = nil
	if ptr := gp.panic_; ptr != nil && (*panicNode)(ptr).defer_ == gp.defer_ {
		gp.recoverPanic = ptr
	}
	return old
}

// EndRecoverFrame restores direct recover permission after a deferred call.
func EndRecoverFrame(state recoverState) {
	gp := getg()
	gp.recoverFrame = state.frame
	gp.recoverPanic = state.panic_
}

// BindRecoverFrame replaces a deferred function's code token with the unique
// stack token for this invocation. A recursive invocation of the same function
// sees the already-bound outer token and is therefore not allowed to recover.
func BindRecoverFrame(function, activation unsafe.Pointer) {
	gp := getg()
	if gp.recoverFrame == function {
		gp.recoverFrame = activation
	}
}

// StartRecoverFrameAlias maps a direct deferred transparent wrapper to the
// wrapped function while the wrapper calls into it. This is the explicit-defer
// analogue of gorecover ignoring abi.FuncIDWrapper frames.
func StartRecoverFrameAlias(from, to unsafe.Pointer) unsafe.Pointer {
	gp := getg()
	old := gp.recoverFrame
	if old == from {
		gp.recoverFrame = to
	}
	return old
}

// EndRecoverFrameAlias restores only the direct-call token. Transparent
// wrappers share their caller's eligible panic rather than opening a nested
// defer scope.
func EndRecoverFrameAlias(frame unsafe.Pointer) {
	getg().recoverFrame = frame
}

// panicIsSuspended reports whether ptr belongs to an outer panic whose direct
// deferred function is still running. It must not resume from one of that
// function's nested defer frames; the outer dispatcher resumes it after the
// direct call returns and restores its caller's recover scope.
func (gp *g) panicIsSuspended(ptr unsafe.Pointer) bool {
	return ptr != nil && ptr == gp.recoverPanic
}

// abortPanics discards panics superseded by Goexit. The standard runtime
// represents Goexit by a goexit _panic that aborts the linked panic records
// while unwinding. LLGo stores Goexit separately on g, so it performs the
// equivalent state transition before starting its longjmp unwind.
func (gp *g) abortPanics() {
	discarded := gp.panic_ != nil
	for gp.panic_ != nil {
		node := (*panicNode)(gp.panic_)
		gp.panic_ = node.prev
		c.Free(unsafe.Pointer(node))
	}
	gp.recoverFrame = nil
	gp.recoverPanic = nil
	if discarded && PanicRecovered != nil {
		PanicRecovered()
	}
}

// RecoverMark, set by the public runtime package, records the recovering
// frame for panic-snapshot splicing.
var RecoverMark func()

const (
	// LLGoFiles: the frame-pointer helper must live in the runtime core —
	// programs that never import "runtime" still link Recover.
	LLGoFiles = "_wrap/fp.c"
)

//go:linkname c_framepointer C.llgo_framepointer
func c_framepointer() unsafe.Pointer

// Panic panics with a value.
func Panic(v any) {
	if v == nil {
		v = &PanicNilError{}
	}
	SavePanicCallerFrames()
	gp := getg()
	ptr := (*panicNode)(c.Malloc(unsafe.Sizeof(panicNode{})))
	ptr.prev = gp.panic_
	ptr.arg = v
	ptr.defer_ = gp.defer_
	gp.panic_ = unsafe.Pointer(ptr)

	Rethrow(gp.defer_)
}
func Goexit() {
	gp := getg()
	gp.abortPanics()
	gp.goexit = true
	Rethrow(gp.defer_)
}

func init() {
	getg().isMain = true
}

// -----------------------------------------------------------------------------

// TracePanic prints panic message.
func TracePanic(v any) {
	print("panic: ")
	printany(v)
	println("\n")
}

// PanicTraceback, when set by the public runtime package, prints a
// Go-style stack trace for an unrecovered panic and reports whether it
// printed anything; the clite frame dump remains the fallback.
var PanicTraceback func(skip int) bool

// PanicRecovered, when set by the public runtime package, releases auxiliary
// traceback state associated with a recovered panic.
var PanicRecovered func()

// PanicSignal converts a hardware signal into the same Go panic the
// legacy signal handler raised.
func PanicSignal(sig int) {
	switch sig {
	case 8: // SIGFPE
		panic(errorString("integer divide by zero"))
	default: // SIGSEGV, SIGBUS
		panic(errorString("invalid memory address or nil pointer dereference"))
	}
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
