package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestDirectWasmFinalizer(t *testing.T) {
	const src = `package finalizers
import ("runtime"; _ "unsafe")
type Item struct { N int }
func register(p *Item) { runtime.SetFinalizer(p, func(*Item) {}) }
func closure(p *Item, seen *int) { runtime.SetFinalizer(p, func(p *Item) { *seen += p.N }) }
func result(p *Item) { runtime.SetFinalizer(p, func(*Item) error { return nil }) }
func cancel(p *Item) { runtime.SetFinalizer(p, nil) }
func shared(p *Item) any { v := any(p); runtime.SetFinalizer(v, nil); return v }
func dynamic(p, fn any) { runtime.SetFinalizer(p, fn) }
func nonpointer() { runtime.SetFinalizer(1, func(int) {}) }
func wrong(p *Item) { runtime.SetFinalizer(p, func(*int) {}) }
func iface(p *Item) { runtime.SetFinalizer(p, func(any) {}) }
func scalar(p *Item) { runtime.SetFinalizer(p, func(*Item) bool { return true }) }
func tuple(p *Item) { runtime.SetFinalizer(p, func(*Item) (int,int) { return 0,0 }) }
func variadic(p *Item) { runtime.SetFinalizer(p, func(...*Item) {}) }
func notfunc(p *Item) { runtime.SetFinalizer(p, 42) }
func deferred(p *Item) { defer runtime.SetFinalizer(p, nil) }
func async(p *Item) { go runtime.SetFinalizer(p, nil) }
func indirect(p *Item, register func(any,any)) { register(p, nil) }
func SetFinalizer(any, any) {}
func sameName(p *Item) { SetFinalizer(p, func(*Item) {}) }
//go:linkname foreign C.finalize
func foreign(*Item)
func nativeCall(p *Item) { runtime.SetFinalizer(p, foreign) }
//llgo:type C
type nativeCallback func(*Item)
func nativeValue(p *Item, f nativeCallback) { runtime.SetFinalizer(p, f) }
`
	for _, tc := range []struct {
		name, goos, arch string
		dce, gc, debug   bool
	}{
		{"enabled", "wasip1", "wasm", true, true, false},
		{"no-dce", "wasip1", "wasm", false, true, false},
		{"no-gc", "wasip1", "wasm", true, false, false},
		{"native", "linux", "amd64", true, true, false},
		{"debugger", "wasip1", "wasm", true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: tc.goos, GOARCH: tc.arch})
			defer prog.Dispose()
			prog.EnableDeadcodeDrop(tc.dce)
			prog.EnableGCRoots(tc.gc)
			pkg, _, err := NewPackageExWithEmbedMetaOptions(prog, nil, nil, nil, ssapkg, files, nil, false, Options{DebuggerFrames: tc.debug})
			if err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"register", "closure", "result", "cancel", "shared"} {
				ir := pkg.Module().NamedFunction("finalizers." + name).String()
				if got := strings.Contains(ir, "runtime.llgoSetFinalizerPointer"); got != (tc.name == "enabled") {
					t.Errorf("%s direct = %v:\n%s", name, got, ir)
				}
				if tc.name == "enabled" && name != "shared" && (strings.Contains(ir, "@_llgo_") || strings.Contains(ir, `@"_llgo_`)) {
					t.Errorf("%s still references argument type metadata:\n%s", name, ir)
				}
				if name == "shared" && !strings.Contains(ir, "_llgo_finalizers.Item") {
					t.Errorf("observable interface lost type metadata:\n%s", ir)
				}
			}
			for _, name := range []string{"dynamic", "nonpointer", "wrong", "iface", "scalar", "tuple", "variadic", "notfunc", "deferred", "async", "indirect", "sameName", "nativeCall", "nativeValue"} {
				ir := pkg.Module().NamedFunction("finalizers." + name).String()
				if strings.Contains(ir, "runtime.llgoSetFinalizerPointer") {
					t.Errorf("%s lost its dynamic finalizer path:\n%s", name, ir)
				}
			}
		})
	}
}
