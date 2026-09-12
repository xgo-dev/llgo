package funcattrs

import (
	"encoding/json"
	"fmt"
	"go/types"

	"github.com/xgo-dev/llvm"
)

const valuePlanMetadata = "llgo.value.contracts.v1"

// valueSubject describes the logical LLVM signature before any ABI conversion.
// Parameter -1 denotes the return value; Path selects a leaf within that value.
// This backend-only binding is never the exported source contract model.
type valueSubject struct {
	Parameter int
	Path      []int
}

type boundValueContract struct {
	Source Attribute
	Target valueSubject
	From   *valueSubject
	Bits   int
	Lower  uint64
	Upper  uint64
}

type valuePlan struct {
	Version   int
	Contracts []boundValueContract
}

func isValueContract(name string) bool {
	switch name {
	case "nonnull", "range", "nonnegative", "same_as":
		return true
	}
	return false
}

func bindValueSubject(sig *types.Signature, target Target, environment int, resolver ResultPathResolver) (valueSubject, types.Type, error) {
	leaf, err := ResolveTarget(sig, target)
	if err != nil {
		return valueSubject{}, nil, err
	}
	bound := valueSubject{Parameter: -1}
	switch target.Scope {
	case Receiver:
		bound.Parameter = environment
	case Parameter:
		bound.Parameter = environment + target.Index
		if sig.Recv() != nil {
			bound.Parameter++
		}
	}
	if target.Scope == Result && sig.Results().Len() > 1 {
		bound.Path = []int{target.Index}
		if resolver != nil {
			bound.Path = resolver(target.Index)
		}
	}
	return bound, leaf, err
}

func subjectLLVMType(fn llvm.Value, subject valueSubject) (llvm.Type, error) {
	var typ llvm.Type
	if subject.Parameter < 0 {
		typ = fn.GlobalValueType().ReturnType()
	} else {
		params := fn.GlobalValueType().ParamTypes()
		if subject.Parameter >= len(params) {
			return llvm.Type{}, fmt.Errorf("source parameter has no logical LLVM value")
		}
		typ = params[subject.Parameter]
	}
	for _, index := range subject.Path {
		switch typ.TypeKind() {
		case llvm.StructTypeKind:
			fields := typ.StructElementTypes()
			if index < 0 || index >= len(fields) {
				return llvm.Type{}, fmt.Errorf("source field has no LLVM representation")
			}
			typ = fields[index]
		case llvm.ArrayTypeKind:
			if index < 0 || index >= typ.ArrayLength() {
				return llvm.Type{}, fmt.Errorf("source element has no LLVM representation")
			}
			typ = typ.ElementType()
		default:
			return llvm.Type{}, fmt.Errorf("source selector traverses a nonaggregate LLVM value")
		}
	}
	return typ, nil
}

func prepareValueContracts(ctx llvm.Context, fn llvm.Value, sig *types.Signature, attrs []Attribute, environment, intBits int, resolver ResultPathResolver) error {
	plan := valuePlan{Version: 1}
	for _, source := range attrs {
		if !isValueContract(source.Name) {
			continue
		}
		target, leaf, err := bindValueSubject(sig, source.Target, environment, resolver)
		if err != nil {
			return source.Error("%v", err)
		}
		typ, err := subjectLLVMType(fn, target)
		if err != nil {
			return source.Error("%v", err)
		}
		bound := boundValueContract{Source: source, Target: target}
		index := target.Parameter + 1
		direct := len(target.Path) == 0
		switch source.Name {
		case "nonnull":
			if typ.TypeKind() != llvm.PointerTypeKind {
				return source.Error("pointer contract does not select a logical LLVM pointer")
			}
			if direct {
				fn.AddAttributeAtIndex(index, ctx.CreateEnumAttribute(llvm.AttributeKindID("nonnull"), 0))
			}
		case "range", "nonnegative":
			bits, bounds, full, err := IntegerRange(source, leaf, intBits)
			if err != nil {
				return err
			}
			if typ.TypeKind() != llvm.IntegerTypeKind || typ.IntTypeWidth() != bits {
				return source.Error("integer contract does not select its source-width LLVM integer")
			}
			if full {
				continue
			}
			bound.Bits, bound.Lower, bound.Upper = bits, bounds[0], bounds[1]
			if direct {
				fn.AddAttributeAtIndex(index, ctx.CreateConstantRangeAttribute(llvm.AttributeKindID("range"), bits, bounds[:1], bounds[1:]))
			}
		case "same_as":
			if source.From == nil {
				return source.Error("same_as is missing its input selector")
			}
			from, _, err := bindValueSubject(sig, *source.From, environment, resolver)
			if err != nil {
				return source.Error("%v", err)
			}
			fromType, err := subjectLLVMType(fn, from)
			if err != nil || fromType != typ {
				return source.Error("same_as values do not share a logical LLVM representation")
			}
			bound.From = &from
			// Current LLGo collectors do not relocate objects or native stacks.
			// The native shortcut is valid only while these machine values are
			// stable; logical forwarding below is the general representation.
			if direct && len(from.Path) == 0 && from.Parameter >= 0 {
				fn.AddAttributeAtIndex(from.Parameter+1, ctx.CreateEnumAttribute(llvm.AttributeKindID("returned"), 0))
			}
		}
		plan.Contracts = append(plan.Contracts, bound)
	}
	if len(plan.Contracts) != 0 {
		data, err := json.Marshal(plan)
		if err != nil {
			return err
		}
		fn.AddFunctionAttr(ctx.CreateStringAttribute(valuePlanMetadata, string(data)))
	}
	return nil
}

// MaterializeValueContracts runs once while function signatures and call values
// still have their logical representation. Subsequent ABI passes already replace
// old parameters and call results with reconstructed values, so these facts follow
// byval, packed, split, and sret conversions without another reconstruction layer.
// Temporary plans are removed before ABI conversion; source contracts remain.
func MaterializeValueContracts(m llvm.Module) error {
	plans := make(map[llvm.Value]valuePlan)
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		attr := fn.GetStringAttributeAtIndex(-1, valuePlanMetadata)
		if attr.IsNil() {
			continue
		}
		var plan valuePlan
		if err := json.Unmarshal([]byte(attr.GetStringValue()), &plan); err != nil {
			return fmt.Errorf("invalid value contract plan for %s: %w", fn.Name(), err)
		}
		if plan.Version != 1 {
			return fmt.Errorf("unsupported value contract plan version %d", plan.Version)
		}
		plans[fn] = plan
	}
	if len(plans) == 0 {
		return nil
	}
	ctx := m.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	for fn, plan := range plans {
		if fn.IsDeclaration() {
			continue
		}
		b.SetInsertPointBefore(fn.EntryBasicBlock().FirstInstruction())
		for _, contract := range plan.Contracts {
			if contract.Target.Parameter >= 0 {
				value := extractValue(b, fn.Param(contract.Target.Parameter), contract.Target.Path)
				emitValueFact(b, value, contract)
			}
		}
	}
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
		plan := plans[call.CalledValue()]
		if call.CalledFunctionType() != call.CalledValue().GlobalValueType() {
			return fmt.Errorf("value contract call to %s has an incompatible logical prototype", call.CalledValue().Name())
		}
		materializeCallResults(b, call, plan)
	}
	for fn := range plans {
		fn.RemoveStringAttributeAtIndex(-1, valuePlanMetadata)
	}
	return nil
}

func extractValue(b llvm.Builder, value llvm.Value, path []int) llvm.Value {
	for _, index := range path {
		value = b.CreateExtractValue(value, index, "contract.value")
	}
	return value
}

func materializeCallResults(b llvm.Builder, call llvm.Value, plan valuePlan) {
	var results []boundValueContract
	for _, contract := range plan.Contracts {
		if contract.Target.Parameter < 0 {
			results = append(results, contract)
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
		if contract.From == nil {
			continue
		}
		// Arguments are already evaluated SSA snapshots. In particular, never
		// reload a mutable parameter home after the call. Pointer forwarding
		// retains the original access provenance, unlike address equality alone.
		input := extractValue(b, call.Operand(contract.From.Parameter), contract.From.Path)
		if len(contract.Target.Path) == 0 {
			result = input
			continue
		}
		// Forward existing leaf projections, not the aggregate's unrelated
		// fields or padding. Rebuilding a giant aggregate with insertvalue would
		// defeat the ABI's indirect copies solely to carry a relation. Whole
		// value transfers can retain their original result: its field already
		// has the promised identity. The equality fact also relates later
		// projections without inventing pointer access provenance.
		leaves := matchingProjections(call, contract.Target.Path)
		actual := extractValue(b, call, contract.Target.Path)
		equal := b.CreateICmp(llvm.IntEQ, actual, input, "contract.same")
		b.CreateIntrinsic(call.Type().Context().VoidType(), llvm.LookupIntrinsicID("llvm.assume"), []llvm.Value{equal}, "")
		for _, leaf := range leaves {
			leaf.ReplaceAllUsesWith(input)
		}
	}
	for _, contract := range results {
		if contract.From == nil {
			value := extractValue(b, result, contract.Target.Path)
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

func emitValueFact(b llvm.Builder, value llvm.Value, contract boundValueContract) {
	ctx := value.Type().Context()
	var predicate llvm.Value
	switch contract.Source.Name {
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
