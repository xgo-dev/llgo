package fmtspecialize

import (
	"go/constant"
	"go/types"
	"slices"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/types/typeutil"
)

type ValuePlan struct {
	callee *ssa.Function
	names  map[*types.Package]*typeutil.Map
	calls  []formatRewrite
}

type formatRewrite struct {
	call     *ssa.Call
	function *ssa.Function
	input    ssa.Value
	packed   []ssa.Instruction
	saved    ssa.CallCommon
}

// NewValuePlan tracks replacements for default-value formatting.
func NewValuePlan(prog *ssa.Program, names map[*types.Package]*typeutil.Map) *ValuePlan {
	pkg := prog.ImportedPackage("fmt")
	if len(names) == 0 || pkg == nil || pkg.Func("Sprintf") == nil {
		return nil
	}
	return &ValuePlan{callee: pkg.Func("Sprintf"), names: names}
}

// HasCalls reports whether the plan found any eligible call sites.
func (plan *ValuePlan) HasCalls() bool {
	return plan != nil && len(plan.calls) != 0
}

// IsGenericFormatter identifies functions that keep the generic formatter.
func (plan *ValuePlan) IsGenericFormatter(fn *ssa.Function) bool {
	return fn.Pkg == plan.callee.Pkg && (fn.Name() == "printArg" || fn.Name() == "printValue")
}

// Apply temporarily rewrites the calls while accept inspects the call graph.
// Return the packages whose calls were replaced, or nil after restoring SSA.
func (plan *ValuePlan) Apply(accept func() bool) map[*types.Package]bool {
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
	if !ok || owner == nil || plan.names[owner.Pkg] == nil {
		return
	}
	common := call.Common()
	if common.StaticCallee() != plan.callee || len(common.Args) != 2 {
		return
	}
	format, ok := common.Args[0].(*ssa.Const)
	if !ok || format.Value == nil || format.Value.Kind() != constant.String || constant.StringVal(format.Value) != "%v" {
		return
	}
	values := formatValues(call, common.Args[1])
	if len(values) != 1 || values[0] == nil {
		return
	}
	box := values[0]
	if refs := box.Referrers(); refs == nil || len(*refs) != 1 {
		return
	}
	name, ok := plan.names[owner.Pkg].At(box.X.Type()).(string)
	if !ok {
		return
	}
	helper := owner.Func(name)
	if helper == nil {
		return
	}
	varargs := common.Args[1].(*ssa.Slice)
	array := varargs.X.(*ssa.Alloc)
	packed := []ssa.Instruction{box, array, varargs}
	for _, ref := range *array.Referrers() {
		if addr, ok := ref.(*ssa.IndexAddr); ok {
			packed = append(packed, addr, (*addr.Referrers())[0])
		}
	}
	plan.calls = append(plan.calls, formatRewrite{call: call, function: helper, input: box.X, packed: packed})
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
func defaultFormatType(t types.Type, top bool, visiting map[types.Type]bool) bool {
	t = types.Unalias(t)
	if visiting[t] {
		return false
	}
	visiting[t] = true
	defer delete(visiting, t)
	methods := types.NewMethodSet(t)
	for i := 0; i < methods.Len(); i++ {
		switch methods.At(i).Obj().Name() {
		case "Format", "String", "Error":
			return false
		}
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		return u.Kind() >= types.Bool && u.Kind() <= types.UnsafePointer
	case *types.Array:
		return defaultFormatType(u.Elem(), false, visiting)
	case *types.Slice:
		return defaultFormatType(u.Elem(), false, visiting)
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			f := u.Field(i)
			if !defaultFormatType(f.Type(), false, visiting) {
				return false
			}
		}
		return true
	case *types.Map:
		return defaultFormatType(u.Key(), false, visiting) && defaultFormatType(u.Elem(), false, visiting)
	case *types.Chan:
		return true
	case *types.Pointer:
		return !top || !formatDereference(u.Elem()) || defaultFormatType(u.Elem(), false, visiting)
	default:
		return false
	}
}

func (plan *ValuePlan) tentativeRewrite() (map[*ssa.BasicBlock][]ssa.Instruction, map[ssa.Instruction]bool) {
	remove := make(map[ssa.Instruction]bool)
	blocks := make(map[*ssa.BasicBlock][]ssa.Instruction)
	for i := range plan.calls {
		rewrite := &plan.calls[i]
		common := rewrite.call.Common()
		rewrite.saved = *common
		common.Value = rewrite.function
		common.Args = []ssa.Value{rewrite.input}
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
		for _, arg := range rewrite.saved.Args {
			removeFormatRef(arg, rewrite.call)
		}
		if refs := rewrite.input.Referrers(); refs != nil {
			*refs = append(*refs, rewrite.call)
		}
		owners[rewrite.call.Parent().Pkg.Pkg] = true
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
