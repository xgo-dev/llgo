//go:build !llgo
// +build !llgo

package cabi

import (
	"testing"

	llssa "github.com/goplus/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestDevLTOGlobalDCETargetArchAndNewTransformerArchSelection(t *testing.T) {
	if got := targetArch("riscv64-unknown-linux-gnu"); got != "riscv64" {
		t.Fatalf("targetArch(triple) = %q, want riscv64", got)
	}
	if got := targetArch("wasm"); got != "wasm" {
		t.Fatalf("targetArch(single arch) = %q, want wasm", got)
	}

	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	prog := llssa.NewProgram(nil)
	tests := []struct {
		target string
		abi    string
		arch   string
		check  func(TypeInfoSys) bool
	}{
		{"xtensa-esp32-none-elf", "", "xtensa", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoEsp32); return ok }},
		{"riscv32-unknown-elf", "ilp32f", "riscv32", func(sys TypeInfoSys) bool {
			rv, ok := sys.(*TypeInfoRiscv32)
			return ok && rv.mabi == "ilp32f"
		}},
		{"amd64-unknown-linux-gnu", "", "amd64", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoAmd64); return ok }},
		{"arm64-apple-darwin", "", "arm64", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoArm64); return ok }},
		{"arm-unknown-linux-gnueabihf", "", "arm", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoArm); return ok }},
		{"wasm-unknown-wasip1", "", "wasm", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoWasm); return ok }},
		{"riscv64-unknown-linux-gnu", "lp64d", "riscv64", func(sys TypeInfoSys) bool {
			rv, ok := sys.(*TypeInfoRiscv64)
			return ok && rv.mabi == "lp64d"
		}},
		{"386-unknown-linux-gnu", "", "386", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfo386); return ok }},
	}
	for _, tc := range tests {
		tr := NewTransformer(prog, tc.target, tc.abi, ModeCFunc, true)
		if tr.arch != tc.arch {
			t.Fatalf("NewTransformer(%q).arch = %q, want %q", tc.target, tr.arch, tc.arch)
		}
		if tr.mode != ModeCFunc || !tr.optimize {
			t.Fatalf("NewTransformer did not preserve mode/optimize: mode=%v optimize=%v", tr.mode, tr.optimize)
		}
		if !tc.check(tr.sys) {
			t.Fatalf("NewTransformer(%q) selected unexpected sys implementation %T", tc.target, tr.sys)
		}
	}
}

func TestDevLTOGlobalDCEFuncNoUnwindCreatesNounwindAttribute(t *testing.T) {
	ctx := llvm.NewContext()
	attr := funcNoUnwind(ctx)
	if attr.IsNil() {
		t.Fatal("funcNoUnwind returned nil attribute")
	}
	if got, want := attr.GetEnumKind(), int(llvm.AttributeKindID("nounwind")); got != want {
		t.Fatalf("funcNoUnwind kind = %d, want %d", got, want)
	}
	if got := attr.GetEnumValue(); got != 0 {
		t.Fatalf("funcNoUnwind value = %d, want 0", got)
	}
}

func TestSetSkipFuncsAndShouldSkipCall(t *testing.T) {
	tr := &Transformer{}
	tr.SetSkipFuncs([]string{" foo ", "", "bar"})

	if !tr.shouldSkipFunc("foo") {
		t.Fatalf("shouldSkipFunc(foo) = false, want true")
	}
	if !tr.shouldSkipFunc("bar") {
		t.Fatalf("shouldSkipFunc(bar) = false, want true")
	}
	if tr.shouldSkipFunc("") {
		t.Fatalf("shouldSkipFunc(\"\") = true, want false")
	}
	if tr.shouldSkipFunc("baz") {
		t.Fatalf("shouldSkipFunc(baz) = true, want false")
	}

	ctx := llvm.NewContext()
	mod := ctx.NewModule("m")
	fty := llvm.FunctionType(ctx.VoidType(), nil, false)

	callee := llvm.AddFunction(mod, "foo", fty)
	caller := llvm.AddFunction(mod, "caller", fty)
	b := ctx.NewBuilder()
	entry := ctx.AddBasicBlock(caller, "entry")
	b.SetInsertPointAtEnd(entry)
	directCall := llvm.CreateCall(b, fty, callee, nil)
	b.CreateRetVoid()
	if !tr.shouldSkipCall(directCall) {
		t.Fatalf("shouldSkipCall(direct call to foo) = false, want true")
	}

	ptrTy := llvm.PointerType(fty, 0)
	caller2Ty := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptrTy}, false)
	caller2 := llvm.AddFunction(mod, "caller2", caller2Ty)
	b2 := ctx.NewBuilder()
	entry2 := ctx.AddBasicBlock(caller2, "entry")
	b2.SetInsertPointAtEnd(entry2)
	indirectCall := b2.CreateCall(fty, caller2.Param(0), nil, "")
	b2.CreateRetVoid()
	if tr.shouldSkipCall(indirectCall) {
		t.Fatalf("shouldSkipCall(indirect call) = true, want false")
	}
}

func TestRuntimeHeaderWrapAndTypeInfo(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	prog := llssa.NewProgram(nil)
	tr := NewTransformer(prog, "", "", ModeAllFunc, false)

	ctx := llvm.NewContext()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	sliceTy := ctx.StructCreateNamed("github.com/goplus/llgo/runtime/internal/runtime.Slice")
	sliceTy.StructSetBody([]llvm.Type{ptr, ctx.Int64Type(), ctx.Int64Type()}, false)

	if !tr.IsWrapType(ctx, llvm.FunctionType(ctx.VoidType(), nil, false), sliceTy, 1) {
		t.Fatalf("IsWrapType should be true for runtime Slice header")
	}
	info := tr.GetTypeInfo(ctx, llvm.FunctionType(ctx.VoidType(), nil, false), sliceTy, 1)
	if info.Kind == AttrNone {
		t.Fatalf("GetTypeInfo should not keep AttrNone for runtime Slice")
	}
	if info.Size == 0 || info.Align == 0 {
		t.Fatalf("GetTypeInfo size/align should be non-zero, got size=%d align=%d", info.Size, info.Align)
	}
}
