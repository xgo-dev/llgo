package cabi

import (
	"testing"

	"github.com/xgo-dev/llgo/internal/abi/abitest"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestAggregateLoweringFunctionProperties(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	abitest.CheckFunctionProperties(t, "{ i64, i64, i64 }", func(mod llvm.Module) {
		prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
		defer prog.Dispose()
		NewTransformer(prog, "x86_64-unknown-linux-gnu", "", true).TransformModule("test", mod)
	})
}
