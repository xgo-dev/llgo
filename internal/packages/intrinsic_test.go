package packages

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gopackages "golang.org/x/tools/go/packages"
)

func TestLLVMIntrinsicDeclarationsLoadAcrossPackages(t *testing.T) {
	dir := t.TempDir()
	writeLoadTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/intrinsics\ngo 1.24\n")
	if err := os.Mkdir(filepath.Join(dir, "bits"), 0755); err != nil {
		t.Fatal(err)
	}
	writeLoadTestFile(t, filepath.Join(dir, "bits", "bits.go"), `package bits
//llgo:link Count llvm.ctpop
func Count(x uint32) uint32
`)
	writeLoadTestFile(t, filepath.Join(dir, "main.go"), `package main
import "example.com/intrinsics/bits"
func main() { println(bits.Count(7)) }
`)
	pkgs, err := LoadEx(NewDeduper(), nil, loadTestConfig(dir), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range []*Package{pkgs[0], pkgs[0].Imports["example.com/intrinsics/bits"]} {
		if pkg.IllTyped || len(pkg.Errors) != 0 {
			t.Fatalf("%s remains ill-typed: %v", pkg.PkgPath, pkg.Errors)
		}
	}
}

func TestLLVMIntrinsicDeclarationsKeepUnrelatedDiagnostics(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "/src/p/decl.go", `package p
//llgo:link Count llvm.ctpop
func Count(x uint32) uint32
func Missing()
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &Package{
		Syntax: []*ast.File{file},
		Errors: []gopackages.Error{{Kind: gopackages.ListError, Msg: "# p\n./decl.go:3:6: missing function body\n./decl.go:4:6: missing function body\n./decl.go:5:1: undefined: bad"}},
	}
	allowLLVMIntrinsicDeclarations(pkg, fset)
	if len(pkg.Errors) != 1 || strings.Contains(pkg.Errors[0].Msg, "3:6") ||
		!strings.Contains(pkg.Errors[0].Msg, "4:6: missing function body") || !strings.Contains(pkg.Errors[0].Msg, "undefined: bad") {
		t.Fatalf("unexpected filtered diagnostics: %v", pkg.Errors)
	}
}
