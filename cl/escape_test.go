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

func TestEscapeTransientStorageTargets(t *testing.T) {
	const src = `package transient
func bytesUse(b []byte)int{return int(b[0])+len(b)}
func stringUse(s string)int{return int(s[0])+len(s)}
func fixed(x byte)int{b:=make([]byte,16);b[0]=x;return bytesUse(b)}
func dynamic(n int)int{b:=make([]byte,n);return bytesUse(b)}
func wide(n int64)int{b:=make([]byte,n);return bytesUse(b)}
func returning(n int)[]byte{return make([]byte,n)}
func fromBytes(b []byte)int{return stringUse(string(b))}
func toBytes(s string)int{b:=[]byte(s);b[0]++;return bytesUse(b)}
func readOnly(s string)int{return bytesUse([]byte(s))}
func compare(b []byte,s string)bool{return string(b)==s}
func lookup(m map[string]int,b []byte)int{return m[string(b)]}
func fromRunes(r []rune)int{return stringUse(string(r))}
func toRunes(s string)int{return len([]rune(s))}
func concat(a,b string)int{return stringUse(a+b)}
func localMap(n int)int{m:=make(map[int]int);m[0]=n;return m[0]}
func returnedMap()map[int]int{return make(map[int]int)}
func pointers(n int)int{s:=make([]*int,n);return len(s)}
`
	for _, tc := range []struct {
		os, arch string
		roots    bool
	}{
		{"darwin", "arm64", false}, {"linux", "amd64", false}, {"windows", "386", false}, {"wasip1", "wasm", true},
	} {
		t.Run(tc.os+"/"+tc.arch, func(t *testing.T) {
			t.Setenv("GOOS", tc.os)
			t.Setenv("GOARCH", tc.arch)
			source := src
			// This harness's source runtime importer uses the host build
			// context. Wide helper declarations are checked by the target-aware
			// SSA make test and by actual Wasm execution instead.
			if tc.arch == "386" || tc.arch == "wasm" {
				source = strings.ReplaceAll(source, "func wide(n int64)int{b:=make([]byte,n);return bytesUse(b)}", "")
			}
			ir := compileInterfaceBoxIR(t, source, ssa.InstantiateGenerics, &llssa.Target{GOOS: tc.os, GOARCH: tc.arch}, tc.roots)
			for name, helper := range map[string]string{
				"dynamic": "MakeSliceBuffer", "fromBytes": "StringFromBytesBuffer", "toBytes": "StringToBytesBuffer",
				"fromRunes": "StringFromRunesBuffer", "toRunes": "StringToRunesBuffer", "concat": "StringCatBuffer", "readOnly": "StringToBytesReadOnly", "compare": "StringFromBytesReadOnly", "lookup": "StringFromBytesReadOnly",
			} {
				if body := llvmFunction(t, ir, "transient."+name); !strings.Contains(body, "runtime."+helper) {
					t.Errorf("%s lost buffer helper:\n%s", name, body)
				}
			}
			if body := llvmFunction(t, ir, "transient.fixed"); strings.Contains(body, "runtime.Alloc") {
				t.Errorf("fixed make stayed on heap:\n%s", body)
			}
			if body := llvmFunction(t, ir, "transient.returning"); strings.Contains(body, "MakeSliceBuffer") {
				t.Errorf("returned slice uses local buffer:\n%s", body)
			}
			if body := llvmFunction(t, ir, "transient.returnedMap"); strings.Contains(body, "MakeMapBuffer") {
				t.Errorf("returned map uses local header:\n%s", body)
			}
			if body := llvmFunction(t, ir, "transient.localMap"); strings.Contains(body, "MakeMapBuffer") == tc.roots {
				t.Errorf("wrong map storage with roots=%v:\n%s", tc.roots, body)
			}
			if body := llvmFunction(t, ir, "transient.pointers"); strings.Contains(body, "MakeSliceBuffer") == tc.roots {
				t.Errorf("wrong pointer slice storage with roots=%v:\n%s", tc.roots, body)
			}
			if tc.arch != "386" && tc.arch != "wasm" {
				if body := llvmFunction(t, ir, "transient.wide"); !strings.Contains(body, "MakeSliceBuffer") {
					t.Errorf("wide make lost checked helper:\n%s", body)
				}
			}
		})
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

func TestEscapeFunctionStackBudget(t *testing.T) {
	const src = `package stackbudget
func use(p *[32768]byte) byte { return p[0] }
func objects(x byte)byte{a:=new([32768]byte);b:=new([32768]byte);c:=new([32768]byte);a[0]=x;b[0]=x;c[0]=x;return use(a)+use(b)+use(c)}
func reserve(p *[65500]byte) byte{return p[0]}
func runes(s string)rune{a:=new([65500]byte);x:=reserve(a);return []rune(s)[0]+rune(x)}
func bytes(s string)byte{a:=new([65500]byte);x:=reserve(a);b:=[]byte(s);b[0]++;return b[0]+x}
func maps(x int)int{a:=new([65500]byte);n:=reserve(a);m:=make(map[int]int);m[x]=x;return m[x]+int(n)}
type I interface{M()}
func full(p *[65536]byte)byte{return p[0]}
func receiver(v I)byte{a:=new([65536]byte);x:=full(a);v.M();return x}
`
	ir := compileInterfaceBoxIR(t, src, ssa.InstantiateGenerics, nil, false)
	if body := llvmFunction(t, ir, "stackbudget.objects"); strings.Count(body, "runtime.AllocZ") != 1 {
		t.Fatalf("function budget did not retain one large object on heap:\n%s", body)
	}
	if body := llvmFunction(t, ir, "stackbudget.runes"); strings.Contains(body, "StringToRunesBuffer") || !strings.Contains(body, "StringToRunes") {
		t.Fatalf("rune buffer charged as bytes:\n%s", body)
	}
	if body := llvmFunction(t, ir, "stackbudget.bytes"); !strings.Contains(body, "StringToBytesBuffer") {
		t.Fatalf("small byte buffer lost remaining budget:\n%s", body)
	}
	if body := llvmFunction(t, ir, "stackbudget.maps"); strings.Contains(body, "MakeMapBuffer") {
		t.Fatalf("map header/bucket ignored the budget:\n%s", body)
	}
	if body := llvmFunction(t, ir, "stackbudget.receiver"); strings.Contains(body, "IfacePtrDataBuffer") || !strings.Contains(body, "IfacePtrData") {
		t.Fatalf("receiver adapter ignored the shared budget:\n%s", body)
	}
}
