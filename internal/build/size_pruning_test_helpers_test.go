package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

type codecInitTestImporter map[string]*types.Package

func (i codecInitTestImporter) Import(path string) (*types.Package, error) {
	return i[path], nil
}

func checkFormatTestPackage(t *testing.T, fset *token.FileSet, path, source string, conf types.Config) *packages.Package {
	t.Helper()
	file, err := parser.ParseFile(fset, path+".go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Implicits: make(map[ast.Node]types.Object),
		Instances: make(map[*ast.Ident]types.Instance), Scopes: make(map[ast.Node]*types.Scope),
		Selections: make(map[*ast.SelectorExpr]*types.Selection), FileVersions: make(map[*ast.File]string),
	}
	pkg, err := conf.Check(path, fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	return &packages.Package{
		PkgPath: path, Name: pkg.Name(), Types: pkg, TypesInfo: info, Fset: fset,
		TypesSizes: types.SizesFor("gc", "wasm"), Syntax: []*ast.File{file},
	}
}
