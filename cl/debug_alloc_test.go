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

func TestCollectDebugAllocVariables(t *testing.T) {
	const source = `package debugalloc
type item struct { value int }
func named(items [3]item) {
	items[0].value = 1
	println(items[0].value)
}
var anonymous = func(values [2]int) {
	values[0] = 1
	println(values[0])
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "debug_alloc.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset,
		types.NewPackage("debugalloc", "debugalloc"),
		[]*ast.File{file},
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics|ssa.GlobalDebug,
	)
	if err != nil {
		t.Fatal(err)
	}

	assertDebugAllocParameter(t, pkg.Func("named"), "items")
	initFn := pkg.Func("init")
	for _, fn := range initFn.AnonFuncs {
		if fn.Syntax() != nil {
			assertDebugAllocParameter(t, fn, "values")
			return
		}
	}
	t.Fatal("anonymous function not found")
}

func assertDebugAllocParameter(t *testing.T, fn *ssa.Function, name string) {
	t.Helper()
	variables := collectDebugAllocVariables(fn)
	if len(fn.Params) != 1 || fn.Params[0].Name() != name {
		t.Fatalf("unexpected parameters for %s: %v", fn, fn.Params)
	}
	variable := fn.Params[0].Object().(*types.Var)
	if !hasDebugAlloc(variables, variable) {
		t.Fatalf("%s parameter %q has no debug alloca", fn, name)
	}
	if !collectDebugAllocObjects(variables)[variable] {
		t.Fatalf("%s parameter %q is missing from the debug alloca object set", fn, name)
	}
	if got := debugParameterArgNo(fn, variable); got != 1 {
		t.Fatalf("debugParameterArgNo(%s) = %d, want 1", name, got)
	}
	missing := types.NewVar(token.NoPos, nil, "missing", types.Typ[types.Int])
	if got := debugParameterArgNo(fn, missing); got != 0 {
		t.Fatalf("debugParameterArgNo(missing) = %d, want 0", got)
	}
}
