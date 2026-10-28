//go:build go1.27 && ((windows && 386) || llgo_windows_386_cross)

package build

import (
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/xtool/nm"
	"github.com/xgo-dev/llvm"
)

// Compile the complete selected math package, not a reduced rounding fixture.
// COFF definitions are object evidence; the native 386 driver owns execution.
func TestMathRuntime386CompletePackage(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "386" {
		cc, cxx := os.Getenv("LLGO_WINDOWS_CROSS_CC"), os.Getenv("LLGO_WINDOWS_CROSS_CXX")
		if cc == "" || cxx == "" {
			t.Fatal("the explicit Windows 386 cross role requires its real CC/CXX and sysroot")
		}
		// Select target tools only after Go has built the host test runner.
		t.Setenv("CC", cc)
		t.Setenv("CXX", cxx)
	}
	wantSources := map[string][]string{
		"floor_386.s": {"archCeil", "archFloor", "archTrunc"},
		"hypot_386.s": {"archHypot"},
	}
	names := []string{"archCeil", "archFloor", "archTrunc", "archHypot"}
	text := regexp.MustCompile(`(?m)^\s*TEXT\s+·([^\s(,]+)\(SB\),`)
	conf := NewDefaultConf(ModeBuild)
	conf.Goos, conf.Goarch = "windows", "386"
	conf.OptLevel, conf.LTO = optlevel.O0, lto.Off
	conf.ForceRebuild, conf.BinPath = true, t.TempDir()
	modules := 0
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath != "math" {
			return
		}
		modules++
		files := selectedSFiles(pkg.OtherFiles)
		if len(files) != len(wantSources) {
			t.Fatal("the complete selected SDK math assembly inventory changed", files)
		}
		seen := make(map[string]bool)
		for _, file := range files {
			name := filepath.Base(file)
			want, ok := wantSources[name]
			if !ok || seen[name] || file != filepath.Join(runtime.GOROOT(), "src", "math", name) {
				t.Fatal("unexpected or duplicate selected SDK math assembly", file)
			}
			seen[name] = true
			source, err := os.ReadFile(file)
			if err != nil {
				t.Fatal("read the actual selected SDK math assembly", err)
			}
			var entries []string
			for _, match := range text.FindAllSubmatch(source, -1) {
				entries = append(entries, string(match[1]))
			}
			if !reflect.DeepEqual(entries, want) {
				t.Fatal("the actual math TEXT inventory changed", name, entries, want)
			}
			t.Logf("actual SDK %s TEXT entries: %v", file, entries)
		}
		if pkg.Types == nil || pkg.Types.Path() != "math" {
			t.Fatal("the actual SDK math package lacks its typed declarations")
		}
		for _, name := range names {
			function, ok := pkg.Types.Scope().Lookup(name).(*types.Func)
			if !ok || function.Pkg() != pkg.Types {
				t.Fatal("the actual SDK math package lacks its assembly declaration", name)
			}
			parameters := 1
			if name == "archHypot" {
				parameters = 2
			}
			signature := function.Type().(*types.Signature)
			if signature.Recv() != nil || signature.Variadic() || signature.Params().Len() != parameters ||
				signature.Results().Len() != 1 || !types.Identical(signature.Results().At(0).Type(), types.Typ[types.Float64]) {
				t.Fatal("the actual SDK math assembly signature changed", name, signature)
			}
			for i := 0; i < parameters; i++ {
				if !types.Identical(signature.Params().At(i).Type(), types.Typ[types.Float64]) {
					t.Fatal("the actual SDK math assembly argument changed", name, i, signature)
				}
			}
			declarations := 0
			for _, source := range pkg.Syntax {
				for _, declaration := range source.Decls {
					entry, ok := declaration.(*ast.FuncDecl)
					if ok && entry.Name.Name == name {
						declarations++
						if entry.Body != nil {
							t.Fatal("the original math assembly entry was replaced by a Go body", name)
						}
					}
				}
			}
			if declarations != 1 {
				t.Fatal("the selected SDK lacks its unique original math declaration", name, declarations)
			}
		}
		// Assembly is emitted in separate modules after this live Go-IR hook.
		if err := llvm.VerifyModule(pkg.LPkg.Module(), llvm.ReturnStatusAction); err != nil {
			t.Fatal("the complete SDK math Go module has invalid LLVM IR", err)
		}
	}
	pkgs, err := Build(Invocation{Args: []string{"math"}, Config: conf})
	if err != nil {
		t.Fatal("compile the complete selected Windows 386 SDK math package", err)
	}
	if modules != 1 {
		t.Fatal("the complete SDK math package was not observed exactly once", modules)
	}
	pkg := findPkg(pkgs, "math")
	if pkg == nil || pkg.ArchiveFile == "" {
		t.Fatal("the complete SDK math package did not publish its object archive")
	}
	info, err := os.Stat(pkg.ArchiveFile)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 8 {
		t.Fatal("the complete SDK math package has no nonempty object archive", err)
	}
	_, readobj := windowsStdCallTools(t)
	headers := windowsStdCallCommand(t, readobj, "--file-headers", pkg.ArchiveFile)
	formats, machines := 0, 0
	for _, line := range strings.Split(headers, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Format:") {
			formats++
			if line != "Format: COFF-i386" {
				t.Fatal("the complete math archive contains a non-386 object", line)
			}
		}
		if strings.HasPrefix(line, "Machine:") {
			machines++
			if line != "Machine: IMAGE_FILE_MACHINE_I386 (0x14C)" {
				t.Fatal("the complete math archive has the wrong machine", line)
			}
		}
	}
	if formats == 0 || machines != formats {
		t.Fatal("the complete math archive lacks actual I386 COFF member headers", headers)
	}
	nmTool := filepath.Join(filepath.Dir(readobj), "llvm-nm")
	if runtime.GOOS == "windows" {
		nmTool += ".exe"
	}
	if version := windowsStdCallCommand(t, nmTool, "--version"); !strings.Contains(version, "version 22.") {
		t.Fatal("the actual math archive requires LLVM22 llvm-nm", nmTool, version)
	}
	objects, err := nm.New(nmTool).List(pkg.ArchiveFile, "--defined-only", "--extern-only")
	if err != nil {
		t.Fatal("inspect the complete SDK math archive definitions", err)
	}
	for _, name := range names {
		definitions := 0
		for _, object := range objects {
			for _, symbol := range object.Symbols {
				if symbol.Name == "_math."+name && symbol.Type == nm.Text && symbol.FAddr {
					definitions++
				}
			}
		}
		if definitions != 1 {
			t.Fatal("the actual math archive lacks its unique assembly definition", name, definitions)
		}
	}
}
