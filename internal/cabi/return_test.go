//go:build !llgo

package cabi_test

import (
	"testing"

	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llvm"
)

// Compile Go source through the normal ABI lowering before optimizing it. A
// return conversion temporary in each branch prevents SROA from eliminating
// the store/load pairs and can leave dynamic stack adjustments in leaf code.
func TestAggregateReturnStoragePromotes(t *testing.T) {
	for _, target := range []struct{ goos, goarch string }{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"linux", "386"},
		{"linux", "arm"},
		{"linux", "riscv64"},
		{"windows", "amd64"},
		{"windows", "arm64"},
		{"windows", "386"},
		{"wasip1", "wasm"},
	} {
		t.Run(target.goos+"/"+target.goarch, func(t *testing.T) {
			conf := build.NewDefaultConf(build.ModeGen)
			conf.Goos, conf.Goarch = target.goos, target.goarch
			pkgs, err := build.Do([]string{"./_testdata/returns/returns.go"}, conf)
			if err != nil {
				t.Fatal(err)
			}
			pkg := pkgs[0].LPkg
			defer pkg.Prog.Dispose()
			mod := pkg.Module()
			mod.SetDataLayout(pkg.Prog.DataLayout())
			mod.SetTarget(pkg.Prog.Target().Spec().Triple)
			pbo := llvm.NewPassBuilderOptions()
			defer pbo.Dispose()
			pbo.SetVerifyEach(true)
			if err := mod.RunPasses("default<O2>", pkg.Prog.TargetMachine(), pbo); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"Width", "Pair", "PairBool"} {
				fn := mod.NamedFunction(pkgs[0].PkgPath + "." + name)
				if fn.IsNil() {
					t.Fatalf("function %s not found:\n%s", name, mod.String())
				}
				for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
					for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
						if instr.IsAAllocaInst().IsNil() {
							continue
						}
						// LLVM may retain a byte slot for the i1 store followed by
						// an ABI-width load. It must be static and scalar; the
						// integer-only returns should need no stack storage.
						if name != "PairBool" || bb != fn.EntryBasicBlock() || instr.AllocatedType().TypeKind() != llvm.IntegerTypeKind {
							t.Fatalf("aggregate return storage was not promoted after O2:\n%s", fn.String())
						}
					}
				}
			}
			asm, err := pkg.Prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatalf("emit aggregate return assembly: %v", err)
			}
			asm.Dispose()
		})
	}
}
