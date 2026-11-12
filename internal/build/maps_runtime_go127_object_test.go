//go:build go1.27

package build

import (
	"bytes"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	llvmenv "github.com/xgo-dev/llgo/xtool/env/llvm"
	"github.com/xgo-dev/llgo/xtool/nm"
	"github.com/xgo-dev/llvm"
)

// Keep the complete selected SDK package and original AES declarations on each
// target. Object definitions are not native AES execution evidence. ModeBuild
// releases the program after the live hook.
func mapsRuntimeCompletePackage(t *testing.T, goos, goarch string) string {
	t.Helper()
	const mapsPackage = "internal/runtime/maps"
	names := []string{"memHashAES", "memHash32AES", "memHash64AES"}
	conf := NewDefaultConf(ModeBuild)
	conf.Goos, conf.Goarch = goos, goarch
	conf.OptLevel, conf.ForceRebuild = optlevel.O0, true
	conf.LTO = lto.Off
	conf.BinPath = t.TempDir()
	modules := 0
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath != mapsPackage {
			return
		}
		modules++
		if pkg.Types == nil || pkg.Types.Path() != mapsPackage {
			t.Fatal("the selected SDK maps module lacks its actual typed package")
		}
		// This live hook observes Go IR, before independent assembly modules
		// are lowered. Inspect original declarations here and object definitions
		// below, rather than demanding assembly bodies in the Go module.
		parameters := [][]types.BasicKind{
			{types.UnsafePointer, types.Uintptr, types.Uintptr},
			{types.Uint32, types.Uintptr},
			{types.Uint64, types.Uintptr},
		}
		for index, name := range names {
			function, ok := pkg.Types.Scope().Lookup(name).(*types.Func)
			if !ok || function.Pkg() != pkg.Types {
				t.Fatal("the actual selected SDK lacks its AES declaration", name)
			}
			signature := function.Type().(*types.Signature)
			if signature.Recv() != nil || signature.Variadic() ||
				signature.Params().Len() != len(parameters[index]) || signature.Results().Len() != 1 ||
				!types.Identical(signature.Results().At(0).Type(), types.Typ[types.Uintptr]) {
				t.Fatal("the actual SDK AES signature changed", name, signature)
			}
			for argument, kind := range parameters[index] {
				if !types.Identical(signature.Params().At(argument).Type(), types.Typ[kind]) {
					t.Fatal("the actual SDK AES argument changed", name, argument, signature)
				}
			}
			declarations := 0
			for _, source := range pkg.Syntax {
				for _, declaration := range source.Decls {
					entry, ok := declaration.(*ast.FuncDecl)
					if !ok || entry.Name.Name != name {
						continue
					}
					declarations++
					if entry.Body != nil {
						t.Fatal("the AES assembly entry was substituted by a Go body", name)
					}
				}
			}
			if declarations != 1 {
				t.Fatal("the selected SDK lacks its unique original AES declaration", name, declarations)
			}
		}
		if err := llvm.VerifyModule(pkg.LPkg.Module(), llvm.ReturnStatusAction); err != nil {
			t.Fatal("the complete selected SDK maps package has invalid LLVM IR", err)
		}
	}
	pkgs, err := Build(Invocation{Args: []string{mapsPackage}, Config: conf})
	if err != nil {
		t.Fatal("compile the complete selected SDK maps package", goos, goarch, err)
	}
	if modules != 1 {
		t.Fatal("the complete SDK maps package was not observed exactly once", modules)
	}
	pkg := findPkg(pkgs, mapsPackage)
	if pkg == nil || pkg.ArchiveFile == "" {
		t.Fatal("the complete SDK maps package did not publish its object archive")
	}
	info, err := os.Stat(pkg.ArchiveFile)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 8 {
		t.Fatal("the complete SDK maps package has no nonempty object archive", err)
	}
	tool := filepath.Join(llvmenv.New("").BinDir(), "llvm-nm")
	if runtime.GOOS == "windows" {
		tool += ".exe"
	}
	var version bytes.Buffer
	if err := runRunnerCommand(commandEnv{environ: os.Environ()}, tool, []string{"--version"},
		runnerDetails{phase: "maps-aes-object-symbols", timeout: time.Minute}, &version, &version); err != nil {
		t.Fatal("the actual maps archive requires LLVM22 llvm-nm", err, version.String())
	}
	if !strings.Contains(version.String(), "version 22.") {
		t.Fatal("the actual maps archive requires LLVM22 llvm-nm", tool, version.String())
	}
	objects, err := nm.New(tool).List(pkg.ArchiveFile, "--defined-only", "--extern-only")
	if err != nil {
		t.Fatal("inspect actual complete SDK maps archive definitions", err)
	}
	for _, name := range names {
		symbolName := mapsPackage + "." + name
		if conf.Goos == "darwin" || conf.Goos == "windows" && conf.Goarch == "386" {
			symbolName = "_" + symbolName
		}
		definitions := 0
		for _, object := range objects {
			for _, symbol := range object.Symbols {
				if symbol.Name == symbolName && symbol.Type == nm.Text && symbol.FAddr {
					definitions++
				}
			}
		}
		if definitions != 1 {
			t.Fatal("the real SDK archive lacks its unique compiled AES code definition", symbolName, definitions)
		}
	}
	return pkg.ArchiveFile
}
