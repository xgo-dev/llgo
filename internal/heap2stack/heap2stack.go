// Package heap2stack applies LLGo's storage-placement policy around LLVM's
// heap-to-stack analysis. Attributor remains responsible for escape analysis.
package heap2stack

import "github.com/xgo-dev/llvm"

// Run preserves heap storage for repeated allocations whose instances cannot
// be proven disjoint, then moves newly promoted constant-size slots to entry.
// Initialization stays at the original allocation site. No lifetime markers
// are needed for this placement, and pre-existing allocas are left alone.
func Run(mod llvm.Module, tm llvm.TargetMachine) error {
	if mod.IsNil() {
		return mod.RunAttributorHeapToStack(tm, nil)
	}
	var functions, excluded []llvm.Value
	existing := make(map[llvm.Value]bool)
	optnone := llvm.AttributeKindID("optnone")
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.FirstBasicBlock().IsNil() || !fn.GetEnumFunctionAttribute(optnone).IsNil() {
			continue
		}
		functions = append(functions, fn)
		cycles := cyclicBlocks(fn.EntryBasicBlock())
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
			if order == nil {
				continue
			}
			for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
				if !isCall(i) || i.Type().TypeKind() != llvm.PointerTypeKind {
					continue
				}
				if _, known := i.ConstantAllocationSize(); !known || !canReuseAllocation(i, order) {
					excluded = append(excluded, i)
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

func isCall(i llvm.Value) bool {
	switch i.InstructionOpcode() {
	case llvm.Call, llvm.Invoke, llvm.CallBr:
		return true
	}
	return false
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
				if user.InstructionOpcode() == llvm.Call && user.IsTailCall() {
					return false
				}
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
