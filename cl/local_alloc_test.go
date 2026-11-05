package cl

import (
	"fmt"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestCompileBorrowedLocals(t *testing.T) {
	const src = `package localalloc
type Pair struct { x, y uint64 }
func (p *Pair) put(x uint64) { p.y = x }
func forward(p *Pair, x uint64) { p.put(x) }
func ordinary(x uint64) uint64 { var p Pair; forward(&p, x); return p.x + p.y }
func literal(x uint64) uint64 { p := &Pair{x: x}; p.put(x); return p.x + p.y }
func scalar(p *uint64) { *p = 42 }
func field(x uint64) uint64 { p := new(Pair); scalar(&p.x); return p.x + x }
func array(i int) uint64 { var p [2]uint64; scalar(&p[i & 1]); return p[0] + p[1] }
func self(p *Pair, n int) { if n > 0 { self(p, n-1) }; p.y++ }
func recursive(n int) uint64 { var p Pair; self(&p, n); return p.y }
func first(p *Pair, n int) { if n > 0 { second(p, n-1) }; p.y++ }
func second(p *Pair, n int) { if n > 0 { first(p, n-1) }; p.y++ }
func mutual(n int) uint64 { var p Pair; first(&p, n); return p.y }
func named(n int) (r Pair) { self(&r, n); return }
type Pointers struct { p, q *uint64 }
func putPointer(p *Pointers, q *uint64) { p.p = q }
func pointer(q *uint64) *uint64 { var p Pointers; putPointer(&p, q); return p.p }
`
	for _, target := range []llssa.Target{
		{GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"}, {GOOS: "wasip1", GOARCH: "wasm"},
	} {
		for _, mode := range []ssa.BuilderMode{0, ssa.GlobalDebug} {
			t.Run(fmt.Sprintf("%s/%s/%v", target.GOOS, target.GOARCH, mode), func(t *testing.T) {
				ir := compileWithRewritesModeTarget(t, src, nil, mode, &target)
				for _, name := range []string{"ordinary", "literal", "field", "array", "recursive", "mutual", "pointer"} {
					body := llvmFunction(t, ir, "localalloc."+name)
					if strings.Contains(body, "runtime.Alloc") || !strings.Contains(body, "alloca ") {
						t.Errorf("%s should borrow stack storage:\n%s", name, body)
					}
				}
			})
		}
	}
}

func TestCompileEscapingLocals(t *testing.T) {
	const prefix = `package localalloc
import "unsafe"
var _ unsafe.Pointer
var saved *uint64
func borrow(p *uint64) { *p = 1 }
func save(p *uint64) { saved = p }
func unknown(*uint64)
func first(p *uint64, n int) { if n > 0 { second(p, n-1) }; saved = p }
func second(p *uint64, n int) { if n > 0 { first(p, n-1) }; *p = 2 }
func mixed(p, q *uint64) { *p = 1; saved = q }
`
	for _, tc := range []struct{ name, decl string }{
		{"global", `func local() { var x uint64; saved = &x }`},
		{"return", `func local() *uint64 { var x uint64; return &x }`},
		{"transitive", `func forward(p *uint64) { save(p) }; func local() { var x uint64; forward(&x) }`},
		{"recursive escape", `func local(n int) { var x uint64; second(&x, n) }`},
		{"both arguments", `func local() { var x uint64; mixed(&x, &x) }`},
		{"unknown", `func local() { var x uint64; unknown(&x) }`},
		{"dynamic", `func local(f func(*uint64)) { var x uint64; f(&x) }`},
		{"closure", `func local() func() { var x uint64; return func(){ x++ } }`},
		{"go", `func local() { var x uint64; go borrow(&x) }`},
		{"defer", `func local() { var x uint64; defer borrow(&x) }`},
		{"slice", `func local() []uint64 { var x [2]uint64; return x[:] }`},
		{"uintptr", `func local() uintptr { var x uint64; return uintptr(unsafe.Pointer(&x)) }`},
		{"loop", `func local(n int) { for i:=0; i<n; i++ { var x uint64; borrow(&x) } }`},
		{"large", `func local() uint64 { var x [16385]uint64; borrow(&x[0]); return x[0] }`},
		{"recover", `func local() { defer func(){ recover() }(); var x uint64; borrow(&x) }`},
	} {
		for _, mode := range []ssa.BuilderMode{0, ssa.GlobalDebug} {
			t.Run(fmt.Sprintf("%s/%v", tc.name, mode), func(t *testing.T) {
				ir := compileWithRewritesMode(t, prefix+tc.decl, nil, mode)
				if body := llvmFunction(t, ir, "localalloc.local"); !strings.Contains(body, "runtime.AllocZ") {
					t.Fatalf("local must retain heap storage:\n%s", body)
				}
			})
		}
	}
}

func TestBorrowedParameterSummaries(t *testing.T) {
	pkg, _, _ := buildGoSSAPkg(t, `package borrowcache
var saved *int
func goodA(p *int, n int) { if n > 0 { goodB(p, n-1) }; *p = 1 }
func goodB(p *int, n int) { if n > 0 { goodA(p, n-1) } }
func badA(p *int, n int) { if n > 0 { badB(p, n-1) }; saved = p }
func badB(p *int, n int) { if n > 0 { badC(p, n-1) } }
func badC(p *int, n int) { if n > 0 { badA(p, n-1) } }
`)
	prog := newLLSSAProg(t)
	t.Cleanup(prog.Dispose)
	ctx := &context{prog: prog, goTyps: pkg.Pkg}
	for _, order := range [][]string{{"badA", "badB", "badC", "goodA", "goodB"}, {"goodB", "badC", "badB", "badA", "goodA"}} {
		cache := make(map[*ssa.Parameter]bool)
		for _, name := range order {
			a := localEscapeAnalysis{context: ctx, pkg: pkg, params: cache, budget: 1024}
			if got, want := a.borrowed(pkg.Func(name).Params[0]), strings.HasPrefix(name, "good"); got != want {
				t.Fatalf("%s borrowed=%t, want %t (order %v)", name, got, want, order)
			}
		}
		for param, safe := range cache {
			if want := strings.HasPrefix(param.Parent().Name(), "good"); safe != want {
				t.Errorf("cached %s.%s=%t, want %t", param.Parent().Name(), param.Name(), safe, want)
			}
		}
	}
	cache := make(map[*ssa.Parameter]bool)
	a := localEscapeAnalysis{context: ctx, pkg: pkg, params: cache, budget: 3}
	if a.borrowed(pkg.Func("goodA").Params[0]) || len(cache) != 0 {
		t.Fatal("exhausted analysis published incomplete summaries")
	}
	a.budget = 1024
	if !a.borrowed(pkg.Func("goodA").Params[0]) {
		t.Fatal("budget exhaustion poisoned a later complete analysis")
	}
}

func TestBorrowedLocalRejectsInterfaceBox(t *testing.T) {
	pkg, _, _ := buildGoSSAPkg(t, `package localbox
func box() any { var x int; return &x }
`)
	for _, instr := range pkg.Func("box").Blocks[0].Instrs {
		if alloc, ok := instr.(*ssa.Alloc); ok {
			a := localEscapeAnalysis{budget: 1024}
			if a.borrowed(alloc) {
				t.Fatal("interface box retained a pointer to stack storage")
			}
			return
		}
	}
	t.Fatal("missing interface-boxed allocation")
}

func TestBorrowedLocalExplicitRoots(t *testing.T) {
	const src = `package localroots
type Pointers struct { p *uint64 }
func put(p *Pointers, q *uint64) { p.p = q }
func local(q *uint64) *uint64 { var p Pointers; put(&p, q); return p.p }
func inc(p *uint64) { *p++ }
func scalar() uint64 { var n uint64; inc(&n); return n }
`
	pkg, _, files := buildGoSSAPkg(t, src)
	prog := newLLSSAProg(t)
	t.Cleanup(prog.Dispose)
	prog.EnableGCRoots(true)
	llpkg, _, err := NewPackageEx(prog, nil, nil, pkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := llpkg.String()
	if body := llvmFunction(t, ir, "localroots.local"); !strings.Contains(body, "runtime.AllocZ") {
		t.Fatalf("pointer fields need scanned heap storage with explicit roots:\n%s", body)
	}
	if body := llvmFunction(t, ir, "localroots.scalar"); strings.Contains(body, "runtime.Alloc") {
		t.Fatalf("pointer-free local should use stack storage:\n%s", body)
	}
}
