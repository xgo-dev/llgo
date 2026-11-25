package cl

import (
	"testing"

	"github.com/xgo-dev/llgo/internal/llvmgen"
	"github.com/xgo-dev/llvm"
)

func TestWasm32NativeCallbackRoundTripAcrossStructTags(t *testing.T) {
	const src = `package callback
import _ "unsafe"

type Before = struct{ Text string "label:\"before\"" }
type After = struct{ Text string "label:\"after\"" }
type Retagged func(After) string

//llgo:cdecl
type Callback func(After) string

//go:linkname original C.original
func original(Before) string

func convert(f func(Before) string) Callback { return Callback(Retagged(f)) }
func RoundTrip() Callback { return convert(original) }
`
	source, _, files := buildGoSSAPkg(t, src)
	prog := newLLSSAProgForTarget(t, &llvmgen.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"})
	defer prog.Dispose()
	pkg, err := NewPackage(prog, source, files)
	if err != nil {
		t.Fatal(err)
	}
	mod := pkg.Module()
	options := llvm.NewPassBuilderOptions()
	defer options.Dispose()
	options.SetVerifyEach(true)
	if err := mod.RunPasses("default<O2>", prog.TargetMachine(), options); err != nil {
		t.Fatal(err)
	}

	// Once the two Go function conversions fold, the caller must get the
	// original C entry and pass any remaining closure-environment check.
	fn := mustNamedFunction(t, mod, "callback.RoundTrip")
	original := mod.NamedFunction("original")
	ret := fn.FirstBasicBlock().LastInstruction()
	if original.IsNil() || ret.IsAReturnInst().IsNil() || ret.OperandsCount() != 1 || ret.Operand(0) != original {
		t.Fatalf("round trip did not return the original C callback:\n%s", fn)
	}
	for inst := fn.FirstBasicBlock().FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
		if inst.IsACallInst().IsNil() || inst.CalledValue().Name() != llvmgen.PkgRuntime+".AssertRuntimeError" {
			continue
		}
		if !inst.Operand(0).IsNull() {
			t.Fatalf("round trip triggered the closure capture check:\n%s", fn)
		}
	}
}
