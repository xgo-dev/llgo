package llvmfmv

import (
	"fmt"
	"strings"

	"github.com/xgo-dev/llvm"
)

// Only a private call boundary can have different operands. The application
// callee and its calling convention are identical in every machine-code body.
type outlineKey struct {
	intrinsic llvm.Value
	required  uint64
	signature llvm.Type
}

type outlineSite struct {
	call     llvm.Value
	width    int
	required uint64
}

// A narrower baseline body contains every site that later variants could
// need to outline. Check this view before committing any IR transformation.
func (a *wideABI) validateOutlines(fn llvm.Value, baseline string, queries map[llvm.Value]uint64, enabled uint64) error {
	floor := a.registerFloor(fn, baseline)
	for _, s := range a.outlineSites(fn, floor, queries, enabled) {
		call := s.call
		if hasOutlineBundles(call) {
			return fmt.Errorf("SIMD FMV cannot outline call with an operand bundle in %s", fn.Name())
		}
		if !call.IsACallInst().IsNil() && call.TailCallKind() == llvm.TailCallKindMustTail {
			return fmt.Errorf("SIMD FMV cannot outline musttail call in %s", fn.Name())
		}
	}
	return nil
}

func (a *wideABI) outlineBody(fn llvm.Value, baseline string, queries map[llvm.Value]uint64, enabled uint64) {
	floor := a.registerFloor(fn, baseline)
	for _, s := range a.outlineSites(fn, floor, queries, enabled) {
		a.outlineCall(s.call, floor, s.width, s.required)
	}
}

func (a *wideABI) outlineSites(fn llvm.Value, floor int, queries map[llvm.Value]uint64, enabled uint64) []outlineSite {
	facts := blockGuards(fn, queries)
	// An invoke result may need a new normal edge. Visit the original sites once.
	var sites []outlineSite
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
			if w := a.callWidth(i); w > floor {
				sites = append(sites, outlineSite{call: i, width: w})
				continue
			}
			c := directGuardCallee(i)
			if c.IsNil() || c.IntrinsicID() == 0 {
				continue
			}
			req := a.intrinsicGuards(i, c)
			if req&^enabled != 0 && req&^(facts[bb]|enabled) == 0 {
				sites = append(sites, outlineSite{call: i, required: req})
			}
		}
	}
	return sites
}

// A vector mask must cross a target-feature boundary as its scalar bits. LLVM
// may otherwise allocate it in a vector register on one side and a mask
// register on the other. A wide value uses a local slot only when the caller
// cannot use its register ABI.
func (a *wideABI) outlineType(t llvm.Type, floor int) llvm.Type {
	ctx := a.mod.Context()
	if t.TypeKind() == llvm.VectorTypeKind && t.ElementType() == ctx.Int1Type() {
		return ctx.IntType(t.VectorSize())
	}
	if a.valueWidth(t) > floor {
		return llvm.PointerType(ctx.Int8Type(), 0)
	}
	return t
}

func outlineSlot(b llvm.Builder, fn llvm.Value, site llvm.Value, t llvm.Type) llvm.Value {
	b.SetInsertPointBefore(fn.EntryBasicBlock().FirstInstruction())
	slot := b.CreateAlloca(t, "")
	b.SetInsertPointBefore(site)
	return slot
}

// Pure overloaded intrinsics have the same ABI and semantics at every use.
// Other calls retain their original attributes and an indirect target, if any.
func reusableOperation(c llvm.Value) bool {
	if c.IsNil() {
		return false
	}
	for _, prefix := range []string{"llvm.fma.", "llvm.fmuladd.", "llvm.ctpop.", "llvm.experimental.vector.compress."} {
		if strings.HasPrefix(c.Name(), prefix) {
			return true
		}
	}
	return false
}

func (a *wideABI) outlineCall(inst llvm.Value, floor, width int, required uint64) {
	ctx := a.mod.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	caller := inst.InstructionParent().Parent()
	b.SetInsertPointBefore(inst)
	result := inst.Type()
	ret := a.outlineType(result, floor)
	out := llvm.Value{}
	var args []llvm.Value
	if ret != result && ret.TypeKind() == llvm.PointerTypeKind {
		out = outlineSlot(b, caller, inst, result)
		args = append(args, out)
		ret = ctx.VoidType()
	}
	callee := inst.CalledValue()
	// An intrinsic immediate is part of its opcode. Preserve constants in
	// per-site wrappers, including immediate and metadata operands.
	share := required != 0 && reusableOperation(callee) && !hasOutlineAttrs(inst)
	operands := make([]int, inst.ArgOperandsCount())
	for i := range operands {
		v := inst.Operand(i)
		if !share && !v.IsAConstant().IsNil() {
			operands[i] = -1
			continue
		}
		operands[i] = len(args)
		t := a.outlineType(v.Type(), floor)
		if t != v.Type() {
			if t.TypeKind() == llvm.PointerTypeKind {
				slot := outlineSlot(b, caller, inst, v.Type())
				b.CreateStore(v, slot)
				v = slot
			} else {
				v = b.CreateBitCast(v, t, "")
			}
		}
		args = append(args, v)
	}
	indirect := callee.IsAConstant().IsNil()
	if indirect {
		args = append(args, callee)
	}
	ts := make([]llvm.Type, len(args))
	for i, v := range args {
		ts[i] = v.Type()
	}
	ft := llvm.FunctionType(ret, ts, false)
	key := outlineKey{callee, required, ft}
	leaf := llvm.Value{}
	if share {
		leaf = a.outlines[key]
	}
	if leaf.IsNil() {
		name := caller.Name() + symbolPrefix + "call"
		if share {
			name = symbolPrefix + "op_" + strings.TrimPrefix(callee.Name(), "llvm.")
		}
		leaf = llvm.AddFunction(a.mod, name, ft)
		leaf.SetLinkage(llvm.InternalLinkage)
		leaf.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
		profile{width: max(width, a.signature(ft)), isa: required}.attributes(leaf)
		if a.signature(ft) != 0 {
			a.abi.applyWideCallingConvention(leaf)
		}
		if share {
			if a.outlines == nil {
				a.outlines = map[outlineKey]llvm.Value{}
			}
			a.outlines[key] = leaf
		}
		b.SetInsertPointAtEnd(ctx.AddBasicBlock(leaf, "entry"))
		var innerArgs []llvm.Value
		for i, index := range operands {
			original := inst.Operand(i)
			if index < 0 {
				innerArgs = append(innerArgs, original)
				continue
			}
			t := original.Type()
			v := leaf.Param(index)
			if v.Type() != t {
				if v.Type().TypeKind() == llvm.PointerTypeKind {
					v = b.CreateLoad(t, v, "")
				} else {
					v = b.CreateBitCast(v, t, "")
				}
			}
			innerArgs = append(innerArgs, v)
		}
		target := callee
		if indirect {
			target = leaf.Param(len(args) - 1)
		}
		call := b.CreateCall(inst.CalledFunctionType(), target, innerArgs, "")
		call.SetInstructionCallConv(inst.InstructionCallConv())
		copyOutlineCallAttrs(inst, call)
		if width != 0 {
			a.abi.applyWideCallingConvention(call)
		}
		switch {
		case !out.IsNil():
			b.CreateStore(call, leaf.Param(0))
			b.CreateRetVoid()
		case result.TypeKind() == llvm.VoidTypeKind:
			b.CreateRetVoid()
		case ret != result:
			b.CreateRet(b.CreateBitCast(call, ret, ""))
		default:
			b.CreateRet(call)
		}
	}
	b.SetInsertPointBefore(inst)
	var replacement llvm.Value
	if !inst.IsAInvokeInst().IsNil() {
		normal := inst.Successor(0)
		resume := normal
		if !out.IsNil() || ret != result {
			resume = ctx.AddBasicBlock(caller, "fmv.return")
			moveOutlinePHIs(normal, inst.InstructionParent(), resume, b)
		}
		b.SetInsertPointBefore(inst)
		replacement = b.CreateInvoke(ft, leaf, args, resume, inst.Successor(1), "")
		replacement.SetInstructionCallConv(leaf.FunctionCallConv())
		replacement.InstructionSetDebugLoc(inst.InstructionDebugLoc())
		if resume != normal {
			b.SetInsertPointAtEnd(resume)
			if !out.IsNil() {
				replacement = b.CreateLoad(result, out, "")
			} else {
				replacement = b.CreateBitCast(replacement, result, "")
			}
			b.CreateBr(normal)
		}
	} else {
		replacement = b.CreateCall(ft, leaf, args, "")
		replacement.SetInstructionCallConv(leaf.FunctionCallConv())
		replacement.InstructionSetDebugLoc(inst.InstructionDebugLoc())
		if !out.IsNil() {
			replacement = b.CreateLoad(result, out, "")
		} else if ret != result {
			replacement = b.CreateBitCast(replacement, result, "")
		}
	}
	if result.TypeKind() != llvm.VoidTypeKind {
		inst.ReplaceAllUsesWith(replacement)
	}
	inst.EraseFromParentAsInstruction()
}

// Change the incoming edge of each PHI when a wide invoke result is loaded
// in a private normal-return block. The exceptional edge never observes it.
func moveOutlinePHIs(block, from, to llvm.BasicBlock, b llvm.Builder) {
	for phi := block.FirstInstruction(); !phi.IsNil() && !phi.IsAPHINode().IsNil(); {
		next := llvm.NextInstruction(phi)
		b.SetInsertPointBefore(phi)
		p := b.CreatePHI(phi.Type(), "")
		for i := 0; i < phi.IncomingCount(); i++ {
			pred := phi.IncomingBlock(i)
			if pred == from {
				pred = to
			}
			p.AddIncoming([]llvm.Value{phi.IncomingValue(i)}, []llvm.BasicBlock{pred})
		}
		phi.ReplaceAllUsesWith(p)
		phi.EraseFromParentAsInstruction()
		phi = next
	}
}
