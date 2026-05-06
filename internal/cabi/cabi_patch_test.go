//go:build !llgo
// +build !llgo

package cabi

import (
	"testing"

	llssa "github.com/goplus/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

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
