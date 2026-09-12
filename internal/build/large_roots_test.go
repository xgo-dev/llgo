package build

import (
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestLowerLargeAggregatesGCPolicy(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	for _, enabled := range []bool{false, true} {
		func() {
			prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
			defer prog.Dispose()
			prog.EnableGCRoots(enabled)
			mod := prog.NewPackage("large", "large").Module()
			ctx := mod.Context()
			typ := llvm.ArrayType(ctx.Int8Type(), 65537)
			fn := llvm.AddFunction(mod, "large", llvm.FunctionType(typ, nil, false))
			b := ctx.NewBuilder()
			defer b.Dispose()
			b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "entry"))
			b.CreateRet(llvm.ConstNull(typ))
			if lowerMainCExportAggregates(prog, mod, nil) {
				t.Fatal("empty C export set requested ABI lowering")
			}
			if !lowerMainCExportAggregates(prog, mod, []cExport{{}}) {
				t.Fatal("non-empty C export set skipped ABI lowering")
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			if got := !mod.NamedGlobal("llvm_gc_root_chain").IsNil(); got != enabled {
				t.Fatalf("GC roots enabled=%v: root chain present=%v", enabled, got)
			}
		}()
	}
}
