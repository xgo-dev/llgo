//go:build !llgo
// +build !llgo

package cl_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func compileSource(t *testing.T, src, file string) string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	files := []*ast.File{f}
	pkg := types.NewPackage(f.Name.Name, f.Name.Name)
	imp := packages.NewImporter(fset)
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	foo, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, pkg, files, mode)
	if err != nil {
		t.Fatalf("BuildPackage failed: %v", err)
	}

	prog := ssatest.NewProgramEx(t, nil, imp)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	ret, err := cl.NewPackage(prog, foo, files)
	if err != nil {
		t.Fatalf("cl.NewPackage failed: %v", err)
	}
	return ret.String()
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, data := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

func mustPanicContains(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		msg := fmt.Sprint(r)
		if !strings.Contains(msg, want) {
			t.Fatalf("panic = %q, want substring %q", msg, want)
		}
	}()
	fn()
}

func TestCompileEmbedInitPaths(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	writeFiles(t, dir, map[string]string{
		"assets/hello.txt": "Hello, Embed!",
	})

	src := `package foo

import "embed"

//go:embed assets/hello.txt
var text string

//go:embed assets/hello.txt
var bytes []byte

//go:embed assets
var files embed.FS

func use() int { return len(text) + len(bytes) }
`

	ir := compileSource(t, src, mainFile)
	if !strings.Contains(ir, "assets/hello.txt") {
		t.Fatalf("missing embedded fs entry in IR:\n%s", ir)
	}
	if !strings.Contains(ir, "Hello, Embed!") {
		t.Fatalf("missing embedded file content in IR:\n%s", ir)
	}
}

func TestCompileEmbedStringMustResolveSingleFile(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	writeFiles(t, dir, map[string]string{
		"assets/a.txt": "a",
		"assets/b.txt": "b",
	})

	src := `package foo

import _ "embed"

//go:embed assets/*.txt
var text string
`

	mustPanicContains(t, "string variable must resolve to one file", func() {
		compileSource(t, src, mainFile)
	})
}

func TestCompileEmbedBytesMustResolveSingleFile(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	writeFiles(t, dir, map[string]string{
		"assets/a.txt": "a",
		"assets/b.txt": "b",
	})

	src := `package foo

import _ "embed"

//go:embed assets/*.txt
var data []byte
`

	mustPanicContains(t, "[]byte variable must resolve to one file", func() {
		compileSource(t, src, mainFile)
	})
}
