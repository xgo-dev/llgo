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

package ssa

import (
	"go/token"
	"go/types"
	"unsafe"

	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

// func setjmp(env unsafe.Pointer) c.Int
func (p Program) tySetjmp() *types.Signature {
	if p.setjmpTy == nil {
		paramPtr := types.NewParam(token.NoPos, nil, "", p.VoidPtr().raw.Type)
		paramCInt := types.NewParam(token.NoPos, nil, "", p.CInt().raw.Type)
		params := types.NewTuple(paramPtr)
		results := types.NewTuple(paramCInt)
		p.setjmpTy = types.NewSignatureType(nil, nil, nil, params, results, false)
	}
	return p.setjmpTy
}

// func longjmp(env unsafe.Pointer, retval c.Int)
func (p Program) tyLongjmp() *types.Signature {
	if p.longjmpTy == nil {
		paramPtr := types.NewParam(token.NoPos, nil, "", p.VoidPtr().raw.Type)
		paramCInt := types.NewParam(token.NoPos, nil, "", p.CInt().raw.Type)
		params := types.NewTuple(paramPtr, paramCInt)
		p.longjmpTy = types.NewSignatureType(nil, nil, nil, params, nil, false)
	}
	return p.longjmpTy
}

// func() unsafe.Pointer
func (p Program) tyStacksave() *types.Signature {
	if p.stackSaveTy == nil {
		paramPtr := types.NewParam(token.NoPos, nil, "", p.VoidPtr().raw.Type)
		params := types.NewTuple(paramPtr)
		p.stackSaveTy = types.NewSignatureType(nil, nil, nil, nil, params, false)
	}
	return p.stackSaveTy
}

func (b Builder) AllocaSigjmpBuf() Expr {
	prog := b.Prog
	sigjmpBufTy := prog.rtType("SigjmpBuf") // Get type from runtime (target architecture)
	n := prog.SizeOf(sigjmpBufTy)           // Get size for target architecture
	size := prog.IntVal(n, prog.Uintptr())
	return b.Alloca(size)
}

// declare ptr @llvm.stacksave.p0()
func (b Builder) StackSave() Expr {
	fn := b.Pkg.cFunc("llvm.stacksave", b.Prog.tyStacksave())
	return b.InlineCall(fn)
}

// addReturnsTwiceAttr adds the returns_twice attribute to a function.
// This attribute tells LLVM that the function returns twice (once directly, once via longjmp),
// ensuring that variables used across setjmp/longjmp boundaries are placed in
// callee-saved registers or spilled to stack, preventing them from becoming invalid
// after longjmp returns (e.g., the caller's DeferFrame pointer).
func (b Builder) addReturnsTwiceAttr(fn Expr) {
	ctx := b.Pkg.mod.Context()
	attr := ctx.CreateEnumAttribute(llvm.AttributeKindID("returns_twice"), 0)
	fn.impl.AddFunctionAttr(attr)
}

func (b Builder) Sigsetjmp(jb, savemask Expr) Expr {
	// Use setjmp for wasm or targets specified via -target flag (baremetal, etc.)
	if b.Prog.target.GOARCH == "wasm" || b.Prog.target.Target != "" {
		return b.Setjmp(jb)
	}
	fn := b.Pkg.rtFunc("Sigsetjmp")
	b.addReturnsTwiceAttr(fn)
	return b.Call(fn, jb, savemask)
}

func (b Builder) Siglongjmp(jb, retval Expr) {
	// Use longjmp for wasm or targets specified via -target flag (baremetal, etc.)
	if b.Prog.target.GOARCH == "wasm" || b.Prog.target.Target != "" {
		b.Longjmp(jb, retval)
		return
	}
	fn := b.Pkg.rtFunc("Siglongjmp") // TODO(xsw): mark as noreturn
	b.Call(fn, jb, retval)
	// b.Unreachable()
}

func (b Builder) Setjmp(jb Expr) Expr {
	fn := b.Pkg.cFunc("setjmp", b.Prog.tySetjmp())
	b.addReturnsTwiceAttr(fn)
	return b.Call(fn, jb)
}

func (b Builder) Longjmp(jb, retval Expr) {
	fn := b.Pkg.cFunc("longjmp", b.Prog.tyLongjmp())
	b.Call(fn, jb, retval)
	// b.Unreachable()
}

// -----------------------------------------------------------------------------

func (p Function) deferInitBuilder() (b Builder, next BasicBlock) {
	b = p.NewBuilder()
	next = b.setBlockMoveLast(p.blks[0])
	p.blks[0].last = next.last
	return
}

type aDefer struct {
	nextBit   int          // next defer bit
	data      Expr         // pointer to runtime.Defer
	bitsPtr   Expr         // pointer to defer bits
	rethPtr   Expr         // next block of Rethrow
	rundPtr   Expr         // next block of RunDefers
	argsPtr   Expr         // func and args links
	procBlk   BasicBlock   // deferProc block
	panicBlk  BasicBlock   // panic block (runDefers and rethrow)
	rundsNext []BasicBlock // next blocks of RunDefers
	// loopDrainerGenerated marks whether we've already generated the loop-defer
	// drain loop for the current contiguous run of DeferInLoop statements (when
	// walking defers in reverse order in endDefer).
	loopDrainerGenerated bool
	loopCases            []loopDeferCase
	stmts                []func(bits Expr)
}

// loopDeferCase represents a defer statement inside a loop.
// The id uniquely identifies the defer call site for dispatch during drain.
// typ is the node struct type needed to decode the linked-list node.
type loopDeferCase struct {
	id        Expr
	typ       Type
	fn        Expr
	args      []Expr
	buildCall func(Builder, Expr, ...Expr) Expr
}

const (
	// 0: addr sigjmpbuf
	// 1: bits uintptr
	// 2: link *Defer
	// 3: reth voidptr: block address after Rethrow
	// 4: rund voidptr: block address after RunDefers
	// 5: func and args links
	deferSigjmpbuf = iota
	deferBits
	deferLink
	deferRethrow
	deferRunDefers
	deferArgs
)

func (b Builder) getDefer(kind DoAction) *aDefer {
	if b.Func.recov == nil {
		// b.Func.recov maybe nil in ssa.NaiveForm
		return nil
	}
	self := b.Func
	if self.defer_ == nil {
		// TODO(xsw): check if in pkg.init
		var next, panicBlk BasicBlock
		if kind != DeferAlways {
			b, next = self.deferInitBuilder()
		}

		blks := self.MakeBlocks(2)
		procBlk, rethrowBlk := blks[0], blks[1]
		deferState, link, retval := b.initDeferState(procBlk, rethrowBlk)
		czero := b.Prog.IntVal(0, b.Prog.CInt())
		if kind != DeferAlways {
			panicBlk = self.MakeBlock()
		} else {
			blks = self.MakeBlocks(2)
			next, panicBlk = blks[0], blks[1]
		}
		b.If(b.BinOp(token.EQL, retval, czero), next, panicBlk)
		deferState.panicBlk = panicBlk

		b.SetBlockEx(rethrowBlk, AtEnd, false) // rethrow
		b.Call(b.Pkg.rtFunc("Rethrow"), link)
		b.Jump(self.recov)

		if kind == DeferAlways {
			b.SetBlockEx(next, AtEnd, false)
			b.blk.last = next.last
		}
	}
	return self.defer_
}

// DeferData returns the defer data (*runtime.Defer).
func (b Builder) DeferData() Expr {
	return b.Call(b.Pkg.rtFunc("GetThreadDefer"))
}

func (b Builder) getDeferInCurrentBlock() *aDefer {
	if b.Func.recov == nil {
		return nil
	}
	self := b.Func
	if self.defer_ != nil {
		return self.defer_
	}

	// Range-over-func yield bodies need an addressable defer stack in the
	// current block so the synthetic frame can push into its enclosing owner.
	logicalBlk := b.blk
	blks := self.MakeBlocks(4)
	procBlk, rethrowBlk, next, panicBlk := blks[0], blks[1], blks[2], blks[3]
	deferState, link, retval := b.initDeferState(procBlk, rethrowBlk)
	czero := b.Prog.IntVal(0, b.Prog.CInt())
	b.If(b.BinOp(token.EQL, retval, czero), next, panicBlk)
	deferState.panicBlk = panicBlk

	b.SetBlockEx(rethrowBlk, AtEnd, false)
	b.Call(b.Pkg.rtFunc("Rethrow"), link)
	b.Jump(self.recov)

	b.SetBlockEx(next, AtEnd, false)
	if logicalBlk != nil {
		logicalBlk.last = next.last
	}
	b.blk = logicalBlk
	return self.defer_
}

func (b Builder) initDeferState(procBlk, rethrowBlk BasicBlock) (*aDefer, Expr, Expr) {
	self := b.Func
	prog := b.Prog
	zero := prog.Val(uintptr(0))
	link := b.Call(b.Pkg.rtFunc("GetThreadDefer"))
	jb := b.AllocaSigjmpBuf()
	ptr := b.aggregateAllocU(prog.Defer(), jb.impl, zero.impl, link.impl, procBlk.Addr().impl)
	deferData := Expr{ptr, prog.DeferPtr()}
	b.Call(b.Pkg.rtFunc("SetThreadDefer"), deferData)
	bitsPtr := b.FieldAddr(deferData, deferBits)
	rethPtr := b.FieldAddr(deferData, deferRethrow)
	rundPtr := b.FieldAddr(deferData, deferRunDefers)
	argsPtr := b.FieldAddr(deferData, deferArgs)
	// Initialize the args list so later guards (e.g. DeferAlways/DeferInLoop)
	// can safely detect an empty chain without a prior push.
	b.Store(argsPtr, prog.Nil(prog.VoidPtr()))

	czero := prog.IntVal(0, prog.CInt())
	retval := b.Sigsetjmp(jb, czero)

	self.defer_ = &aDefer{
		data:      deferData,
		bitsPtr:   bitsPtr,
		rethPtr:   rethPtr,
		rundPtr:   rundPtr,
		argsPtr:   argsPtr,
		procBlk:   procBlk,
		rundsNext: []BasicBlock{rethrowBlk},
	}
	if len(self.pendingLoopCases) > 0 {
		self.defer_.loopCases = append(self.defer_.loopCases, self.pendingLoopCases...)
		self.pendingLoopCases = nil
	}
	return self.defer_, link, retval
}

// DeferStack returns a stable handle to the current function's explicit
// loop-defer list.
func (b Builder) DeferStack() Expr {
	self := b.getDeferInCurrentBlock()
	if self == nil {
		return b.Prog.Nil(b.Prog.VoidPtr())
	}
	return b.PtrCast(b.Prog.VoidPtr(), self.argsPtr)
}

// DeferStackDrain registers a drain point for defers pushed through an
// explicit stack, such as range-over-func yield bodies.
func (b Builder) DeferStackDrain() {
	self := b.getDefer(DeferInLoop)
	if self == nil {
		return
	}
	b.appendLoopDeferDrainer(self)
}

// Defer emits a defer instruction.
func (b Builder) Defer(kind DoAction, fn Expr, buildCall func(Builder, Expr, ...Expr) Expr, args ...Expr) {
	dbgInstrCall("Defer", fn, args)
	var prog Program
	var nextbit Expr
	var self = b.getDefer(kind)
	if self == nil {
		return
	}
	id := b.Prog.Val(b.Func.nextDeferID)
	b.Func.nextDeferID++
	switch kind {
	case DeferInCond:
		prog = b.Prog
		next := self.nextBit
		if uintptr(next) >= unsafe.Sizeof(uintptr(0))*8 {
			panic("too many conditional defers")
		}
		self.nextBit++
		bits := b.Load(self.bitsPtr)
		nextbit = prog.Val(uintptr(1 << next))
		b.Store(self.bitsPtr, b.BinOp(token.OR, bits, nextbit))
	case DeferAlways:
		// nothing to do
	case DeferInLoop:
		// Loop defers rely on a dedicated drain loop inserted below.
	}
	typ := b.saveDeferArgs(self, kind, id, fn, args)
	if kind == DeferInLoop {
		loopCase := loopDeferCase{id: id, typ: typ, fn: fn, args: args, buildCall: buildCall}
		self.loopCases = append(self.loopCases, loopCase)
	}
	b.appendDeferStmt(self, kind, typ, buildCall, fn, args, nextbit)
}

// DeferTo emits a defer instruction into an explicit runtime defer stack.
func (b Builder) DeferTo(owner Function, stack Expr, fn Expr, buildCall func(Builder, Expr, ...Expr) Expr, args ...Expr) {
	if debugInstr {
		logCall("DeferTo", fn, args)
	}
	if owner == nil {
		b.Defer(DeferInLoop, fn, buildCall, args...)
		return
	}
	self := owner.defer_
	id := b.Prog.Val(owner.nextDeferID)
	owner.nextDeferID++
	argsPtr := b.PtrCast(b.Prog.Pointer(b.Prog.VoidPtr()), stack)
	typ := b.saveDeferArgsTo(argsPtr, DeferInLoop, id, fn, args)
	loopCase := loopDeferCase{
		id:        id,
		typ:       typ,
		fn:        fn,
		args:      args,
		buildCall: buildCall,
	}
	if self == nil {
		owner.pendingLoopCases = append(owner.pendingLoopCases, loopCase)
		return
	}
	self.loopCases = append(self.loopCases, loopCase)
}

func (b Builder) appendDeferStmt(self *aDefer, kind DoAction, typ Type, buildCall func(Builder, Expr, ...Expr) Expr, fn Expr, args []Expr, nextbit Expr) {
	self.stmts = append(self.stmts, func(bits Expr) {
		switch kind {
		case DeferInCond:
			// Leaving a run of loop defers; allow the next loop-defer statement
			// (earlier in source order) to generate its own drainer.
			self.loopDrainerGenerated = false
			prog := b.Prog
			zero := prog.Val(uintptr(0))
			has := b.BinOp(token.NEQ, b.BinOp(token.AND, bits, nextbit), zero)
			b.IfThen(has, func() {
				b.callDefer(self, typ, buildCall, fn, args)
			})
		case DeferAlways:
			// Leaving a run of loop defers; allow the next loop-defer statement
			// (earlier in source order) to generate its own drainer.
			self.loopDrainerGenerated = false
			b.callDefer(self, typ, buildCall, fn, args)
		case DeferInLoop:
			b.loopDeferDrainer(self)
		}
	})
}

func (b Builder) appendLoopDeferDrainer(self *aDefer) {
	self.stmts = append(self.stmts, func(Expr) {
		b.loopDeferDrainer(self)
	})
}

func (b Builder) loopDeferDrainer(self *aDefer) {
	if self.loopDrainerGenerated {
		return
	}
	self.loopDrainerGenerated = true
	if len(self.loopCases) == 0 {
		return
	}

	drainEntry := b.blk
	drainEntryAddr := drainEntry.Addr()

	prog := b.Prog
	condBlk := b.Func.MakeBlock()
	exitBlk := b.Func.MakeBlock()
	idBlk := b.Func.MakeBlock()

	b.Jump(condBlk)
	b.SetBlockEx(condBlk, AtEnd, true)
	list := b.Load(self.argsPtr)
	has := b.BinOp(token.NEQ, list, prog.Nil(prog.VoidPtr()))
	b.If(has, idBlk, exitBlk)

	b.SetBlockEx(idBlk, AtEnd, true)
	hdr := prog.Struct(prog.VoidPtr(), prog.Uintptr())
	hdrData := b.Load(Expr{list.impl, prog.Pointer(hdr)})
	nodeID := b.getField(hdrData, 1)

	chkBlks := make([]BasicBlock, len(self.loopCases))
	caseBlks := make([]BasicBlock, len(self.loopCases))
	for i := range self.loopCases {
		chkBlks[i] = b.Func.MakeBlock()
		caseBlks[i] = b.Func.MakeBlock()
	}

	b.Jump(chkBlks[0])
	for i, c := range self.loopCases {
		nextBlk := exitBlk
		if i+1 < len(self.loopCases) {
			nextBlk = chkBlks[i+1]
		}
		b.SetBlockEx(chkBlks[i], AtEnd, true)
		match := b.BinOp(token.EQL, nodeID, c.id)
		b.If(match, caseBlks[i], nextBlk)

		b.SetBlockEx(caseBlks[i], AtEnd, true)
		b.Store(self.rethPtr, drainEntryAddr)
		b.callDefer(self, c.typ, c.buildCall, c.fn, c.args)
		b.Jump(condBlk)
	}

	b.SetBlockEx(exitBlk, AtEnd, true)
}

/*
type node struct {
	prev *node
	id   uintptr // identifies defer statement for dispatch
	fn   func()  // (only if closure)
	args ...
}
// push
defer.Args = &node{defer.Args, id, fn, args...}
// pop
node := defer.Args
defer.Args = node.prev
free(node)
*/

func (b Builder) saveDeferArgs(self *aDefer, kind DoAction, id Expr, fn Expr, args []Expr) Type {
	return b.saveDeferArgsTo(self.argsPtr, kind, id, fn, args)
}

func (b Builder) saveDeferArgsTo(argsPtr Expr, kind DoAction, id Expr, fn Expr, args []Expr) Type {
	if kind != DeferInLoop && fn != Nil && fn.kind != vkClosure && len(args) == 0 {
		return nil
	}
	prog := b.Prog
	offset := 2 // prev + id
	if fn != Nil && fn.kind == vkClosure {
		offset++
	}
	typs := make([]Type, len(args)+offset)
	flds := make([]llvm.Value, len(args)+offset)
	typs[0] = prog.VoidPtr()
	flds[0] = b.Load(argsPtr).impl
	typs[1] = prog.Uintptr()
	flds[1] = id.impl
	if fn != Nil && fn.kind == vkClosure {
		typs[2] = fn.Type
		flds[2] = fn.impl
	}
	for i, arg := range args {
		typs[i+offset] = arg.Type
		flds[i+offset] = arg.impl
	}
	typ := prog.Struct(typs...)
	ptr := Expr{b.aggregateAllocU(typ, flds...), prog.VoidPtr()}
	b.Store(argsPtr, ptr)
	return typ
}

func (b Builder) callDefer(self *aDefer, typ Type, buildCall func(Builder, Expr, ...Expr) Expr, fn Expr, args []Expr) {
	if typ == nil {
		buildCall(b, fn, args...)
		return
	}
	prog := b.Prog
	zero := prog.Nil(prog.VoidPtr())
	list := b.Load(self.argsPtr)
	has := b.BinOp(token.NEQ, list, zero)
	// The guard is required because callDefer is reused by endDefer() after the
	// list has been drained. Without this check we would dereference a nil
	// pointer when no loop defers were recorded.
	b.IfThen(has, func() {
		ptr := b.Load(self.argsPtr)
		data := b.Load(Expr{ptr.impl, prog.Pointer(typ)})
		offset := 2 // prev + id
		b.Store(self.argsPtr, Expr{b.getField(data, 0).impl, prog.VoidPtr()})
		if fn != Nil && fn.kind == vkClosure {
			fn = b.getField(data, 2)
			offset++
		}
		for i := 0; i < len(args); i++ {
			args[i] = b.getField(data, i+offset)
		}
		buildCall(b, fn, args...)
		b.Call(b.Pkg.rtFunc("FreeDeferNode"), ptr)
	})
}

// RunDefers emits instructions to run deferred instructions.
func (b Builder) RunDefers() {
	self := b.getDefer(DeferInCond)
	if self == nil {
		return
	}
	blk := b.Func.MakeBlock()
	self.rundsNext = append(self.rundsNext, blk)

	b.Store(self.rundPtr, blk.Addr())
	b.Jump(self.procBlk)

	b.SetBlockEx(blk, AtEnd, false)
	b.blk.last = blk.last
}

func (p Function) endDefer(b Builder) {
	self := p.defer_
	if self == nil {
		return
	}
	nexts := self.rundsNext
	if len(nexts) == 0 {
		return
	}

	rethrowBlk := nexts[0]
	procBlk := self.procBlk
	panicBlk := self.panicBlk
	rethPtr := self.rethPtr
	rundPtr := self.rundPtr
	bitsPtr := self.bitsPtr

	stmts := self.stmts
	n := len(stmts)
	rethsNext := make([]BasicBlock, n+1)
	blks := p.MakeBlocks(n - 1)
	copy(rethsNext[1:], blks)
	rethsNext[0] = rethrowBlk
	rethsNext[n] = procBlk

	for i := n - 1; i >= 0; i-- {
		rethNext := rethsNext[i]
		b.SetBlockEx(rethsNext[i+1], AtEnd, true)
		b.Store(rethPtr, rethNext.Addr())
		stmts[i](b.Load(bitsPtr))
		if i != 0 {
			b.Jump(rethNext)
		}
	}
	link := b.getField(b.Load(self.data), deferLink)
	b.Call(b.Pkg.rtFunc("SetThreadDefer"), link)
	b.IndirectJump(b.Load(rundPtr), nexts)

	b.SetBlockEx(panicBlk, AtEnd, false) // panicBlk: exec runDefers and rethrow
	b.Store(rundPtr, rethrowBlk.Addr())
	b.IndirectJump(b.Load(rethPtr), rethsNext)
}

// -----------------------------------------------------------------------------

// Unreachable emits an unreachable instruction.
func (b Builder) Unreachable() {
	b.impl.CreateUnreachable()
}

// Recover emits a recover instruction.
func (b Builder) Recover() Expr {
	dbgInstrln("Recover")
	// TODO(xsw): recover can't be a function call in Go
	return b.Call(b.Pkg.rtFunc("Recover"))
}

// Panic emits a panic instruction.
func (b Builder) Panic(v Expr) {
	b.Call(b.Pkg.rtFunc("Panic"), v)
	b.Unreachable() // TODO: func supports noreturn attribute
}

// -----------------------------------------------------------------------------
