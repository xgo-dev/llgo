package funcattrs

import (
	"fmt"
	"go/types"

	"github.com/xgo-dev/llvm"
)

type resultFact struct {
	Entry        bool // From identifies a logical input, rather than a result relation.
	Name         string
	Path         []int // The whole Go result within the logical return tuple.
	From         int   // LLVM input index for sameas, or -1 for an independent value fact.
	Lower, Upper uint64
}

// ValuePlan belongs to one LLVM function before ABI conversion.
type ValuePlan []resultFact

// ResultPathResolver accounts for padding wrappers in a logical Go result tuple.
type ResultPathResolver func(index int) []int

func PrepareResultAttributes(ctx llvm.Context, fn llvm.Value, sig *types.Signature, attrs []Attribute, environment, intBits int, resolver ResultPathResolver) (ValuePlan, error) {
	var plan ValuePlan
	for _, source := range attrs {
		if source.Target.Scope != Result {
			continue
		}
		fact := resultFact{Name: source.Name, From: -1}
		if sig.Results().Len() > 1 {
			fact.Path = []int{source.Target.Index}
			if resolver != nil {
				fact.Path = resolver(source.Target.Index)
			}
		}
		typ := fn.GlobalValueType().ReturnType()
		for _, index := range fact.Path {
			if typ.TypeKind() != llvm.StructTypeKind {
				return nil, source.Error("selected result has no LLVM tuple")
			}
			fields := typ.StructElementTypes()
			if index < 0 || index >= len(fields) {
				return nil, source.Error("selected result has no LLVM value")
			}
			typ = fields[index]
		}
		selected, err := ResolveTarget(sig, source.Target)
		if err != nil {
			return nil, source.Error("%v", err)
		}
		switch source.Name {
		case "nonnull":
			if typ.TypeKind() != llvm.PointerTypeKind {
				return nil, source.Error("selected result is not an LLVM pointer")
			}
		case "range", "nonnegative":
			bits, bounds, full, err := IntegerRange(source, selected, intBits)
			if err != nil {
				return nil, err
			}
			if typ.TypeKind() != llvm.IntegerTypeKind || typ.IntTypeWidth() != bits {
				return nil, source.Error("selected result does not have its source integer width")
			}
			if full {
				continue
			}
			fact.Lower, fact.Upper = bounds[0], bounds[1]
		case "sameas":
			if source.From == nil {
				return nil, source.Error("sameas requires a parameter name")
			}
			fact.From = source.From.Index + environment
			if sig.Recv() != nil {
				fact.From++
			}
			if fact.From < 0 || fact.From >= fn.ParamsCount() || fn.Param(fact.From).Type() != typ {
				return nil, source.Error("sameas values do not share an LLVM representation")
			}
			// Current collectors do not move objects or native stacks.
			if len(fact.Path) == 0 {
				fn.AddAttributeAtIndex(fact.From+1, ctx.CreateEnumAttribute(llvm.AttributeKindID("returned"), 0))
			}
		}
		plan = append(plan, fact)
	}
	for _, source := range attrs {
		if source.Target.Scope == Result || source.Name == "access" || source.Name == "noalias" {
			continue
		}
		index := environment
		if source.Target.Scope == Parameter {
			index += source.Target.Index
			if sig.Recv() != nil {
				index++
			}
		}
		if index >= fn.ParamsCount() {
			return nil, source.Error("selected parameter has no LLVM value")
		}
		fact := resultFact{Entry: true, Name: source.Name, From: index}
		if source.Name != "nonnull" {
			selected, _ := ResolveTarget(sig, source.Target)
			_, bounds, full, err := IntegerRange(source, selected, intBits)
			if err != nil {
				return nil, err
			}
			if full {
				continue
			}
			fact.Lower, fact.Upper = bounds[0], bounds[1]
		}
		plan = append(plan, fact)
	}
	return plan, nil
}

// MaterializeValueContracts applies result guarantees on normal continuations
// before ABI conversion reconstructs the values. No plan is serialized to bitcode.
func MaterializeValueContracts(m llvm.Module, plans map[llvm.Value]ValuePlan) error {
	if len(plans) == 0 {
		return nil
	}
	ctx := m.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	var calls []llvm.Value
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
				if !instr.IsACallInst().IsNil() || !instr.IsAInvokeInst().IsNil() {
					if _, ok := plans[instr.CalledValue()]; ok {
						calls = append(calls, instr)
					}
				}
			}
		}
	}
	for _, call := range calls {
		if call.CalledFunctionType() != call.CalledValue().GlobalValueType() {
			return fmt.Errorf("value contract call to %s has an incompatible logical prototype", call.CalledValue().Name())
		}
	}
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IsDeclaration() {
			continue
		}
		plan := plans[fn]
		if len(plan) == 0 {
			continue
		}
		b.SetInsertPointBefore(fn.EntryBasicBlock().FirstInstruction())
		for _, fact := range plan {
			if fact.Entry {
				emitValueFact(b, fn.Param(fact.From), fact)
			}
		}
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			ret := bb.LastInstruction()
			if ret.IsNil() || ret.IsAReturnInst().IsNil() || ret.OperandsCount() == 0 {
				continue
			}
			b.SetInsertPointBefore(ret)
			for _, fact := range plan {
				if fact.Entry {
					continue
				}
				value := extractValue(b, ret.Operand(0), fact.Path)
				if fact.From >= 0 {
					equal := b.CreateICmp(llvm.IntEQ, value, fn.Param(fact.From), "contract.same")
					b.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.assume"), []llvm.Value{equal}, "")
				} else {
					emitValueFact(b, value, fact)
				}
			}
		}
	}
	for _, call := range calls {
		materializeCallResults(b, call, plans[call.CalledValue()])
	}

	return nil
}

func extractValue(b llvm.Builder, value llvm.Value, path []int) llvm.Value {
	for _, index := range path {
		value = b.CreateExtractValue(value, index, "contract.value")
	}
	return value
}

func materializeCallResults(b llvm.Builder, call llvm.Value, plan ValuePlan) {
	var results []resultFact
	for _, fact := range plan {
		if !fact.Entry {
			results = append(results, fact)
		}
	}
	if len(results) == 0 {
		return
	}
	continuation := normalContinuation(b, call)
	// Capture existing users before constructing replacement aggregates: a
	// blanket RAUW would replace their old-call operand with themselves.
	var users []llvm.Value
	seen := make(map[llvm.Value]bool)
	for use := call.FirstUse(); !use.IsNil(); use = use.NextUse() {
		if user := use.User(); !seen[user] {
			seen[user] = true
			users = append(users, user)
		}
	}
	b.SetInsertPointBefore(continuation)
	result := call
	for _, contract := range results {
		if contract.From < 0 {
			continue
		}
		// Arguments are already evaluated SSA snapshots. In particular, never
		// reload a mutable parameter home after the call. Pointer forwarding
		// retains the original access provenance, unlike address equality alone.
		input := call.Operand(contract.From)
		if len(contract.Path) == 0 {
			result = input
			continue
		}
		// Forward existing leaf projections, not the aggregate's unrelated
		// fields or padding. Rebuilding a giant aggregate with insertvalue would
		// defeat the ABI's indirect copies solely to carry a relation. Whole
		// value transfers can retain their original result: its field already
		// has the promised identity. The equality fact also relates later
		// projections without inventing pointer access provenance.
		leaves := matchingProjections(call, contract.Path)
		actual := extractValue(b, call, contract.Path)
		equal := b.CreateICmp(llvm.IntEQ, actual, input, "contract.same")
		b.CreateIntrinsic(call.Type().Context().VoidType(), llvm.LookupIntrinsicID("llvm.assume"), []llvm.Value{equal}, "")
		for _, leaf := range leaves {
			leaf.ReplaceAllUsesWith(input)
		}
	}
	for _, contract := range results {
		if contract.From < 0 {
			value := extractValue(b, result, contract.Path)
			emitValueFact(b, value, contract)
		}
	}
	if result != call {
		for _, user := range users {
			for i := 0; i < user.OperandsCount(); i++ {
				if user.Operand(i) == call {
					user.SetOperand(i, result)
				}
			}
		}
	}
	// Postconditions require a normal continuation; retaining a mandatory
	// tail-call form would leave no legal place for reconstruction and facts.
	if !call.IsACallInst().IsNil() {
		call.SetTailCall(false)
	}
}

func matchingProjections(value llvm.Value, path []int) []llvm.Value {
	var matches []llvm.Value
	for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
		extract := use.User().IsAExtractValueInst()
		if extract.IsNil() {
			continue
		}
		indices := extract.Indices()
		if len(indices) > len(path) {
			continue
		}
		prefix := true
		for i, index := range indices {
			if int(index) != path[i] {
				prefix = false
				break
			}
		}
		if !prefix {
			continue
		}
		if len(indices) == len(path) {
			matches = append(matches, extract)
		} else {
			matches = append(matches, matchingProjections(extract, path[len(indices):])...)
		}
	}
	return matches
}

// normalContinuation makes invoke facts local to its normal edge. Splitting
// unconditionally also handles shared normal destinations and their PHIs.
func normalContinuation(b llvm.Builder, call llvm.Value) llvm.Value {
	if call.IsAInvokeInst().IsNil() {
		return llvm.NextInstruction(call)
	}
	parent := call.InstructionParent()
	normal := call.Successor(0)
	ctx := call.Type().Context()
	edge := ctx.AddBasicBlock(parent.Parent(), "contract.normal")
	b.SetInsertPointAtEnd(edge)
	branch := b.CreateBr(normal)
	for i := 0; i < call.OperandsCount(); i++ {
		if call.Operand(i) == normal.AsValue() {
			call.SetOperand(i, edge.AsValue())
			break
		}
	}
	for phi := normal.FirstInstruction(); !phi.IsNil() && !phi.IsAPHINode().IsNil(); {
		next := llvm.NextInstruction(phi)
		values := make([]llvm.Value, phi.IncomingCount())
		blocks := make([]llvm.BasicBlock, len(values))
		for i := range values {
			values[i], blocks[i] = phi.IncomingValue(i), phi.IncomingBlock(i)
			if blocks[i] == parent {
				blocks[i] = edge
			}
		}
		b.SetInsertPointBefore(phi)
		replacement := b.CreatePHI(phi.Type(), "contract.merge")
		replacement.AddIncoming(values, blocks)
		phi.ReplaceAllUsesWith(replacement)
		phi.EraseFromParentAsInstruction()
		phi = next
	}
	return branch
}

func emitValueFact(b llvm.Builder, value llvm.Value, contract resultFact) {
	ctx := value.Type().Context()
	var predicate llvm.Value
	switch contract.Name {
	case "nonnull":
		predicate = b.CreateICmp(llvm.IntNE, value, llvm.ConstNull(value.Type()), "contract.nonnull")
	case "range", "nonnegative":
		// Subtraction in the source bit width makes signed intervals that cross
		// zero work without constraining any ABI carrier or its padding bits.
		lower := llvm.ConstInt(value.Type(), contract.Lower, false)
		width := llvm.ConstInt(value.Type(), contract.Upper-contract.Lower, false)
		distance := b.CreateSub(value, lower, "contract.range.offset")
		predicate = b.CreateICmp(llvm.IntULT, distance, width, "contract.range")
	default:
		return
	}
	b.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.assume"), []llvm.Value{predicate}, "")
}
