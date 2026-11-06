package ssa

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
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

func TestMapLitArrayRejectsEmpty(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("empty map literal array accepted")
		}
	}()
	var b Builder
	b.mapLitArray("k", nil)
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

func TestMapLitArrayNameCompatibility(t *testing.T) {
	prog := NewProgram(nil)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	defer prog.Dispose()
	pkg := prog.NewPackage("maplit", "maplit")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	b := pkg.NewFunc("build", sig, GoDecl).MakeBody(1)
	int32Type := prog.Type(types.Typ[types.Int32], GoDecl)
	namedType := prog.Type(types.NewNamed(types.NewTypeName(0,
		types.NewPackage("example.com/maplit", "maplit"), "Key", nil), types.Typ[types.Int], nil), GoDecl)
	large := make([]Expr, 1024)
	for i := range large {
		large[i] = prog.Val(i)
	}
	for _, tt := range []struct {
		name  string
		elems []Expr
	}{
		{"single", []Expr{prog.Val(0)}},
		{"ints", []Expr{prog.Val(-1), prog.Val(0), prog.Val(42)}},
		{"reordered", []Expr{prog.Val(42), prog.Val(0), prog.Val(-1)}},
		{"int32", []Expr{prog.IntVal(1, int32Type), prog.IntVal(2, int32Type)}},
		{"named", []Expr{prog.IntVal(1, namedType), prog.IntVal(2, namedType)}},
		{"bools", []Expr{prog.Val(true), prog.Val(false)}},
		{"floats", []Expr{prog.Val(-1.5), prog.Val(0.0), prog.Val(3.25)}},
		{"large", large},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, suffix := range []string{"k", "v", "", "k\x00v", strings.Repeat("long", 32)} {
				// Preserve the original concatenation as the compatibility oracle.
				nameInput := tt.elems[0].raw.Type.String() + "\x00" + suffix
				for _, e := range tt.elems {
					nameInput += "\x00" + e.impl.String()
				}
				sum := sha256.Sum256([]byte(nameInput))
				want := "_llgo_maplit$" + base64.RawURLEncoding.EncodeToString(sum[:])
				got := b.mapLitArray(suffix, tt.elems)
				if got.impl.Name() != want {
					t.Fatalf("suffix %q: name = %q, want %q", suffix, got.impl.Name(), want)
				}
				if again := b.mapLitArray(suffix, tt.elems); again.impl != got.impl {
					t.Fatalf("suffix %q: identical arrays did not reuse the global", suffix)
				}
				if got.impl.Linkage() != llvm.PrivateLinkage || !got.impl.IsGlobalConstant() || !strings.Contains(got.impl.String(), "unnamed_addr constant") {
					t.Fatalf("suffix %q: unexpected global attributes: %s", suffix, got.impl.String())
				}
			}
		})
	}
	b.Return()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkMapLitArray(b *testing.B) {
	for _, n := range []int{16, 256, 4096} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			prog := NewProgram(nil)
			prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
			defer prog.Dispose()
			pkg := prog.NewPackage("maplit", "maplit")
			sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
			builder := pkg.NewFunc("build", sig, GoDecl).MakeBody(1)
			elems := make([]Expr, n)
			for i := range elems {
				elems[i] = prog.Val(i)
			}
			// Reuse the global so the module does not grow with the iteration count.
			builder.mapLitArray("k", elems)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				builder.mapLitArray("k", elems)
			}
			b.StopTimer()
			builder.Return()
		})
	}
}
