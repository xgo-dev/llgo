package wasmresume

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLayoutFramesUsesRuntimeHeaderAndStableSlots(t *testing.T) {
	for _, test := range []struct {
		name       string
		dataLayout string
		wantSize   uint64
		wantAlign  int
		wantOffset []uint64
	}{
		{
			name:       "wasm32",
			dataLayout: "e-m:e-p:32:32-i64:64-n32:64-S128",
			wantSize:   32,
			wantAlign:  8,
			wantOffset: []uint64{0, 4, 8, 16, 24},
		},
		{
			name:       "wasm64",
			dataLayout: "e-m:e-p:64:64-i64:64-n32:64-S128",
			wantSize:   40,
			wantAlign:  8,
			wantOffset: []uint64{0, 8, 16, 24, 32},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			mod := ctx.NewModule(test.name)
			defer mod.Dispose()
			targetData := llvm.NewTargetData(test.dataLayout)
			defer targetData.Dispose()

			i64 := ctx.Int64Type()
			fn := llvm.AddFunction(mod, "leaf", llvm.FunctionType(i64, []llvm.Type{i64}, false))
			markFunction(ctx, fn)
			block := ctx.AddBasicBlock(fn, "entry")
			builder := ctx.NewBuilder()
			defer builder.Dispose()
			builder.SetInsertPointAtEnd(block)
			builder.CreateRet(fn.Param(0))

			layouts, err := layoutFrames(mod, targetData)
			if err != nil {
				t.Fatal(err)
			}
			if len(layouts) != 1 {
				t.Fatalf("layouts = %+v", layouts)
			}
			layout := layouts[0]
			if layout.size != test.wantSize || layout.alignment != test.wantAlign {
				t.Fatalf("layout size/alignment = %d/%d, want %d/%d",
					layout.size, layout.alignment, test.wantSize, test.wantAlign)
			}
			for field, want := range test.wantOffset {
				if got := targetData.ElementOffset(layout.typ, field); got != want {
					t.Errorf("field %d offset = %d, want %d", field, got, want)
				}
			}
			if layout.fieldIndex(1) != 3 || layout.fieldIndex(2) != 4 {
				t.Fatalf("slot fields = %d/%d, want 3/4", layout.fieldIndex(1), layout.fieldIndex(2))
			}
			if layout.fieldIndex(0) != -1 || layout.fieldIndex(3) != -1 {
				t.Fatalf("invalid slot fields = %d/%d, want -1/-1",
					layout.fieldIndex(0), layout.fieldIndex(3))
			}
		})
	}
}

func TestLayoutFramesKeepsDynamicAllocaAsPointer(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("dynamic")
	defer mod.Dispose()
	targetData := llvm.NewTargetData("e-m:e-p:32:32-i64:64-n32:64-S128")
	defer targetData.Dispose()

	i32 := ctx.Int32Type()
	ptr := llvm.PointerType(i32, 0)
	calleeType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr}, false)
	callee := llvm.AddFunction(mod, "callee", calleeType)
	fn := llvm.AddFunction(mod, "caller", llvm.FunctionType(ctx.VoidType(), []llvm.Type{i32}, false))
	markFunction(ctx, fn)
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	local := builder.CreateArrayAlloca(i32, fn.Param(0), "local")
	call := builder.CreateCall(calleeType, callee, []llvm.Value{local}, "")
	markCall(ctx, call)
	builder.CreateRetVoid()

	layouts, err := layoutFrames(mod, targetData)
	if err != nil {
		t.Fatal(err)
	}
	layout := layouts[0]
	fields := layout.typ.StructElementTypes()
	if len(fields) != 5 || fields[4].TypeKind() != llvm.PointerTypeKind {
		t.Fatalf("frame fields = %v, want dynamic alloca pointer at field 4", fields)
	}
}

func TestLayoutFramesPreservesAllocaAlignment(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("aligned")
	defer mod.Dispose()
	targetData := llvm.NewTargetData("e-m:e-p:64:64-i64:64-n32:64-S128")
	defer targetData.Dispose()

	i32 := ctx.Int32Type()
	ptr := llvm.PointerType(i32, 0)
	calleeType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr}, false)
	callee := llvm.AddFunction(mod, "callee", calleeType)
	fn := llvm.AddFunction(mod, "caller", llvm.FunctionType(ctx.VoidType(), nil, false))
	markFunction(ctx, fn)
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	local := builder.CreateAlloca(i32, "local")
	local.SetAlignment(32)
	call := builder.CreateCall(calleeType, callee, []llvm.Value{local}, "")
	markCall(ctx, call)
	builder.CreateRetVoid()

	layouts, err := layoutFrames(mod, targetData)
	if err != nil {
		t.Fatal(err)
	}
	layout := layouts[0]
	if layout.alignment != 32 {
		t.Fatalf("frame alignment = %d, want 32", layout.alignment)
	}
	slot := layout.plan.slots[0]
	offset := targetData.ElementOffset(layout.typ, layout.fieldIndex(slot.id))
	if offset%32 != 0 {
		t.Fatalf("aligned alloca offset = %d, want a multiple of 32", offset)
	}
}
