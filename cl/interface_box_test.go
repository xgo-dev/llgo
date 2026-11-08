package cl

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa/ssautil"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

const interfaceBoxDecl = `package ifacebox
type F func() int
type Pair struct { A, B uint64 }
var saved any
func extract(x any) F { return x.(F) }
func forward(x any) F { return extract(x) }
func keep(x any) { saved = x }
func mixed(x, y any) F { saved = y; return x.(F) }
func unknown(any)
func recursive(x any, n int) F { if n > 0 { return recursive(x, n-1) }; return x.(F) }
`

func TestCompileBorrowedInterfaceBoxes(t *testing.T) {
	const src = interfaceBoxDecl + `
func direct(f F) F { return any(f).(F) }
func helper(f F) F { return extract(f) }
func forwarded(f F) F { return forward(f) }
func commaOK(f F) (F, bool) { v, ok := any(f).(F); return v, ok }
func nilComparison(f F) bool { return any(f) != nil }
func integer(x uint64) uint64 { return any(x).(uint64) }
func pair(x Pair) Pair { return any(x).(Pair) }
func branch(f F, choose bool) F {
 x := any(f)
 if choose { return x.(F) }
 return extract(x)
}
func phi(f F, other any, choose bool) F {
 x := any(f)
 if choose { x = other }
 return extract(x)
}
func separate(f F, other any) F { return mixed(f, other) }
`
	for _, target := range []llssa.Target{
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		for _, mode := range []ssa.BuilderMode{0, ssa.GlobalDebug} {
			t.Run(fmt.Sprintf("%s/%s/%v", target.GOOS, target.GOARCH, mode), func(t *testing.T) {
				ir := compileInterfaceBoxIR(t, src, mode, &target, false)
				for _, name := range []string{"direct", "helper", "forwarded", "commaOK", "nilComparison", "integer", "pair", "branch", "phi", "separate"} {
					body := llvmFunction(t, ir, "ifacebox."+name)
					if strings.Contains(body, "runtime.Alloc") || !strings.Contains(body, "alloca ") {
						t.Errorf("%s should use stack box storage:\n%s", name, body)
					}
				}
			})
		}
	}
}

func TestCompileInterfaceBoxPlacement(t *testing.T) {
	for _, tc := range []struct{ name, decl string }{
		{"return", `func box(f F) any { return f }`},
		{"global", `func box(f F) { saved = f }`},
		{"helper", `func box(f F) { keep(f) }`},
		{"second argument", `func box(f F) F { x := any(f); return mixed(x, x) }`},
		{"unknown", `func box(f F) { unknown(f) }`},
		{"indirect", `func box(f F, use func(any)) { use(f) }`},
		{"defer", `func box(f F) { defer extract(f) }`},
		{"go", `func box(f F) { go extract(f) }`},
		{"closure", `func box(f F) func() F { x := any(f); return func() F { return x.(F) } }`},
		{"recursive", `func box(f F) F { return recursive(f, 1) }`},
		{"interface assertion", `func box(f F) any { return any(f).(interface{}) }`},
		{"loop", `func box(f F, n int) int { sum := 0; for i:=0;i<n;i++ { sum += extract(f)() }; return sum }`},
		{"recover", `func box(f F) F { defer func() { recover() }(); return extract(f) }`},
		{"large", `func box(x [16385]uint64) [16385]uint64 { return any(x).([16385]uint64) }`},
		{"budget", "func box(f F) { x := any(f);" + strings.Repeat("extract(x);", 1025) + "}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ir := compileInterfaceBoxIR(t, interfaceBoxDecl+tc.decl, ssa.InstantiateGenerics, nil, false)
			body := llvmFunction(t, ir, "ifacebox.box")
			// These cases exceeded the old use walker's scope, but their boxes
			// stay within the current invocation/iteration in the full graph.
			stack := tc.name == "defer" || tc.name == "recursive" || tc.name == "loop" || tc.name == "recover" || tc.name == "budget"
			if heap := strings.Contains(body, "runtime.Alloc"); heap == stack {
				t.Fatalf("box heap=%v, want %v:\n%s", heap, !stack, body)
			}
		})
	}
}

func compileInterfaceBoxIR(t *testing.T, src string, mode ssa.BuilderMode, target *llssa.Target, roots bool) string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "ifacebox.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: packages.NewImporter(fset)}, fset, types.NewPackage(file.Name.Name, file.Name.Name), files, mode)
	if err != nil {
		t.Fatal(err)
	}
	// Source import keeps the runtime's private ABI aliases needed for boxing.
	prog := newLLSSAProgForTarget(t, target)
	defer prog.Dispose()
	if target == nil {
		prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	}
	prog.EnableGCRoots(roots)
	compiled, _, err := NewPackageEx(prog, nil, nil, pkg, files)
	if err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(compiled.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	return compiled.String()
}

func TestInterfaceBoxExplicitRoots(t *testing.T) {
	ir := compileInterfaceBoxIR(t, interfaceBoxDecl+`func box(f F) F { return extract(f) }`, ssa.InstantiateGenerics, &llssa.Target{GOOS: "wasip1", GOARCH: "wasm"}, true)
	body := llvmFunction(t, ir, "ifacebox.box")
	if strings.Contains(body, "runtime.AllocU") || !strings.Contains(ir, "__llgo_gcroot_stack_pointer_") {
		t.Fatalf("borrowed box must use typed stack roots:\n%s", body)
	}
	for _, want := range []string{"gcroot.init", "llvm_gc_root_chain"} {
		if !strings.Contains(body, want) {
			t.Fatalf("explicit-root box lost %s:\n%s", want, body)
		}
	}
}
