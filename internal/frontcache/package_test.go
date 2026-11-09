package frontcache

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestPackageExportPreservesPrivateTypesConstantsAndNames(t *testing.T) {
	const src = `package p
 type node struct{next *node; Value int "\xff\x00"}
 func (n *node) private(x int) (answer int) {return x}
 type Alias = node
 var Root *node
 const Fraction = 1.0/3
 const Complex = Fraction+2i
 const Big = 1<<200
 const Truth = true
 const Text = "hello\nworld\xff\x00"
 func Read(p *node, count int) (answer int) {return count}
 `
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := new(types.Config).Check("example/p", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	index := NewTypes(map[*types.Package]string{checked: checked.Path()}, fset)
	index.AddPackage(checked, nil)
	data := ExportPackage(checked, index, fset)
	wire, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	data = new(PackageExport)
	if err := json.Unmarshal(wire, data); err != nil {
		t.Fatal(err)
	}
	fresh := types.NewPackage(checked.Path(), checked.Name())
	bindings := NewTypes(map[*types.Package]string{fresh: fresh.Path()}, fset)
	restored, err := ImportPackage(data, &TypeImporter{Packages: map[string]*types.Package{fresh.Path(): fresh}, Existing: bindings, Position: func(string) token.Pos { return token.NoPos }})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range checked.Scope().Names() {
		before, after := checked.Scope().Lookup(name), restored.Scope().Lookup(name)
		if after == nil {
			t.Fatalf("missing %s", name)
		}
		if a, b := index.Key(before.Type()), bindings.Key(after.Type()); a != b {
			t.Fatalf("type %s changed: %s != %s", name, a, b)
		}
		if a, ok := before.(*types.Const); ok {
			b := after.(*types.Const)
			if a.Val().ExactString() != b.Val().ExactString() {
				t.Fatalf("constant %s: %s != %s", name, a.Val(), b.Val())
			}
		}
	}
	sig := restored.Scope().Lookup("Read").Type().(*types.Signature)
	if sig.Params().At(0).Name() != "p" || sig.Params().At(1).Name() != "count" || sig.Results().At(0).Name() != "answer" {
		t.Fatal("declaration parameter names were lost")
	}
	node := restored.Scope().Lookup("node").Type().(*types.Named)
	if node.Underlying().(*types.Struct).Tag(1) != "\xff\x00" {
		t.Fatal("non-UTF-8 struct tag changed")
	}
	if node.NumMethods() != 1 || node.Method(0).Name() != "private" || node.Method(0).Pkg() != restored {
		t.Fatal("private method identity was lost")
	}
	if field := node.Underlying().(*types.Struct).Field(0).Type().(*types.Pointer).Elem(); field != node {
		t.Fatal("recursive identity changed")
	}
}
