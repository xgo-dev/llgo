package ssa

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

func testWasmStringProfiles(t *testing.T, shape string, test func(*testing.T, Program, Package)) {
	t.Helper()
	for _, target := range []Target{
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
	} {
		t.Run(shape+"/"+target.WasmProfile, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			setTestRuntime(t, prog)
			test(t, prog, prog.NewPackage("p", "example.com/p"))
		})
	}
}

func checkNativeStringZeroResults(t *testing.T, pkg Package, observers ...llvm.Value) {
	t.Helper()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid string adapter: %v\n%s", err, pkg.String())
	}
	options := llvm.NewPassBuilderOptions()
	defer options.Dispose()
	if err := pkg.Module().RunPasses("default<O2>", pkg.Prog.TargetMachine(), options); err != nil {
		t.Fatal(err)
	}
	for _, observer := range observers {
		returns := 0
		for block := observer.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			ret := block.LastInstruction()
			if ret.IsNil() || ret.InstructionOpcode() != llvm.Ret {
				continue
			}
			returns++
			if ret.OperandsCount() != 1 || !ret.Operand(0).IsNull() {
				t.Errorf("string padding, payload, or callback identity changed in %s:\n%s", observer.Name(), observer)
			}
		}
		if returns == 0 {
			t.Errorf("observer %s has no return sites", observer.Name())
		}
	}
}
