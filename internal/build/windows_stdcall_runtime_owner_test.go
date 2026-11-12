//go:build go1.27 && (windows || llgo_windows_cross)

package build

import (
	"os"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

func windowsStdCallCompletePackage(t *testing.T, arch string) {
	t.Helper()
	const path = "internal/runtime/syscall/windows"
	conf := NewDefaultConf(ModeBuild)
	conf.Goos, conf.Goarch = "windows", arch
	// Inherit the lane's actual CC/CXX and sysroot. In particular, do not
	// replace native MSVC with a bare GNU target lacking its dependency tree.
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
		t.Fatal("compile the complete Windows syscall package", arch, err)
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
