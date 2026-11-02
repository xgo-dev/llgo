package build

import (
	"go/constant"
	"strings"

	"github.com/xgo-dev/llgo/internal/optlevel"
	"golang.org/x/tools/go/ssa"
)

func canSpecializeLiteralFormats(ctx *context) bool {
	conf := ctx.buildConf
	return conf != nil && conf.OptLevel != optlevel.O0 &&
		ctx.mode != ModeGen && !ctx.frontendOptions.DebuggerFrames
}

// Rewrite during the instruction-counting pass, before reachability analysis,
// so a literal write need not retain the general formatter and reflection.
func specializeLiteralFormat(instruction ssa.Instruction) {
	call, ok := instruction.(ssa.CallInstruction)
	if !ok {
		return
	}
	common := call.Common()
	callee := common.StaticCallee()
	if callee == nil || callee.Pkg == nil || callee.Pkg.Pkg.Path() != "fmt" {
		return
	}
	// An arbitrary Writer can inspect native or Wasm logical callers. Keep
	// Fprintf until a helper can preserve its original frame identity.
	switch callee.Name() {
	case "Printf", "Sprintf":
	default:
		return
	}
	if len(common.Args) != 2 {
		return
	}
	args, ok := common.Args[1].(*ssa.Const)
	if !ok || !args.IsNil() {
		return
	}
	format, ok := common.Args[0].(*ssa.Const)
	if !ok || format.Value == nil || format.Value.Kind() != constant.String {
		return
	}
	text, ok := literalFormat(constant.StringVal(format.Value))
	if !ok {
		return
	}
	helper := callee.Pkg.Func("llgo" + callee.Name() + "Literal")
	if helper == nil {
		return
	}
	common.Value = helper
	common.Args = common.Args[:1]
	common.Args[0] = ssa.NewConst(constant.MakeString(text), format.Type())
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
