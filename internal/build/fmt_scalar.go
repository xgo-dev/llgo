package build

import (
	"go/constant"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

func specializeScalarFormat(call ssa.CallInstruction, callee *ssa.Function, format string) {
	helper := callee.Pkg.Func("llgo" + callee.Name() + "Scalars")
	if helper == nil {
		return
	}
	operands := scalarFormatOperands(call, call.Common().Args[1])
	if len(operands) == 0 {
		return
	}
	if text, ok := scalarFormatString(format, operands); ok {
		common := call.Common()
		common.Value = helper
		if text != format {
			common.Args[0] = ssa.NewConst(constant.MakeString(text), common.Args[0].Type())
		}
	}
}

func specializeScalarPrintln(call ssa.CallInstruction, callee *ssa.Function) {
	helperName := "llgoPrintfScalars"
	if callee.Name() == "Sprintln" {
		helperName = "llgoSprintfScalars"
	}
	helper := callee.Pkg.Func(helperName)
	if helper == nil {
		return
	}
	common := call.Common()
	args := common.Args[0]
	operands := scalarFormatOperands(call, args)
	if len(operands) == 0 {
		return
	}
	// Println formats every operand with %v, with one space between
	// operands and a final newline. Resolve the verbs from the known types.
	format := make([]byte, 0, len(operands)*3)
	for i, kind := range operands {
		verb := scalarDefaultVerb(kind)
		if verb == 0 {
			return
		}
		if i != 0 {
			format = append(format, ' ')
		}
		format = append(format, '%', verb)
	}
	format = append(format, '\n')
	common.Value = helper
	common.Args = []ssa.Value{ssa.NewConst(constant.MakeString(string(format)), types.Typ[types.String]), args}
}

// x/tools emits this array for individually supplied variadic arguments.
// Require every element to come from an undisguised scalar conversion.
func scalarFormatOperands(call ssa.CallInstruction, variadic ssa.Value) []types.BasicKind {
	args, ok := variadic.(*ssa.Slice)
	if !ok || args.Low != nil || args.High != nil || args.Max != nil {
		return nil
	}
	array, ok := args.X.(*ssa.Alloc)
	if !ok || array.Comment != "varargs" {
		return nil
	}
	if refs := args.Referrers(); len(*refs) != 1 || (*refs)[0] != call {
		return nil
	}
	n := array.Type().(*types.Pointer).Elem().(*types.Array).Len()
	kinds := make([]types.BasicKind, n)
	for _, ref := range *array.Referrers() {
		if ref == args {
			continue
		}
		addr, ok := ref.(*ssa.IndexAddr)
		if !ok || len(*addr.Referrers()) != 1 {
			return nil
		}
		index, ok := addr.Index.(*ssa.Const)
		if !ok {
			return nil
		}
		i, ok := constant.Int64Val(index.Value)
		if !ok || i < 0 || i >= n || kinds[i] != types.Invalid {
			return nil
		}
		store, ok := (*addr.Referrers())[0].(*ssa.Store)
		if !ok || store.Addr != addr {
			return nil
		}
		value, ok := store.Val.(*ssa.MakeInterface)
		if !ok {
			return nil
		}
		// A defined type may implement Formatter, Stringer or error.
		basic, ok := types.Unalias(value.X.Type()).(*types.Basic)
		if !ok {
			return nil
		}
		kinds[i] = basic.Kind()
	}
	return kinds
}

func scalarFormatString(format string, operands []types.BasicKind) (string, bool) {
	var resolved []byte
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		if i == len(format) {
			return "", false
		}
		if format[i] == '%' {
			continue
		}
		// Support optional zero padding and decimal widths through 99.
		// Widths, flags and verbs outside this subset retain the normal path.
		if format[i] == '0' {
			i++
		}
		start := i
		for i < len(format) && '0' <= format[i] && format[i] <= '9' {
			i++
		}
		if i == len(format) || i-start > 2 || len(operands) == 0 {
			return "", false
		}
		kind := operands[0]
		switch format[i] {
		case 'v':
			verb := scalarDefaultVerb(kind)
			if verb == 0 {
				return "", false
			}
			if resolved == nil {
				resolved = []byte(format)
			}
			resolved[i] = verb
		case 's':
			if kind != types.String {
				return "", false
			}
		case 't':
			if kind != types.Bool {
				return "", false
			}
		case 'b', 'd', 'o', 'x', 'X':
			if scalarDefaultVerb(kind) != 'd' {
				return "", false
			}
		default:
			return "", false
		}
		operands = operands[1:]
	}
	if len(operands) != 0 {
		return "", false
	}
	if resolved != nil {
		return string(resolved), true
	}
	return format, true
}

func scalarDefaultVerb(kind types.BasicKind) byte {
	switch {
	case kind == types.String:
		return 's'
	case kind == types.Bool:
		return 't'
	case kind >= types.Int && kind <= types.Uintptr:
		return 'd'
	default:
		return 0
	}
}
