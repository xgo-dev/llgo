package calltemplate

import (
	"go/constant"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ssa"
)

// Rewrite during the instruction-counting pass, before reachability analysis,
// so a specialized format can release the general formatter and reflection.
func (b *Bindings) RewriteFormatCall(instruction ssa.Instruction, scalarFormats bool) {
	call, ok := instruction.(ssa.CallInstruction)
	if b == nil || !ok {
		return
	}
	common := call.Common()
	callee := common.StaticCallee()
	if callee == nil {
		return
	}
	fn, _ := callee.Object().(*types.Func)
	rule, ok := b.formats[fn]
	if !ok || (rule.literal == "" && rule.scalar == "") || len(common.Args) != rule.args+1 {
		return
	}
	if rule.format < 0 {
		if scalarFormats {
			specializeScalarPrintln(call, callee.Pkg.Func(rule.scalar))
		}
		return
	}
	format, ok := common.Args[rule.format].(*ssa.Const)
	if !ok || format.Value == nil || format.Value.Kind() != constant.String {
		return
	}
	if args, ok := common.Args[rule.args].(*ssa.Const); !ok || !args.IsNil() {
		if scalarFormats && rule.scalar != "" {
			specializeScalarFormat(call, callee.Pkg.Func(rule.scalar), constant.StringVal(format.Value))
		}
		return
	}
	text, ok := literalFormat(constant.StringVal(format.Value))
	if !ok || rule.literal == "" {
		return
	}
	helper := callee.Pkg.Func(rule.literal)
	if helper == nil {
		return
	}
	common.Value = helper
	common.Args = []ssa.Value{ssa.NewConst(constant.MakeString(text), format.Type())}
}

// Only %% is interpreted here. Verbs, indexes, width/precision and malformed
// formats continue through fmt, including its missing-argument diagnostics.
func literalFormat(format string) (string, bool) {
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		if i+1 == len(format) || format[i+1] != '%' {
			return "", false
		}
		i++
	}
	return strings.ReplaceAll(format, "%%", "%"), true
}
