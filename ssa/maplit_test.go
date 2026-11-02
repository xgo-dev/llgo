package ssa

import (
	"go/importer"
	"go/types"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestMapLitLoopArrays(t *testing.T) {
	prog := NewProgram(nil)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	defer prog.Dispose()
	prog.SetRuntime(func() *types.Package {
		p, e := importer.For("source", nil).Import(PkgRuntime)
		if e != nil {
			t.Fatal(e)
		}
		return p
	})
	pkg := prog.NewPackage("maplit", "maplit")
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "value", types.Typ[types.Int])), nil, false)
	fn := pkg.NewFunc("build", sig, GoDecl)
	b := fn.MakeBody(1)
	typ := prog.Type(types.NewMap(types.Typ[types.Int], types.Typ[types.Int]), GoDecl)
	// Parameters exercise the local-array fallback; constants use private globals.
	m := b.MakeMap(typ, prog.Val(2))
	b.MapLitLoop(m, []Expr{prog.Val(1), prog.Val(2)}, []Expr{prog.Val(3), prog.Val(4)})
	b.MapLitLoop(m, []Expr{prog.Val(5), prog.Val(6)}, []Expr{b.Param(0), prog.Val(7)})
	b.MapLitLoop(m, []Expr{prog.Val(8), prog.Val(9)}, []Expr{b.Param(0), prog.Val(10)})
	b.Return()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	for _, want := range []string{"private unnamed_addr constant [2 x", "alloca [2 x"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %q:\n%s", want, ir)
		}
	}
}

func TestMapLitLoopRejectsInvalidLengths(t *testing.T) {
	for _, n := range []int{0, 1} {
		t.Run(map[int]string{0: "empty", 1: "mismatched"}[n], func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid lengths accepted")
				}
			}()
			var b Builder
			b.MapLitLoop(Expr{}, make([]Expr, n), nil)
		})
	}
}
