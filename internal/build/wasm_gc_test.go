package build

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestWasmGCRootFrameLinksRuntimeChain(t *testing.T) {
	conf := NewDefaultConf(ModeGen)
	conf.Target = "wasi"
	conf.Tags = "llgo.wasm.gc.linear"
	modules := make(map[string]string)
	var copyContract, lengthContract bool
	conf.ModuleHook = func(pkg Package) {
		switch pkg.PkgPath {
		case "github.com/xgo-dev/llgo/runtime/internal/gcroot",
			"github.com/xgo-dev/llgo/internal/build/testdata/wasm-gc-liveness":
			modules[pkg.PkgPath] = pkg.LPkg.String()
		case "github.com/xgo-dev/llgo/runtime/internal/runtime":
			mod := pkg.LPkg.Module()
			copy := mod.NamedFunction(pkg.PkgPath + ".CStrCopy")
			length := mod.NamedFunction(pkg.PkgPath + ".MapLen")
			copyContract = !copy.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("memory")).IsNil()
			lengthContract = !length.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("range")).IsNil()
		}
	}
	if _, err := Do([]string{"./testdata/wasm-gc-liveness"}, conf); err != nil {
		t.Fatal(err)
	}
	if len(modules) != 2 {
		t.Fatalf("observed %d relevant modules, want 2", len(modules))
	}
	if copyContract || !lengthContract {
		t.Fatalf("linear GC contracts: CStrCopy memory=%v, MapLen range=%v; want conservative copy and preserved scalar range", copyContract, lengthContract)
	}
	mainIR := modules["github.com/xgo-dev/llgo/internal/build/testdata/wasm-gc-liveness"]
	if !strings.Contains(mainIR, "@llvm_gc_root_chain") {
		t.Fatalf("main package does not publish the compiler root chain:\n%s", mainIR)
	}
	runtimeIR := modules["github.com/xgo-dev/llgo/runtime/internal/gcroot"]
	if !strings.Contains(runtimeIR, "@llvm_gc_root_chain") {
		t.Fatalf("runtime package does not consume the compiler root chain:\n%s", runtimeIR)
	}
}
