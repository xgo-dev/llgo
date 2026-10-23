//go:build go1.26

package build

import (
	"os"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	xtoolllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
	"github.com/xgo-dev/llvm"
)

func TestWindowsStdCallARM64CompletePackage(t *testing.T) {
	const path = "internal/runtime/syscall/windows"
	conf := NewDefaultConf(ModeBuild)
	conf.Goos, conf.Goarch = "windows", "arm64"
	t.Setenv("CCFLAGS", "--target="+xtoolllvm.GetTargetTriple(conf.Goos, conf.Goarch))
	conf.OptLevel, conf.ForceRebuild = optlevel.O0, true
	conf.BinPath = t.TempDir()
	modules := 0
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath != path {
			return
		}
		modules++
		module := pkg.LPkg.Module()
		for _, name := range []string{"StdCall", "AsmStdCallAddr"} {
			function := module.NamedFunction(path + "." + name)
			if function.IsNil() || function.IsDeclaration() {
				t.Error("Windows call entry has no actual implementation", name)
			}
		}
		if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
			t.Error("complete Windows syscall module is invalid", err)
		}
	}
	pkgs, err := Build(Invocation{Args: []string{path}, Config: conf})
	if err != nil {
		t.Fatal("compile the complete Windows ARM64 syscall package", err)
	}
	if modules != 1 {
		t.Fatal("complete Windows syscall owner was not compiled exactly once", modules)
	}
	pkg := findPkg(pkgs, path)
	if pkg == nil || pkg.ArchiveFile == "" {
		t.Fatal("missing Windows syscall object archive")
	}
	info, err := os.Stat(pkg.ArchiveFile)
	if err != nil || info.Size() <= 8 {
		t.Fatal("empty Windows syscall object archive", err)
	}
}
