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
	"strings"
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

// func(unsafe.Pointer)
func (p Program) tyStackrestore() *types.Signature {
	if p.stackRestoreTy == nil {
		paramPtr := types.NewParam(token.NoPos, nil, "", p.VoidPtr().raw.Type)
		params := types.NewTuple(paramPtr)
		p.stackRestoreTy = types.NewSignatureType(nil, nil, nil, params, nil, false)
	}
	return p.stackRestoreTy
}

func (b Builder) AllocaSigjmpBuf() Expr {
	prog := b.Prog
	sigjmpBufTy := prog.rtType("SigjmpBuf") // Get type from runtime (target architecture)
	n := prog.SizeOf(sigjmpBufTy)           // Get size for target architecture
	size := prog.IntVal(n, prog.Uintptr())
	ret := b.Alloca(size)
	if prog.target.effectiveGOOS() == "windows" {
		// UCRT jmp_buf uses up to 16-byte alignment on supported Windows
		// architectures. CreateArrayAlloca otherwise inherits byte alignment
		// from its i8 element.
		ret.impl.SetAlignment(16)
	}
	return ret
}

// declare ptr @llvm.stacksave.p0()
func (b Builder) StackSave() Expr {
	return Expr{
		b.impl.CreateIntrinsic(b.Prog.VoidPtr().ll, llvm.LookupIntrinsicID("llvm.stacksave"), nil, ""),
		b.Prog.VoidPtr(),
	}
}

// declare void @llvm.stackrestore.p0(ptr)
func (b Builder) StackRestore(sp Expr) {
	b.impl.CreateIntrinsic(b.Prog.Void().ll, llvm.LookupIntrinsicID("llvm.stackrestore"), []llvm.Value{sp.impl}, "")
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
	if b.Prog.target.effectiveGOOS() == "windows" {
		return b.windowsSetjmp(jb)
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
	if b.Prog.target.effectiveGOOS() == "windows" {
		b.windowsLongjmp(jb, retval)
		return
	}
	fn := b.Pkg.rtFunc("Siglongjmp") // TODO(xsw): mark as noreturn
	b.Call(fn, jb, retval)
	// b.Unreachable()
}

func (b Builder) windowsSetjmp(jb Expr) Expr {
	prog := b.Prog
	ptrParam := types.NewParam(token.NoPos, nil, "", prog.VoidPtr().raw.Type)
	intResult := types.NewParam(token.NoPos, nil, "", prog.CInt().raw.Type)
	goarch := prog.target.effectiveGOARCH()
	var name string
	var params *types.Tuple
	var args []Expr

	switch goarch {
	case "386":
		name = "_setjmp3"
		zero := prog.IntVal(0, prog.CInt())
		params = types.NewTuple(
			ptrParam,
			types.NewParam(token.NoPos, nil, "", prog.CInt().raw.Type),
		)
		args = []Expr{jb, zero}
	case "amd64":
		// A nil frame tells longjmp to restore the saved context directly.
		// LLGo owns Go defer unwinding, and RtlUnwind cannot leave a vectored
		// exception handler that interrupted generated Go code reliably.
		name = "_setjmpex"
		triple := strings.ToLower(prog.target.Spec().Triple)
		if strings.Contains(triple, "windows-gnu") || strings.Contains(triple, "mingw") {
			// MinGW exposes the same intrinsic entry point under its GNU CRT
			// spelling; unlike the setjmp macro this lets LLGo pass a nil frame.
			name = "__intrinsic_setjmpex"
		}
		params = types.NewTuple(
			ptrParam,
			types.NewParam(token.NoPos, nil, "", prog.VoidPtr().raw.Type),
		)
		args = []Expr{jb, prog.Nil(prog.VoidPtr())}
	case "arm64":
		// UCRT longjmp performs a Windows virtual unwind before restoring the
		// saved context. That cannot cross third-party assembly without .pdata,
		// including libffi's ARM64 closure entry. LLGo owns defer unwinding, so
		// use the runtime's ABI-only context save/restore pair instead.
		name = "llgo_setjmp"
		params = types.NewTuple(ptrParam)
		args = []Expr{jb}
	default:
		panic("ssa: unsupported Windows architecture for setjmp: " + goarch)
	}

	sig := types.NewSignatureType(
		nil, nil, nil, params, types.NewTuple(intResult), false,
	)
	fn := b.Pkg.cFunc(name, sig)
	b.addReturnsTwiceAttr(fn)
	return b.Call(fn, args...)
}

func (b Builder) windowsLongjmp(jb, retval Expr) {
	goarch := b.Prog.target.effectiveGOARCH()
	if goarch != "amd64" && goarch != "arm64" {
		b.cLongjmp(jb, retval)
		return
	}
	fn := b.Pkg.cFunc("llgo_longjmp", b.Prog.tyLongjmp())
	b.Call(fn, jb, retval)
}

func (b Builder) Setjmp(jb Expr) Expr {
	if b.Prog.target.effectiveGOOS() == "windows" {
		return b.windowsSetjmp(jb)
	}
	return b.cSetjmp(jb)
}

func (b Builder) cSetjmp(jb Expr) Expr {
	fn := b.Pkg.cFunc("setjmp", b.Prog.tySetjmp())
	b.addReturnsTwiceAttr(fn)
	return b.Call(fn, jb)
}

func (b Builder) Longjmp(jb, retval Expr) {
	if b.Prog.target.effectiveGOOS() == "windows" {
		b.windowsLongjmp(jb, retval)
		return
	}
	b.cLongjmp(jb, retval)
}

func (b Builder) cLongjmp(jb, retval Expr) {
	fn := b.Pkg.cFunc("longjmp", b.Prog.tyLongjmp())
	b.Call(fn, jb, retval)
	// b.Unreachable()
}

// -----------------------------------------------------------------------------

func (p Function) deferInitBuilder(from Builder) (b Builder, next BasicBlock) {
	b = p.NewBuilder()
	if p.diFunc != nil {
		loc := from.impl.GetCurrentDebugLocation()
		if !loc.Scope.IsNil() {
			b.impl.SetCurrentDebugLocation(loc.Line, loc.Col, loc.Scope, loc.InlinedAt)
		}
	}
	next = b.setBlockMoveLast(p.blks[0])
	p.blks[0].last = next.last
	return
}

type aDefer struct {
	nextBit     int        // next defer bit
	data        Expr       // pointer to runtime.Defer
	bitsPtr     Expr       // pointer to defer bits
	rethPtr     Expr       // native block address or wasm Rethrow selector
	rundPtr     Expr       // native block address or wasm RunDefers selector
	argsPtr     Expr       // func and args links
	procBlk     BasicBlock // deferProc block
	panicBlk    BasicBlock // panic block (runDefers and rethrow)
	rundTargets []deferTarget
	// loopDrainerGenerated marks whether we've already generated the loop-defer
	// drain loop for the current contiguous run of DeferInLoop statements (when
	// walking defers in reverse order in endDefer).
	loopDrainerGenerated bool
	loopCases            []loopDeferCase
	stmts                []func(bits Expr, resume deferTarget)
}

// deferTarget pairs a wasm selector with the corresponding local block.
// Native targets store the block address directly; wasm selectors are dense
// within each dispatch so LLVM can lower them efficiently.
type deferTarget struct {
	index int
	block BasicBlock
}

// loopDeferCase represents a defer statement inside a loop.
// The id uniquely identifies the defer call site for dispatch during drain.
// typ is the node struct type needed to decode the linked-list node.
type loopDeferCase struct {
	id         Expr
	typ        Type
	mayRecover bool
	fn         Expr
	args       []Expr
	buildCall  func(Builder, Expr, ...Expr) Expr
}

const (
	// 0: addr sigjmpbuf
	// 1: bits uintptr
	// 2: link *Defer
	// 3: reth voidptr: native block address or wasm Rethrow selector
	// 4: rund voidptr: native block address or wasm RunDefers selector
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
			b, next = self.deferInitBuilder(b)
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
	// Wasm Reth selector 0 is reserved for procBlk; Rund selector 0 is
	// rethrowBlk. Native targets store the corresponding block addresses.
	initialReth := deferTarget{index: 0, block: procBlk}
	ptr := b.aggregateAllocU(prog.Defer(), jb.impl, zero.impl, link.impl, b.deferTargetValue(initialReth).impl)
	deferData := Expr{ptr, prog.DeferPtr()}
	b.Call(b.Pkg.rtFunc("SetThreadDefer"), deferData)
	if prog.GCRootsEnabled() {
		b.Call(b.Pkg.rtFunc("SetDeferGCRoot"), deferData, b.currentGCRootChain())
	}
	bitsPtr := b.FieldAddr(deferData, deferBits)
	rethPtr := b.FieldAddr(deferData, deferRethrow)
	rundPtr := b.FieldAddr(deferData, deferRunDefers)
	argsPtr := b.FieldAddr(deferData, deferArgs)
	if prog.target.GOARCH == "wasm" {
		b.storeDeferTarget(rundPtr, deferTarget{index: 0, block: rethrowBlk})
	}
	// Initialize the args list so later guards (e.g. DeferAlways/DeferInLoop)
	// can safely detect an empty chain without a prior push.
	b.Store(argsPtr, prog.Nil(prog.VoidPtr()))

	czero := prog.IntVal(0, prog.CInt())
	retval := b.Sigsetjmp(jb, czero)
	if prog.GCRootsEnabled() {
		b.Call(b.Pkg.rtFunc("RestoreDeferGCRoot"), deferData)
	}

	self.defer_ = &aDefer{
		data:        deferData,
		bitsPtr:     bitsPtr,
		rethPtr:     rethPtr,
		rundPtr:     rundPtr,
		argsPtr:     argsPtr,
		procBlk:     procBlk,
		rundTargets: []deferTarget{{index: 0, block: rethrowBlk}},
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
	b.DeferRecover(kind, deferMayRecover(fn), fn, buildCall, args...)
}

// DeferRecover emits a defer instruction with explicit recover capability.
func (b Builder) DeferRecover(kind DoAction, mayRecover bool, fn Expr, buildCall func(Builder, Expr, ...Expr) Expr, args ...Expr) {
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
		loopCase := loopDeferCase{id: id, typ: typ, mayRecover: mayRecover, fn: fn, args: args, buildCall: buildCall}
		self.loopCases = append(self.loopCases, loopCase)
	}
	b.appendDeferStmt(self, kind, typ, mayRecover, buildCall, fn, args, nextbit)
}

// DeferTo emits a defer instruction into an explicit runtime defer stack.
func (b Builder) DeferTo(owner Function, stack Expr, fn Expr, buildCall func(Builder, Expr, ...Expr) Expr, args ...Expr) {
	b.DeferToRecover(owner, stack, deferMayRecover(fn), fn, buildCall, args...)
}

// DeferToRecover emits a defer instruction into an explicit runtime defer
// stack with explicit recover capability.
func (b Builder) DeferToRecover(owner Function, stack Expr, mayRecover bool, fn Expr, buildCall func(Builder, Expr, ...Expr) Expr, args ...Expr) {
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
		id:         id,
		typ:        typ,
		mayRecover: mayRecover,
		fn:         fn,
		args:       args,
		buildCall:  buildCall,
	}
	if self == nil {
		owner.pendingLoopCases = append(owner.pendingLoopCases, loopCase)
		return
	}
	self.loopCases = append(self.loopCases, loopCase)
}

func (b Builder) appendDeferStmt(self *aDefer, kind DoAction, typ Type, mayRecover bool, buildCall func(Builder, Expr, ...Expr) Expr, fn Expr, args []Expr, nextbit Expr) {
	self.stmts = append(self.stmts, func(bits Expr, resume deferTarget) {
		switch kind {
		case DeferInCond:
			// Leaving a run of loop defers; allow the next loop-defer statement
			// (earlier in source order) to generate its own drainer.
			self.loopDrainerGenerated = false
			prog := b.Prog
			zero := prog.Val(uintptr(0))
			has := b.BinOp(token.NEQ, b.BinOp(token.AND, bits, nextbit), zero)
			b.IfThen(has, func() {
				b.callDefer(self, typ, mayRecover, buildCall, fn, args)
			})
		case DeferAlways:
			// Leaving a run of loop defers; allow the next loop-defer statement
			// (earlier in source order) to generate its own drainer.
			self.loopDrainerGenerated = false
			b.callDefer(self, typ, mayRecover, buildCall, fn, args)
		case DeferInLoop:
			b.loopDeferDrainer(self, resume)
		}
	})
}

func (b Builder) appendLoopDeferDrainer(self *aDefer) {
	self.stmts = append(self.stmts, func(_ Expr, resume deferTarget) {
		b.loopDeferDrainer(self, resume)
	})
}

func (b Builder) loopDeferDrainer(self *aDefer, resume deferTarget) {
	if self.loopDrainerGenerated {
		return
	}
	self.loopDrainerGenerated = true
	if len(self.loopCases) == 0 {
		return
	}

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
		b.storeDeferTarget(self.rethPtr, resume)
		b.callDefer(self, c.typ, c.mayRecover, c.buildCall, c.fn, c.args)
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
	saveFn := fn != Nil && (fn.kind == vkClosure || fn.kind == vkIfaceMethod)
	if kind != DeferInLoop && fn != Nil && !saveFn && len(args) == 0 {
		return nil
	}
	prog := b.Prog
	offset := 2 // prev + id
	if saveFn {
		offset++
	}
	typs := make([]Type, len(args)+offset)
	flds := make([]llvm.Value, len(args)+offset)
	typs[0] = prog.VoidPtr()
	flds[0] = b.Load(argsPtr).impl
	typs[1] = prog.Uintptr()
	flds[1] = id.impl
	if saveFn {
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

func (b Builder) callDefer(self *aDefer, typ Type, mayRecover bool, buildCall func(Builder, Expr, ...Expr) Expr, fn Expr, args []Expr) {
	if typ == nil {
		b.callRecoverScopedDefer(fn, mayRecover, func() {
			buildCall(b, fn, args...)
		})
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
		if fn != Nil && (fn.kind == vkClosure || fn.kind == vkIfaceMethod) {
			savedType := fn.Type
			fn = b.getField(data, 2)
			// A transient interface invocation has the same physical pair as a
			// funcval, so aggregate field reconstruction sees vkClosure. Keep
			// its call semantics: the saved data word is an ordinary receiver,
			// not a hidden closure environment.
			if savedType.kind == vkIfaceMethod {
				fn.Type = savedType
			}
			offset++
		}
		for i := 0; i < len(args); i++ {
			args[i] = b.getField(data, i+offset)
		}
		b.Call(b.Pkg.rtFunc("FreeDeferNode"), ptr)
		b.callRecoverScopedDefer(fn, mayRecover, func() {
			buildCall(b, fn, args...)
		})
	})
}

func (b Builder) callRecoverScopedDefer(fn Expr, mayRecover bool, call func()) {
	if fn.IsNil() || fn.impl.IsNil() || isRecoverBuiltin(fn) {
		call()
		return
	}
	token := b.recoverDeferToken(fn, mayRecover)
	if token.IsNil() {
		call()
		return
	}
	prev := b.Call(b.Pkg.rtFunc("StartRecoverFrame"), token)
	call()
	b.Call(b.Pkg.rtFunc("EndRecoverFrame"), prev)
}

// CallRecoverAlias invokes fn through a compiler-generated wrapper while
// preserving the wrapped function as the direct deferred call for recover.
func (b Builder) CallRecoverAlias(from Expr, mayRecover bool, fn Expr, buildCall func(Builder, Expr, ...Expr) Expr, args ...Expr) Expr {
	token := b.recoverDeferToken(fn, mayRecover)
	if from.IsNil() || token.IsNil() {
		return buildCall(b, fn, args...)
	}
	prev := b.Call(
		b.Pkg.rtFunc("StartRecoverFrameAlias"),
		b.PtrCast(b.Prog.VoidPtr(), from),
		token,
	)
	ret := buildCall(b, fn, args...)
	b.Call(b.Pkg.rtFunc("EndRecoverFrameAlias"), prev)
	return ret
}

func (b Builder) recoverDeferToken(fn Expr, mayRecover bool) Expr {
	switch fn.kind {
	case vkClosure, vkIfaceMethod:
		if !mayRecover {
			return Nil
		}
		return b.PtrCast(b.Prog.VoidPtr(), b.Field(fn, 0))
	case vkFuncDecl:
		if !mayRecover {
			return Nil
		}
		return b.PtrCast(b.Prog.VoidPtr(), fn)
	case vkFuncPtr:
		return b.PtrCast(b.Prog.VoidPtr(), fn)
	}
	return Nil
}

func deferMayRecover(fn Expr) bool {
	if fn.IsNil() || fn.Type == nil {
		return false
	}
	switch fn.kind {
	case vkClosure, vkFuncDecl, vkIfaceMethod, vkFuncPtr:
		// The lower-level Builder API has no Go SSA or compilation-wide
		// analysis cache. Conservatively scope every callable value; the Go
		// frontend uses DeferRecover/DeferToRecover with its precise result.
		return true
	}
	return false
}

func isRecoverBuiltin(fn Expr) bool {
	if fn.IsNil() || fn.kind != vkBuiltin {
		return false
	}
	bi, ok := fn.raw.Type.(*builtinTy)
	return ok && bi.name == "recover"
}

// RunDefers emits instructions to run deferred instructions.
func (b Builder) RunDefers() {
	self := b.getDefer(DeferInCond)
	if self == nil {
		return
	}
	blk := b.Func.MakeBlock()
	target := deferTarget{index: len(self.rundTargets), block: blk}
	self.rundTargets = append(self.rundTargets, target)

	b.storeDeferTarget(self.rundPtr, target)
	b.Jump(self.procBlk)

	b.SetBlockEx(blk, AtEnd, false)
	b.blk.last = blk.last
}

func (b Builder) storeDeferTarget(ptr Expr, target deferTarget) {
	b.Store(ptr, b.deferTargetValue(target))
}

func (b Builder) deferTargetValue(target deferTarget) Expr {
	if b.Prog.target.GOARCH == "wasm" {
		return b.PtrCast(b.Prog.VoidPtr(), b.Prog.Val(uintptr(target.index)))
	}
	return target.block.Addr()
}

func (b Builder) jumpDeferTarget(ptr Expr, targets []deferTarget) {
	loaded := b.Load(ptr)
	if b.Prog.target.GOARCH != "wasm" {
		blocks := make([]BasicBlock, len(targets))
		for i, target := range targets {
			blocks[i] = target.block
		}
		b.IndirectJump(loaded, blocks)
		return
	}

	selector := b.Convert(b.Prog.Uintptr(), loaded)
	invalid := b.Func.MakeBlock()
	sw := b.impl.CreateSwitch(selector.impl, invalid.first, len(targets))
	for _, target := range targets {
		sw.AddCase(b.Prog.Val(uintptr(target.index)).impl, target.block.first)
	}
	b.SetBlockEx(invalid, AtEnd, false)
	b.Unreachable()
}

func (p Function) endDefer(b Builder) {
	self := p.defer_
	if self == nil {
		return
	}
	rundTargets := self.rundTargets
	// A partially constructed defer state has no dispatch target yet.
	// initDeferState seeds selector 0 with the terminal rethrow target.
	if len(rundTargets) == 0 {
		return
	}
	rethrowTarget := rundTargets[0]
	rethrowBlk := rethrowTarget.block
	procBlk := self.procBlk
	panicBlk := self.panicBlk
	rethPtr := self.rethPtr
	rundPtr := self.rundPtr
	bitsPtr := self.bitsPtr

	stmts := self.stmts
	n := len(stmts)
	var blks []BasicBlock
	if n > 1 {
		blks = p.MakeBlocks(n - 1)
	}
	// Reth selector 0 must remain procBlk because initDeferState installs it
	// before the final number of deferred statements is known. Selector 1 is
	// the terminal rethrow, followed by the intermediate continuation blocks.
	// Keep the slice in native continuation order; wasm dispatch uses index.
	rethTargets := make([]deferTarget, n+1)
	if n > 0 {
		rethTargets[0] = deferTarget{index: 1, block: rethrowBlk}
		for i, blk := range blks {
			rethTargets[i+1] = deferTarget{index: i + 2, block: blk}
		}
	}
	rethTargets[n] = deferTarget{index: 0, block: procBlk}

	for i := n - 1; i >= 0; i-- {
		rethNext := rethTargets[i]
		resume := rethTargets[i+1]
		b.SetBlockEx(resume.block, AtEnd, true)
		b.storeDeferTarget(rethPtr, rethNext)
		stmts[i](b.Load(bitsPtr), resume)
		if i != 0 {
			b.Jump(rethNext.block)
		}
	}
	link := b.getField(b.Load(self.data), deferLink)
	b.Call(b.Pkg.rtFunc("SetThreadDefer"), link)
	b.jumpDeferTarget(rundPtr, rundTargets)

	b.SetBlockEx(panicBlk, AtEnd, false) // panicBlk: exec runDefers and rethrow
	b.storeDeferTarget(rundPtr, rethrowTarget)
	b.jumpDeferTarget(rethPtr, rethTargets)
}

// -----------------------------------------------------------------------------

// Unreachable emits an unreachable instruction.
func (b Builder) Unreachable() {
	b.impl.CreateUnreachable()
}

// BindRecoverFrame gives this invocation a stack-local identity. The runtime
// replaces a pending deferred-function token with this activation token only
// when this invocation is the direct deferred call. Recursive calls to the
// same function therefore cannot recover the caller's panic.
func (b Builder) BindRecoverFrame() {
	token := b.AllocaT(b.Prog.Byte())
	token = b.PtrCast(b.Prog.VoidPtr(), token)
	b.Func.recoverToken = token
	b.Call(
		b.Pkg.rtFunc("BindRecoverFrame"),
		b.PtrCast(b.Prog.VoidPtr(), b.Func.Expr),
		token,
	)
}

// Recover emits a recover instruction.
func (b Builder) Recover() Expr {
	dbgInstrln("Recover")
	token := b.Func.recoverToken
	if token.IsNil() {
		// Keep the lower-level Builder API usable for callers that emit recover
		// without the Go frontend's function-entry binding.
		token = b.PtrCast(b.Prog.VoidPtr(), b.Func.Expr)
	}
	return b.Call(b.Pkg.rtFunc("Recover"), token)
}

// Panic emits a panic instruction.
func (b Builder) Panic(v Expr) {
	b.Call(b.Pkg.rtFunc("Panic"), v)
	b.Unreachable() // TODO: func supports noreturn attribute
}

// -----------------------------------------------------------------------------
