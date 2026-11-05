package ssa

import (
	"fmt"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestRotateLeftRejectsInvalidOperands(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("rotate", "rotate")
	b := pkg.NewFunc("test", NoArgsNoRet, CDecl).MakeBody(1)
	integer := prog.toType(types.Typ[types.Uint32])
	for _, typ := range []types.Type{
		types.Typ[types.Float64],
		types.Typ[types.Bool],
		types.NewPointer(types.Typ[types.Int]),
		types.NewArray(types.Typ[types.Uint32], 1),
	} {
		invalid := prog.toType(typ)
		for _, pair := range [][2]Type{{invalid, integer}, {integer, invalid}} {
			t.Run(fmt.Sprint(pair[0].RawType(), "/", pair[1].RawType()), func(t *testing.T) {
				defer func() {
					if got := fmt.Sprint(recover()); !strings.Contains(got, "expected scalar integer arguments") {
						t.Fatalf("diagnostic = %q", got)
					}
				}()
				b.RotateLeft(Expr{llvm.Undef(pair[0].ll), pair[0]}, Expr{llvm.Undef(pair[1].ll), pair[1]})
			})
		}
	}
	b.Return()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}
