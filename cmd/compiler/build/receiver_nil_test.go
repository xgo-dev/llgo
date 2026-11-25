package build

import (
	"github.com/xgo-dev/llgo/types"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/cmd/cl"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

func TestCollectReceiverNilChecksAcrossPackages(t *testing.T) {
	fset := token.NewFileSet()
	checked := func(path string) *packages.Package {
		t.Helper()
		file, err := parser.ParseFile(fset, path+".go", `package p
type T struct{}
func (*T) M() {}
func Invoke[A any](p *T, _ A) { (*p).M() }
`, 0)
		if err != nil {
			t.Fatal(err)
		}
		info := &types.Info{Selections: make(map[*ast.SelectorExpr]*types.Selection)}
		typ, err := new(types.Config).Check(path, fset, []*ast.File{file}, info)
		if err != nil {
			t.Fatal(err)
		}
		return &packages.Package{ID: path, PkgPath: path, Types: typ, TypesInfo: info, Syntax: []*ast.File{file}}
	}
	dependency := checked("dep")
	overlay := checked("overlay")
	root := &packages.Package{ID: "root", Imports: map[string]*packages.Package{"dep": dependency}}
	got := collectReceiverNilChecks([]*packages.Package{root}, []*packages.Package{overlay, dependency})
	files := append(append([]*ast.File{}, dependency.Syntax...), overlay.Syntax...)
	want := cl.CollectReceiverNilChecks(files, dependency.TypesInfo, overlay.TypesInfo)
	if want == nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("shared receiver metadata = %#v, want dependency and overlay selections %#v", got, want)
	}
	if collectReceiverNilChecks() != nil {
		t.Fatal("empty package graph must not invent receiver metadata")
	}
}
