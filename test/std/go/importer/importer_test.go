package importer_test

import (
	"go/importer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultImporter(t *testing.T) {
	imp := importer.Default()
	if imp == nil {
		t.Fatalf("Default returned nil importer")
	}
	if runtime.GOARCH == "wasm" {
		checkWasmImporter(t, imp)
	} else if _, err := imp.Import("fmt"); err != nil {
		t.Fatalf("Default importer failed to import fmt: %v", err)
	}
}

func TestForAndForCompiler(t *testing.T) {
	imp := importer.For("gc", nil)
	if imp == nil {
		t.Fatalf("For(gc,nil) returned nil importer")
	}
	if runtime.GOARCH == "wasm" {
		checkWasmImporter(t, imp)
	} else if _, err := imp.Import("math"); err != nil {
		t.Fatalf("For(gc,nil) failed to import math: %v", err)
	}

	fset := token.NewFileSet()
	imp2 := importer.ForCompiler(fset, "gc", nil)
	if imp2 == nil {
		t.Fatalf("ForCompiler returned nil importer")
	}
	if runtime.GOARCH == "wasm" {
		checkWasmImporter(t, imp2)
	} else if _, err := imp2.Import("strings"); err != nil {
		t.Fatalf("ForCompiler failed to import strings: %v", err)
	}

	// Reference exported Lookup type explicitly.
	var _ importer.Lookup
}

func checkWasmImporter(t *testing.T, imp types.Importer) {
	t.Helper()
	if pkg, err := imp.Import("unsafe"); err != nil || pkg != types.Unsafe {
		t.Fatalf("builtin unsafe import = %v, %v", pkg, err)
	}
	// Default gc archive resolution invokes the host go command. The guest
	// cannot spawn it; this is not an assertion that archive decoding succeeded.
	if _, err := imp.Import("fmt"); err == nil || !strings.Contains(strings.ToLower(err.Error()), "not implemented") {
		t.Fatalf("gc archive resolution = %v, want unavailable host compiler", err)
	}
}

func TestSourceImporterReadsPackage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture")
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "fixture.go"), []byte("package fixture\nconst Answer = 42\ntype Value struct { N int64 }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	imp := importer.ForCompiler(token.NewFileSet(), "source", nil).(types.ImporterFrom)
	pkg, err := imp.ImportFrom("./fixture", dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !pkg.Complete() || pkg.Name() != "fixture" || pkg.Scope().Lookup("Value") == nil {
		t.Fatalf("source importer did not type-check the fixture: %v", pkg)
	}
	answer, ok := pkg.Scope().Lookup("Answer").(*types.Const)
	if !ok || answer.Val().ExactString() != "42" {
		t.Fatalf("source importer lost the exported constant: %v", answer)
	}
	if again, err := imp.ImportFrom("./fixture", dir, 0); err != nil || again != pkg {
		t.Fatalf("source import cache = %v, %v; want original package", again, err)
	}
}
