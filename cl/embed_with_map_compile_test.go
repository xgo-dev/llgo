//go:build !llgo
// +build !llgo

package cl_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	"github.com/goplus/llgo/cl"
	"github.com/goplus/llgo/internal/goembed"
	"github.com/goplus/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestCompileWithPreloadedEmbedMap(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	writeFiles(t, dir, map[string]string{
		"assets/hello.txt": "Hello, Preloaded!",
	})

	src := `package foo

import "embed"

//go:embed assets
var files embed.FS
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, mainFile, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	files := []*ast.File{f}

	imp := packages.NewImporter(fset)
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	fooPkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, types.NewPackage(f.Name.Name, f.Name.Name), files, mode)
	if err != nil {
		t.Fatalf("BuildPackage failed: %v", err)
	}

	embedMap, err := goembed.LoadDirectives(fset, files)
	if err != nil {
		t.Fatalf("LoadDirectives failed: %v", err)
	}

	prog := ssatest.NewProgramEx(t, nil, imp)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	ret, _, err := cl.NewPackageExWithEmbed(prog, nil, nil, fooPkg, files, embedMap, nil)
	if err != nil {
		t.Fatalf("NewPackageExWithEmbed failed: %v", err)
	}

	ir := ret.String()
	if !strings.Contains(ir, "assets/hello.txt") {
		t.Fatalf("missing embedded fs entry in IR:\n%s", ir)
	}
	if !strings.Contains(ir, "Hello, Preloaded!") {
		t.Fatalf("missing embedded file content in IR:\n%s", ir)
	}
}

func TestCompileWithInvalidEmbedDirectivePanics(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	src := `package foo

import _ "embed"

//go:embed "bad
var text string
`
	mustPanicContains(t, "invalid //go:embed quoted pattern", func() {
		_ = compileSource(t, src, mainFile)
	})
}
