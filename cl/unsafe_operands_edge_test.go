package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/goplus/gogen/packages"
)

func TestRewriteUnsafeOperandsSkipsUnsupportedForms(t *testing.T) {
	RewriteUnsafeOperands(nil, nil)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", `package foo
import "unsafe"
type S struct { Value int }
func (s S) Method() int { return 0 }
func f(s S) {
	_ = unsafe.Offsetof(s.Method)
	_ = unsafe.Sizeof(s.Value)
	_ = unsafe.Offsetof(s.Value)
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Instances:  make(map[*ast.Ident]types.Instance),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Implicits:  make(map[ast.Node]types.Object),
	}
	conf := types.Config{Importer: packages.NewImporter(fset)}
	_, _ = conf.Check("foo", fset, []*ast.File{file}, info)
	ast.Inspect(file, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok && selector.Sel.Name == "Value" {
			delete(info.Selections, selector)
		}
		return true
	})
	RewriteUnsafeOperands([]*ast.File{file}, info)

	file, err = parser.ParseFile(fset, "pointer.go", `package foo
import "unsafe"
type S[T any] struct { Value T }
func pointer[T any](p *S[T]) uintptr { return unsafe.Offsetof(p.Value) }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info = &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Instances: make(map[*ast.Ident]types.Instance),
		Selections: make(map[*ast.SelectorExpr]*types.Selection), Scopes: make(map[ast.Node]*types.Scope),
		Implicits: make(map[ast.Node]types.Object),
	}
	_, _ = conf.Check("foo", fset, []*ast.File{file}, info)
	ast.Inspect(file, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if selection := info.Selections[selector]; selection != nil && selection.Kind() == types.FieldVal {
				info.Selections[selector] = nil
			}
		}
		return true
	})
	RewriteUnsafeOperands([]*ast.File{file}, info)
}

func TestRewriteUnsafeOperandsSkipsFilesWithoutUnsafe(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{"no import", "package foo\nfunc f() {}"},
		{"blank unsafe", "package foo\nimport _ \"unsafe\"\nfunc f() {}"},
		{"blank unsafe raw string", "package foo\nimport _ `unsafe`\nfunc f() {}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "foo.go", tc.src, 0)
			if err != nil {
				t.Fatal(err)
			}
			// An incomplete declaration makes an accidental AST walk fail.
			file.Decls = append(file.Decls, (*ast.FuncDecl)(nil))
			RewriteUnsafeOperands([]*ast.File{file}, &types.Info{})
		})
	}
}

func TestContainsTypeParamRecursiveAndAlias(t *testing.T) {
	const src = `package p
	type Node struct { Next *Node }
	type Generic[T any] struct { Next *Generic[T]; Value T }
	type Alias[T any] = Generic[T]
	type Phantom[T any] = int
	func f[T any](a Node, b Generic[T], c Generic[int], d Alias[T], e Alias[int], f Phantom[T], g Phantom[int]) {}
	`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "types.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := new(types.Config).Check("p", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	params := pkg.Scope().Lookup("f").Type().(*types.Signature).Params()
	want := []bool{false, true, false, true, false, true, false}
	for i, expected := range want {
		param := params.At(i)
		if got := containsTypeParam(param.Type()); got != expected {
			t.Errorf("%s (%s): got %v, want %v", param.Name(), param.Type(), got, expected)
		}
	}
}
