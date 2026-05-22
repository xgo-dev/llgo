//go:build !llgo
// +build !llgo

package ssa_test

import (
	"go/types"
	"strings"
	"testing"

	"github.com/goplus/llgo/ssa"
	"github.com/goplus/llgo/ssa/ssatest"
)

func TestGoClosureStartupUsesGCManagedMemory(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	ctxFields := []*types.Var{
		types.NewField(0, nil, "x", types.Typ[types.Int], false),
	}
	ctxStruct := types.NewStruct(ctxFields, nil)
	ctxParam := types.NewParam(0, nil, "__llgo_ctx", types.NewPointer(ctxStruct))
	innerSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(ctxParam), nil, false)
	inner := pkg.NewFunc("inner", innerSig, ssa.InGo)
	ib := inner.MakeBody(1)
	ib.Return()

	outer := pkg.NewFunc("outer", ssa.NoArgsNoRet, ssa.InGo)
	ob := outer.MakeBody(1)
	closure := ob.MakeClosure(inner.Expr, []ssa.Expr{prog.Val(42)})
	ob.Go(closure, func(b ssa.Builder, fn ssa.Expr, args ...ssa.Expr) ssa.Expr {
		return b.Call(fn, args...)
	})
	ob.Return()

	ir := pkg.String()
	if strings.Contains(ir, "@malloc") {
		t.Fatalf("goroutine startup data should not use malloc:\n%s", ir)
	}
	if strings.Contains(ir, "@free") {
		t.Fatalf("goroutine startup data should not use free:\n%s", ir)
	}
	if !strings.Contains(ir, `"github.com/goplus/llgo/runtime/internal/runtime.AllocRoot"`) {
		t.Fatalf("goroutine startup data should use scanned uncollectable memory:\n%s", ir)
	}
	if !strings.Contains(ir, `"github.com/goplus/llgo/runtime/internal/runtime.FreeRoot"`) {
		t.Fatalf("goroutine startup data should be freed after the entry call returns:\n%s", ir)
	}
	// The closure context must remain visible to the runtime GC until the
	// uncollectable startup record is initialized.
	if got := strings.Count(ir, `"github.com/goplus/llgo/runtime/internal/runtime.AllocU"`); got < 1 {
		t.Fatalf("expected closure ctx to use AllocU, got %d:\n%s", got, ir)
	}
}
