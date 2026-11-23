package calltemplate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"go/version"
	"runtime"
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
		{"scalar comparison", "var a = 1; var b = a == 1", true},
		{"copy string as bytes", "var s = \"text\"; var b = []byte(s)", true},
		{"length of private slice", "var a = [2]int{}; var b = a[:]; var n = len(b)", true},
		{"fixed index", "var a = [2]int{}; var p = &a[1]", true},
		{"slice with unknown backing array", "var a = [2]int{}; var b = a[:]; var c = b[:]", false},
		{"unmodeled builtin", "var a = [2]int{}; var b = a[:]; var n = cap(b)", false},
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

func TestJSONExportGuard(t *testing.T) {
	const guard = `if p != &allow { panic("unauthorized") }; `
	for _, test := range []struct {
		name, body, token string
		want              bool
	}{
		{"shared token", guard + "return nil", "&allow", true},
		{"different token", guard + "return nil", "&other", false},
		{"missing guard", "return nil", "&allow", false},
		{"different guard", "if p == &allow { return nil }; panic(0)", "&allow", false},
		{"write before guard", "other = 1; " + guard + "return nil", "&allow", false},
		{"write after guard", guard + "other = 1; return nil", "&allow", false},
		{"nonconstant success value", guard + "return &other", "&allow", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			source := `package jsontext
var allow, other int
type exporter struct{}
func (exporter) Export(p *int) *int { ` + test.body + ` }
func example() { exporter{}.Export(` + test.token + `) }
`
			file, err := parser.ParseFile(fset, "export.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			typed := types.NewPackage("encoding/json/jsontext", "jsontext")
			pkg, _, err := ssautil.BuildPackage(&types.Config{}, fset, typed, []*ast.File{file}, ssa.SanityCheckFunctions)
			if err != nil {
				t.Fatal(err)
			}
			call, ok := pkg.Func("example").Blocks[0].Instrs[0].(*ssa.Call)
			if !ok || call.Common().StaticCallee() == nil {
				t.Fatal("test call lacks a static callee")
			}
			if got := jsonExportGuard(call.Common().StaticCallee(), call.Common().Args); got != test.want {
				t.Fatalf("discardable guard = %v, want %v", got, test.want)
			}
		})
	}
}

func TestJSONV2Initializers(t *testing.T) {
	if version.Lang(runtime.Version()) != "go1.27" {
		t.Skip("requires Go 1.27")
	}
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, "encoding/json")
	if err != nil || len(pkgs) != 1 || packages.PrintErrors(pkgs) != 0 {
		t.Fatalf("load standard encoding/json: %v", err)
	}
	if pkgs[0].Imports["encoding/json/v2"] == nil {
		t.Skip("requires the default JSON v2 implementation")
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

func TestStatelessTokenizerHandle(t *testing.T) {
	pkg := types.NewPackage("encoding/json/v2", "json")
	if StatelessTokenizerHandle(pkg) {
		t.Fatal("accepted missing tokenizer handle")
	}
	typ := types.NewStruct(nil, nil)
	pkg.Scope().Insert(types.NewVar(token.NoPos, pkg, "export", typ))
	if !StatelessTokenizerHandle(pkg) {
		t.Fatal("rejected zero-sized tokenizer handle")
	}
	pkg = types.NewPackage("encoding/json/v2", "json")
	state := types.NewField(token.NoPos, pkg, "state", types.Typ[types.Int], false)
	typ = types.NewStruct([]*types.Var{state}, nil)
	pkg.Scope().Insert(types.NewVar(token.NoPos, pkg, "export", typ))
	if StatelessTokenizerHandle(pkg) {
		t.Fatal("accepted a tokenizer handle that requires state")
	}
}
