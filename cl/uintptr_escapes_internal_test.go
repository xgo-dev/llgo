//go:build !llgo

package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestUintptrEscapesRootsWasmStorage(t *testing.T) {
	const src = `package roots
import "unsafe"
func collect()
//go:uintptrescapes
func marked(p uintptr) { collect(); _ = *(*int)(unsafe.Pointer(p)) }
func integer(p uintptr) { collect() }
func source() unsafe.Pointer
func laterArgument() uintptr
//go:uintptrescapes
func two(uintptr, uintptr)
func caller() { two(uintptr(source()), laterArgument()) }
`
	for _, tc := range []struct {
		profile, goos, triple, rootArray, rootStore string
		pointerSize                                 int
	}{
		{"j32", "js", "wasm32-unknown-emscripten", "[1 x { ptr, i32 }]", "store i64 ", 4},
		{"j64", "js", "wasm64-unknown-emscripten", "[1 x ptr]", "store ptr ", 8},
		{"w32", "wasip1", "wasm32-unknown-wasip1", "[1 x { ptr, i32 }]", "store i64 ", 4},
	} {
		t.Run(tc.profile, func(t *testing.T) {
			ssaPkg, files := buildCallerFrameSSAPackage(t, "roots", src)
			prog := newLLSSAProgForTarget(t, &llssa.Target{
				GOOS: tc.goos, GOARCH: "wasm", LLVMTarget: tc.triple, WasmProfile: tc.profile,
			})
			defer prog.Dispose()
			prog.EnableGCRoots(true)
			if prog.PointerSize() != tc.pointerSize || prog.GoWordSize() != 8 {
				t.Fatalf("physical pointer/Go word sizes = %d/%d, want %d/8", prog.PointerSize(), prog.GoWordSize(), tc.pointerSize)
			}
			pkg, err := NewPackage(prog, ssaPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid root module: %v\n%s", err, pkg.String())
			}
			marked := mustNamedFunction(t, pkg.Module(), "roots.marked")
			if got := marked.Param(0).Type().IntTypeWidth(); got != 64 {
				t.Fatalf("Go uintptr parameter width = %d, want 64", got)
			}
			for _, want := range []string{tc.rootArray, "inttoptr i64", "@llvm_gc_root_chain"} {
				if !strings.Contains(marked.String(), want) {
					t.Fatalf("uintptr root is missing %q:\n%s", want, marked)
				}
			}
			integer := mustNamedFunction(t, pkg.Module(), "roots.integer").String()
			if strings.Contains(integer, "@llvm_gc_root_chain") {
				t.Fatalf("ordinary uintptr became a root:\n%s", integer)
			}
			caller := mustNamedFunction(t, pkg.Module(), "roots.caller").String()
			source := strings.Index(caller, "call ptr @roots.source()")
			later := strings.Index(caller, "call i64 @roots.laterArgument()")
			if source < 0 || later <= source || !strings.Contains(caller[source:later], tc.rootStore) {
				t.Fatalf("source pointer was not published in a Go root slot before the next argument:\n%s", caller)
			}
		})
	}
}

func uintptrEscapesPackage(t *testing.T, src string) *ssa.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "uintptr_escapes.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: importer.Default()}, fset,
		types.NewPackage("uintptr_escapes", "p"), []*ast.File{file}, ssa.InstantiateGenerics|ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func TestUintptrEscapesRoots(t *testing.T) {
	pkg := uintptrEscapesPackage(t, `package p
import "unsafe"
//go:uintptrescapes
func marked(p uintptr, rest ...uintptr) {}
func ordinary(p uintptr) {}
func direct() { marked(uintptr(unsafe.Pointer(new(int))), uintptr(unsafe.Pointer(new(int)))) }
func goroutine() { go marked(uintptr(unsafe.Pointer(new(int)))) }
func deferred() { defer marked(uintptr(unsafe.Pointer(new(int)))) }
//go:uintptrescapes
func forward(p uintptr, rest ...uintptr) { marked(p, rest...) }
func unmarked() { ordinary(uintptr(unsafe.Pointer(new(int)))) }
func integer(p uintptr) { ordinary(p) }
func arithmetic(p unsafe.Pointer) { marked(uintptr(p) + 1) }
type MyInt int
func integerConversion(p unsafe.Pointer) { marked(uintptr(MyInt(uintptr(p)))) }
func narrowConversion(p unsafe.Pointer) { marked(uintptr(uint32(uintptr(p)))) }
func allocatedSlice(p unsafe.Pointer) {
 rest := make([]uintptr, 1)
 rest[0] = uintptr(p)
 marked(0, rest...)
}
func savedSlice(rest []uintptr) { marked(0, rest...) }
func repeated(p unsafe.Pointer) { marked(uintptr(p)); marked(uintptr(p)) }
func merged(p, q unsafe.Pointer, choose bool) {
 value := uintptr(p)
 if choose { value = uintptr(q) }
 marked(value, value)
}
type Word uintptr
//go:uintptrescapes
func named(Word) {}
func convert(p unsafe.Pointer) { named(Word(uintptr(p))) }
`)
	for _, test := range []struct {
		name string
		want int
	}{
		{"marked", 1}, {"direct", 2}, {"goroutine", 1}, {"deferred", 1},
		{"forward", 1}, {"unmarked", 0}, {"integer", 0}, {"arithmetic", 0},
		{"integerConversion", 0}, {"narrowConversion", 0}, {"allocatedSlice", 0},
		{"savedSlice", 0}, {"repeated", 1},
		{"merged", 2}, {"named", 1}, {"convert", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			fn := pkg.Func(test.name)
			roots := uintptrEscapesRoots(fn)
			if len(roots) != test.want {
				t.Fatalf("roots = %v, want %d", roots, test.want)
			}
		})
	}
}

func TestUintptrEscapesHeapAllocation(t *testing.T) {
	pkg := uintptrEscapesPackage(t, `package p
import "unsafe"
//go:uintptrescapes
func marked(p uintptr, rest ...uintptr) {}
func escaping() {
 var local int
 var fields struct { n int }
 var array [2]int
 marked(uintptr(unsafe.Pointer(&local)), uintptr(unsafe.Pointer(&fields.n)), uintptr(unsafe.Pointer(&array[1])))
 go marked(uintptr(unsafe.Pointer(&local)))
 defer marked(uintptr(unsafe.Pointer(&local)))
}
`)
	// The SSA builder conservatively escapes address-taken locals, including
	// field/array bases and variadic arrays. Do not add a second escape pass or
	// turn every ordinary uintptr argument into a pointer to satisfy this.
	allocations := 0
	for _, block := range pkg.Func("escaping").Blocks {
		for _, instr := range block.Instrs {
			if alloc, ok := instr.(*ssa.Alloc); ok {
				allocations++
				if !alloc.Heap {
					t.Errorf("uintptr conversion source/backing array remains on stack: %s", alloc)
				}
			}
		}
	}
	if allocations < 4 {
		t.Fatalf("only %d allocations, want locals and the variadic backing array", allocations)
	}
}

func TestUintptrEscapesMethodWrappers(t *testing.T) {
	pkg := uintptrEscapesPackage(t, `package p
import "unsafe"
type Leaf struct{}
//go:uintptrescapes
func (Leaf) Marked(p uintptr) {}
func (Leaf) Ordinary(p uintptr) {}
type Outer struct { Leaf }
func method() { Outer{}.Marked(uintptr(unsafe.Pointer(new(int)))) }
func expression() { Outer.Marked(Outer{}, uintptr(unsafe.Pointer(new(int)))) }
func bound() { f := Outer{}.Marked; f(uintptr(unsafe.Pointer(new(int)))) }
func unmarkedBound() { f := Outer{}.Ordinary; f(uintptr(unsafe.Pointer(new(int)))) }
func invoked(i interface{ Marked(uintptr) }, p unsafe.Pointer) { i.Marked(uintptr(p)) }
`)
	for _, name := range []string{"method", "expression", "bound"} {
		if roots := uintptrEscapesRoots(pkg.Func(name)); len(roots) != 1 {
			t.Errorf("%s roots = %v, want one conversion source", name, roots)
		}
	}
	if hasUintptrEscapesDirective(nil) {
		t.Fatal("nil callee has directive")
	}
	if roots := uintptrEscapesRoots(pkg.Func("unmarkedBound")); len(roots) != 0 {
		t.Fatalf("ordinary bound method inherited pointer semantics: %v", roots)
	}
	if roots := uintptrEscapesRoots(pkg.Func("invoked")); len(roots) != 0 {
		t.Fatalf("interface call inherited implementation pragma: %v", roots)
	}
}

func TestUintptrEscapesCrossPackageGeneric(t *testing.T) {
	_, pkg := buildCallerFrameSSAProgram(t, "example.com/dep", `package dep
//go:uintptrescapes
func Marked(p uintptr) {}
//go:uintptrescapes
func Generic[T any](p uintptr) {}
func Ordinary(p uintptr) {}
func OrdinaryGeneric[T any](p uintptr) {}
`, "example.com/caller", `package caller
import (
 "unsafe"
 "example.com/dep"
)
func direct() { dep.Marked(uintptr(unsafe.Pointer(new(int)))) }
func generic() { dep.Generic[int](uintptr(unsafe.Pointer(new(int)))) }
func ordinary() { dep.Ordinary(uintptr(unsafe.Pointer(new(int)))) }
func ordinaryGeneric() { dep.OrdinaryGeneric[int](uintptr(unsafe.Pointer(new(int)))) }
`)
	for _, name := range []string{"direct", "generic"} {
		if roots := uintptrEscapesRoots(pkg.Func(name)); len(roots) != 1 {
			t.Errorf("cross-package %s roots = %v, want one conversion source", name, roots)
		}
	}
	for _, name := range []string{"ordinary", "ordinaryGeneric"} {
		if roots := uintptrEscapesRoots(pkg.Func(name)); len(roots) != 0 {
			t.Fatalf("ordinary imported %s inherited pointer semantics: %v", name, roots)
		}
	}
}
