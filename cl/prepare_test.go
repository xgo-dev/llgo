package cl

import (
	"go/ast"
	"go/types"

	"github.com/xgo-dev/llgo/internal/testsyntax"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func prepareTestSyntax(prog llssa.Program, pkg *ssa.Package, files []*ast.File, options Options) error {
	return testsyntax.Prepare(pkg, files, func(dep *types.Package, syntax []*ast.File) error {
		if err := ParsePkgSyntaxWithOptions(prog, pkg.Prog.Fset, dep, syntax, options); err != nil {
			return err
		}
		testsyntax.BindScope(prog.PackageDirectives(dep), pkg.Prog.Fset, dep)
		return nil
	})
}

func compileTestPackage(prog llssa.Program, pkg *ssa.Package, files []*ast.File) (llssa.Package, error) {
	if err := prepareTestSyntax(prog, pkg, files, Options{}); err != nil {
		return nil, err
	}
	prog.Directives().Freeze()
	return NewPackage(prog, pkg, files)
}
