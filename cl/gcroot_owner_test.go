package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

func TestGCRootOwner(t *testing.T) {
	const src = `package roots
import "unsafe"
type Pair struct { A, B int }
type Pointer *int
type Text string
type Bytes []byte
type Raw unsafe.Pointer
var global Pair
func field(p *Pair) *int { return &p.B }
func index(p *[4]int) *int { return &p[2] }
func slice(p []byte) []byte { return p[1:][1:] }
func array(p []byte) *[2]byte { return (*[2]byte)(p[1:]) }
func text(p string) string { return p[1:] }
func named(p *int) Pointer { return Pointer(p) }
func namedtext(p Text) string { return string(p) }
func namedbytes(p Bytes) []byte { return []byte(p) }
func raw(p Raw) unsafe.Pointer { return unsafe.Pointer(p) }
func cast(p *int) *uint { return (*uint)(unsafe.Pointer(p)) }
func constant() string { return "abcd"[1:] }
func static() *int { return &global.B }
func load(p **int) *int { return *p }
func copy(p []byte) string { return string(p) }
func integer(p uintptr) unsafe.Pointer { return unsafe.Pointer(p) }
func merge(p, q *int, yes bool) *int { if yes { p = q }; return p }
func box(p *int) any { return p }
func synthetic(n int) []byte { return make([]byte, n, 256) }
type Function func()
func function(p Function) func() { return (func())(p) }
func closure() func() *int { p := Pair{}; return func() *int { return &p.B } }
`
	pkg, _, _ := buildGoSSAPkg(t, src)
	for _, name := range []string{"field", "index", "slice", "array", "text", "named", "cast", "namedtext", "namedbytes", "raw"} {
		fn := pkg.Func(name)
		ret := fn.Blocks[len(fn.Blocks)-1].Instrs[len(fn.Blocks[len(fn.Blocks)-1].Instrs)-1].(*ssa.Return)
		if got := gcRootOwner(ret.Results[0]); got != fn.Params[0] {
			t.Errorf("%s owner = %s, want parameter", name, got)
		}
	}
	for _, name := range []string{"load", "copy", "integer", "merge", "box", "synthetic", "function"} {
		fn := pkg.Func(name)
		for _, block := range fn.Blocks {
			if ret, ok := block.Instrs[len(block.Instrs)-1].(*ssa.Return); ok {
				if got := gcRootOwner(ret.Results[0]); got != ret.Results[0] {
					t.Errorf("%s crossed allocation or provenance boundary: %s", name, got)
				}
			}
		}
	}
	closure := pkg.Func("closure").AnonFuncs[0]
	ret := closure.Blocks[0].Instrs[len(closure.Blocks[0].Instrs)-1].(*ssa.Return)
	if got := gcRootOwner(ret.Results[0]); got != closure.FreeVars[0] {
		t.Errorf("captured aggregate owner = %s, want closure free variable", got)
	}
	for name, kind := range map[string]string{"constant": "constant", "static": "global"} {
		fn := pkg.Func(name)
		ret := fn.Blocks[0].Instrs[len(fn.Blocks[0].Instrs)-1].(*ssa.Return)
		owner := gcRootOwner(ret.Results[0])
		_, constant := owner.(*ssa.Const)
		_, global := owner.(*ssa.Global)
		if kind == "constant" && !constant || kind == "global" && !global {
			t.Errorf("%s owner = %T", name, owner)
		}
	}
}

func TestCompileSharedGCRoots(t *testing.T) {
	const src = `package roots
type Pair struct { A, B int }
var global Pair
func use(*int)
func useBytes([]byte)
func fields(p *Pair) { use(&p.A); use(&p.B) }
func slices(p []byte) { useBytes(p[1:]); useBytes(p[2:]) }
func static() { use(&global.A); use(&global.B) }
`
	for _, tc := range []struct {
		name, goos, arch string
		dce, roots       bool
	}{
		{"enabled", "wasip1", "wasm", true, true},
		{"no-dce", "wasip1", "wasm", false, true},
		{"no-roots", "wasip1", "wasm", true, false},
		{"native", "linux", "amd64", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: tc.goos, GOARCH: tc.arch})
			defer prog.Dispose()
			prog.EnableGCRoots(tc.roots)
			prog.EnableDeadcodeDrop(tc.dce)
			pkg, _, err := NewPackageExWithEmbedMetaOptions(prog, nil, nil, nil, ssapkg, files, nil, false, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"fields", "slices", "static"} {
				want := uint64(3)
				if name == "static" {
					want = 2
				}
				if tc.name == "enabled" {
					want = 1
					if name == "static" {
						want = 0
					}
				}
				if !tc.roots {
					want = 0
				}
				g := pkg.Module().NamedGlobal("roots." + name + "$gcmap")
				if want == 0 {
					if !g.IsNil() {
						t.Errorf("%s retained a root map: %s", name, g)
					}
					continue
				}
				if g.IsNil() {
					t.Fatalf("%s missing root map", name)
				}
				if g.Initializer().Operand(0).ZExtValue() != want {
					t.Fatalf("%s root count, want %d: %s", name, want, g)
				}
				if !strings.Contains(g.String(), "unnamed_addr constant") {
					t.Errorf("%s map cannot be pooled: %s", name, g)
				}
			}
		})
	}
}
