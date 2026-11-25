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
					if _, known := i.ConstantAllocationSize(); !known || !canReuseAllocation(i, bb, order) {
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
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for i := bb.FirstInstruction(); !i.IsNil(); {
				next := llvm.NextInstruction(i)
				if i.InstructionOpcode() == llvm.Alloca && !existing[i] {
					clearTailOnMemIntrinsics(i)
					if bb != entry && !i.Operand(0).IsAConstantInt().IsNil() {
						i.RemoveFromParentAsInstruction()
						b.Insert(i)
					}
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

// isNonTailMemIntrinsic reports whether call is a non-musttail memory
// intrinsic (memset/memcpy/memmove) that is not in tail-return position.
// Post-inline optimization marks such intrinsics "tail" whenever a function
// has no existing allocas; promoting an allocation clears "tail" on them.
func isNonTailMemIntrinsic(i llvm.Value) bool {
	if i.InstructionOpcode() != llvm.Call || i.TailCallKind() != llvm.TailCallKindTail || i.IsAMemIntrinsic().IsNil() {
		return false
	}
	next := llvm.NextInstruction(i)
	return !next.IsNil() && next.InstructionOpcode() != llvm.Ret
}

// clearTailOnMemIntrinsics drops the non-semantic "tail" hint from memory
// intrinsics that access a newly promoted alloca so the IR remains valid.
func clearTailOnMemIntrinsics(alloca llvm.Value) {
	work := []llvm.Value{alloca}
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
			switch user.InstructionOpcode() {
			case llvm.GetElementPtr, llvm.BitCast, llvm.PHI, llvm.Select:
				work = append(work, user)
			case llvm.Call:
				if user.TailCallKind() == llvm.TailCallKindTail && !user.IsAMemIntrinsic().IsNil() {
					user.SetTailCallKind(llvm.TailCallKindNone)
				}
			}
		}
	}
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
	hasTailCall := false
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
			if i.InstructionOpcode() == llvm.Call && i.IsTailCall() && !isNonTailMemIntrinsic(i) {
				hasTailCall = true
				for n := 0; n < i.OperandsCount(); n++ {
					operand := i.Operand(n)
					if !operand.IsAInstruction().IsNil() {
						work = append(work, operand)
					}
				}
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
	if hasTailCall {
		work = append(work, published...)
	}
	for len(work) != 0 {
		i := work[len(work)-1]
		work = work[:len(work)-1]
		if inputs[i] {
			continue
		}
		inputs[i] = true
		if i.InstructionOpcode() == llvm.Load {
			// Loading through a pointer reads stored contents, not the pointer
			// address itself; any pointer stored into memory is already seeded
			// via published above.
			continue
		}
		for n := 0; n < i.OperandsCount(); n++ {
			operand := i.Operand(n)
			if !operand.IsAInstruction().IsNil() {
				work = append(work, operand)
			}
		}
	}
	return inputs
}

// canReuseAllocation proves a sufficient condition for reusing a single entry
// stack slot across loop iterations: no pointer derived from call is stored to
// memory, observed as an address, passed to a cross-block free, or live at the
// program point immediately before call executes. Attributor still validates
// capture/free behavior of calls.
func canReuseAllocation(call llvm.Value, allocBB llvm.BasicBlock, order map[llvm.Value]int) bool {
	work := []llvm.Value{call}
	derived := make(map[llvm.Value]bool)
	allocPos := order[call]
	freeKind := llvm.AttributeKindID("allockind")
	for len(work) != 0 {
		v := work[len(work)-1]
		work = work[:len(work)-1]
		if derived[v] {
			continue
		}
		derived[v] = true
		for use := v.FirstUse(); !use.IsNil(); use = use.NextUse() {
			user := use.User()
			if user.IsAInstruction().IsNil() {
				return false
			}
			userBB := user.InstructionParent()
			if userBB == allocBB && user.InstructionOpcode() != llvm.PHI {
				if pos, ok := order[user]; !ok || pos <= allocPos {
					return false
				}
			}
			switch user.InstructionOpcode() {
			case llvm.GetElementPtr, llvm.BitCast, llvm.PHI, llvm.Select:
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
				if userBB != allocBB && isFreeCall(user, freeKind) {
					return false
				}
			default:
				return false
			}
		}
	}
	// Verify no derived value is live immediately before call executes.
	for v := range derived {
		if v == call {
			continue
		}
		defBB := v.InstructionParent()
		if defBB == allocBB {
			if pos, ok := order[v]; !ok || pos < allocPos {
				return false
			}
			continue
		}
		if isDerivedLiveAtAlloc(v, defBB, allocBB, allocPos, order) {
			return false
		}
	}
	return true
}

func isFreeCall(call llvm.Value, allocKind uint) bool {
	if !call.GetCallSiteEnumAttribute(-1, allocKind).IsNil() {
		return true
	}
	if callee := call.CalledValue().IsAFunction(); !callee.IsNil() {
		if !callee.GetEnumFunctionAttribute(allocKind).IsNil() {
			return true
		}
	}
	return false
}

// isDerivedLiveAtAlloc reports whether v (defined in defBB != allocBB) can
// reach any of its uses from the point immediately after allocBB's allocation
// without first passing through defBB (which redefines v).
func isDerivedLiveAtAlloc(v llvm.Value, defBB, allocBB llvm.BasicBlock, allocPos int, order map[llvm.Value]int) bool {
	// Forward BFS from allocBB without stepping into defBB.
	reachable := map[llvm.BasicBlock]bool{allocBB: true}
	queue := []llvm.BasicBlock{allocBB}
	for len(queue) > 0 {
		bb := queue[0]
		queue = queue[1:]
		term := bb.LastInstruction()
		for i := 0; i < term.SuccessorsCount(); i++ {
			succ := term.Successor(i)
			if succ == defBB || reachable[succ] {
				continue
			}
			reachable[succ] = true
			queue = append(queue, succ)
		}
	}
	for use := v.FirstUse(); !use.IsNil(); use = use.NextUse() {
		user := use.User()
		if user.InstructionOpcode() == llvm.PHI {
			for i := 0; i < user.IncomingCount(); i++ {
				if user.IncomingValue(i) == v && reachable[user.IncomingBlock(i)] {
					return true
				}
			}
			continue
		}
		userBB := user.InstructionParent()
		if userBB == allocBB {
			if pos, ok := order[user]; !ok || pos > allocPos {
				return true
			}
			continue
		}
		if userBB != defBB && reachable[userBB] {
			return true
		}
	}
	return false
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
