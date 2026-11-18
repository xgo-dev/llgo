package cabi

import (
	"fmt"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func BenchmarkAMD64SharedAggregateLowering(b *testing.B) {
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
	ftyp := llvm.FunctionType(ctx.VoidType(), []llvm.Type{typ}, false)
	tr := NewTransformer(prog, prog.Target().Spec().Triple, "", false)
	names := make([]string, 32)
	for i := range names {
		names[i] = fmt.Sprintf("consume%d", i)
	}
	b.ResetTimer()
	for range b.N {
		mod := ctx.NewModule("shared-aggregate")
		mod.SetDataLayout(prog.TargetData().String())
		for _, name := range names {
			llvm.AddFunction(mod, name, ftyp)
		}
		tr.TransformModule("shared-aggregate", mod)
		mod.Dispose()
	}
}
