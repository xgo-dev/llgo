package wasmresume

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestInventoryNumbersDirectAndIndirectCalls(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("resume")
	defer mod.Dispose()

	voidFn := llvm.FunctionType(ctx.VoidType(), nil, false)
	callee := llvm.AddFunction(mod, "callee", voidFn)
	callerType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{callee.Type()}, false)
	fn := llvm.AddFunction(mod, "caller", callerType)
	fn.AddFunctionAttr(ctx.CreateStringAttribute(FunctionAttribute, "1"))
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)

	direct := builder.CreateCall(voidFn, callee, nil, "")
	markCall(ctx, direct)
	target := fn.Param(0)
	indirect := builder.CreateCall(voidFn, target, nil, "")
	markCall(ctx, indirect)
	builder.CreateRetVoid()

	functions, err := Inventory(mod)
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) != 1 || functions[0].Name != "caller" {
		t.Fatalf("functions = %+v", functions)
	}
	calls := functions[0].Calls
	if len(calls) != 2 {
		t.Fatalf("calls = %+v", calls)
	}
	if calls[0].ID != 1 || calls[0].Callee != "callee" || calls[0].Indirect {
		t.Fatalf("direct call = %+v", calls[0])
	}
	if calls[1].ID != 2 || calls[1].Callee != "" || !calls[1].Indirect {
		t.Fatalf("indirect call = %+v", calls[1])
	}

	ir := mod.String()
	if !strings.Contains(ir, "!"+CallMetadata+" !0") ||
		!strings.Contains(ir, "!"+CallMetadata+" !1") ||
		!strings.Contains(ir, "!0 = !{i32 1, i32 1}") ||
		!strings.Contains(ir, "!1 = !{i32 1, i32 2}") {
		t.Fatalf("resume IDs were not written to call metadata:\n%s", ir)
	}
}

func TestInventoryRejectsMarkerInUnmarkedFunction(t *testing.T) {
	ctx, mod, fn, builder := newInventoryTestFunction(t, false)
	defer ctx.Dispose()
	defer mod.Dispose()
	defer builder.Dispose()
	call := builder.CreateCall(llvm.FunctionType(ctx.VoidType(), nil, false), fn, nil, "")
	markCall(ctx, call)
	builder.CreateRetVoid()

	if _, err := Inventory(mod); err == nil || !strings.Contains(err.Error(), "unmarked function") {
		t.Fatalf("Inventory error = %v", err)
	}
}

func TestInventoryRejectsMarkerOnNonCall(t *testing.T) {
	ctx, mod, _, builder := newInventoryTestFunction(t, true)
	defer ctx.Dispose()
	defer mod.Dispose()
	defer builder.Dispose()
	ret := builder.CreateRetVoid()
	markCall(ctx, ret)

	if _, err := Inventory(mod); err == nil || !strings.Contains(err.Error(), "non-call") {
		t.Fatalf("Inventory error = %v", err)
	}
}

func TestInventoryIgnoresUnmarkedDeclarations(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("empty")
	defer mod.Dispose()
	llvm.AddFunction(mod, "declaration", llvm.FunctionType(ctx.VoidType(), nil, false))
	functions, err := Inventory(mod)
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) != 0 {
		t.Fatalf("functions = %+v, want empty", functions)
	}
}

func TestInventoryIncludesMarkedLeafFunction(t *testing.T) {
	ctx, mod, _, builder := newInventoryTestFunction(t, true)
	defer ctx.Dispose()
	defer mod.Dispose()
	defer builder.Dispose()
	builder.CreateRetVoid()

	functions, err := Inventory(mod)
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) != 1 || functions[0].Name != "function" || len(functions[0].Calls) != 0 {
		t.Fatalf("functions = %+v", functions)
	}
}

func TestInventoryRejectsInvalidMarker(t *testing.T) {
	ctx, mod, fn, builder := newInventoryTestFunction(t, true)
	defer ctx.Dispose()
	defer mod.Dispose()
	defer builder.Dispose()
	call := builder.CreateCall(llvm.FunctionType(ctx.VoidType(), nil, false), fn, nil, "")
	kind := ctx.MDKindID(CallMetadata)
	version := llvm.ConstInt(ctx.Int32Type(), MarkerVersion+1, false).ConstantAsMetadata()
	call.SetMetadata(kind, ctx.MDNode([]llvm.Metadata{version}))
	builder.CreateRetVoid()

	if _, err := Inventory(mod); err == nil || !strings.Contains(err.Error(), "invalid resumable call marker") {
		t.Fatalf("Inventory error = %v", err)
	}
}

func newInventoryTestFunction(t *testing.T, marked bool) (llvm.Context, llvm.Module, llvm.Value, llvm.Builder) {
	t.Helper()
	ctx := llvm.NewContext()
	mod := ctx.NewModule("invalid")
	fn := llvm.AddFunction(mod, "function", llvm.FunctionType(ctx.VoidType(), nil, false))
	if marked {
		fn.AddFunctionAttr(ctx.CreateStringAttribute(FunctionAttribute, "1"))
	}
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	builder.SetInsertPointAtEnd(block)
	return ctx, mod, fn, builder
}

func markCall(ctx llvm.Context, instr llvm.Value) {
	kind := ctx.MDKindID(CallMetadata)
	version := llvm.ConstInt(ctx.Int32Type(), MarkerVersion, false).ConstantAsMetadata()
	instr.SetMetadata(kind, ctx.MDNode([]llvm.Metadata{version}))
}
