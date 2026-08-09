package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestDebugFunctionScope(t *testing.T) {
	if debugFunctionScope(nil) != nil || debugFunctionScope(new(ssa.Function)) != nil {
		t.Fatal("function without source scope should return nil")
	}
	const source = `package scope

func named(p int) int {
	if p != 0 {
		v := p + 1
		return v
	}
	return p
}

var anonymous = func() int {
	if true {
		v := 1
		return v
	}
	return 0
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "scope.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset,
		types.NewPackage("scope", "scope"),
		[]*ast.File{file},
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics|ssa.GlobalDebug,
	)
	if err != nil {
		t.Fatal(err)
	}

	named := pkg.Func("named")
	if got, want := debugFunctionScope(named), named.Object().(*types.Func).Scope(); got != want {
		t.Fatalf("named function scope = %p, want %p", got, want)
	}

	initFn := pkg.Func("init")
	var anonymous *ssa.Function
	for _, child := range initFn.AnonFuncs {
		if child.Syntax() != nil {
			anonymous = child
			break
		}
	}
	if anonymous == nil {
		t.Fatal("anonymous function not found")
	}
	root := debugFunctionScope(anonymous)
	if root == nil {
		t.Fatal("anonymous function scope is nil")
	}
	if root.Pos() < anonymous.Syntax().Pos() || root.End() > anonymous.Syntax().End() {
		t.Fatalf("anonymous function scope %s is outside function %s", root, anonymous.Syntax())
	}
	lit, ok := anonymous.Syntax().(*ast.FuncLit)
	if !ok || len(lit.Body.List) == 0 {
		t.Fatal("anonymous function syntax not found")
	}
	ifStmt, ok := lit.Body.List[0].(*ast.IfStmt)
	if !ok || len(ifStmt.Body.List) == 0 {
		t.Fatal("anonymous function inner scope not found")
	}
	sourcePos := ifStmt.Body.List[0].Pos()
	want := root.Innermost(sourcePos)
	if got := (&context{}).getDebugLocScope(anonymous, sourcePos); got != want || got == nil {
		t.Fatalf("anonymous instruction scope = %p, want %p", got, want)
	}
}
