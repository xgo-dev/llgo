package cltest

import (
	"go/ast"
	"go/types"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/testsyntax"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// PrepareSyntax collects and binds source records before a test lowers Go SSA.
func PrepareSyntax(prog llssa.Program, pkg *ssa.Package, files []*ast.File, options cl.Options) error {
	return testsyntax.Prepare(pkg, files, func(dep *types.Package, syntax []*ast.File) error {
		if err := cl.ParsePkgSyntaxWithOptions(prog, pkg.Prog.Fset, dep, syntax, options); err != nil {
			return err
		}
		testsyntax.BindScope(prog.PackageDirectives(dep), pkg.Prog.Fset, dep)
		return nil
	})
}
