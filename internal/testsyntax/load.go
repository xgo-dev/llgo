// Package testsyntax loads dependency syntax for tests that construct Go SSA
// with an export-data importer. The build driver already owns this syntax.
package testsyntax

import (
	"go/ast"
	"go/types"
	"sort"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

// Prepare passes the supplied package and its LLGo source dependencies to
// collect. Dependency files follow the host build selection used by the tests'
// export-data importer; directives are interpreted by the caller's collector.
func Prepare(pkg *ssa.Package, files []*ast.File, collect func(*types.Package, []*ast.File) error) error {
	if err := collect(pkg.Pkg, files); err != nil {
		return err
	}
	seen := make(map[*types.Package]bool)
	imports := make(map[string]*types.Package)
	var visit func(*types.Package)
	visit = func(dep *types.Package) {
		if seen[dep] {
			return
		}
		seen[dep] = true
		if dep != pkg.Pkg && dep.Scope().Lookup("LLGoPackage") != nil {
			imports[dep.Path()] = dep
		}
		for _, next := range dep.Imports() {
			visit(next)
		}
	}
	visit(pkg.Pkg)
	if len(imports) == 0 {
		return nil
	}
	paths := make([]string, 0, len(imports))
	for path := range imports {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	loaded, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax,
		Fset: pkg.Prog.Fset,
	}, paths...)
	if err != nil {
		return err
	}
	for _, source := range loaded {
		if len(source.Errors) != 0 {
			return source.Errors[0]
		}
		if err := collect(imports[source.PkgPath], source.Syntax); err != nil {
			return err
		}
	}
	return nil
}
