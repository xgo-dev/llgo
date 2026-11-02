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
