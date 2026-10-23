//go:build go1.27 && (amd64 || llgo_amd64_cross)

package build

import (
	"os"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	xtoolllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
	"github.com/xgo-dev/llvm"
)

// Native AMD64 CI owns this complete selected-SDK package. Other hosts require
// the explicit cross role and actual AMD64 tools; this is object evidence, not
// native AES execution. ModeBuild releases the program after the live hook.
func TestMapsRuntimeAMD64CompletePackage(t *testing.T) {
	const mapsPackage = "internal/runtime/maps"
	names := []string{"memHashAES", "memHash32AES", "memHash64AES"}
	conf := NewDefaultConf(ModeBuild)
	conf.Goos, conf.Goarch = runtime.GOOS, "amd64"
	t.Setenv("CCFLAGS", "--target="+xtoolllvm.GetTargetTriple(conf.Goos, conf.Goarch))
	conf.OptLevel, conf.ForceRebuild = optlevel.O0, true
	conf.BinPath = t.TempDir()
	bodies := map[string]bool{}
	modules := 0
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath != mapsPackage {
			return
		}
		modules++
		if pkg.Types == nil || pkg.Types.Path() != mapsPackage {
			t.Fatal("the selected SDK maps module lacks its actual typed package")
		}
		module := pkg.LPkg.Module()
		for _, name := range names {
			function := module.NamedFunction(mapsPackage + "." + name)
			bodies[name] = !function.IsNil() && !function.IsDeclaration() && function.BasicBlocksCount() > 0
		}
		if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
			t.Fatal("the complete selected SDK maps package has invalid LLVM IR", err)
		}
	}
	pkgs, err := Build(Invocation{Args: []string{mapsPackage}, Config: conf})
	if err != nil {
		t.Fatal("compile the complete selected AMD64 SDK maps package", err)
	}
	if modules != 1 {
		t.Fatal("the complete SDK maps package was not observed exactly once", modules)
	}
	for _, name := range names {
		if !bodies[name] {
			t.Fatal("the complete SDK maps package lacks its actual AES definition", name)
		}
	}
	pkg := findPkg(pkgs, mapsPackage)
	if pkg == nil || pkg.ArchiveFile == "" {
		t.Fatal("the complete SDK maps package did not publish its object archive")
	}
	info, err := os.Stat(pkg.ArchiveFile)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 8 {
		t.Fatal("the complete SDK maps package has no nonempty object archive", err)
	}
}
