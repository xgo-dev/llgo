//go:build !llgo
// +build !llgo

package ssa_test

import (
	"strings"
	"testing"

	"github.com/goplus/llgo/ssa"
	"github.com/goplus/llgo/ssa/ssatest"
)

func TestExplicitDeferStackIR(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	callee := pkg.NewFunc("callee", ssa.NoArgsNoRet, ssa.InGo)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())

	stack := b.BuiltinCall("ssa:deferstack")
	b.Return()
	b.SetBlockEx(fn.Block(0), ssa.BeforeLast, true)
	b.DeferTo(fn, stack, callee.Expr, ssa.Builder.Call)
	b.DeferStackDrain()
	b.RunDefers()
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if !strings.Contains(ir, "FreeDeferNode") {
		t.Fatalf("expected explicit defer stack node cleanup in IR, got:\n%s", ir)
	}
	if !strings.Contains(ir, "sigsetjmp") && !strings.Contains(ir, "setjmp") {
		t.Fatalf("expected explicit defer stack setup in IR, got:\n%s", ir)
	}
}

func TestExplicitDeferStackFallbackAndNilBuiltin(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	callee := pkg.NewFunc("callee", ssa.NoArgsNoRet, ssa.InGo)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	stack := b.BuiltinCall("ssa:deferstack")
	if stack.Type != prog.VoidPtr() {
		t.Fatalf("ssa:deferstack without recover returned %v, want %v", stack.Type, prog.VoidPtr())
	}
	b.DeferTo(nil, stack, callee.Expr, ssa.Builder.Call)
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if strings.Contains(ir, "sigsetjmp") || strings.Contains(ir, "setjmp") {
		t.Fatalf("unexpected defer stack setup without recover, got:\n%s", ir)
	}
}

func TestExplicitDeferStackDrainWithoutLoopCases(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())

	_ = b.BuiltinCall("ssa:deferstack")
	b.DeferStackDrain()
	b.RunDefers()
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if strings.Contains(ir, "FreeDeferNode") {
		t.Fatalf("unexpected explicit defer cleanup without loop cases, got:\n%s", ir)
	}
	if !strings.Contains(ir, "sigsetjmp") && !strings.Contains(ir, "setjmp") {
		t.Fatalf("expected defer stack setup with recover, got:\n%s", ir)
	}
}

func TestExplicitDeferStackDrainWithoutRecoverNoop(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	b.DeferStackDrain()
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if strings.Contains(ir, "FreeDeferNode") || strings.Contains(ir, "sigsetjmp") || strings.Contains(ir, "setjmp") {
		t.Fatalf("unexpected defer stack machinery without recover, got:\n%s", ir)
	}
}

func TestPlainDeferWithoutSavedArgsIR(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	callee := pkg.NewFunc("callee", ssa.NoArgsNoRet, ssa.InGo)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())
	b.Defer(ssa.DeferAlways, callee.Expr, ssa.Builder.Call)
	b.RunDefers()
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if strings.Contains(ir, "FreeDeferNode") {
		t.Fatalf("plain zero-arg defer should not allocate defer nodes, got:\n%s", ir)
	}
	if !strings.Contains(ir, "call void @callee()") {
		t.Fatalf("expected direct deferred call in IR, got:\n%s", ir)
	}
}

func TestConditionalDeferIR(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	callee := pkg.NewFunc("callee", ssa.NoArgsNoRet, ssa.InGo)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())
	b.Return()
	b.SetBlockEx(fn.Block(0), ssa.BeforeLast, true)
	b.Defer(ssa.DeferInCond, callee.Expr, ssa.Builder.Call)
	b.RunDefers()
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if !strings.Contains(ir, "or i64") || !strings.Contains(ir, "and i64") {
		t.Fatalf("expected conditional defer bitmask operations in IR, got:\n%s", ir)
	}
}
