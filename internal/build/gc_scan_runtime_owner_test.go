package build

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	xtoolllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
	"github.com/xgo-dev/llvm"
)

const gcScanRuntimePackage = "internal/runtime/gc/scan"

// These are the complete supported expansion factors, not a sample of the
// first failing private entry. The SDK's PCs carry AX input and Z1/Z2 output;
// they are not ordinary void(ptr) Go functions.
func gcScanRuntimeFactors() []int {
	return []int{
		1, 2, 3, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32,
		36, 40, 44, 48, 52, 56, 60, 64,
	}
}

func TestGCScanRuntimeAMD64CompleteOriginalFamily(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src", "internal", "runtime", "gc", "scan")
	var expanders []string
	for _, factor := range gcScanRuntimeFactors() {
		expanders = append(expanders, fmt.Sprintf("expandAVX512_%d<>", factor))
	}
	text := regexp.MustCompile(`(?m)^\s*TEXT\s+·([^\s(,]+)\(SB\),`)
	for _, owner := range []struct {
		file string
		want []string
	}{
		{"expand_amd64.s", expanders},
		{"scan_amd64.s", []string{"ExpandAVX512", "scanSpanPackedAVX512"}},
		{"filter_amd64.s", []string{"FilterNilAVX512"}},
	} {
		t.Run(owner.file, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(dir, owner.file))
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, match := range text.FindAllSubmatch(source, -1) {
				names = append(names, string(match[1]))
			}
			if !reflect.DeepEqual(names, owner.want) {
				t.Fatalf("complete original SDK owner changed: got %v, want %v", names, owner.want)
			}
			if owner.file == "expand_amd64.s" {
				pc := regexp.MustCompile(`\$·(expandAVX512_[0-9]+<>)\(SB\)`)
				var entries []string
				for _, match := range pc.FindAllSubmatch(source, -1) {
					entries = append(entries, string(match[1]))
				}
				if !reflect.DeepEqual(entries, expanders) {
					t.Fatal("the actual PC table does not own every private expander exactly once", entries)
				}
			}
		})
	}
}

// Compile the actual complete SDK package on every LLVM host. ModeBuild owns
// and releases its program; inspect the live module only inside ModuleHook.
// This is object/owner evidence, not execution of the original AVX512 source.
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
