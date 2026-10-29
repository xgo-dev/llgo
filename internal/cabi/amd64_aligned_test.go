package cabi

import (
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

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
	for _, tc := range []struct {
		name   string
		typ    llvm.Type
		memory bool
	}{
		{"unaligned", packed, true},
		{"nested", ctx.StructType([]llvm.Type{packed}, false), true},
		{"array", llvm.ArrayType(packed, 1), true},
		{"array stride", llvm.ArrayType(ctx.StructType([]llvm.Type{ctx.Int32Type(), ctx.Int8Type()}, true), 2), true},
		{"aligned", ctx.StructType([]llvm.Type{ctx.Int8Type(), ctx.Int64Type()}, false), false},
		{"packed aligned member", ctx.StructType([]llvm.Type{ctx.Int64Type()}, true), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ftyp := llvm.FunctionType(tc.typ, []llvm.Type{tc.typ}, false)
			for _, index := range []int{0, 1} {
				info := tr.GetTypeInfo(ctx, ftyp, tc.typ, index)
				if got := info.Kind == AttrPointer; got != tc.memory {
					t.Fatalf("index %d: memory = %t, want %t", index, got, tc.memory)
				}
				if tc.memory && info.ByValAlign != 8 {
					t.Fatalf("index %d: byval alignment = %d, want 8", index, info.ByValAlign)
				}
			}
		})
	}
}
