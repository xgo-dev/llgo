package cabi_test

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llgo/internal/cabi"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestFunctionAttributesSurviveABI(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmPrinters()
	for _, target := range []ssa.Target{{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "windows", GOARCH: "386"}, {GOOS: "wasip1", GOARCH: "wasm32"}} {
		t.Run(target.GOARCH, func(t *testing.T) {
			prog := ssa.NewProgram(&target)
			defer prog.Dispose()
			prog.SetFunctionAttributes("p.Stop", ssa.FunctionCold|ssa.FunctionNoReturn)
			pkg := prog.NewPackage("p", "p")
			// An ordinary aggregate argument and a large result exercise both
			// existing ABI transformations without any attribute-specific mapping.
			param := types.NewArray(types.Typ[types.Int64], 4)
			result := types.NewArray(types.Typ[types.Byte], 65537)
			sig := types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewParam(token.NoPos, nil, "value", param)),
				types.NewTuple(types.NewParam(token.NoPos, nil, "", result)), false)
			pkg.NewFunc("p.Stop", sig, ssa.InGo)
			abi.LowerLargeAggregates(prog.TargetData(), pkg.Module())
			cabi.NewTransformer(prog, "", "", false).TransformModule("p", pkg.Module())
			fn := pkg.Module().NamedFunction("p.Stop")
			for _, attr := range []string{"cold", "noreturn"} {
				if fn.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(attr)).IsNil() {
					t.Fatalf("ABI dropped %s:\n%s", attr, pkg.String())
				}
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
