package cl_test

import (
	"github.com/xgo-dev/llgo/types"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/cl"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen/ssatest"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
)

// TestNewPackageExWithEmbedSharedTracking compiles with a caller-provided
// CallerTracking, the multi-package driver pattern where one instance is
// shared across every package of a compilation (like patches).
func TestNewPackageExWithEmbedSharedTracking(t *testing.T) {
	src := `package foo

import "runtime"

func Where() string {
	_, file, _, _ := runtime.Caller(0)
	return file
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
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

	prog := ssatest.NewProgramEx(t, nil, imp)
	ct := cl.NewCallerTracking()
	ret, _, err := cl.NewPackageExWithEmbed(prog, ct, nil, nil, fooPkg, files, nil)
	if err != nil {
		t.Fatalf("NewPackageExWithEmbed failed: %v", err)
	}
	if ret.String() == "" {
		t.Fatal("empty IR")
	}
}
