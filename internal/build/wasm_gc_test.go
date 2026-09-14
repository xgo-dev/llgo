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
	var sawRuntime, copyRestricted, lengthGuaranteed bool
	conf.ModuleHook = func(pkg Package) {
		switch pkg.PkgPath {
		case "github.com/xgo-dev/llgo/runtime/internal/gcroot",
			"github.com/xgo-dev/llgo/internal/build/testdata/wasm-gc-liveness":
			modules[pkg.PkgPath] = pkg.LPkg.String()
		case "github.com/xgo-dev/llgo/runtime/internal/runtime":
			sawRuntime = true
			mod := pkg.LPkg.Module()
			copy := mod.NamedFunction(pkg.PkgPath + ".CStrCopy")
			length := mod.NamedFunction(pkg.PkgPath + ".MapLen")
			copyRestricted = !copy.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("writeonly")).IsNil()
			lengthGuaranteed = !length.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("range")).IsNil()
		}
	}
	if _, err := Do([]string{"./testdata/wasm-gc-liveness"}, conf); err != nil {
		t.Fatal(err)
	}
	if len(modules) != 2 {
		t.Fatalf("observed %d relevant modules, want 2", len(modules))
	}
	if !sawRuntime || copyRestricted || !lengthGuaranteed {
		t.Fatalf("linear GC attributes: runtime=%v, copy writeonly=%v, length range=%v", sawRuntime, copyRestricted, lengthGuaranteed)
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
