package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

func TestOutlinedGCRoots(t *testing.T) {
	for _, tc := range []struct {
		name, arch, triple string
		level              optlevel.Level
		dce, tls, outline  bool
	}{
		{"wasm32-size", "wasm", "wasm32-unknown-emscripten", optlevel.Oz, true, false, true},
		{"wasm64-size", "wasm", "wasm64-unknown-emscripten", optlevel.Oz, true, false, true},
		{"wasm32-tls", "wasm", "wasm32-unknown-emscripten", optlevel.Oz, true, true, true},
		{"wasm64-tls", "wasm", "wasm64-unknown-emscripten", optlevel.Oz, true, true, true},
		{"balanced-size", "wasm", "wasm32-unknown-emscripten", optlevel.Os, true, false, false},
		{"speed", "wasm", "wasm32-unknown-emscripten", optlevel.O2, true, false, false},
		{"debug", "wasm", "wasm32-unknown-emscripten", optlevel.O0, true, false, false},
		{"no-dce", "wasm", "wasm32-unknown-emscripten", optlevel.Oz, false, false, false},
		{"native", "amd64", "x86_64-unknown-linux", optlevel.Oz, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			goos := "js"
			if tc.arch == "amd64" {
				goos = "linux"
			}
			prog := NewProgram(&Target{GOOS: goos, GOARCH: tc.arch, LLVMTarget: tc.triple, OptLevel: tc.level})
			defer prog.Dispose()
			prog.EnableDeadcodeDrop(tc.dce)
			prog.EnableGCRoots(true)
			prog.EnableThreadLocalGCRoots(tc.tls)
			pkg := prog.NewPackage("p", "example.com/roots")
			params := types.NewTuple(types.NewVar(token.NoPos, nil, "p", types.Typ[types.UnsafePointer]))
			sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
			for _, name := range []string{"first", "second"} {
				fn := pkg.NewFunc(name, sig, GoDecl)
				b := fn.MakeBody(1)
				roots := fn.NewGCRoots(2)
				b.SetGCRoot(roots[0], b.Param(0))
				b.Return()
				b.EndBuild()
				calls := strings.Contains(fn.impl.String(), "call void @__llgo_gcroot_")
				if calls != tc.outline {
					t.Fatalf("outlined=%v, want %v:\n%s", calls, tc.outline, fn.impl)
				}
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("%v:\n%s", err, pkg.Module())
			}
			if !tc.outline {
				return
			}
			suffix := ""
			if tc.tls {
				suffix = "_tls"
			}
			for _, name := range []string{"push_2", "pop"} {
				fn := pkg.Module().NamedFunction("__llgo_gcroot_" + name + suffix)
				if fn.IsNil() {
					t.Fatal("missing shared helper", name)
				}
				if fn.Linkage() != llvm.LinkOnceODRLinkage || fn.Visibility() != llvm.HiddenVisibility {
					t.Fatal("helper cannot be shared privately across packages")
				}
				if tc.tls && strings.Contains(fn.String(), "@llvm_gc_root_chain") {
					t.Fatal("worker helper used the single-worker chain")
				}
			}
		})
	}
}
