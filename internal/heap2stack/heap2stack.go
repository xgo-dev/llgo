// Package heap2stack applies LLGo's storage-placement policy around LLVM's
// heap-to-stack analysis. Attributor remains responsible for escape analysis.
package heap2stack

import "github.com/xgo-dev/llvm"

// Bound recursive attribute analysis on native stacks, especially Windows'
// 1 MiB threads. This is a conservative stack-safety cutoff, not a correctness
// limit of the analysis. Larger functions retain heap storage; later LLVM
// passes still optimize them normally.
const maxFunctionInstructions = 1024

type skippedFunction struct {
	fn              llvm.Value
	hadNoinline     bool
	savedAttributes []llvm.Attribute
}

func saveSkippedFunction(fn llvm.Value) skippedFunction {
	entry := skippedFunction{
		fn:          fn,
		hadNoinline: !fn.GetEnumFunctionAttribute(llvm.AttributeKindID("noinline")).IsNil(),
	}
	for _, name := range []string{"alwaysinline", "optsize", "minsize"} {
		if attr := fn.GetEnumFunctionAttribute(llvm.AttributeKindID(name)); !attr.IsNil() {
			entry.savedAttributes = append(entry.savedAttributes, attr)
		}
	}
	return entry
}

func (entry skippedFunction) skipAnalysis(ctx llvm.Context) {
	// optnone also stops dependencies requested from other functions. Remove
	// incompatible attributes so the temporary state stays verifier-valid.
	for _, attr := range entry.savedAttributes {
		entry.fn.RemoveEnumFunctionAttribute(uint(attr.GetEnumKind()))
	}
	entry.fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
	entry.fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("optnone"), 0))
}

func (entry skippedFunction) restore() {
	entry.fn.RemoveEnumFunctionAttribute(llvm.AttributeKindID("optnone"))
	if !entry.hadNoinline {
		entry.fn.RemoveEnumFunctionAttribute(llvm.AttributeKindID("noinline"))
	}
	for _, attr := range entry.savedAttributes {
		entry.fn.AddFunctionAttr(attr)
	}
}

// Run preserves heap storage in oversized functions, for tail-call inputs,
// and for repeated allocations whose instances cannot be proven disjoint,
// then moves new fixed slots to entry.
// Initialization stays at the original allocation site. No lifetime markers
// are needed for this placement, and pre-existing allocas are left alone.
func Run(mod llvm.Module, tm llvm.TargetMachine) error {
	if mod.IsNil() {
		return mod.RunAttributorHeapToStack(tm, nil)
	}
	var functions, excluded []llvm.Value
	existing := make(map[llvm.Value]bool)
	optnone := llvm.AttributeKindID("optnone")
	var skipped []skippedFunction
	defer func() {
		for _, entry := range skipped {
			entry.restore()
		}
	}()
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.FirstBasicBlock().IsNil() || !fn.GetEnumFunctionAttribute(optnone).IsNil() {
			continue
		}
		if exceedsAnalysisBudget(fn) {
			entry := saveSkippedFunction(fn)
			skipped = append(skipped, entry)
			entry.skipAnalysis(mod.Context())
			continue
		}
		functions = append(functions, fn)
		cycles := cyclicBlocks(fn.EntryBasicBlock())
		tailInputs := tailCallInputs(fn)
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			var order map[llvm.Value]int
			if cycles[bb] {
				order = make(map[llvm.Value]int)
			}
			for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
				if i.InstructionOpcode() == llvm.Alloca {
					existing[i] = true
				}
				if order != nil {
					order[i] = len(order)
				}
			}
			for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
				if !isCall(i) || i.Type().TypeKind() != llvm.PointerTypeKind {
					continue
				}
				if tailInputs[i] {
					excluded = append(excluded, i)
					continue
				}
				if order != nil {
					if _, known := i.ConstantAllocationSize(); !known || !canReuseAllocation(i, order) {
						excluded = append(excluded, i)
					}
				}
			}
		}
	}
	if err := mod.RunAttributorHeapToStack(tm, excluded); err != nil {
		return err
	}
	b := mod.Context().NewBuilder()
	defer b.Dispose()
	for _, fn := range functions {
		entry := fn.EntryBasicBlock()
		first := entry.FirstInstruction()
		for !first.IsNil() && first.InstructionOpcode() == llvm.PHI {
			first = llvm.NextInstruction(first)
		}
		b.SetInsertPointBefore(first)
		for bb := llvm.NextBasicBlock(entry); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for i := bb.FirstInstruction(); !i.IsNil(); {
				next := llvm.NextInstruction(i)
				if i.InstructionOpcode() == llvm.Alloca && !existing[i] && !i.Operand(0).IsAConstantInt().IsNil() {
					i.RemoveFromParentAsInstruction()
					b.Insert(i)
				}
				i = next
			}
		}
	}
	return nil
}

func exceedsAnalysisBudget(fn llvm.Value) bool {
	n := 0
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
			n++
			if n > maxFunctionInstructions {
				return true
			}
		}
	}
	return false
}

func isCall(i llvm.Value) bool {
	switch i.InstructionOpcode() {
	case llvm.Call, llvm.Invoke, llvm.CallBr:
		return true
	}
	return false
}

// tailCallInputs finds SSA inputs to tail/musttail calls across all blocks.
// A tail callee may execute after this frame has been released, even when it
// does not capture its arguments. Follow operands backwards so GEPs, phis,
// selects, casts and returned aliases cannot hide a newly promoted stack
// address. Conservatively include scalar/aggregate dependencies too. Attributor
// can also forward pointers through local memory, so exclude pointer spills
// and address conversions whenever a tail call is present. This avoids needing
// alias analysis here. The visited set bounds cyclic phis and shared chains.
func tailCallInputs(fn llvm.Value) map[llvm.Value]bool {
	inputs := make(map[llvm.Value]bool)
	var work, published []llvm.Value
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
			if i.InstructionOpcode() == llvm.Call && i.IsTailCall() {
				work = append(work, i)
			}
			switch i.InstructionOpcode() {
			case llvm.Store:
				value := i.Operand(0)
				switch value.Type().TypeKind() {
				case llvm.IntegerTypeKind, llvm.FloatTypeKind, llvm.DoubleTypeKind,
					llvm.X86_FP80TypeKind, llvm.FP128TypeKind, llvm.PPC_FP128TypeKind:
					// Scalar addresses are already covered by PtrToInt below.
				default:
					// Includes pointer vectors and unknown aggregate kinds.
					if !value.IsAInstruction().IsNil() {
						published = append(published, value)
					}
				}
			case llvm.PtrToInt:
				published = append(published, i)
			}
		}
	}
	if len(work) != 0 {
		work = append(work, published...)
	}
	for len(work) != 0 {
		i := work[len(work)-1]
		work = work[:len(work)-1]
		if inputs[i] {
			continue
		}
		inputs[i] = true
		for n := 0; n < i.OperandsCount(); n++ {
			operand := i.Operand(n)
			if !operand.IsAInstruction().IsNil() {
				work = append(work, operand)
			}
		}
	}
	return inputs
}

// canReuseAllocation proves a sufficient condition: all pointer uses finish
// after the allocation in this block, without a phi, pointer publication, or
// address observation. Attributor must still validate capture/free behavior
// of calls. Cross-block uses remain on the heap until we have a broader proof.
func canReuseAllocation(call llvm.Value, order map[llvm.Value]int) bool {
	work := []llvm.Value{call}
	seen := make(map[llvm.Value]bool)
	for len(work) != 0 {
		v := work[len(work)-1]
		work = work[:len(work)-1]
		if seen[v] {
			continue
		}
		seen[v] = true
		for use := v.FirstUse(); !use.IsNil(); use = use.NextUse() {
			user := use.User()
			if pos, ok := order[user]; !ok || pos <= order[call] {
				return false
			}
			switch user.InstructionOpcode() {
			case llvm.GetElementPtr, llvm.BitCast:
				work = append(work, user)
			case llvm.Load:
				if user.Operand(0) != v {
					return false
				}
			case llvm.Store:
				if user.Operand(1) != v || user.Operand(0) == v {
					return false
				}
			case llvm.Call, llvm.Invoke, llvm.CallBr:
				argument := false
				for arg := 0; arg < user.ArgOperandsCount(); arg++ {
					if user.OperandUse(arg) == use {
						argument = true
						break
					}
				}
				if !argument {
					return false
				}
			default:
				return false
			}
		}
	}
	return true
}

// cyclicBlocks uses Tarjan's SCC traversal rather than natural-loop headers,
// so multiple-entry (irreducible) cycles are covered too. Unreachable blocks
// cannot repeatedly allocate at runtime and are not visited.
func cyclicBlocks(entry llvm.BasicBlock) map[llvm.BasicBlock]bool {
	index := make(map[llvm.BasicBlock]int)
	low := make(map[llvm.BasicBlock]int)
	onStack := make(map[llvm.BasicBlock]bool)
	cyclic := make(map[llvm.BasicBlock]bool)
	var stack []llvm.BasicBlock
	var visit func(llvm.BasicBlock)
	visit = func(bb llvm.BasicBlock) {
		index[bb] = len(index) + 1
		low[bb] = index[bb]
		stack = append(stack, bb)
		onStack[bb] = true
		term := bb.LastInstruction()
		self := false
		for i := 0; i < term.SuccessorsCount(); i++ {
			next := term.Successor(i)
			self = self || next == bb
			if index[next] == 0 {
				visit(next)
				low[bb] = min(low[bb], low[next])
			} else if onStack[next] {
				low[bb] = min(low[bb], index[next])
			}
		}
		if low[bb] != index[bb] {
			return
		}
		start := len(stack) - 1
		for stack[start] != bb {
			start--
		}
		for _, member := range stack[start:] {
			onStack[member] = false
			if self || len(stack)-start > 1 {
				cyclic[member] = true
			}
		}
		stack = stack[:start]
	}
	visit(entry)
	return cyclic
}
