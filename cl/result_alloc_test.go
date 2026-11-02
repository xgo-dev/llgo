package cl

import (
	"fmt"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestCompileNonEscapingNamedResults(t *testing.T) {
	const src = `package resultalloc
import "unsafe"
type Pair struct { Lo, Hi uint64 }
func put(p *Pair, x uint64) { p.Hi = x }
func forward(p *Pair, x uint64) { put(p, x) }
var saved *uint64
func mixed(first, second *uint64) { *first = 1; saved = second }
func scalar() (r uint64) { mixed(&r, saved); return }
func pointerBranch(op int, x uint64) (r Pair) {
	if op == 0 { return Pair{Lo: x} }
	forward(&r, x)
	return r
}

func fieldBranch(op int, x uint64) (r Pair) {
	if op == 0 { return Pair{Lo: x} }
	p := &r.Hi
	*p = x
	return
}
type words struct { typ, data unsafe.Pointer }
func interfaceBranch(typ, data unsafe.Pointer) (val any) {
	if typ == nil { return nil }
	p := (*words)(unsafe.Pointer(&val))
	p.typ = typ
	p.data = data
	return
}
func arrayBranch(op int, x uint64) (r [2]uint64) {
	if op != 0 { r[op & 1] = x }
	return
}
func phiBranch(op int, x uint64) (r Pair) {
	p := &r.Lo
	if op != 0 { p = &r.Hi }
	*p = x
	return
}
func compareBranch(x uint64) (r Pair) {
	if p := &r; p != nil { p.Hi = x }
	return
}
type PairPtr *Pair
func changeType(x uint64) (r Pair) {
	p := PairPtr(&r)
	p.Hi = x
	return
}
func borrowZero(*struct{}) {}
func zero() (r struct{}) { borrowZero(&r); return }
`
	for _, target := range []llssa.Target{
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		for _, mode := range []ssa.BuilderMode{0, ssa.GlobalDebug} {
			t.Run(fmt.Sprintf("%s/%s/%v", target.GOOS, target.GOARCH, mode), func(t *testing.T) {
				ir := compileWithRewritesModeTarget(t, src, nil, mode, &target)
				for _, name := range []string{"pointerBranch", "fieldBranch", "interfaceBranch", "arrayBranch", "phiBranch", "scalar", "compareBranch", "changeType"} {
					body := llvmFunction(t, ir, "resultalloc."+name)
					if strings.Contains(body, "runtime.Alloc") || !strings.Contains(body, "alloca ") {
						t.Errorf("%s should use stack result storage:\n%s", name, body)
					}
				}
				if body := llvmFunction(t, ir, "resultalloc.zero"); strings.Contains(body, "alloca ") {
					t.Errorf("zero-sized result must retain the shared heap sentinel:\n%s", body)
				}
			})
		}
	}
}

func TestCompileEscapingNamedResults(t *testing.T) {
	const prefix = `package resultalloc
import "unsafe"
var _ unsafe.Pointer
var saved *uint64
func save(p *uint64) { saved = p }
func identity(p *uint64) *uint64 { return p }
func unknown(*uint64)
func recursive(p *uint64, n int) { if n != 0 { recursive(p, n-1) }; *p = 1 }
func borrow(p *uint64) { *p = 1 }
func mixed(first, second *uint64) { *first = 1; saved = second }
`
	for _, tc := range []struct{ name, decl string }{
		{"global", `func result() (r uint64) { saved = &r; return }`},
		{"helper", `func result() (r uint64) { save(&r); return }`},
		{"second argument", `func result() (r uint64) { mixed(&r, &r); return }`},
		{"returned alias", `func result() (r uint64) { _ = identity(&r); return }`},
		{"unknown", `func result() (r uint64) { unknown(&r); return }`},
		{"indirect", `func result(f func(*uint64)) (r uint64) { f(&r); return }`},
		{"closure", `func result() (r uint64, f func()) { f = func() { r++ }; return }`},
		{"go", `func result() (r uint64) { go borrow(&r); return }`},
		{"defer", `func result() (r uint64) { defer borrow(&r); return }`},
		{"uintptr", `func result() (r uint64, p uintptr) { p = uintptr(unsafe.Pointer(&r)); return }`},
		{"hidden uintptr", `func toInt(p *uint64) { _ = uintptr(unsafe.Pointer(p)) }; func result() (r uint64) { toInt(&r); return }`},
		{"unsafe pointer", `func result() (r uint64, p unsafe.Pointer) { p = unsafe.Pointer(&r); return }`},
		{"pointer", `func result() (r uint64, p *uint64) { p = &r; return }`},
		{"field", `func result() (r struct { x uint64 }) { saved = &r.x; return }`},
		{"index", `func result() (r [2]uint64) { saved = &r[1]; return }`},
		{"named pointer", `type Ptr *uint64; var named Ptr; func result() (r uint64) { named = Ptr(&r); return }`},
		{"phi", `func result(b bool) (r uint64) { p := &r; if b { p = saved }; saved = p; return }`},
		{"slice", `func result() (r [2]uint64, s []uint64) { s = r[:]; return }`},
		{"recursive", `func result(n int) (r uint64) { recursive(&r, n); return }`},
		{"large", `func result() (r [16385]uint64) { borrow(&r[0]); return }`},
		{"analysis limit", "func result() (r uint64) {" + strings.Repeat("borrow(&r);", 1025) + "return }"},
		{"intrinsic body", `//go:linkname intrinsic llgo.atomicLoad
func intrinsic(p *uint64) uint64 { return *p }
func result() (r uint64) { _ = intrinsic(&r); return }`},
	} {
		for _, mode := range []ssa.BuilderMode{0, ssa.GlobalDebug} {
			t.Run(fmt.Sprintf("%s/%v", tc.name, mode), func(t *testing.T) {
				ir := compileWithRewritesMode(t, prefix+tc.decl, nil, mode)
				if body := llvmFunction(t, ir, "resultalloc.result"); !strings.Contains(body, "runtime.Alloc") {
					t.Fatalf("result must retain heap storage:\n%s", body)
				}
			})
		}
	}
}

func TestResultEscapeAnalysisRejectsUnsupportedUses(t *testing.T) {
	pkg, _, _ := buildGoSSAPkg(t, `package resultalloc
func receive(c <-chan int) { <-c }
func add(n int) int { return n + 1 }
`)
	// Missing referrers and operations outside the supported pointer-use
	// subset must never be taken as proof that a value is only borrowed.
	for _, value := range []ssa.Value{
		ssa.NewConst(nil, pkg.Func("receive").Params[0].Type()),
		pkg.Func("receive").Params[0],
		pkg.Func("add").Params[0],
	} {
		analysis := resultEscapeAnalysis{budget: 1024}
		if analysis.borrowed(value, make(map[ssa.Value]bool)) {
			t.Errorf("unsupported use of %s was classified as borrowed", value)
		}
	}
}

func TestNamedResultExplicitRoots(t *testing.T) {
	const src = `package resultalloc
type Pair struct { x, y uint64 }
func put(p *Pair) { p.y = 1 }
func scalar() (r Pair) { put(&r); return }
type Pointers struct { p, q *uint64 }
func putPointer(p *Pointers, q *uint64) { p.p = q }
func pointer(q *uint64) (r Pointers) { putPointer(&r, q); return }
`
	ssaPkg, _, files := buildGoSSAPkg(t, src)
	prog := newLLSSAProg(t)
	t.Cleanup(prog.Dispose)
	prog.EnableGCRoots(true)
	pkg, _, err := NewPackageEx(prog, nil, nil, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	if body := llvmFunction(t, pkg.String(), "resultalloc.scalar"); strings.Contains(body, "runtime.Alloc") {
		t.Fatalf("pointer-free result should use stack storage with explicit roots:\n%s", body)
	}
	if body := llvmFunction(t, pkg.String(), "resultalloc.pointer"); !strings.Contains(body, "runtime.AllocZ") {
		t.Fatalf("pointer-containing result needs scanned heap storage:\n%s", body)
	}
}
