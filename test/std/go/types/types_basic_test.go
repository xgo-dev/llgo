package types_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"testing"
)

func parseAndCheckPackage(t *testing.T, filename, src string, info *gotypes.Info) (*token.FileSet, *gotypes.Package, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", filename, err)
	}
	conf := gotypes.Config{Importer: testImporter(t)}
	pkg, err := conf.Check("example.org/p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("Config.Check: %v", err)
	}
	return fset, pkg, file
}

func TestConfigCheckAndInfo(t *testing.T) {
	src := `package p
import "example.org/fixture/fmt"
type S struct{ X int }
func (S) M() string { return fmt.Sprint("m") }
var _ = S{}.M()
func F[T ~int](v T) T { return v + 1 }
var G = F[int](1)
`
	info := &gotypes.Info{
		Types:      map[ast.Expr]gotypes.TypeAndValue{},
		Defs:       map[*ast.Ident]gotypes.Object{},
		Uses:       map[*ast.Ident]gotypes.Object{},
		Implicits:  map[ast.Node]gotypes.Object{},
		Selections: map[*ast.SelectorExpr]*gotypes.Selection{},
		Scopes:     map[ast.Node]*gotypes.Scope{},
	}

	fset, pkg, _ := parseAndCheckPackage(t, "p.go", src, info)
	if pkg.Name() != "p" || pkg.Path() != "example.org/p" {
		t.Fatalf("package = (%q, %q), want (%q, %q)", pkg.Name(), pkg.Path(), "p", "example.org/p")
	}
	if pkg.Scope().Lookup("S") == nil || pkg.Scope().Lookup("F") == nil || pkg.Scope().Lookup("G") == nil {
		t.Fatal("expected exported objects missing from package scope")
	}
	if len(info.Defs) == 0 || len(info.Uses) == 0 || len(info.Types) == 0 {
		t.Fatalf("Info maps should be non-empty: defs=%d uses=%d types=%d", len(info.Defs), len(info.Uses), len(info.Types))
	}
	if len(info.Selections) == 0 {
		t.Fatal("expected at least one method selection in Info.Selections")
	}
	if len(info.Scopes) == 0 {
		t.Fatal("expected non-empty Info.Scopes")
	}

	fObj := pkg.Scope().Lookup("F")
	if fObj == nil {
		t.Fatal("lookup F returned nil")
	}
	sig, ok := fObj.Type().(*gotypes.Signature)
	if !ok {
		t.Fatalf("F type = %T, want *types.Signature", fObj.Type())
	}
	if sig.TypeParams().Len() != 1 {
		t.Fatalf("F type params len = %d, want 1", sig.TypeParams().Len())
	}

	if got := gotypes.ObjectString(pkg.Scope().Lookup("G"), gotypes.RelativeTo(pkg)); got == "" {
		t.Fatal("ObjectString returned empty for G")
	}
	if got := gotypes.TypeString(sig, gotypes.RelativeTo(pkg)); got == "" {
		t.Fatal("TypeString returned empty for F signature")
	}
	var selExpr ast.Expr
	for expr := range info.Selections {
		selExpr = expr
		break
	}
	if selExpr == nil {
		t.Fatal("expected selection expression for ExprString")
	}
	if got := gotypes.ExprString(selExpr); got == "" {
		t.Fatal("ExprString returned empty")
	}

	tv, err := gotypes.Eval(fset, pkg, token.NoPos, "F[int](2)")
	if err != nil {
		t.Fatalf("Eval(F[int](2)): %v", err)
	}
	if tv.Type != nil && tv.Type.String() != "int" {
		t.Fatalf("Eval type = %v, want int when non-nil", tv.Type)
	}
}

func TestSelectionsAndMethodSet(t *testing.T) {
	src := `package p
type I interface{ M() string }
type T struct{}
func (T) M() string { return "ok" }
func use(v I) string { return v.M() }
`
	info := &gotypes.Info{
		Types:      map[ast.Expr]gotypes.TypeAndValue{},
		Defs:       map[*ast.Ident]gotypes.Object{},
		Uses:       map[*ast.Ident]gotypes.Object{},
		Selections: map[*ast.SelectorExpr]*gotypes.Selection{},
	}
	_, pkg, _ := parseAndCheckPackage(t, "sel.go", src, info)

	tObj := pkg.Scope().Lookup("T")
	iObj := pkg.Scope().Lookup("I")
	if tObj == nil || iObj == nil {
		t.Fatal("expected T and I in scope")
	}
	ms := gotypes.NewMethodSet(tObj.Type())
	if ms.Len() != 1 {
		t.Fatalf("method set len = %d, want 1", ms.Len())
	}
	if ms.At(0).Obj().Name() != "M" {
		t.Fatalf("method name = %q, want M", ms.At(0).Obj().Name())
	}

	iface, ok := iObj.Type().Underlying().(*gotypes.Interface)
	if !ok {
		t.Fatalf("I underlying = %T, want *types.Interface", iObj.Type().Underlying())
	}
	if !gotypes.Implements(tObj.Type(), iface) {
		t.Fatal("T should implement I")
	}
	if !gotypes.AssignableTo(tObj.Type(), iface) {
		t.Fatal("T should be assignable to I")
	}
}
