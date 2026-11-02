package cabi

import (
	"fmt"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestAMD64ZeroLengthMemberLowering(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	for _, goos := range []string{"linux", "darwin"} {
		t.Run(goos, func(t *testing.T) {
			prog := llssa.NewProgram(&llssa.Target{GOOS: goos, GOARCH: "amd64"})
			defer prog.Dispose()
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			tr := NewTransformer(prog, prog.Target().Spec().Triple, "", false)
			// Clang passes packed { char tag[N]; long long tail[0]; } in
			// memory for N=1 or 2, but in a register for aligned N=8.
			for _, size := range []int{1, 2, 8} {
				t.Run(fmt.Sprint(size), func(t *testing.T) {
					typ := ctx.StructType([]llvm.Type{
						llvm.ArrayType(ctx.Int8Type(), size),
						llvm.ArrayType(ctx.Int64Type(), 0),
					}, true)
					ftyp := llvm.FunctionType(typ, []llvm.Type{typ}, false)
					mod := ctx.NewModule("zero-length-member")
					defer mod.Dispose()
					mod.SetDataLayout(prog.TargetData().String())
					callee := llvm.AddFunction(mod, "identity", ftyp)
					b := ctx.NewBuilder()
					defer b.Dispose()
					b.SetInsertPointAtEnd(ctx.AddBasicBlock(callee, "entry"))
					b.CreateRet(callee.Param(0))
					ptr := llvm.PointerType(ctx.Int8Type(), 0)
					caller := llvm.AddFunction(mod, "caller", llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr, ptr, ptr}, false))
					b.SetInsertPointAtEnd(ctx.AddBasicBlock(caller, "entry"))
					value := b.CreateLoad(typ, caller.Param(1), "")
					direct := b.CreateCall(ftyp, callee, []llvm.Value{value}, "")
					indirect := b.CreateCall(ftyp, caller.Param(0), []llvm.Value{direct}, "")
					b.CreateStore(indirect, caller.Param(2))
					b.CreateRetVoid()
					if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
						t.Fatal(err)
					}
					tr.TransformModule("zero-length-member", mod)
					if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
						t.Fatalf("invalid lowered module: %v\n%s", err, mod.String())
					}
					check := func(value llvm.Value, call bool) {
						t.Helper()
						var ft llvm.Type
						getAttr := value.GetEnumAttributeAtIndex
						if call {
							ft = value.CalledFunctionType()
							getAttr = value.GetCallSiteEnumAttribute
						} else {
							ft = value.GlobalValueType()
						}
						if size == 8 {
							if ft.ReturnType() != ctx.Int64Type() || ft.ParamTypesCount() != 1 || ft.ParamTypes()[0] != ctx.Int64Type() {
								t.Fatalf("aligned empty member should retain register ABI:\n%s", value.String())
							}
							return
						}
						if ft.ReturnType() != ctx.VoidType() || ft.ParamTypesCount() != 2 ||
							getAttr(1, llvm.AttributeKindID("sret")).IsNil() ||
							getAttr(2, llvm.AttributeKindID("byval")).IsNil() {
							t.Fatalf("unaligned empty member should use sret/byval:\n%s", value.String())
						}
						align := getAttr(2, llvm.AttributeKindID("align"))
						if align.IsNil() || align.GetEnumValue() != 8 {
							t.Fatalf("byval alignment should be 8:\n%s", value.String())
						}
					}
					check(mod.NamedFunction("identity"), false)
					calls := 0
					for bb := caller.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
						for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
							if !inst.IsACallInst().IsNil() && (inst.CalledValue() == mod.NamedFunction("identity") || inst.CalledValue() == caller.Param(0)) {
								check(inst, true)
								calls++
							}
						}
					}
					if calls != 2 {
						t.Fatalf("checked %d calls, want direct and indirect calls:\n%s", calls, mod.String())
					}
				})
			}
		})
	}
}

func TestAMD64AlignedIntegerAggregate(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	tr := NewTransformer(prog, "x86_64-unknown-linux-gnu", "", false)
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	marker := llvm.ArrayType(llvm.VectorType(ctx.Int32Type(), 4), 0)
	for _, laneBits := range []int{8, 16, 32} {
		typ := ctx.StructType([]llvm.Type{
			marker, llvm.ArrayType(ctx.IntType(laneBits), 128/laneBits),
		}, false)
		ftyp := llvm.FunctionType(typ, []llvm.Type{typ}, false)
		for _, index := range []int{0, 1} {
			info := tr.GetTypeInfo(ctx, ftyp, typ, index)
			if info.Kind != AttrWidthType2 || info.Type1 != ctx.Int64Type() || info.Type2 != ctx.Int64Type() {
				t.Fatalf("i%d aggregate at index %d: kind %v, components %s, %s; want two i64",
					laneBits, index, info.Kind, info.Type1, info.Type2)
			}
		}
	}
}

func BenchmarkAMD64NestedArrayClassification(b *testing.B) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	typ := ctx.Int8Type()
	for depth := 0; depth < 18; depth++ {
		typ = llvm.ArrayType(typ, 2)
	}
	ftyp := llvm.FunctionType(typ, []llvm.Type{typ}, false)
	tr := NewTransformer(prog, prog.Target().Spec().Triple, "", false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.GetTypeInfo(ctx, ftyp, typ, 1)
	}
}

func TestAMD64PackedAggregate(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	tr := NewTransformer(prog, "x86_64-unknown-linux-gnu", "", false)
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	packed := ctx.StructType([]llvm.Type{ctx.Int8Type(), ctx.Int64Type()}, true)
	marker := llvm.ArrayType(llvm.VectorType(ctx.Int32Type(), 4), 0)
	for _, tc := range []struct {
		name       string
		typ        llvm.Type
		byvalAlign int
	}{
		{"unaligned", packed, 8},
		{"nested", ctx.StructType([]llvm.Type{packed}, false), 8},
		{"array", llvm.ArrayType(packed, 1), 8},
		{"array stride", llvm.ArrayType(ctx.StructType([]llvm.Type{ctx.Int32Type(), ctx.Int8Type()}, true), 2), 8},
		{"nested array stride", llvm.ArrayType(llvm.ArrayType(ctx.StructType([]llvm.Type{ctx.Int16Type(), ctx.Int8Type()}, true), 2), 2), 8},
		{"aligned nested array", llvm.ArrayType(llvm.ArrayType(ctx.Int16Type(), 2), 2), 0},
		{"aligned", ctx.StructType([]llvm.Type{ctx.Int8Type(), ctx.Int64Type()}, false), 0},
		{"packed aligned member", ctx.StructType([]llvm.Type{ctx.Int64Type()}, true), 0},
		{"unaligned empty member", ctx.StructType([]llvm.Type{ctx.Int8Type(), llvm.ArrayType(ctx.Int64Type(), 0)}, true), 8},
		{"unaligned empty member after array", ctx.StructType([]llvm.Type{llvm.ArrayType(ctx.Int8Type(), 2), llvm.ArrayType(ctx.Int64Type(), 0)}, true), 8},
		{"aligned empty member", ctx.StructType([]llvm.Type{llvm.ArrayType(ctx.Int8Type(), 8), llvm.ArrayType(ctx.Int64Type(), 0)}, true), 0},
		{"over-aligned packed member", ctx.StructType([]llvm.Type{marker, packed}, false), 16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ftyp := llvm.FunctionType(tc.typ, []llvm.Type{tc.typ}, false)
			for _, index := range []int{0, 1} {
				info := tr.GetTypeInfo(ctx, ftyp, tc.typ, index)
				memory := tc.byvalAlign != 0
				if got := info.Kind == AttrPointer; got != memory {
					t.Fatalf("index %d: memory = %t, want %t", index, got, memory)
				}
				if memory && info.Type1 != llvm.PointerType(tc.typ, 0) {
					t.Fatalf("index %d: memory type = %s, want pointer", index, info.Type1)
				}
				if memory && index == 1 && info.ByValAlign != tc.byvalAlign {
					t.Fatalf("parameter byval alignment = %d, want %d", info.ByValAlign, tc.byvalAlign)
				}
			}
		})
	}
}
