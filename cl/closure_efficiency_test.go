package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

const closureEfficiencySource = `package closureperf
var Sink func(int64) int64
var Evaluations int64
//go:noinline
func argument(value int64) int64 { Evaluations++; return value }
func Known(base, value int64) int64 {
	fn := func(delta int64) int64 { return base + delta }
	base++
	Sink = fn
	return fn(argument(value))
}
//go:noinline
func Dynamic(fn func(int64) int64, value int64) int64 { return fn(value) }
type Handler struct { Fn func() int64 }
func (h *Handler) Set(fn func() int64) { h.Fn = fn }
func Schedule(h *Handler) { defer h.Set(func() int64 { return 17 }) }
`

func closureEfficiencyCalls(function llvm.Value) (allocations []uint64, indirect int) {
	for _, block := range function.BasicBlocks() {
		for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			call := inst.IsACallInst()
			if call.IsNil() {
				continue
			}
			callee := call.CalledValue()
			if callee.IsAFunction().IsNil() {
				indirect++
			} else if strings.HasSuffix(callee.Name(), ".AllocU") {
				allocations = append(allocations, call.Operand(0).ZExtValue())
			}
		}
	}
	return
}

func testClosureEfficiency(t *testing.T, check func(*testing.T, llvm.Module, llssa.Program)) {
	t.Helper()
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ssapkg, _, files := buildGoSSAPkg(t, closureEfficiencySource)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				t.Fatal(err)
			}
			module := pkg.Module()
			module.SetDataLayout(prog.DataLayout())
			module.SetTarget(target.Spec().Triple)
			for _, stage := range []string{"O0", "O2"} {
				t.Run(stage, func(t *testing.T) {
					if stage == "O2" {
						options := llvm.NewPassBuilderOptions()
						defer options.Dispose()
						options.SetVerifyEach(true)
						if err := module.RunPasses("default<O2>", prog.TargetMachine(), options); err != nil {
							t.Fatal(err)
						}
					}
					t.Logf("%s module:\n%s", stage, module.String())
					check(t, module, prog)
					if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
						t.Fatal(err)
					}
				})
			}
			assembly, err := prog.TargetMachine().EmitToMemoryBuffer(module, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			assembly.Dispose()
		})
	}
}

func TestCapturedLiteralClosureKeepsKnownEntry(t *testing.T) {
	testClosureEfficiency(t, func(t *testing.T, module llvm.Module, prog llssa.Program) {
		known := mustNamedFunction(t, module, "closureperf.Known")
		allocations, indirect := closureEfficiencyCalls(known)
		if indirect != 0 {
			t.Errorf("known captured entry acquired %d indirect/barrier calls", indirect)
		}
		if len(allocations) != 1 {
			t.Errorf("captured header/payload allocations = %v, want exactly one", allocations)
		}
		dynamic := mustNamedFunction(t, module, "closureperf.Dynamic")
		if _, dynamicCalls := closureEfficiencyCalls(dynamic); dynamicCalls == 0 {
			t.Error("unknown function parameter lost its dynamic dispatch")
		}
	})
}

func TestDeferredFunctionArgumentAvoidsLateHeaderAllocation(t *testing.T) {
	testClosureEfficiency(t, func(t *testing.T, module llvm.Module, prog llssa.Program) {
		function := mustNamedFunction(t, module, "closureperf.Schedule")
		allocations, _ := closureEfficiencyCalls(function)
		for _, size := range allocations {
			if size == 2*uint64(prog.GoWordSize()) {
				t.Errorf("deferred static function was boxed again during unwind: allocations %v", allocations)
			}
		}
	})
}
