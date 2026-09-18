package types_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

type checkedPackage struct {
	pkg  *types.Package
	fset *token.FileSet
	file *ast.File
}

func typeCheck(t *testing.T, src string, info *types.Info) checkedPackage {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.AllErrors|parser.ParseComments)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if info == nil {
		info = &types.Info{}
	}
	ensureInfoMaps(info)

	conf := types.Config{Importer: testImporter(t)}
	pkg, err := conf.Check("test/p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("type-check failed: %v", err)
	}

	return checkedPackage{pkg: pkg, fset: fset, file: file}
}

func ensureInfoMaps(info *types.Info) {
	if info.Types == nil {
		info.Types = make(map[ast.Expr]types.TypeAndValue)
	}
	if info.Defs == nil {
		info.Defs = make(map[*ast.Ident]types.Object)
	}
	if info.Uses == nil {
		info.Uses = make(map[*ast.Ident]types.Object)
	}
	if info.Implicits == nil {
		info.Implicits = make(map[ast.Node]types.Object)
	}
	if info.Selections == nil {
		info.Selections = make(map[*ast.SelectorExpr]*types.Selection)
	}
	if info.Scopes == nil {
		info.Scopes = make(map[ast.Node]*types.Scope)
	}
	if info.Instances == nil {
		info.Instances = make(map[*ast.Ident]types.Instance)
	}
}
