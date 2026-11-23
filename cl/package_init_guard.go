package cl

import (
	"slices"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func PackageInitGuardName(path string) string {
	return "_llgo_init_pkg." + path
}

func (p *context) compileGuardedPackageInit(b llssa.Builder, call *ssa.Call) bool {
	callee := call.Common().StaticCallee()
	if call.Parent().Synthetic != "package initializer" || callee == nil || callee.Pkg == nil || callee.Synthetic != "package initializer" {
		return false
	}
	path := callee.Pkg.Pkg.Path()
	if !slices.Contains(p.options.GuardPackageInitCalls, path) {
		return false
	}
	guard := DefineInitializerGuard(p.pkg, PackageInitGuardName(path), true)
	run := p.fn.MakeBlock()
	done := p.fn.MakeBlock()
	b.If(b.Load(guard.Expr), run, done)
	b.SetBlock(run)
	p.call(b, llssa.Call, call.Common())
	b.Jump(done)
	b.SetBlock(done)
	return true
}
