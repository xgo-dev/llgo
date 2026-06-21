//go:build !llgo
// +build !llgo

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

func buildLinkOnceSSAPackage(t *testing.T, src string) *ssa.Package {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("p", "p")
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	ssapkg, _, err := ssautil.BuildPackage(&types.Config{Importer: importer.Default()}, fset, pkg, []*ast.File{f}, mode)
	if err != nil {
		t.Fatal(err)
	}
	return ssapkg
}

func linkOnceTestMethodValue(t *testing.T, ssapkg *ssa.Package, recv types.Type, name string) (*context, *ssa.Function) {
	t.Helper()
	sel := ssapkg.Prog.MethodSets.MethodSet(recv).Lookup(nil, name)
	if sel == nil {
		t.Fatalf("%v has no method %s", recv, name)
	}
	ctx := &context{
		goProg:      ssapkg.Prog,
		goTyps:      ssapkg.Pkg,
		linkOnceFns: make(map[*ssa.Function]none),
	}
	fn := ctx.methodValue(sel)
	if fn == nil {
		t.Fatalf("MethodValue(%v.%s) returned nil", recv, name)
	}
	return ctx, fn
}

func TestNeedsLinkOnceSkipsNonGenericPromotedWrapper(t *testing.T) {
	ssapkg := buildLinkOnceSSAPackage(t, `package p

type Inner struct{}

func (Inner) M() {}

type Outer struct {
	Inner
}
`)
	outer := ssapkg.Pkg.Scope().Lookup("Outer").(*types.TypeName).Type()
	ctx, fn := linkOnceTestMethodValue(t, ssapkg, types.NewPointer(outer), "M")
	if fn.Pkg != nil || fn.Origin() != nil {
		t.Fatalf("test expected a non-generic on-demand wrapper, got pkg=%v origin=%v", fn.Pkg, fn.Origin())
	}
	if ctx.needsLinkOnce(fn) {
		t.Fatalf("non-generic promoted wrapper %s should not need linkonce", fn)
	}
}

func TestNeedsLinkOnceMarksGenericPromotedWrapper(t *testing.T) {
	ssapkg := buildLinkOnceSSAPackage(t, `package p

type Inner[T any] struct{}

func (*Inner[T]) M() {}

type Outer struct {
	*Inner[int]
}
`)
	outer := ssapkg.Pkg.Scope().Lookup("Outer").(*types.TypeName).Type()
	ctx, fn := linkOnceTestMethodValue(t, ssapkg, types.NewPointer(outer), "M")
	if fn.Origin() != nil {
		t.Fatalf("test expected a wrapper around a generic method, got generic instance %s", fn)
	}
	if !ctx.needsLinkOnce(fn) {
		t.Fatalf("generic promoted wrapper %s should need linkonce", fn)
	}
}

func TestNeedsLinkOnceMarksGenericMethodInstance(t *testing.T) {
	ssapkg := buildLinkOnceSSAPackage(t, `package p

type Box[T any] struct{}

func (Box[T]) M() {}
`)
	box := ssapkg.Pkg.Scope().Lookup("Box").(*types.TypeName).Type().(*types.Named)
	boxInt, err := types.Instantiate(nil, box, []types.Type{types.Typ[types.Int]}, true)
	if err != nil {
		t.Fatal(err)
	}
	ctx, fn := linkOnceTestMethodValue(t, ssapkg, boxInt, "M")
	if fn.Origin() == nil {
		t.Fatalf("test expected a generic method instance, got %s", fn)
	}
	if !ctx.needsLinkOnce(fn) {
		t.Fatalf("generic method instance %s should need linkonce", fn)
	}
}
