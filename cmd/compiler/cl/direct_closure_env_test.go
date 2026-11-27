package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
	"github.com/xgo-dev/llgo/types"
	"github.com/xgo-dev/llvm"
)

func TestDirectSinglePtrClosureEnvIR(t *testing.T) {
	const source = `package directenv

import "unsafe"

type box struct{ v int }

func (b *box) add(d int) int {
	if b == nil {
		return -d
	}
	return b.v + d
}

func makePtr(b *box) func(int) int {
	return func(d int) int {
		if b == nil {
			return -d
		}
		return b.v + d
	}
}

func makeBound(b *box) func(int) int {
	return b.add
}

func makeMap(m map[int]int) func(int) int {
	return func(k int) int { return m[k] }
}

func makeUnsafe(p unsafe.Pointer) func() uintptr {
	return func() uintptr { return uintptr(p) }
}

func makeTwo(a, b *box) func() int {
	return func() int { return a.v + b.v }
}

func makeInt(x int) func() int {
	return func() int { return x }
}
`
	for _, tc := range []struct {
		target     llvmgen.Target
		wantDirect bool
	}{
		{llvmgen.Target{GOOS: "darwin", GOARCH: "arm64"}, true},
		{llvmgen.Target{GOOS: "linux", GOARCH: "amd64"}, true},
		{llvmgen.Target{GOOS: "windows", GOARCH: "arm64"}, true},
		{llvmgen.Target{GOOS: "windows", GOARCH: "386"}, true},
		{llvmgen.Target{GOOS: "wasip1", GOARCH: "wasm"}, false},
		{llvmgen.Target{GOOS: "js", GOARCH: "wasm"}, false},
	} {
		t.Run(tc.target.GOOS+"/"+tc.target.GOARCH, func(t *testing.T) {
			ir := compileInterfaceBoxIR(t, source, ssa.InstantiateGenerics, &tc.target, tc.target.GOARCH == "wasm")
			for _, fnName := range []string{"directenv.makePtr", "directenv.makeBound", "directenv.makeMap", "directenv.makeUnsafe"} {
				fnIR := llvmFunction(t, ir, fnName)
				hasAlloc := strings.Contains(fnIR, "AllocU")
				if tc.wantDirect && hasAlloc {
					t.Fatalf("%s unexpectedly allocated closure env on hidden-env target:\n%s", fnName, fnIR)
				}
				if !tc.wantDirect && !hasAlloc {
					t.Fatalf("%s omitted boxed closure env on explicit-env target:\n%s", fnName, fnIR)
				}
			}
			// Multi-capture and non-pointer captures must still box their env.
			for _, fnName := range []string{"directenv.makeTwo", "directenv.makeInt"} {
				fnIR := llvmFunction(t, ir, fnName)
				if !strings.Contains(fnIR, "AllocU") && !strings.Contains(fnIR, "AllocAtomicU") {
					t.Fatalf("%s unexpectedly elided boxed closure env:\n%s", fnName, fnIR)
				}
			}
		})
	}
}

func TestDirectSinglePtrClosureEnvDebugInfo(t *testing.T) {
	const source = `package directenvdbg

type box struct{ v int }

func makeLoop(b *box) func(int) int {
	return func(n int) int {
		sum := 0
		for i := 0; i < n; i++ {
			if b != nil {
				sum += b.v + i
			}
		}
		return sum
	}
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "direct_env_dbg.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	ssaPkg, _, err := ssautil.BuildPackage(
		&types.Config{},
		fset,
		types.NewPackage("directenvdbg", "directenvdbg"),
		[]*ast.File{file},
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics|ssa.GlobalDebug,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, level := range []optlevel.Level{optlevel.O0, optlevel.O2} {
		t.Run(level.String(), func(t *testing.T) {
			prog := newLLSSAProgForTarget(t, &llvmgen.Target{
				GOOS:     "darwin",
				GOARCH:   "arm64",
				OptLevel: level,
			})
			defer prog.Dispose()
			pkg, _, err := newPackageEx(prog, nil, nil, nil, ssaPkg, []*ast.File{file}, nil, false, Options{
				Debug:        true,
				DebugSymbols: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("debug module invalid: %v\n%s", err, pkg.Module().String())
			}
			fn := pkg.Module().NamedFunction("directenvdbg.makeLoop$1")
			if fn.IsNil() {
				t.Fatalf("missing directenvdbg.makeLoop$1:\n%s", pkg.Module().String())
			}
			fnIR := fn.String()
			if strings.Contains(fnIR, "alloca ptr") {
				t.Fatalf("direct capture-by-value debug ref emitted unexpected pointer alloca:\n%s", fnIR)
			}
			if !strings.Contains(fnIR, "#dbg_value(ptr %0,") || strings.Count(fnIR, "#dbg_value(ptr %") < 2 {
				t.Fatalf("expected #dbg_value for both $env and decoded captured b:\n%s", fnIR)
			}
		})
	}

	if isDirectClosurePointerType(nil) {
		t.Fatal("isDirectClosurePointerType(nil) = true, want false")
	}
	var emptyCtx context
	if emptyCtx.canUseDirectSinglePtrClosureEnv(nil) {
		t.Fatal("canUseDirectSinglePtrClosureEnv(nil) = true, want false")
	}
	if !emptyCtx.directCaptureByValueUsesValid(&ssa.FreeVar{}) {
		t.Fatal("directCaptureByValueUsesValid(empty FreeVar) = false, want true")
	}
	// Verify non-MUL and non-CaptureByValue-forwarding uses reject direct env.
	fv := &ssa.FreeVar{}
	*fv.Referrers() = []ssa.Instruction{&ssa.UnOp{Op: token.SUB, X: fv}}
	if emptyCtx.directCaptureByValueUsesValid(fv) {
		t.Fatal("non-MUL UnOp unexpectedly accepted")
	}
	*fv.Referrers() = []ssa.Instruction{&ssa.Return{}}
	if emptyCtx.directCaptureByValueUsesValid(fv) {
		t.Fatal("arbitrary referrer unexpectedly accepted")
	}
	*fv.Referrers() = []ssa.Instruction{&ssa.MakeClosure{Fn: &ssa.Builtin{}, Bindings: []ssa.Value{fv}}}
	if emptyCtx.directCaptureByValueUsesValid(fv) {
		t.Fatal("non-Function MakeClosure unexpectedly accepted")
	}
	*fv.Referrers() = []ssa.Instruction{&ssa.MakeClosure{Fn: &ssa.Function{}, Bindings: []ssa.Value{fv}}}
	if emptyCtx.directCaptureByValueUsesValid(fv) {
		t.Fatal("MakeClosure without matching CaptureByValue freevar unexpectedly accepted")
	}
}
