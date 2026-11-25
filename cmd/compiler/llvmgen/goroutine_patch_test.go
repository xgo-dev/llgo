package llvmgen_test

import (
	"github.com/xgo-dev/llgo/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen/ssatest"
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
	inner := pkg.NewEnvFunc("inner", llvmgen.NoArgsNoRet, llvmgen.GoDecl, ctxParam, false)
	ib := inner.MakeBody(1)
	ib.Return()

	outer := pkg.NewFunc("outer", llvmgen.NoArgsNoRet, llvmgen.GoDecl)
	ob := outer.MakeBody(1)
	closure := ob.MakeClosure(inner.Expr, []llvmgen.Expr{prog.Val(42)})
	ob.Go(closure, func(b llvmgen.Builder, fn llvmgen.Expr, args ...llvmgen.Expr) llvmgen.Expr {
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
	if strings.Contains(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.FreeRoot"`) {
		t.Fatalf("the runtime, not the compiler wrapper, should release goroutine startup data:\n%s", ir)
	}
	// The closure context must remain visible to the runtime GC until the
	// uncollectable startup record is initialized.
	if got := strings.Count(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.AllocAtomicU"`); got < 1 {
		t.Fatalf("expected closure ctx to use AllocAtomicU, got %d:\n%s", got, ir)
	}
	if strings.Contains(ir, "EnterLocalContext") {
		t.Fatalf("program without context-backed locals paid locality entry cost:\n%s", ir)
	}
}

func TestGoInstallsContextForContextBackedLocals(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	prog.SetLocalityInfo("example.com/state.Value", llvmgen.LocalityInfo{Locality: llvmgen.GoroutineLocal})
	prog.SetLocalStorage("example.com/state.Value", llvmgen.LocalStoragePackage)
	pkg := prog.NewPackage("bar", "foo/bar")
	outer := pkg.NewFunc("outer", llvmgen.NoArgsNoRet, llvmgen.GoDecl)
	b := outer.MakeBody(1)
	b.Go(llvmgen.Nil, func(b llvmgen.Builder, _ llvmgen.Expr, args ...llvmgen.Expr) llvmgen.Expr {
		return llvmgen.Expr{}
	})
	b.Return()

	ir := pkg.String()
	for _, want := range []string{"LocalContext", "EnterLocalContext", "LeaveLocalContext"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("goroutine wrapper missing %q:\n%s", want, ir)
		}
	}
}

func TestWASIGoInstallsContextWithoutUserLocality(t *testing.T) {
	prog := ssatest.NewProgram(t, &llvmgen.Target{GOOS: "wasip1", GOARCH: "wasm", Target: "wasi"})
	pkg := prog.NewPackage("bar", "foo/bar")
	outer := pkg.NewFunc("outer", llvmgen.NoArgsNoRet, llvmgen.GoDecl)
	b := outer.MakeBody(1)
	b.Go(llvmgen.Nil, func(b llvmgen.Builder, _ llvmgen.Expr, args ...llvmgen.Expr) llvmgen.Expr {
		return llvmgen.Expr{}
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

	outer := pkg.NewFunc("outer", llvmgen.NoArgsNoRet, llvmgen.GoDecl)
	ob := outer.MakeBody(1)
	ob.Go(llvmgen.Nil, func(b llvmgen.Builder, _ llvmgen.Expr, args ...llvmgen.Expr) llvmgen.Expr {
		b.Panic(args[0])
		return llvmgen.Expr{}
	}, prog.Zero(prog.Any()))
	ob.Return()

	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}

	ir := pkg.String()
	if strings.Contains(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.FreeRoot"`) {
		t.Fatalf("goroutine wrapper should leave panic-path startup cleanup to the runtime:\n%s", ir)
	}
	if strings.Contains(ir, "unreachable\n  ret ptr null") {
		t.Fatalf("goroutine wrapper should not return after unreachable:\n%s", ir)
	}
}

func TestGoDoesNotPassConfiguredStackSizeToRuntime(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	prog.SetPthreadStackSize(32 << 20)
	pkg := prog.NewPackage("bar", "foo/bar")

	outer := pkg.NewFunc("outer", llvmgen.NoArgsNoRet, llvmgen.GoDecl)
	ob := outer.MakeBody(1)
	ob.Go(llvmgen.Nil, func(b llvmgen.Builder, _ llvmgen.Expr, args ...llvmgen.Expr) llvmgen.Expr {
		return llvmgen.Expr{}
	})
	ob.Return()

	ir := pkg.String()
	if !strings.Contains(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.NewProc"`) {
		t.Fatalf("goroutine should delegate startup to the runtime:\n%s", ir)
	}
	if strings.Contains(ir, "33554432") {
		t.Fatalf("goroutine caller must not embed the configured stack size:\n%s", ir)
	}
	if fn := pkg.Module().NamedFunction(llvmgen.PkgRuntime + ".NewProc"); fn.ParamsCount() != 2 {
		t.Fatalf("NewProc takes %d parameters, want two", fn.ParamsCount())
	}
	if strings.Contains(ir, "pthread") {
		t.Fatalf("compiler IR should not depend on the pthread backend:\n%s", ir)
	}
}

func TestGoUsesTwoArgumentRuntimeEntryByDefault(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	outer := pkg.NewFunc("outer", llvmgen.NoArgsNoRet, llvmgen.GoDecl)
	ob := outer.MakeBody(1)
	ob.Go(llvmgen.Nil, func(b llvmgen.Builder, _ llvmgen.Expr, args ...llvmgen.Expr) llvmgen.Expr {
		return llvmgen.Expr{}
	})
	ob.Return()

	ir := pkg.String()
	if !strings.Contains(ir, `"github.com/xgo-dev/llgo/runtime/internal/runtime.NewProc"`) {
		t.Fatalf("goroutine should delegate startup to the runtime:\n%s", ir)
	}
	if strings.Contains(ir, "pthread") {
		t.Fatalf("compiler IR should not depend on the pthread backend:\n%s", ir)
	}
	if fn := pkg.Module().NamedFunction(llvmgen.PkgRuntime + ".NewProc"); fn.ParamsCount() != 2 {
		t.Fatalf("NewProc takes %d parameters, want two", fn.ParamsCount())
	}
}

func TestRuntimePthreadStackSizeConstant(t *testing.T) {
	for _, target := range []*llvmgen.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten"},
	} {
		for _, size := range []uint64{0, 32 << 20} {
			prog := ssatest.NewProgram(t, target)
			prog.SetPthreadStackSize(size)
			pkg := prog.NewPackage("runtime", llvmgen.PkgRuntime)
			name := llvmgen.RuntimeGoroutineStackSizeVar
			g := pkg.NewVarEx(name, prog.Pointer(prog.Uintptr()))
			prog.InitPthreadStackSize(g)
			global := pkg.Module().NamedGlobal(name)
			if !global.IsGlobalConstant() || global.Initializer().ZExtValue() != size {
				t.Fatalf("runtime stack size: got %s, want constant %d", global.String(), size)
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		}
	}
}
