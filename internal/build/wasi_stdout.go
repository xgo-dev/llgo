package build

import (
	"github.com/xgo-dev/llgo/cl"
	"golang.org/x/tools/go/ssa"
)

// The descriptor alias lives in the standard library archive. Only calls in
// the executable package change when a closed link proves it can use the alias.
type wasiStdoutPlan struct {
	entry           *ssa.Package
	regular, direct *ssa.Function
	calls           []ssa.CallInstruction
	specialized     bool
}

func newWASIStdoutPlan(ctx *context) *wasiStdoutPlan {
	if ctx.frontend != nil || len(ctx.initial) != 1 || ctx.initial[0].Name != "main" ||
		!wasiGuardedStaticData(ctx.buildConf, ctx.crossCompile, ctx.frontendOptions) {
		return nil
	}
	entry := ctx.progSSA.Package(ctx.initial[0].Types)
	fmtPkg := ctx.progSSA.ImportedPackage("fmt")
	osPkg := ctx.progSSA.ImportedPackage("os")
	if entry == nil || fmtPkg == nil || osPkg == nil {
		return nil
	}
	regular := fmtPkg.Func("llgoPrintfLiteral")
	direct := osPkg.Func("llgoWriteClosedStdout")
	if regular == nil || direct == nil {
		return nil
	}
	return &wasiStdoutPlan{entry: entry, regular: regular, direct: direct}
}

func (p *wasiStdoutPlan) record(instruction ssa.Instruction) {
	if call, ok := instruction.(ssa.CallInstruction); ok && call.Common().StaticCallee() == p.regular {
		p.calls = append(p.calls, call)
	}
}

func (p *wasiStdoutPlan) specialize(unused, candidates map[*ssa.Global]cl.OptionalInitializer, files []cl.OptionalInitializer,
	analyze func() map[*ssa.Global]cl.OptionalInitializer) map[*ssa.Global]cl.OptionalInitializer {
	if len(p.calls) == 0 {
		return unused
	}
	for _, file := range files {
		if _, ok := candidates[file.Global]; !ok {
			return unused
		}
		if _, ok := unused[file.Global]; !ok && file.Global.Name() != "Stdout" {
			return unused
		}
	}
	// The regular formatter reads os.Stdout. Remove those reads tentatively
	// so the second pass can distinguish them from application observations.
	for _, call := range p.calls {
		call.Common().Value = p.direct
	}
	updated := analyze()
	for _, file := range files {
		if _, ok := updated[file.Global]; !ok {
			for _, call := range p.calls {
				call.Common().Value = p.regular
			}
			return unused
		}
	}
	p.specialized = true
	return updated
}
