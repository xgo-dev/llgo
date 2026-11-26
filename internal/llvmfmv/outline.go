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

// Check guarded intrinsics before mutating any public definition. Include
// the widest register body, where additional intrinsics can be outlined.
func (a *wideABI) validateOutlines(fn llvm.Value, queries map[llvm.Value]uint64, enabled uint64) error {
	for _, s := range a.outlineSites(fn, 512, queries, enabled) {
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
	traps := false
	for _, s := range a.outlineSites(fn, floor, queries, enabled) {
		if s.width > floor {
			a.trapBeforeCall(s.call)
			traps = true
		} else {
			a.outlineIntrinsic(s.call, s.required)
		}
	}
	if traps {
		// LLVM truncates after each noreturn trap and removes dead call or
		// invoke edges, including their incoming values in successor PHIs.
		fn.RemoveUnreachableBlocks()
	}
}

func (a *wideABI) outlineSites(fn llvm.Value, floor int, queries map[llvm.Value]uint64, enabled uint64) []outlineSite {
	facts := blockGuards(fn, queries)
	// Visit the original sites before replacing calls or removing dead blocks.
	var sites []outlineSite
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
			if w := a.callWidth(i); w > floor {
				sites = append(sites, outlineSite{call: i, width: w})
				continue
			}
			// The arithmetic intrinsics eligible for ISA outlining are calls.
			// LLVM does not allow them as invoke targets.
			if i.IsACallInst().IsNil() {
				continue
			}
			c := directGuardCallee(i)
			if c.IsNil() || c.IntrinsicID() == 0 {
				continue
			}
			req := a.intrinsicGuards(i, c)
			// A generic intrinsic can be lowered at a narrower width.
			// Outline for its optional ISA only when the caller already
			// has every register needed by the private signature.
			if req&^enabled != 0 && req&^(facts[bb]|enabled) == 0 && a.signature(c.GlobalValueType()) <= floor {
				sites = append(sites, outlineSite{call: i, required: req})
			}
		}
	}
	return sites
}

// This implementation is selected only on machines without the fixed width
// required by the call. Fault at the call itself; scalar exits and earlier
// Go effects retain their source order. The capable version keeps the exact
// original call, including indirect targets, EH edges and ABI attributes.
func (a *wideABI) trapBeforeCall(inst llvm.Value) {
	b := a.mod.Context().NewBuilder()
	defer b.Dispose()
	b.SetInsertPointBefore(inst)
	trap := a.mod.IntrinsicDeclaration(llvm.LookupIntrinsicID("llvm.trap"), nil)
	fault := b.CreateCall(trap.GlobalValueType(), trap, nil, "")
	fault.InstructionSetDebugLoc(inst.InstructionDebugLoc())
}

// A vector mask crosses an ISA boundary as its scalar bits: the two targets
// can otherwise choose different register classes for the same mask.
func outlineType(t llvm.Type) llvm.Type {
	if t.TypeKind() == llvm.VectorTypeKind && t.ElementType() == t.Context().Int1Type() {
		return t.Context().IntType(t.VectorSize())
	}
	return t
}

// Pure overloaded intrinsics have the same ABI and semantics at every use.
// Per-site wrappers retain immediate operands and call-site attributes.
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

func (a *wideABI) outlineIntrinsic(inst llvm.Value, required uint64) {
	ctx := a.mod.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	caller := inst.InstructionParent().Parent()
	b.SetInsertPointBefore(inst)
	result := inst.Type()
	ret := outlineType(result)
	var args []llvm.Value
	callee := inst.CalledValue()
	// An intrinsic immediate is part of its opcode. Preserve constants in
	// per-site wrappers, including immediate and metadata operands.
	share := reusableOperation(callee) && !hasOutlineAttrs(inst)
	operands := make([]int, inst.ArgOperandsCount())
	for i := range operands {
		v := inst.Operand(i)
		if !share && !v.IsAConstant().IsNil() {
			operands[i] = -1
			continue
		}
		operands[i] = len(args)
		if t := outlineType(v.Type()); t != v.Type() {
			v = b.CreateBitCast(v, t, "")
		}
		args = append(args, v)
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
		profile{width: a.signature(ft), isa: required}.attributes(leaf)
		a.abi.applyWideCallingConvention(leaf)
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
				v = b.CreateBitCast(v, t, "")
			}
			innerArgs = append(innerArgs, v)
		}
		call := b.CreateCall(inst.CalledFunctionType(), callee, innerArgs, "")
		copyOutlineCallAttrs(inst, call)
		switch {
		case result.TypeKind() == llvm.VoidTypeKind:
			b.CreateRetVoid()
		case ret != result:
			b.CreateRet(b.CreateBitCast(call, ret, ""))
		default:
			b.CreateRet(call)
		}
	}
	b.SetInsertPointBefore(inst)
	replacement := b.CreateCall(ft, leaf, args, "")
	replacement.SetInstructionCallConv(leaf.FunctionCallConv())
	replacement.InstructionSetDebugLoc(inst.InstructionDebugLoc())
	if ret != result {
		replacement = b.CreateBitCast(replacement, result, "")
	}
	if result.TypeKind() != llvm.VoidTypeKind {
		inst.ReplaceAllUsesWith(replacement)
	}
	inst.EraseFromParentAsInstruction()
}
