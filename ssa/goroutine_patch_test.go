//go:build !llgo
// +build !llgo

package ssa_test

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"github.com/xgo-dev/llvm"
)

func TestGoClosureStartupUsesGCManagedMemory(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	ctxFields := []*types.Var{
		types.NewField(0, nil, "x", types.Typ[types.Int], false),
	}
	ctxStruct := types.NewStruct(ctxFields, nil)
	ctxParam := types.NewParam(0, nil, "$env", types.NewPointer(ctxStruct))
	inner := pkg.NewEnvFunc("inner", ssa.NoArgsNoRet, ssa.InGo, ctxParam, false)
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
	if !strings.Contains(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.AllocRoot"`) {
		t.Fatalf("goroutine startup data should use scanned uncollectable memory:\n%s", ir)
	}
	if !strings.Contains(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.FreeRoot"`) {
		t.Fatalf("goroutine startup data should be freed after the entry call returns:\n%s", ir)
	}
	// The closure context must remain visible to the runtime GC until the
	// uncollectable startup record is initialized.
	if got := strings.Count(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.AllocU"`); got < 1 {
		t.Fatalf("expected closure ctx to use AllocU, got %d:\n%s", got, ir)
	}
	if strings.Contains(ir, "EnterLocalContext") {
		t.Fatalf("program without context-backed locals paid locality entry cost:\n%s", ir)
	}
}

func TestGoInstallsContextForContextBackedLocals(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	prog.SetLocalityInfo("example.com/state.Value", ssa.LocalityInfo{Locality: ssa.GoroutineLocal})
	prog.SetLocalStorage("example.com/state.Value", ssa.LocalStoragePackage)
	pkg := prog.NewPackage("bar", "foo/bar")
	outer := pkg.NewFunc("outer", ssa.NoArgsNoRet, ssa.InGo)
	b := outer.MakeBody(1)
	b.Go(ssa.Nil, func(b ssa.Builder, _ ssa.Expr, args ...ssa.Expr) ssa.Expr {
		return ssa.Expr{}
	})
	b.Return()

	ir := pkg.String()
	for _, want := range []string{"LocalContext", "EnterLocalContext", "LeaveLocalContext"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("goroutine wrapper missing %q:\n%s", want, ir)
		}
	}
}

func TestGoPanicRoutineDoesNotReturnAfterUnreachable(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	outer := pkg.NewFunc("outer", ssa.NoArgsNoRet, ssa.InGo)
	ob := outer.MakeBody(1)
	ob.Go(ssa.Nil, func(b ssa.Builder, _ ssa.Expr, args ...ssa.Expr) ssa.Expr {
		b.Panic(args[0])
		return ssa.Expr{}
	}, prog.Zero(prog.Any()))
	ob.Return()

	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}

	ir := pkg.String()
	freeRoot := strings.Index(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.FreeRoot"`)
	panicCall := strings.Index(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.Panic"`)
	if freeRoot < 0 || panicCall < 0 || freeRoot > panicCall {
		t.Fatalf("goroutine wrapper should free startup data before panic call:\n%s", ir)
	}
	if strings.Contains(ir, "unreachable\n  ret ptr null") {
		t.Fatalf("goroutine wrapper should not return after unreachable:\n%s", ir)
	}
}

func TestGoPassesConfiguredStackSizeToRuntime(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	prog.SetPthreadStackSize(32 << 20)
	pkg := prog.NewPackage("bar", "foo/bar")

	outer := pkg.NewFunc("outer", ssa.NoArgsNoRet, ssa.InGo)
	ob := outer.MakeBody(1)
	ob.Go(ssa.Nil, func(b ssa.Builder, _ ssa.Expr, args ...ssa.Expr) ssa.Expr {
		return ssa.Expr{}
	})
	ob.Return()

	ir := pkg.String()
	if !strings.Contains(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.NewProc"`) {
		t.Fatalf("goroutine should delegate startup to the runtime:\n%s", ir)
	}
	if !strings.Contains(ir, "33554432") {
		t.Fatalf("goroutine should pass the configured stack size:\n%s", ir)
	}
	if strings.Contains(ir, "pthread") {
		t.Fatalf("compiler IR should not depend on the pthread backend:\n%s", ir)
	}
}

func TestGoPassesZeroStackSizeToRuntimeByDefault(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	outer := pkg.NewFunc("outer", ssa.NoArgsNoRet, ssa.InGo)
	ob := outer.MakeBody(1)
	ob.Go(ssa.Nil, func(b ssa.Builder, _ ssa.Expr, args ...ssa.Expr) ssa.Expr {
		return ssa.Expr{}
	})
	ob.Return()

	ir := pkg.String()
	if !strings.Contains(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.NewProc"`) {
		t.Fatalf("goroutine should delegate startup to the runtime:\n%s", ir)
	}
	if strings.Contains(ir, "pthread") {
		t.Fatalf("compiler IR should not depend on the pthread backend:\n%s", ir)
	}
}
