package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/goescape"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestEscapeResolvedInstructionContracts(t *testing.T) {
	const src = `package effects
import "unsafe"
var sink unsafe.Pointer
//go:linkname load llgo.atomicLoad
func load(p *unsafe.Pointer) unsafe.Pointer { panic("not the called body") }
//go:linkname store llgo.atomicStore
func store(p *unsafe.Pointer, v unsafe.Pointer) {}
//go:linkname unordered llgo.atomicStoreUnordered
func unordered(p *unsafe.Pointer, v unsafe.Pointer) {}
//go:linkname exchange llgo.atomicXchg
func exchange(p *unsafe.Pointer, v unsafe.Pointer) unsafe.Pointer { return nil }
//go:linkname compare llgo.atomicCmpXchg
func compare(p *unsafe.Pointer, old, new unsafe.Pointer) (unsafe.Pointer,bool) { return nil,false }
//go:linkname compareOK llgo.atomicCmpXchgOK
func compareOK(p *unsafe.Pointer, old, new unsafe.Pointer) bool {return false}
//go:linkname add llgo.atomicAddReturnNew
func add(p *int64, n int64) int64 {return 0}
//go:linkname ignore llgo.skip
func ignore(any) {panic("not the called body")}
//go:noescape
func borrow(*int)
//go:noescape
func callback(func())
func reader(p unsafe.Pointer) unsafe.Pointer {return load(&p)}
func writer() {x:=1;store(&sink,unsafe.Pointer(&x))}
func unorderedWriter() {x:=1;unordered(&sink,unsafe.Pointer(&x))}
func exchangeWriter() {x:=1;exchange(&sink,unsafe.Pointer(&x))}
func compareWriter() {x:=1;compare(&sink,nil,unsafe.Pointer(&x))}
func compareOKWriter() {x:=1;compareOK(&sink,nil,unsafe.Pointer(&x))}
func adder(x int64)int64 {return add(&x,1)}
func discarded(x int) {ignore(&x)}
func borrowed(x int) {borrow(&x)}
func called(x int) {callback(func(){sink=unsafe.Pointer(&x)})}
`
	ir := compileInterfaceBoxIR(t, src, ssa.InstantiateGenerics, nil, false)
	for _, name := range []string{"reader", "discarded", "borrowed", "adder"} {
		if body := llvmFunction(t, ir, "effects."+name); strings.Contains(body, "runtime.Alloc") {
			t.Errorf("%s retains borrowed storage:\n%s", name, body)
		}
	}
	for _, name := range []string{"writer", "unorderedWriter", "exchangeWriter", "compareWriter", "compareOKWriter", "called"} {
		if body := llvmFunction(t, ir, "effects."+name); !strings.Contains(body, "runtime.AllocZ") {
			t.Errorf("%s lost published pointer:\n%s", name, body)
		}
	}
}

func TestMissingEscapeFactsAreConservative(t *testing.T) {
	var result *goescape.Result
	if decision := result.Decision(&ssa.Alloc{}); !decision.Heap || decision.Reason == "" {
		t.Fatalf("missing facts accepted: %+v", decision)
	}
	ctx := &context{}
	if !ctx.stackStorage(&ssa.Alloc{}) || ctx.stackStorage(&ssa.Alloc{Heap: true}) || ctx.stackStorage(&ssa.MakeClosure{}) {
		t.Fatal("missing analysis must preserve only existing SSA-local allocations")
	}
}

func TestEscapeTargetStorageConstraints(t *testing.T) {
	const src = `package placement
func use(p *[131072]byte) byte { return p[0] }
func large() byte { return use(new([131072]byte)) }
func local(p *int) *int { var values [2]*int; values[0] = p; return values[0] }
func borrow(p **int) *int { return *p }
func promoted(p *int) *int { return borrow(&p) }
`
	target := &llssa.Target{GOOS: "wasip1", GOARCH: "wasm"}
	ir := compileInterfaceBoxIR(t, src, ssa.InstantiateGenerics, target, true)
	for _, name := range []string{"large", "promoted"} {
		if body := llvmFunction(t, ir, "placement."+name); !strings.Contains(body, "runtime.AllocZ") {
			t.Errorf("%s must retain heap storage:\n%s", name, body)
		}
	}
	if body := llvmFunction(t, ir, "placement.local"); strings.Contains(body, "runtime.AllocZ") {
		t.Errorf("SSA-local pointer fields should retain their existing root plan:\n%s", body)
	}
}

func TestEscapeCABIExistingStackStorage(t *testing.T) {
	const src = `package C
var Sink int32
func Reset_Handler() { values := [3]int32{9,10,11}; Sink = values[1] }
`
	ir := compileInterfaceBoxIR(t, src, ssa.GlobalDebug, nil, false)
	if body := llvmFunction(t, ir, "Reset_Handler"); strings.Contains(body, "runtime.Alloc") {
		t.Fatalf("C-ABI locals introduced a Go runtime dependency:\n%s", body)
	}
}

func TestEscapeLayoutDoesNotPopulateBackendTypes(t *testing.T) {
	// Source patches may replace a named type with the same layout and fewer
	// methods. Caching the original type in the coordinator before lowering
	// caused shared/archive builds to emit nonexistent runtime methods.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "layout.go", `package layout
type T struct { X int }
func (T) OriginalOnly() {}
func f() *T { return new(T) }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(&types.Config{}, fset, types.NewPackage("layout", "layout"), []*ast.File{file}, ssa.InstantiateGenerics)
	if err != nil {
		t.Fatal(err)
	}
	prog := newLLSSAProgForTarget(t, nil)
	defer prog.Dispose()
	PrepareEscapeAnalysis(prog, pkg.Prog, nil)
	original := pkg.Pkg.Scope().Lookup("T").Type().(*types.Named)
	replacement := types.NewNamed(types.NewTypeName(token.NoPos, pkg.Pkg, "T", nil), original.Underlying(), nil)
	if got := prog.Type(replacement, llssa.GoDecl).RawType().(*types.Named); got.NumMethods() != 0 {
		t.Fatal("layout queries cached the original type's methods in the backend")
	}
}
