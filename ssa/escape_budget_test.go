package ssa_test

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
)

func TestDeferBudgetKeepsRegistrationOwnership(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("budget", "budget")
	params := types.NewTuple(types.NewParam(token.NoPos, nil, "x", types.Typ[types.Int]))
	callee := pkg.NewFunc("callee", types.NewSignatureType(nil, nil, nil, params, nil, false), ssa.GoDecl)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()
	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.GoDecl)
	node := prog.Struct(prog.VoidPtr(), prog.Uintptr(), prog.Int())
	fn.SetEscapeStackBytes(llabi.MaxEscapeStackBytes - prog.SizeOf(prog.Defer()) - 3*prog.SizeOf(node))
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())
	for i := 0; i < 5; i++ {
		b.Defer(ssa.DeferAlways, callee.Expr, ssa.Builder.Call, prog.Val(i))
	}
	b.RunDefers()
	b.Return()
	b.EndBuild()
	ir := pkg.Module().String()
	// Leave room for the frame and three records. Only the two rejected
	// records may be freed, regardless of when their pop paths are generated.
	if got := strings.Count(ir, "call void @\"github.com/xgo-dev/llgo/runtime/internal/runtime.FreeDeferNode\""); got != 2 {
		t.Fatalf("heap cleanup count=%d, want 2:\n%s", got, ir)
	}
}
