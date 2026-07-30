package wasmresume

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestEmitLeafEntriesLoadsParametersAndStoresResult(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("leaf")
	defer mod.Dispose()
	mod.SetDataLayout("e-m:e-p:32:32-i64:64-n32:64-S128")
	targetData := llvm.NewTargetData(mod.DataLayout())
	defer targetData.Dispose()

	i32 := ctx.Int32Type()
	fn := llvm.AddFunction(mod, "leaf", llvm.FunctionType(i32, []llvm.Type{i32}, false))
	markFunction(ctx, fn)
	fn.Param(0).SetName("input")
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	builder.CreateRet(builder.CreateAdd(fn.Param(0), llvm.ConstInt(i32, 1, false), "value"))

	lowered, err := emitLeafEntries(mod, targetData)
	if err != nil {
		t.Fatal(err)
	}
	if len(lowered) != 1 || lowered[0].layout.size != 20 || lowered[0].layout.alignment != 4 {
		t.Fatalf("lowered leaves = %+v", lowered)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify lowered leaf: %v\n%s", err, mod.String())
	}

	ir := mod.String()
	for _, want := range []string{
		`@__llgo_wasm_resume_desc.leaf = constant { ptr, i32, i32, i32, i32 }`,
		`{ ptr @__llgo_wasm_resume.leaf, i32 20, i32 4, i32 0, i32 0 }`,
		`define internal i8 @__llgo_wasm_resume.leaf(ptr %0, ptr %1)`,
		`load i32, ptr %2`,
		`call i32 @leaf(i32 %input)`,
		`store i32 %3, ptr %4`,
		`ret i8 1`,
	} {
		if !strings.Contains(ir, want) {
			t.Errorf("lowered leaf is missing %q:\n%s", want, ir)
		}
	}
}

func TestEmitLeafEntriesSkipsNonLeafAndDeclarations(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("skip")
	defer mod.Dispose()
	targetData := llvm.NewTargetData("e-m:e-p:32:32-i64:64-n32:64-S128")
	defer targetData.Dispose()

	voidFn := llvm.FunctionType(ctx.VoidType(), nil, false)
	declaration := llvm.AddFunction(mod, "declaration", voidFn)
	markFunction(ctx, declaration)
	callee := llvm.AddFunction(mod, "callee", voidFn)
	nonLeaf := llvm.AddFunction(mod, "nonleaf", voidFn)
	markFunction(ctx, nonLeaf)
	block := ctx.AddBasicBlock(nonLeaf, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	call := builder.CreateCall(voidFn, callee, nil, "")
	markCall(ctx, call)
	builder.CreateRetVoid()

	lowered, err := emitLeafEntries(mod, targetData)
	if err != nil {
		t.Fatal(err)
	}
	if len(lowered) != 0 {
		t.Fatalf("lowered leaves = %+v, want empty", lowered)
	}
}

func TestEmitLeafEntriesRejectsDuplicateSymbols(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("duplicate")
	defer mod.Dispose()
	targetData := llvm.NewTargetData("e-m:e-p:32:32-i64:64-n32:64-S128")
	defer targetData.Dispose()

	voidFn := llvm.FunctionType(ctx.VoidType(), nil, false)
	fn := llvm.AddFunction(mod, "leaf", voidFn)
	markFunction(ctx, fn)
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	builder.CreateRetVoid()
	llvm.AddFunction(mod, resumeEntryPrefix+"leaf", llvm.FunctionType(ctx.Int8Type(), nil, false))

	if _, err := emitLeafEntries(mod, targetData); err == nil ||
		!strings.Contains(err.Error(), "duplicate resumable descriptor") {
		t.Fatalf("emitLeafEntries error = %v", err)
	}
}
