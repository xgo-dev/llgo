package jsonspecialize

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestDiscardableJSONInit(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want bool
	}{
		{"private graph", "var a = [2]int{1, 2}; var b = a[:]", true},
		{"alloc and function", "var a = func() *int { return new(int) }()", true},
		{"private append", "var a = []int{1}; var b = append(a, 2)", true},
		{"panic", "func init() { panic(1) }", false},
		{"invalid destination", "var p *int; func init() { *p = 1 }", false},
		{"external call", "var f func(); func init() { f() }", false},
		{"loop", "func init() { for {} }", false},
		{"append can mutate existing storage", "var a = [2]int{}; var b = a[:1]; var c = append(b, 2)", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "init.go", "package json\n"+tc.body, 0)
			if err != nil {
				t.Fatal(err)
			}
			pkg := types.NewPackage("encoding/json", "json")
			ssaPkg, _, err := ssautil.BuildPackage(&types.Config{}, fset, pkg, []*ast.File{file}, ssa.SanityCheckFunctions)
			if err != nil {
				t.Fatal(err)
			}
			if got := DiscardableInit(ssaPkg.Func("init"), func(*types.Package) bool { return true }); got != tc.want {
				t.Fatalf("discardable = %v, want %v", got, tc.want)
			}
			if DiscardableInit(ssaPkg.Func("init"), func(*types.Package) bool { return false }) {
				t.Fatal("accepted untrusted source")
			}
		})
	}
}

func TestJSONV2Initializers(t *testing.T) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, "encoding/json")
	if err != nil || len(pkgs) != 1 || packages.PrintErrors(pkgs) != 0 {
		t.Fatalf("load standard encoding/json: %v", err)
	}
	if pkgs[0].Imports["encoding/json/v2"] == nil || !strings.HasPrefix(pkgs[0].Types.GoVersion(), "go1.27") {
		t.Skip("requires Go 1.27 with the default JSON v2 implementation")
	}
	prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
	prog.Build()
	for _, pkg := range prog.AllPackages() {
		if IsImplementation(pkg.Pkg.Path()) {
			if !DiscardableInit(pkg.Func("init"), func(*types.Package) bool { return true }) {
				t.Errorf("could not verify %s initialization", pkg.Pkg.Path())
			}
		}
	}
}
