package types_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"testing"
)

func mustCheck(t *testing.T, src string) (*token.FileSet, *gotypes.Package, *gotypes.Info) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "rel.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	info := &gotypes.Info{
		Types: map[ast.Expr]gotypes.TypeAndValue{},
		Defs:  map[*ast.Ident]gotypes.Object{},
		Uses:  map[*ast.Ident]gotypes.Object{},
	}
	conf := gotypes.Config{Importer: testImporter(t)}
	pkg, err := conf.Check("example.org/rel", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("Config.Check: %v", err)
	}
	return fset, pkg, info
}

func TestTypeRelationsHelpers(t *testing.T) {
	_, pkg, _ := mustCheck(t, `package p
type I interface{ M() int }
type T struct{}
func (T) M() int { return 1 }
var _ I = T{}
`)
	iObj := pkg.Scope().Lookup("I")
	tObj := pkg.Scope().Lookup("T")
	if iObj == nil || tObj == nil {
		t.Fatal("lookup I/T returned nil")
	}

	iface := iObj.Type().Underlying().(*gotypes.Interface)
	tType := tObj.Type()
	if !gotypes.Implements(tType, iface) {
		t.Fatal("T should implement I")
	}
	if !gotypes.AssignableTo(tType, iface) {
		t.Fatal("T should be assignable to I")
	}
	if !gotypes.ConvertibleTo(gotypes.Typ[gotypes.Int], gotypes.Typ[gotypes.Int64]) {
		t.Fatal("int should be convertible to int64")
	}
	if !gotypes.IdenticalIgnoreTags(
		gotypes.NewStruct([]*gotypes.Var{gotypes.NewField(token.NoPos, nil, "X", gotypes.Typ[gotypes.Int], false)}, []string{`json:"x"`}),
		gotypes.NewStruct([]*gotypes.Var{gotypes.NewField(token.NoPos, nil, "X", gotypes.Typ[gotypes.Int], false)}, []string{`yaml:"x"`}),
	) {
		t.Fatal("IdenticalIgnoreTags should ignore field tags")
	}
}

func TestInstantiateAndEval(t *testing.T) {
	fset, pkg, _ := mustCheck(t, `package p
func Inc[T ~int](v T) T { return v + 1 }
var X = Inc[int](41)
`)
	incObj := pkg.Scope().Lookup("Inc")
	if incObj == nil {
		t.Fatal("Inc object missing")
	}
	sig, ok := incObj.Type().(*gotypes.Signature)
	if !ok {
		t.Fatalf("Inc type = %T, want *types.Signature", incObj.Type())
	}
	inst, err := gotypes.Instantiate(nil, sig, []gotypes.Type{gotypes.Typ[gotypes.Int]}, false)
	if err != nil {
		t.Fatalf("Instantiate(Inc[int]): %v", err)
	}
	if inst.String() == "" {
		t.Fatal("instantiated signature string should not be empty")
	}

	tv, err := gotypes.Eval(fset, pkg, token.NoPos, "Inc[int](1)")
	if err != nil {
		t.Fatalf("Eval(Inc[int](1)): %v", err)
	}
	if tv.Type != nil && tv.Type.String() != "int" {
		t.Fatalf("Eval type = %v, want int when non-nil", tv.Type)
	}
	if tv.Value != nil {
		// This expression is not constant-folded by go/types (call expression), so nil is expected.
		t.Logf("Eval value = %v", tv.Value)
	}
}
