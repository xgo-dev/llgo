//go:build go1.27 && (386 || llgo_386_cross)

package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	xtoolllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
	"github.com/xgo-dev/llvm"
)

// Keep the complete selected SDK source, including aliases preceding callees,
// alignment panic calls, and all non-CAS operations. This is object evidence;
// it does not grant native atomic execution coverage on a cross host.
func TestRuntimeAtomic386CompletePackage(t *testing.T) {
	const atomicPackage = "internal/runtime/atomic"
	names := strings.Fields(`
		Cas Casint32 Casint64 Casuintptr CasRel
		Loaduintptr Loaduint Storeint32 Storeint64 Storeuintptr Xadduintptr
		Loadint32 Loadint64 Xaddint32 Xaddint64 Cas64 Casp1 Xadd Xadd64
		Xchg8 Xchg Xchgint32 Xchgint64 Xchguintptr Xchg64
		StorepNoWB Store StoreRel StoreReluintptr Load64 Store64
		Or8 And8 Store8 Or And And32 Or32 And64 Or64 Anduintptr Oruintptr
	`)
	want := make(map[string]bool, len(names))
	for _, name := range names {
		want[name] = true
	}
	if len(names) != 42 || len(want) != len(names) {
		t.Fatal("the independent complete 386 atomic inventory changed")
	}
	source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src", "internal", "runtime", "atomic", "atomic_386.s"))
	if err != nil {
		t.Fatal("read the actual selected SDK atomic assembly", err)
	}
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(source), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "TEXT" {
			continue
		}
		name, _, found := strings.Cut(strings.TrimPrefix(fields[1], "·"), "(SB)")
		if !found || !want[name] || seen[name] {
			t.Fatal("the actual SDK atomic TEXT inventory differs", line)
		}
		seen[name] = true
	}
	if len(seen) != len(want) {
		t.Fatal("the actual SDK atomic source is incomplete", len(seen), len(want))
	}

	conf := NewDefaultConf(ModeBuild)
	conf.Goos, conf.Goarch = "windows", "386"
	t.Setenv("CCFLAGS", "--target="+xtoolllvm.GetTargetTriple(conf.Goos, conf.Goarch))
	conf.OptLevel, conf.ForceRebuild = optlevel.O0, true
	conf.BinPath = t.TempDir()
	modules := 0
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath != atomicPackage {
			return
		}
		modules++
		module := pkg.LPkg.Module()
		for _, name := range names {
			function := module.NamedFunction(atomicPackage + "." + name)
			if function.IsNil() || function.IsDeclaration() || function.BasicBlocksCount() == 0 {
				t.Error("the complete atomic package lacks its actual definition", name)
			}
		}
		if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
			t.Fatal("the complete 386 SDK atomic package has invalid LLVM IR", err)
		}
	}
	pkgs, err := Build(Invocation{Args: []string{atomicPackage}, Config: conf})
	if err != nil {
		t.Fatal("compile the complete selected 386 SDK atomic package", err)
	}
	if modules != 1 {
		t.Fatal("the complete atomic package was not observed exactly once", modules)
	}
	pkg := findPkg(pkgs, atomicPackage)
	if pkg == nil || pkg.ArchiveFile == "" {
		t.Fatal("the complete atomic package did not publish an object archive")
	}
	info, err := os.Stat(pkg.ArchiveFile)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 8 {
		t.Fatal("the complete atomic package has no nonempty object archive", err)
	}
}
