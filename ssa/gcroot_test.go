//go:build !llgo

package ssa_test

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/goplus/llgo/ssa"
	"github.com/goplus/llgo/ssa/ssatest"
	"github.com/xgo-dev/llvm"
)

func TestGCRootFrameIR(t *testing.T) {
	prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "js", GOARCH: "wasm"})
	pkg := prog.NewPackage("main", "main")

	param := types.NewParam(token.NoPos, nil, "p", types.NewPointer(types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), nil, false)
	fn := pkg.NewFunc("main.keep", sig, ssa.InGo)
	b := fn.MakeBody(1)
	mayGC := pkg.NewFunc("runtime.mayGC", ssa.NoArgsNoRet, ssa.InGo)
	b.Call(mayGC.Expr)
	root := fn.NewGCRoots(1)[0]
	assertPanics(t, func() {
		fn.NewGCRoots(1)
	})
	b.SetGCRoot(root, b.Param(0))
	b.Return()
	b.EndBuild()

	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	ir := pkg.String()
	for _, want := range []string{
		`define void @main.keep(ptr %0)`,
		`@llvm_gc_root_chain`,
		`[1 x ptr]`,
		`store ptr %0`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %q in GC root IR:\n%s", want, ir)
		}
	}
	if strings.Contains(ir, `llvm.gcroot`) || strings.Contains(ir, `gc "shadow-stack"`) {
		t.Fatalf("GC roots must be lowered before optimization:\n%s", ir)
	}
	if push, call := strings.Index(ir, `store ptr %`), strings.Index(ir, `call void @runtime.mayGC`); push < 0 || call < 0 || push > call {
		t.Fatalf("GC root frame must be linked before a safepoint:\n%s", ir)
	}

	mod := pkg.Module()
	mod.SetDataLayout(prog.DataLayout())
	mod.SetTarget(prog.Target().Spec().Triple)
	pbo := llvm.NewPassBuilderOptions()
	defer pbo.Dispose()
	if err := mod.RunPasses("default<O2>", prog.TargetMachine(), pbo); err != nil {
		t.Fatalf("optimize GC root frame: %v", err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify optimized GC root frame: %v", err)
	}
	optimized := mod.String()
	if !strings.Contains(optimized, `@llvm_gc_root_chain`) ||
		strings.Contains(optimized, `llvm.gcroot`) ||
		strings.Contains(optimized, `gc "shadow-stack"`) {
		t.Fatalf("optimization changed the lowered GC root ABI:\n%s", optimized)
	}
}

func TestGCRootReservationAndClosureContext(t *testing.T) {
	prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "js", GOARCH: "wasm"})
	pkg := prog.NewPackage("main", "main")

	fn := pkg.NewFunc("main.empty", ssa.NoArgsNoRet, ssa.InGo)
	if roots := fn.NewGCRoots(0); roots != nil {
		t.Fatalf("NewGCRoots(0) = %v, want nil", roots)
	}
	if context := fn.ClosureContextParam(); !context.IsNil() {
		t.Fatal("ordinary function unexpectedly has a closure context")
	}

	context := types.NewParam(token.NoPos, nil, "__llgo_ctx", types.Typ[types.UnsafePointer])
	closureSig := ssa.FuncAddCtx(context, ssa.NoArgsNoRet)
	closure := pkg.NewFuncEx("main.closure", closureSig, ssa.InGo, true, false)
	if context := closure.ClosureContextParam(); context.IsNil() {
		t.Fatal("closure function is missing its hidden context")
	}
}

func TestAggregateGCRootPointers(t *testing.T) {
	prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "js", GOARCH: "wasm"})
	pkg := prog.NewPackage("main", "main")

	ptr := types.NewPointer(types.Typ[types.Int])
	fnType := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fields := []*types.Var{
		types.NewField(token.NoPos, nil, "p", ptr, false),
		types.NewField(token.NoPos, nil, "s", types.NewSlice(types.Typ[types.Byte]), false),
		types.NewField(token.NoPos, nil, "text", types.Typ[types.String], false),
		types.NewField(token.NoPos, nil, "any", types.NewInterfaceType(nil, nil).Complete(), false),
		types.NewField(token.NoPos, nil, "fn", fnType, false),
		types.NewField(token.NoPos, nil, "array", types.NewArray(ptr, 2), false),
	}
	holder := types.NewStruct(fields, nil)
	param := types.NewParam(token.NoPos, nil, "holder", holder)
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), nil, false)
	fn := pkg.NewFunc("main.aggregate", sig, ssa.InGo)
	b := fn.MakeBody(1)

	value := b.Param(0)
	if got := prog.GCRootCount(value.Type); got != 7 {
		t.Fatalf("GCRootCount(holder) = %d, want 7", got)
	}
	roots := b.GCRootPointers(value)
	if len(roots) != 7 {
		t.Fatalf("GCRootPointers(holder) returned %d roots, want 7", len(roots))
	}
	slots := fn.NewGCRoots(len(roots))
	for i, value := range roots {
		b.SetGCRoot(slots[i], value)
	}
	b.Return()
	b.EndBuild()

	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	if ir := pkg.String(); !strings.Contains(ir, `[7 x ptr]`) {
		t.Fatalf("aggregate did not emit one seven-root frame:\n%s", ir)
	}
}

func TestPatchedNestedGCRootPointers(t *testing.T) {
	prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "js", GOARCH: "wasm"})
	pkg := prog.NewPackage("main", "main")

	originalPkg := types.NewPackage("syscall/js", "js")
	originalName := types.NewTypeName(token.NoPos, originalPkg, "Value", nil)
	original := types.NewNamed(originalName, types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, originalPkg, "pointer", types.NewPointer(types.Typ[types.Int]), false),
		types.NewField(token.NoPos, originalPkg, "data", types.Typ[types.UnsafePointer], false),
	}, nil), nil)
	patchedName := types.NewTypeName(token.NoPos, originalPkg, "Value", nil)
	patched := types.NewNamed(patchedName, types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, originalPkg, "ref", types.Typ[types.Int32], false),
	}, nil), nil)
	prog.SetPatch(func(typ types.Type) types.Type {
		if typ == original {
			return patched
		}
		return typ
	})

	outer := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "value", original, false),
		types.NewField(token.NoPos, nil, "err", types.NewInterfaceType(nil, nil).Complete(), false),
	}, nil)
	param := types.NewParam(token.NoPos, nil, "result", outer)
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), nil, false)
	fn := pkg.NewFunc("main.patched", sig, ssa.InGo)
	b := fn.MakeBody(1)

	roots := b.GCRootPointers(b.Param(0))
	if len(roots) != 1 {
		t.Fatalf("GCRootPointers(patched nested struct) returned %d roots, want 1", len(roots))
	}
	b.Return()
	b.EndBuild()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("operation did not panic")
		}
	}()
	fn()
}
