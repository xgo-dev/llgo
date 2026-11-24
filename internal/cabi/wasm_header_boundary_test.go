package cabi_test

import (
	"strings"
	"testing"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/heap2stack"
	ssa "github.com/xgo-dev/llgo/internal/llvmgen"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

func TestWasmBoundaryHeaderRangeIRSize(t *testing.T) {
	conf := build.NewDefaultConf(build.ModeGen)
	conf.Target = "wasi"
	// The GC root pass independently publishes a root for each string.
	// Isolate the code emitted for ABI normalization in this size budget.
	conf.Tags = "nogc"
	conf.ModuleHook = func(pkg build.Package) {
		if pkg.Name != "main" {
			return
		}
		// Count the call sites and generated loops together before any ABI
		// or optimizer pass can hide per-element instruction expansion.
		n := 0
		for fn := pkg.LPkg.Module().FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
			if fn.Name() != "ReadLargeResult" && fn.Name() != "main.readLargeParam" && fn.Name() != "ReadNestedParam" &&
				!strings.HasPrefix(fn.Name(), "__llgo_clear_header_range") {
				continue
			}
			for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
				for v := bb.FirstInstruction(); !v.IsNil(); v = llvm.NextInstruction(v) {
					n++
				}
			}
		}
		if n == 0 || n > 180 {
			t.Fatalf("boundary readers and range helpers generated %d instructions", n)
		}
	}
	pkgs, err := build.Do([]string{"./_testdata/wasmheaderboundary/main.go"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	pkgs[0].LPkg.Prog.Dispose()
}

func TestWasmSparseHeaderBoundary(t *testing.T) {
	runWasmHeaderConfigs(t, checkWasmSparseHeaderBoundary)
}

func checkWasmSparseHeaderBoundary(t *testing.T, tags string, level optlevel.Level) {
	conf := build.NewDefaultConf(build.ModeGen)
	conf.Target, conf.Tags, conf.OptLevel = "wasi", tags, level
	if level != optlevel.O0 {
		// ModeGen skips optimization. Keep allocation optimization before
		// C ABI and copy lowering, as in a production build.
		conf.ModuleHook = func(pkg build.Package) {
			if pkg.Name != "main" {
				return
			}
			prog, mod := pkg.LPkg.Prog, pkg.LPkg.Module()
			llabi.LowerLargeAggregates(prog.TargetData(), mod, llabi.AggregateLoweringConfig{
				GoWordSize: prog.GoWordSize(), GCRoots: prog.GCRootsEnabled(), Wasm: true,
			})
			if err := heap2stack.Run(mod, prog.TargetMachine()); err != nil {
				t.Fatal(err)
			}
		}
	}
	pkgs, err := build.Do([]string{"./_testdata/wasmheaderboundary/sparse/main.go"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	mod := pkg.Module()
	functions := []struct{ name, foreign string }{
		{"CopySparseParam", ""}, {"CopySparseResult", "sparseResult"},
		{"CopySparseArrayParam", ""}, {"CopySparseArrayResult", "sparseArrayResult"},
		{"CopyWrappedParam", ""}, {"CopyWrappedResult", "wrappedResult"},
	}
	for _, tc := range functions {
		fn := mod.NamedFunction(tc.name)
		var storage llvm.Value
		cleared, copied := false, false
		if tc.foreign == "" {
			if fn.GetEnumAttributeAtIndex(2, llvm.AttributeKindID("byval")).IsNil() {
				t.Fatal("sparse parameter lacks private byval storage")
			}
			storage = fn.Param(1)
		}
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for v := bb.FirstInstruction(); !v.IsNil(); v = llvm.NextInstruction(v) {
				if v.IsACallInst().IsNil() {
					continue
				}
				switch callee := v.CalledValue().Name(); {
				case callee == tc.foreign:
					storage = v.Operand(0)
				case strings.HasPrefix(callee, "__llgo_clear_header_range") && v.Operand(0) == storage:
					cleared = true
				case strings.HasPrefix(callee, "llvm.memcpy"), strings.HasPrefix(callee, "llvm.memmove"):
					copied = copied || v.Operand(0) == fn.Param(0)
				}
			}
		}
		if storage.IsNil() || !cleared || !copied {
			t.Fatalf("%s must clear headers in its private ABI storage and copy the payload:\n%s", tc.name, fn)
		}
	}
	optimizeWasmModule(t, pkg, level)
	text := wasmAssembly(t, pkg)
	for _, tc := range functions {
		body := wasmAssemblyBody(t, text, tc.name)
		if lines := strings.Count(body, "\n"); lines > 512 {
			t.Errorf("%s expanded a sparse header copy to %d assembly lines", tc.name, lines)
		}
	}
}

func TestWasmBoundaryHeaderStorage(t *testing.T) {
	conf := build.NewDefaultConf(build.ModeGen)
	conf.Target = "wasi"
	pkgs, err := build.Do([]string{"./_testdata/wasmheaderboundary/main.go"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	mod := pkg.Module()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.readLargeParam", "ReadNestedParam"} {
		fn := mod.NamedFunction(name)
		var storage llvm.Value
		if !fn.IsNil() {
			for i, param := range fn.Params() {
				if !fn.GetEnumAttributeAtIndex(i+1, llvm.AttributeKindID("byval")).IsNil() {
					storage = param
					break
				}
			}
		}
		if storage.IsNil() {
			t.Fatalf("%s lacks a private byval argument", name)
		}
		var clears int
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for v := bb.FirstInstruction(); !v.IsNil(); v = llvm.NextInstruction(v) {
				if !v.IsACallInst().IsNil() && strings.HasPrefix(v.CalledValue().Name(), "__llgo_clear_header_range") {
					if v.Operand(0) != storage {
						t.Fatalf("%s normalized an extra copy of its byval argument", name)
					}
					clears++
				}
			}
		}
		if clears != 1 {
			t.Fatalf("%s normalized its byval argument %d times", name, clears)
		}
	}
	discard := mod.NamedFunction("DiscardResult")
	if body := discard.String(); !strings.Contains(body, "@heapResult(") ||
		strings.Contains(body, "__llgo_clear_header_range") || strings.Contains(body, "runtime.AllocU") {
		t.Fatalf("discarded result allocated or cleared headers:\n%s", body)
	}
	for _, tc := range []struct {
		read, foreign string
		heap          bool
	}{
		{"ReadStackResult", "stackResult", false},
		{"ReadHeapResult", "heapResult", true},
		{"ReadLargeResult", "largeResult", true},
	} {
		fn := mod.NamedFunction(tc.read)
		var result, normalized llvm.Value
		resultAllocs := 0
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for v := bb.FirstInstruction(); !v.IsNil(); v = llvm.NextInstruction(v) {
				if v.IsACallInst().IsNil() {
					continue
				}
				switch name := v.CalledValue().Name(); {
				case name == ssa.PkgRuntime+".AllocU":
					resultAllocs++
				case name == tc.foreign:
					if v.GetCallSiteEnumAttribute(1, llvm.AttributeKindID("sret")).IsNil() {
						t.Fatalf("%s lacks sret", tc.foreign)
					}
					result = v.Operand(0)
				case strings.HasPrefix(name, "__llgo_clear_header_range"):
					normalized = v.Operand(0)
				}
			}
		}
		if result.IsNil() || result != normalized {
			t.Fatalf("%s did not normalize its own sret storage:\n%s", tc.read, fn)
		}
		if tc.heap {
			if resultAllocs != 1 {
				t.Fatalf("%s allocated %d scanned buffers for one native result", tc.read, resultAllocs)
			}
			if result.IsACallInst().IsNil() || result.CalledValue().Name() != ssa.PkgRuntime+".AllocU" {
				t.Fatalf("%s result is outside scanned storage:\n%s", tc.read, fn)
			}
		} else if result.IsAAllocaInst().IsNil() {
			t.Fatalf("%s should use a stack slot:\n%s", tc.read, fn)
		}
	}
}
