package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/typepatch"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestDebugVariablePackageScope(t *testing.T) {
	const source = `package scope
var global int
type S struct { field int }
func f(param int) (result int) {
	local := global
	{ global := param; local += global }
	closure := func() int { nested := local; return nested }
	return closure()
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "scope.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	pkg, err := new(types.Config).Check("scope", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	check := func() {
		t.Helper()
		for ident, obj := range info.Defs {
			variable, ok := obj.(*types.Var)
			if !ok {
				continue
			}
			want := obj == pkg.Scope().Lookup("global")
			if got := isGlobal(variable); got != want {
				t.Errorf("%s at %s: isGlobal = %v, want %v", ident.Name, fset.Position(ident.Pos()), got, want)
			}
		}
	}
	check()
	// Clone and Merge copy declarations into a new package scope without
	// changing each object's Parent. Globals must remain globals after that
	// scope identity changes.
	before := pkg.Scope()
	merged := typepatch.Clone(pkg)
	typepatch.Merge(merged, types.NewPackage("original", "original"), nil, false)
	if merged.Scope() == before {
		t.Fatal("Clone+Merge did not create an independent package scope")
	}
	pkg = merged
	check()
	global := pkg.Scope().Lookup("global").(*types.Var)
	if allocs := testing.AllocsPerRun(100, func() {
		if !isGlobal(global) {
			t.Fatal("package variable classified as local")
		}
	}); allocs != 0 {
		t.Fatalf("package-scope classification allocated %g objects", allocs)
	}
	if isGlobal(types.NewVar(token.NoPos, pkg, "synthetic", types.Typ[types.Int])) {
		t.Fatal("variable without a parent scope classified as global")
	}
}

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
}
