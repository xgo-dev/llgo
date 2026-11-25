package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestDirectSinglePtrClosureEnvIR(t *testing.T) {
	const source = `package directenv

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

func makeTwo(a, b *box) func() int {
	return func() int { return a.v + b.v }
}

func makeInt(x int) func() int {
	return func() int { return x }
}
`
	for _, tc := range []struct {
		target     llssa.Target
		wantDirect bool
	}{
		{llssa.Target{GOOS: "darwin", GOARCH: "arm64"}, true},
		{llssa.Target{GOOS: "linux", GOARCH: "amd64"}, true},
		{llssa.Target{GOOS: "windows", GOARCH: "arm64"}, true},
		{llssa.Target{GOOS: "wasip1", GOARCH: "wasm"}, false},
		{llssa.Target{GOOS: "linux", GOARCH: "mips64"}, false},
	} {
		t.Run(tc.target.GOOS+"/"+tc.target.GOARCH, func(t *testing.T) {
			ir := compileInterfaceBoxIR(t, source, ssa.InstantiateGenerics, &tc.target, tc.target.GOARCH == "wasm")
			for _, fnName := range []string{"directenv.makePtr", "directenv.makeBound", "directenv.makeMap"} {
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
			sum += b.v + i
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
		&types.Config{Importer: importer.Default()},
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
			prog := newLLSSAProgForTarget(t, &llssa.Target{
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
			if strings.Count(fnIR, "#dbg_value(ptr %0,") < 2 {
				t.Fatalf("expected #dbg_value(ptr %%0, ...) for both $env and captured b:\n%s", fnIR)
			}
		})
	}
}
