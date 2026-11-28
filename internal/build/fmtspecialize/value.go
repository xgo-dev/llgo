package fmtspecialize

import (
	"go/constant"
	"go/types"
	"slices"

	"golang.org/x/tools/go/ssa"
)

type ValuePlan struct {
	callee      *ssa.Function
	builtin     *ssa.Function
	printf      *ssa.Function
	plainPrintf *ssa.Function
	pending     []formatRewrite
	names       map[*types.Package]*ValueFunctions
	calls       []formatRewrite
}

type formatRewrite struct {
	call         *ssa.Call
	function     *ssa.Function
	inputs       []ssa.Value
	builtinKinds uint32
	packed       []ssa.Instruction
	saved        ssa.CallCommon
}

// NewValuePlan tracks replacements for default-value formatting.
func NewValuePlan(prog *ssa.Program, names map[*types.Package]*ValueFunctions) *ValuePlan {
	pkg := prog.ImportedPackage("fmt")
	if pkg == nil || pkg.Func("Sprintf") == nil || len(names) == 0 && pkg.Func(builtinSprintf) == nil && pkg.Func(builtinPrintf) == nil {
		return nil
	}
	return &ValuePlan{callee: pkg.Func("Sprintf"), builtin: pkg.Func(builtinSprintf),
		printf: pkg.Func(builtinPrintf), plainPrintf: pkg.Func("Printf"), names: names}
}

// HasCalls reports whether the plan found any eligible call sites.
func (plan *ValuePlan) HasCalls() bool {
	return plan != nil && (len(plan.calls) != 0 || len(plan.pending) != 0)
}

// isGenericFormatter identifies functions that keep the generic formatter.
func (plan *ValuePlan) isGenericFormatter(fn *ssa.Function) bool {
	return fn.Pkg == plan.callee.Pkg && (fn.Name() == "printArg" || fn.Name() == "printValue")
}

// apply temporarily rewrites the calls while accept inspects the call graph.
// Return the packages whose calls were replaced, or nil after restoring SSA.
func (plan *ValuePlan) apply(accept func() bool) map[*types.Package]bool {
	blocks, removed := plan.tentativeRewrite()
	if !accept() {
		plan.restore(blocks)
		return nil
	}
	return plan.commit(removed)
}

// Record adds a call to the plan when its packed operands are used only here.
func (plan *ValuePlan) Record(instruction ssa.Instruction, owner *ssa.Package) {
	call, ok := instruction.(*ssa.Call)
	if !ok || owner == nil || owner.Pkg.Name() != "main" {
		return
	}
	common := call.Common()
	functions := plan.names[owner.Pkg]
	if callee := common.StaticCallee(); functions != nil && callee != nil && callee.Pkg == plan.callee.Pkg {
		if name := functions.stdout[call.Pos()]; name != "" && (callee.Name() == "Printf" || callee.Name() == "Println") {
			plan.recordTyped(call, owner.Func(name), formatValues(call, common.Args[len(common.Args)-1]))
			return
		}
	}
	callee := common.StaticCallee()
	if len(common.Args) != 2 {
		return
	}
	format, ok := common.Args[0].(*ssa.Const)
	switch callee {
	case plan.callee:
		if plan.recordBuiltin(call, plan.builtin, ok) || !ok {
			return
		}
	case plan.plainPrintf:
		if callee != nil {
			plan.recordBuiltin(call, plan.printf, ok)
		}
		return
	default:
		return
	}
	if functions == nil || !ok || format.Value == nil || format.Value.Kind() != constant.String || constant.StringVal(format.Value) != "%v" {
		return
	}
	values := formatValues(call, common.Args[1])
	if len(values) != 1 || values[0] == nil {
		return
	}
	if name, ok := functions.values.At(values[0].X.Type()).(string); ok {
		plan.recordTyped(call, owner.Func(name), values)
	}
}

func (plan *ValuePlan) recordTyped(call *ssa.Call, helper *ssa.Function, operands []*ssa.MakeInterface) {
	if helper == nil || len(operands) == 0 {
		return
	}
	params := helper.Signature.Params()
	if len(operands) != params.Len() {
		return
	}
	args := call.Common().Args
	varargs := args[len(args)-1].(*ssa.Slice)
	array := varargs.X.(*ssa.Alloc)
	packed := []ssa.Instruction{array, varargs}
	inputs := make([]ssa.Value, len(operands))
	for i, box := range operands {
		if box == nil || !types.Identical(box.X.Type(), params.At(i).Type()) {
			return
		}
		if refs := box.Referrers(); refs == nil || len(*refs) != 1 {
			return
		}
		inputs[i] = box.X
		packed = append(packed, box)
	}
	for _, ref := range *array.Referrers() {
		if addr, ok := ref.(*ssa.IndexAddr); ok {
			packed = append(packed, addr, (*addr.Referrers())[0])
		}
	}
	plan.calls = append(plan.calls, formatRewrite{call: call, function: helper, inputs: inputs, packed: packed})
}

func formatDereference(t types.Type) bool {
	switch t.Underlying().(type) {
	case *types.Array, *types.Slice, *types.Struct, *types.Map:
		return true
	default:
		return false
	}
}

// defaultFormatType checks the actual paths visited by a default formatter.
// A named value can have methods that override formatting at any level.
func defaultFormatType(t types.Type, top bool, visiting map[types.Type]bool, methods map[*types.Func]bool) bool {
	t = types.Unalias(t)
	if visiting[t] {
		return false
	}
	visiting[t] = true
	defer delete(visiting, t)
	method, ok := defaultValueMethod(t, methods)
	if !ok || method != "" {
		return ok
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		return u.Kind() >= types.Bool && u.Kind() <= types.UnsafePointer
	case *types.Array:
		return defaultFormatType(u.Elem(), false, visiting, methods)
	case *types.Slice:
		return defaultFormatType(u.Elem(), false, visiting, methods)
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			f := u.Field(i)
			visible := methods
			if !f.Exported() {
				visible = nil
			}
			if !defaultFormatType(f.Type(), false, visiting, visible) {
				return false
			}
		}
		return true
	case *types.Map:
		return defaultFormatType(u.Key(), false, visiting, methods) && defaultFormatType(u.Elem(), false, visiting, methods)
	case *types.Chan:
		return true
	case *types.Pointer:
		return !top || !formatDereference(u.Elem()) || defaultFormatType(u.Elem(), false, visiting, methods)
	default:
		return false
	}
}

func (plan *ValuePlan) tentativeRewrite() (map[*ssa.BasicBlock][]ssa.Instruction, map[ssa.Instruction]bool) {
	remove := make(map[ssa.Instruction]bool)
	blocks := make(map[*ssa.BasicBlock][]ssa.Instruction)
	// Keep the constant local to each executable package. Its callers pass
	// the same value while fmt's archive stays reusable across executables.
	kinds := make(map[*ssa.Package]uint32)
	for _, rewrite := range plan.calls {
		kinds[rewrite.call.Parent().Pkg] |= rewrite.builtinKinds
	}
	for i := range plan.calls {
		rewrite := &plan.calls[i]
		common := rewrite.call.Common()
		rewrite.saved = *common
		common.Value = rewrite.function
		if rewrite.inputs != nil {
			common.Args = rewrite.inputs
		} else if rewrite.builtinKinds != 0 {
			mask := ssa.NewConst(constant.MakeUint64(uint64(kinds[rewrite.call.Parent().Pkg])), types.Typ[types.Uint32])
			common.Args = []ssa.Value{common.Args[0], mask, common.Args[1]}
		}
		for _, instruction := range rewrite.packed {
			remove[instruction] = true
			block := instruction.Block()
			blocks[block] = block.Instrs
		}
	}
	for block, original := range blocks {
		block.Instrs = slices.DeleteFunc(slices.Clone(original), func(instruction ssa.Instruction) bool {
			return remove[instruction]
		})
	}
	return blocks, remove
}

func (plan *ValuePlan) restore(blocks map[*ssa.BasicBlock][]ssa.Instruction) {
	for block, original := range blocks {
		block.Instrs = original
	}
	for _, rewrite := range plan.calls {
		*rewrite.call.Common() = rewrite.saved
	}
}

func removeFormatRef(value ssa.Value, instruction ssa.Instruction) {
	if refs := value.Referrers(); refs != nil {
		*refs = slices.DeleteFunc(*refs, func(ref ssa.Instruction) bool { return ref == instruction })
	}
}

func (plan *ValuePlan) commit(remove map[ssa.Instruction]bool) map[*types.Package]bool {
	owners := make(map[*types.Package]bool)
	for _, rewrite := range plan.calls {
		owners[rewrite.call.Parent().Pkg.Pkg] = true
		if rewrite.inputs == nil {
			continue
		}
		for _, arg := range rewrite.saved.Args {
			removeFormatRef(arg, rewrite.call)
		}
		for _, input := range rewrite.inputs {
			if refs := input.Referrers(); refs != nil {
				*refs = append(*refs, rewrite.call)
			}
		}
	}
	for instruction := range remove {
		for _, operand := range instruction.Operands(nil) {
			if *operand != nil {
				removeFormatRef(*operand, instruction)
			}
		}
		if alloc, ok := instruction.(*ssa.Alloc); ok {
			fn := alloc.Parent()
			fn.Locals = slices.DeleteFunc(fn.Locals, func(local *ssa.Alloc) bool { return local == alloc })
		}
	}
	return owners
}

// BuiltinPrintfCalls appends the committed builtin calls in the entry package.
func (plan *ValuePlan) BuiltinPrintfCalls(entry *ssa.Package) []*ssa.Call {
	var calls []*ssa.Call
	if plan.printf != nil {
		for _, rewrite := range plan.calls {
			if entry == rewrite.call.Parent().Pkg && rewrite.call.Common().StaticCallee() == plan.printf {
				calls = append(calls, rewrite.call)
			}
		}
	}
	return calls
}
