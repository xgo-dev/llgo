//go:build !llgo

package ssa

import (
	"fmt"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestUMulOverflowResultLayout(t *testing.T) {
	for _, arch := range []string{"amd64", "386"} {
		t.Run(arch, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "windows", GOARCH: arch})
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", arch))
			pkg := prog.NewPackage("overflow", "overflow")
			fn := pkg.NewFunc("test", NoArgsNoRet, InC)
			b := fn.MakeBody(1)
			for _, kind := range []types.BasicKind{types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uint, types.Uintptr} {
				typ := prog.toType(types.Typ[kind])
				a := Expr{llvm.ConstInt(typ.ll, 3, false), typ}
				result := b.UMulOverflow(a, a)
				want := prog.Struct(typ, prog.Bool())
				if result.impl.Type() != want.ll {
					t.Fatalf("%s result LLVM type = %s, want %s", typ.RawType(), result.impl.Type(), want.ll)
				}
				if result.Type.ll != want.ll {
					t.Fatal("result Go and LLVM layouts disagree")
				}
			}
			b.Return()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUMulOverflowRejectsInvalidOperands(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("overflow", "overflow")
	b := pkg.NewFunc("test", NoArgsNoRet, InC).MakeBody(1)
	for _, pair := range [][2]types.BasicKind{{types.Int, types.Int}, {types.Float64, types.Float64}, {types.Uint32, types.Uint64}} {
		t.Run(fmt.Sprint(pair), func(t *testing.T) {
			defer func() {
				if got := fmt.Sprint(recover()); !strings.Contains(got, "same unsigned integer type") {
					t.Fatalf("diagnostic = %q", got)
				}
			}()
			a, c := prog.toType(types.Typ[pair[0]]), prog.toType(types.Typ[pair[1]])
			b.UMulOverflow(Expr{llvm.Undef(a.ll), a}, Expr{llvm.Undef(c.ll), c})
		})
	}
	b.Return()
}
