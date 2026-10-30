//go:build go1.27 && windows && (386 || llgo_windows_386_cross)

package build

import (
	"debug/pe"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

func TestRuntimeAtomic386ActualWrappersMatchOriginalGo(t *testing.T) {
	if runtime.GOARCH != "386" {
		if runtime.GOARCH != "amd64" {
			t.Fatal("the explicit PE32 execution role requires Windows 386 or amd64")
		}
		cc, cxx := os.Getenv("LLGO_WINDOWS_CROSS_CC"), os.Getenv("LLGO_WINDOWS_CROSS_CXX")
		if cc == "" || cxx == "" {
			t.Fatal("the explicit Windows 386 execution role requires its real CC/CXX and sysroot")
		}
		t.Setenv("CC", cc)
		t.Setenv("CXX", cxx)
	}
	t.Setenv("GOOS", "windows")
	t.Setenv("GOARCH", "386")
	dir := t.TempDir()
	mainFile := assemblyLanguageTestWrite(t, dir, "main.go", runtimeAtomic386Program(t))
	original := filepath.Join(dir, "original.exe")
	goTool := filepath.Join(runtime.GOROOT(), "bin", "go.exe")
	runtimeAtomicCommand(t, goTool,
		[]string{"build", "-ldflags=-checklinkname=0", "-o", original, mainFile}, 5*time.Minute)
	runtimeAtomic386RequirePE(t, original)
	want := runtimeAtomicCommand(t, original, nil, 5*time.Minute)
	if want != "atomic 386 wrappers 10 aligned 80 unaligned 560" {
		t.Fatal("the actual original-Go 386 execution denominator changed", want)
	}

	conf := NewDefaultConf(ModeBuild)
	conf.Goos, conf.Goarch = "windows", "386"
	conf.OptLevel, conf.LTO = optlevel.O0, lto.Off
	conf.ForceRebuild = true
	conf.OutFile = filepath.Join(dir, "llgo.exe")
	atomicModules, drivers := 0, 0
	conf.ModuleHook = func(pkg Package) {
		module := pkg.LPkg.Module()
		if pkg.PkgPath == "internal/runtime/atomic" {
			atomicModules++
			runtimeAtomic386RequireOperations(t, module)
			if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
				t.Fatal("the actual complete 386 atomic module is invalid", err)
			}
		}
		if !slices.Contains(pkg.GoFiles, mainFile) {
			return
		}
		drivers++
		for _, row := range runtimeAtomic386AlignmentRows() {
			entry := module.NamedFunction("internal/runtime/atomic." + row.name)
			if entry.IsNil() || entry.FirstUse().IsNil() {
				t.Fatal("the 386 driver did not retain its real SDK call target", row.name)
			}
		}
	}
	if _, err := Do([]string{mainFile}, conf); err != nil {
		t.Fatal("compile the actual native 386 atomic wrappers", err)
	}
	if atomicModules != 1 || drivers != 1 {
		t.Fatal("expected exactly one complete atomic module and one actual driver", atomicModules, drivers)
	}
	runtimeAtomic386RequirePE(t, conf.OutFile)
	if got := runtimeAtomicCommand(t, conf.OutFile, nil, 5*time.Minute); got != want {
		t.Fatalf("LLGo 386 atomic execution %q differs from original Go %q", got, want)
	}
}

func runtimeAtomic386RequirePE(t *testing.T, path string) {
	t.Helper()
	file, err := pe.Open(path)
	if err != nil {
		t.Fatal("open the real 386 executable", err)
	}
	defer file.Close()
	_, pe32 := file.OptionalHeader.(*pe.OptionalHeader32)
	if file.FileHeader.Machine != pe.IMAGE_FILE_MACHINE_I386 || !pe32 ||
		file.FileHeader.Characteristics&pe.IMAGE_FILE_DLL != 0 ||
		file.FileHeader.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE == 0 {
		t.Fatal("native atomic execution requires an actual I386 PE32 executable", file.FileHeader)
	}
}
