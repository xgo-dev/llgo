//go:build !llgo

package build

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/cabi"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/lto"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llvm"
)

func TestHeapToStackFailureStopsABILowering(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	target := &llvmgen.Target{GOOS: "linux", GOARCH: "amd64"}
	prog := llvmgen.NewProgram(target)
	defer prog.Dispose()
	const pkgPath = "example.com/allocfailure"
	lpkg := prog.NewPackage("allocfailure", pkgPath)
	mod := lpkg.Module()
	lc := mod.Context()
	result := lc.StructType([]llvm.Type{lc.Int64Type(), lc.Int64Type(), lc.Int64Type()}, false)
	fn := llvm.AddFunction(mod, "result", llvm.FunctionType(result, nil, false))
	b := lc.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(lc.AddBasicBlock(fn, "entry"))
	b.CreateRet(llvm.ConstNull(result))
	conf := NewDefaultConf(ModeGen)
	conf.Goos, conf.Goarch, conf.OptLevel = target.GOOS, target.GOARCH, optlevel.O2
	ctx := &context{
		prog: prog, buildConf: conf, mode: ModeGen,
		cTransformer: cabi.NewTransformer(prog, target.Spec().Triple, "", true),
	}
	want := errors.New("heap-to-stack failed")
	calls := 0
	old := runHeapToStack
	t.Cleanup(func() { runHeapToStack = old })
	runHeapToStack = func(got llvm.Module, _ llvm.TargetMachine) error {
		calls++
		if got != mod {
			t.Fatal("heap-to-stack received a different module")
		}
		return want
	}
	// Disabled passes and O0 must not enter the optimizer.
	if err := optimizeHeapAllocations(ctx, pkgPath, mod); err != nil {
		t.Fatal(err)
	}
	ctx.passOpt, conf.OptLevel = true, optlevel.O0
	if err := optimizeHeapAllocations(ctx, pkgPath, mod); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("disabled heap-to-stack ran")
	}
	conf.OptLevel = optlevel.O2
	pkg := &aPackage{Package: &packages.Package{PkgPath: pkgPath}, LPkg: lpkg}
	err := compilePackageModule(ctx, pkg, nil, false)
	if !errors.Is(err, want) || !strings.Contains(err.Error(), pkgPath) {
		t.Fatalf("compiler lost the optimizer error or package context: %v", err)
	}
	if calls != 1 {
		t.Fatalf("optimizer called %d times, want 1", calls)
	}
	if mod.NamedFunction("result").GlobalValueType().ReturnType() != result {
		t.Fatal("C ABI lowering continued after the optimizer failed")
	}
}

// ModeGen keeps LLVM passes disabled so IR fixtures can inspect their output.
// Run allocation optimization at its production boundary, then let Build
// perform ABI lowering before the test runs the ordinary LLVM pipeline.
func optimizeTestAllocationsBeforeABI(t *testing.T, conf *Config, before func(llvm.Module)) {
	t.Helper()
	conf.ModuleHook = func(pkg Package) {
		if pkg.Name != "main" {
			return
		}
		mod := pkg.LPkg.Module()
		if before != nil {
			before(mod)
		}
		ctx := &context{prog: pkg.LPkg.Prog, buildConf: conf, passOpt: true}
		// Build repeats this lowering after ModuleHook. These fixtures have no
		// oversized aggregates, so the repeated pass does not change their IR.
		lowerLargeAggregates(ctx.prog, mod)
		if err := optimizeHeapAllocations(ctx, pkg.PkgPath, mod); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPostInlineHeapToStackGuardsAndErrors(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	target := &llvmgen.Target{GOOS: "linux", GOARCH: "amd64"}
	prog := llvmgen.NewProgram(target)
	defer prog.Dispose()
	const pkgPath = "example.com/postinline"
	lpkg := prog.NewPackage("postinline", pkgPath)
	mod := lpkg.Module()
	lc := mod.Context()
	fn := llvm.AddFunction(mod, "noop", llvm.FunctionType(lc.VoidType(), nil, false))
	b := lc.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(lc.AddBasicBlock(fn, "entry"))
	b.CreateRetVoid()

	conf := NewDefaultConf(ModeGen)
	conf.Goos, conf.Goarch, conf.OptLevel = target.GOOS, target.GOARCH, optlevel.O2
	ctx := &context{prog: prog, buildConf: conf, mode: ModeGen}

	// Early return without invoking Attributor when no allockind calls exist.
	ctx.passOpt = true
	if err := optimizePostInlineHeapAllocations(ctx, pkgPath, mod); err != nil {
		t.Fatalf("unexpected error on allocation-free module: %v", err)
	}

	calls := 0
	want := errors.New("post-inline failed")
	old := runPostInlineHeapToStack
	t.Cleanup(func() { runPostInlineHeapToStack = old })
	runPostInlineHeapToStack = func(got llvm.Module, _ llvm.TargetMachine) ([]llvm.Value, error) {
		calls++
		if got != mod {
			t.Fatal("post-inline heap-to-stack received a different module")
		}
		return nil, want
	}

	// Disabled passOpt, O0, and runtime package must skip post-inline analysis.
	ctx.passOpt = false
	if err := optimizePostInlineHeapAllocations(ctx, pkgPath, mod); err != nil {
		t.Fatal(err)
	}
	ctx.passOpt, conf.OptLevel = true, optlevel.O0
	if err := optimizePostInlineHeapAllocations(ctx, pkgPath, mod); err != nil {
		t.Fatal(err)
	}
	conf.OptLevel = optlevel.O2
	if err := optimizePostInlineHeapAllocations(ctx, llvmgen.PkgRuntime, mod); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("post-inline heap-to-stack ran %d times when guarded, want 0", calls)
	}

	err := optimizePostInlineHeapAllocations(ctx, pkgPath, mod)
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "post-inline") || !strings.Contains(err.Error(), pkgPath) {
		t.Fatalf("unexpected wrapped error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("post-inline heap-to-stack called %d times, want 1", calls)
	}
}

func runHeapToStackFixture(t *testing.T, moduleName, binaryPrefix, source, wantOutput string) {
	t.Helper()
	if testing.Short() {
		t.Skip("builds and runs heap-to-stack execution fixture")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	for name, text := range map[string]string{
		"go.mod": "module " + moduleName + "\n\ngo 1.27.0\n", "main.go": source,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []lto.Mode{lto.Off, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.OptLevel, conf.LTO = optlevel.O2, mode
			conf.OutFile = filepath.Join(dir, binaryPrefix+"-"+mode.String())
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
				t.Fatal(err)
			}
			out, err := exec.CommandContext(t.Context(), conf.OutFile).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != wantOutput {
				t.Fatalf("%s execution: %v\n%s", binaryPrefix, err, out)
			}
		})
	}
}

func TestHeapToStackABIExecution(t *testing.T) {
	const source = `package main
type triple struct { a, b, c uint64 }
//go:noinline
func read(p *uint64) triple { return triple{*p, *p + 1, *p + 2} }
//go:noinline
func sum(v triple) uint64 { return v.a + v.b + v.c }
//go:noinline
func local(n uint64) triple {
 p := new(uint64)
 *p = n
 return read(p)
}
func main() {
 for n := uint64(1); n < 100; n++ {
  v := local(n)
  if v != (triple{n, n+1, n+2}) || sum(v) != 3*n+3 { panic("aggregate ABI") }
 }
 println("abi-ok")
}
`
	runHeapToStackFixture(t, "allocabi", "abi", source, "abi-ok")
}

func TestPostInlineHeapToStack(t *testing.T) {
	const source = `package main

import "runtime"

type field struct {
	name string
	val  int
}

type node struct {
	option bool
	elem   int
	cases  []field
}

func cases(n *node) []field {
	if n.option {
		return []field{{}, {val: n.elem}}
	}
	return n.cases
}

//go:noinline
func lookup(n *node, idx int) int {
	return cases(n)[idx].val
}

//go:noinline
func walk(n *node, depth int) int {
	if depth <= 0 {
		return 0
	}
	if n.option {
		return cases(n)[1].val + walk(n, depth-1)
	}
	return 0
}

func main() {
	n := &node{option: true, elem: 7}
	// Warm up before measuring steady-state allocations.
	if lookup(n, 1) != 7 || walk(n, 4) != 28 {
		panic("unexpected result")
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	sum := 0
	for i := 0; i < 4096; i++ {
		sum += lookup(n, 1)
		sum += walk(n, 2)
	}
	runtime.ReadMemStats(&after)
	if sum != 4096*21 {
		panic("bad sum")
	}
	if after.Mallocs != before.Mallocs {
		panic("unexpected heap allocation after inlining")
	}
	println("post-inline-ok")
}
`
	runHeapToStackFixture(t, "postinlineh2s", "h2s", source, "post-inline-ok")
}
