//go:build go1.26 && (amd64 || llgo_amd64_cross)

package build

import (
	"os"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	xtoolllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
	"github.com/xgo-dev/llvm"
)

// The native AMD64 standard-Go CI owns this complete actual SDK object root. Other
// hosts need the explicit llgo_amd64_cross role and a real configured AMD64
// toolchain; an unrelated Windows target profile must not silently qualify it.
// ModeBuild owns/releases its program; the hook only inspects the live module.
// This is object/owner evidence, not original AVX512 execution.
func TestGCScanRuntimeAMD64CompletePackage(t *testing.T) {
	conf := NewDefaultConf(ModeBuild)
	conf.Goos, conf.Goarch = runtime.GOOS, "amd64"
	// The package is cross-compiled on ARM64 hosts. Give the real IR compiler
	// the same AMD64 target, including for dependencies with x86 inline asm.
	t.Setenv("CCFLAGS", "--target="+xtoolllvm.GetTargetTriple(conf.Goos, conf.Goarch))
	conf.OptLevel, conf.ForceRebuild = optlevel.O0, true
	conf.BinPath = t.TempDir()
	bodies := map[string]bool{}
	modules := 0
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath != gcScanRuntimePackage {
			return
		}
		modules++
		module := pkg.LPkg.Module()
		for _, name := range []string{"ExpandAVX512", "scanSpanPackedAVX512", "FilterNilAVX512"} {
			function := module.NamedFunction(gcScanRuntimePackage + "." + name)
			bodies[name] = !function.IsNil() && !function.IsDeclaration()
		}
		if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
			t.Fatal("the complete selected SDK package has invalid LLVM IR", err)
		}
	}
	pkgs, err := Build(Invocation{Args: []string{gcScanRuntimePackage}, Config: conf})
	if err != nil {
		t.Fatal("compile the complete selected AMD64 SDK scan owner", err)
	}
	if modules != 1 {
		t.Fatal("the complete SDK scan package was not observed exactly once", modules)
	}
	for _, name := range []string{"ExpandAVX512", "scanSpanPackedAVX512", "FilterNilAVX512"} {
		if !bodies[name] {
			t.Fatal("the runtime replacement lacks an actual typed Go body", name)
		}
	}
	pkg := findPkg(pkgs, gcScanRuntimePackage)
	if pkg == nil || pkg.ArchiveFile == "" {
		t.Fatal("the complete SDK package did not publish its object archive")
	}
	info, err := os.Stat(pkg.ArchiveFile)
	if err != nil || info.Size() <= 8 {
		t.Fatal("the complete SDK package has no nonempty object archive", err)
	}
}
