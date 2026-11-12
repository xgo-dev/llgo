package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestZeroSizedFieldDerefEmitsBaseNilGuard(t *testing.T) {
	const src = `package zeroderef
type T struct {
	n int
	z struct{}
}
func Eq(p, q *T) bool {
	return p.z == q.z
}
`
	ir := compileWithRewrites(t, src, nil)
	if got := strings.Count(ir, "AssertNilDeref"); got < 4 {
		t.Fatalf("zero-sized field comparison should guard field bases and loads, got %d guards:\n%s", got, ir)
	}
}

func TestZeroSizedSourceClosureElidesEnvironment(t *testing.T) {
	const src = `package zeroclosure

func makeValue() (func() *struct{}, *struct{}) {
	value := struct{}{}
	return func() *struct{} { return &value }, &value
}

func compareLocal() bool {
	value := struct{}{}
	closure := func() *struct{} { return &value }
	return closure() == &value
}

func keepPointer(pointer *struct{}) func() bool {
	return func() bool { return pointer == nil }
}
`
	targets := []struct {
		name   string
		target *llssa.Target
	}{
		{name: "native"},
		{name: "wasm-explicit", target: &llssa.Target{GOOS: "wasip1", GOARCH: "wasm"}},
	}
	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			ir := compileWithRewritesTarget(t, src, nil, target.target)
			for _, want := range []string{
				`{ ptr @"zeroclosure.makeValue$1", ptr null }`,
				`define ptr @"zeroclosure.makeValue$1"()`,
				`define ptr @"zeroclosure.compareLocal$1"()`,
				`ret ptr @"__llgo.moduleZeroSizedAlloc$"`,
			} {
				if !strings.Contains(ir, want) {
					t.Fatalf("zero-sized source closure did not elide its environment; missing %q:\n%s", want, ir)
				}
			}
			module := ownerIRModule(t, ir)
			for _, name := range []string{"zeroclosure.makeValue", "zeroclosure.compareLocal"} {
				function := mustNamedFunction(t, module, name)
				for _, block := range function.BasicBlocks() {
					for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
						if call := inst.IsACallInst(); !call.IsNil() &&
							strings.Contains(call.CalledValue().Name(), ".Alloc") {
							t.Fatalf("zero-sized source closure allocated storage:\n%s", function)
						}
					}
				}
			}

			// A captured pointer value still needs an environment. Its closure
			// header and payload share one scanned allocation. The immutable
			// pointer capture is copied by value from its invocation-local slot.
			function := mustNamedFunction(t, module, "zeroclosure.keepPointer")
			entry := mustNamedFunction(t, module, "zeroclosure.keepPointer$1")
			if entry.ParamsCount() != 1 {
				t.Fatalf("captured pointer entry lost its environment parameter: %s", entry)
			}
			root := func(value llvm.Value) llvm.Value {
				for !value.IsAGetElementPtrInst().IsNil() {
					value = value.Operand(0)
				}
				return value
			}
			var allocation, source llvm.Value
			allocations := 0
			for _, block := range function.BasicBlocks() {
				for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
					if call := inst.IsACallInst(); !call.IsNil() {
						switch {
						case strings.HasSuffix(call.CalledValue().Name(), ".AllocU"):
							allocation = call
							allocations++
						}
					}
				}
			}
			if allocations != 1 {
				t.Fatalf("capture needs one scanned combined header/payload:\n%s", function)
			}
			for _, block := range function.BasicBlocks() {
				for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
					if inst.InstructionOpcode() == llvm.Store && inst.Operand(0) == function.Param(0) {
						source = inst.Operand(1)
					}
				}
			}
			codeStored, envStored, captureStored := false, false, false
			for _, block := range function.BasicBlocks() {
				for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
					if inst.InstructionOpcode() != llvm.Store {
						continue
					}
					value, destination := inst.Operand(0), inst.Operand(1)
					if root(destination) == allocation {
						codeStored = codeStored || value == entry
						envStored = envStored || root(value) == allocation && !value.IsNull()
						captureStored = captureStored || !value.IsALoadInst().IsNil() && value.Operand(0) == source
					}
				}
			}
			if !codeStored || !envStored || source.IsNil() || !captureStored {
				t.Fatalf("captured pointer lost its code, environment or source argument:\n%s", function)
			}
		})
	}
}

func TestUnusedDerefEmitsNilGuard(t *testing.T) {
	const src = `package unusedderef
func LoadArrayElement() {
	var values [2]*int
	_ = *values[1]
}
func LoadPointer(p *int) {
	_ = *p
}
`
	ir := compileWithRewrites(t, src, nil)
	arrayLoad := llvmFunction(t, ir, "unusedderef.LoadArrayElement")
	if !strings.Contains(arrayLoad, "AssertNilDeref") {
		t.Fatalf("unused array-element dereference should retain a nil guard:\n%s", arrayLoad)
	}
	directLoad := llvmFunction(t, ir, "unusedderef.LoadPointer")
	if !strings.Contains(directLoad, "AssertNilDeref") {
		t.Fatalf("unused direct dereference should retain a nil guard:\n%s", directLoad)
	}
}
