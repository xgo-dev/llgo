package frontcache

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/ssa"
	"strconv"
	"strings"
	"testing"
)

func TestTypeGraphPreservesPrivateRecursiveAndGenericTypes(t *testing.T) {
	const source = `package p
type private struct{ N *private; hidden int; Tag string ` + "`json:\"tag\"`" + ` }
func (*private) hiddenMethod(x int) string{return ""}
type Box[T any]struct { Next *Box[T]; Value T }
func (b Box[T]) Get() T{return b.Value}
type Alias[T any] = Box[T]
var A *private
var B Box[int]
var C Alias[string]
type I interface { hiddenMethod(int) string }
func F(a I, b ...string)(int,error){return 0,nil}
`
	prog, pkg, index := fixture(t, source, true)
	graph := NewTypeGraph(index, prog.Fset)
	roots := make(map[string]int)
	for _, name := range pkg.Pkg.Scope().Names() {
		roots[name] = graph.Add(pkg.Pkg.Scope().Lookup(name).Type())
	}
	data, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	var decoded TypeGraph
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	fresh := types.NewPackage(pkg.Pkg.Path(), pkg.Pkg.Name())
	bindings := NewTypes(map[*types.Package]string{fresh: fresh.Path()}, prog.Fset)
	importer := &TypeImporter{Graph: &decoded, Packages: map[string]*types.Package{fresh.Path(): fresh}, Position: func(key string) token.Pos {
		at := strings.LastIndexByte(key, ':')
		if at < 0 {
			return token.NoPos
		}
		offset, err := strconv.Atoi(key[at+1:])
		if err != nil {
			return token.NoPos
		}
		var pos token.Pos
		prog.Fset.Iterate(func(file *token.File) bool {
			if file.Name() == key[:at] && offset <= file.Size() {
				pos = file.Pos(offset)
				return false
			}
			return true
		})
		return pos
	}}
	for name, id := range roots {
		typ, err := importer.Import(id)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		original := pkg.Pkg.Scope().Lookup(name).Type()
		if index.Key(original) != bindings.Key(typ) {
			t.Fatalf("%s: type identity changed:\n%v\n%v", name, original, typ)
		}
		if types.NewMethodSet(original).Len() != types.NewMethodSet(typ).Len() {
			t.Fatalf("%s: methods lost", name)
		}
		if named, ok := typ.(*types.Named); ok && named.Obj().Pkg() != fresh {
			t.Fatalf("%s: old package identity retained", name)
		}
	}
	alias, err := importer.Import(roots["Alias"])
	if err != nil || alias.(*types.Alias).TypeParams().Len() != 1 {
		t.Fatalf("generic alias parameters lost: %v %v", alias, err)
	}
	function, err := importer.Import(roots["F"])
	if err != nil || !types.Identical(function.(*types.Signature).Results().At(1).Type(), types.Universe.Lookup("error").Type()) {
		t.Fatalf("predeclared error identity lost: %v %v", function, err)
	}
}

func TestTypeGraphRejectsCorruptReferences(t *testing.T) {
	for _, graph := range []*TypeGraph{
		{Nodes: []TypeNode{{Kind: "pointer", Elem: 0}}},
		{Nodes: []TypeNode{{Kind: "pointer", Elem: 100}}},
		{Nodes: []TypeNode{{Kind: "basic", Basic: -1}}},
	} {
		p := &TypeImporter{Graph: graph}
		if _, err := p.Import(0); err == nil {
			t.Fatal("invalid type accepted")
		}
		if _, err := p.Import(0); err == nil {
			t.Fatal("partially imported type escaped failed transaction")
		}
	}
}

func fixture(t *testing.T, source string, build bool) (*ssa.Program, *ssa.Package, *Types) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object), Selections: make(map[*ast.SelectorExpr]*types.Selection), Instances: make(map[*ast.Ident]types.Instance)}
	pkg, err := new(types.Config).Check("example/p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	prog := ssa.NewProgram(fset, ssa.InstantiateGenerics)
	sp := prog.CreatePackage(pkg, []*ast.File{file}, info, true)
	if build {
		sp.Build()
	}
	index := NewTypes(map[*types.Package]string{pkg: pkg.Path()}, fset)
	index.AddPackage(pkg, info)
	return prog, sp, index
}
