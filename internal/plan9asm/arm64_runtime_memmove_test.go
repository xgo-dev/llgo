package plan9asm

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llvm"
)

type arm64MemmoveRuntimeImporter struct{ source *types.Package }

func (i arm64MemmoveRuntimeImporter) Import(path string) (*types.Package, error) {
	if path == "runtime" {
		return i.source, nil
	}
	return importer.Default().Import(path)
}

func actualARM64MemmoveRuntime(t *testing.T) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(runtime.GOROOT(), "src/runtime/stubs.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var declaration bytes.Buffer
	for _, node := range file.Decls {
		if fn, ok := node.(*ast.FuncDecl); ok && fn.Name.Name == "memmove" {
			if err := format.Node(&declaration, fset, fn); err != nil {
				t.Fatal(err)
			}
		}
	}
	if declaration.Len() == 0 {
		t.Fatal("selected SDK has no memmove declaration")
	}
	syntax, err := parser.ParseFile(fset, "runtime.go", "package runtime\nimport \"unsafe\"\n"+declaration.String(), 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{Importer: importer.Default(), Sizes: types.SizesFor("gc", "arm64")}).Check("runtime", fset, []*ast.File{syntax}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func TestTranslateARM64RuntimeMemmoveABI0(t *testing.T) {
	directory, err := filepath.Abs("../build/testdata/plan9asm-memmove-arm64")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	source := filepath.Join(directory, "main.go")
	syntax, err := parser.ParseFile(fset, source, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	sizes := types.SizesFor("gc", "arm64")
	typed, err := (&types.Config{Sizes: sizes, Importer: arm64MemmoveRuntimeImporter{actualARM64MemmoveRuntime(t)}}).Check("example.com/arm64memmove", fset, []*ast.File{syntax}, nil)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &packages.Package{PkgPath: typed.Path(), Dir: directory, Types: typed, TypesSizes: sizes, Fset: fset, Syntax: []*ast.File{syntax}, GoFiles: []string{source}, CompiledGoFiles: []string{source}, OtherFiles: []string{filepath.Join(directory, "copy_arm64.s")}, Imports: map[string]*packages.Package{}}
	for _, imp := range typed.Imports() {
		pkg.Imports[imp.Path()] = &packages.Package{PkgPath: imp.Path(), Types: imp}
	}
	for _, goos := range []string{"linux", "darwin", "windows"} {
		t.Run(goos, func(t *testing.T) {
			assembly, err := os.ReadFile(filepath.Join(directory, "copy_arm64.s"))
			if err != nil {
				t.Fatal(err)
			}
			module, err := TranslateSourceModuleForPkgWithOptions(pkg, filepath.Join(directory, "copy_arm64.s"), assembly, goos, "arm64", TranslateOptions{GOROOT: runtime.GOROOT()})
			if err != nil {
				t.Fatal(err)
			}
			defer module.Module.Dispose()
			sig := module.Signatures["runtime.memmove"]
			if sig.Name != "runtime.memmove" || sig.Ret != "void" || len(sig.Frame.Params) != 3 || strings.Contains(module.Module.String(), "@memmove(") {
				t.Fatalf("lost selected Go ABI0 declaration: %+v\n%s", sig, module.Module.String())
			}
			if err := llvm.VerifyModule(module.Module, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
	delete(pkg.Imports, "runtime")
	if _, err := TranslateFileForPkgWithOptions(pkg, filepath.Join(directory, "copy_arm64.s"), "linux", "arm64", nil, TranslateOptions{GOROOT: runtime.GOROOT()}); err == nil || !strings.Contains(err.Error(), "selected Go source declaration") {
		t.Fatalf("missing private declaration must fail before emitting an invented call: %v", err)
	}
}
