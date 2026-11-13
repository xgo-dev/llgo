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
var D map[string][2]chan<- *private
type I interface { hiddenMethod(int) string }
func F(a I, b ...string)(int,error){return 0,nil}
func G[T interface{~int | ~float64}](x T) T{return x}
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
		{Nodes: []TypeNode{{Kind: "unknown"}}},
		{Nodes: []TypeNode{{Kind: "array", Length: -1}}},
		{Nodes: []TypeNode{{Kind: "chan", Direction: -1}}},
		{Nodes: []TypeNode{{Kind: "named", Universe: true, Object: Object{Name: "missing"}}}},
		{Nodes: []TypeNode{{Kind: "union", Types: []int{1}}}},
		{Nodes: []TypeNode{{Kind: "struct", Fields: []Object{{Package: "missing"}}}}},
		{Nodes: []TypeNode{{Kind: "func", Params: []int{1}, Parameters: []Object{{Type: 1}, {Type: 1}}}}},
		{Nodes: []TypeNode{{Kind: "func", Params: []int{1}, Parameters: []Object{{Type: 0}}}}},
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

func TestTypeGraphTupleAndDependencyIdentity(t *testing.T) {
	prog, pkg, index := fixture(t, `package p
type N int
func F(x int) (n N, err error) { return 0, nil }
`, false)
	graph := NewTypeGraph(index, prog.Fset)
	if graph.Add(nil) != -1 {
		t.Fatal("nil type acquired a graph identity")
	}
	sig := pkg.Pkg.Scope().Lookup("F").Type().(*types.Signature)
	id := graph.Add(sig.Results())
	// A dependency's named types must reuse its checked package, rather than
	// inventing another N with the same printed name.
	importer := &TypeImporter{Graph: graph, Existing: index, Packages: map[string]*types.Package{pkg.Pkg.Path(): pkg.Pkg}}
	typ, err := importer.Import(id)
	if err != nil || !types.Identical(typ, sig.Results()) {
		t.Fatalf("dependency result types changed: %v, %v", typ, err)
	}
	if typ.(*types.Tuple).At(0).Name() != "n" {
		t.Fatal("tuple field name changed")
	}
	// Older anonymous signature records contain no parameter-name metadata.
	graph.Nodes[id].Elements = nil
	importer = &TypeImporter{Graph: graph, Existing: index}
	if typ, err := importer.Import(id); err != nil || !types.Identical(typ, sig.Results()) {
		t.Fatalf("anonymous tuple changed: %v, %v", typ, err)
	}
}

func TestTypeIdentityRejectsAmbiguousPackages(t *testing.T) {
	index := NewTypes(nil)
	var key string
	for range 2 {
		pkg := types.NewPackage("example/p", "p")
		named := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "T", nil), types.Typ[types.Int], nil)
		key = index.Key(named)
		index.Add(named)
	}
	if _, ok := index.Lookup(key); ok || index.Bind(key, types.Typ[types.Int]) {
		t.Fatal("ambiguous named identity was reused")
	}
	if index.Key(nil) != "" || FunctionID(nil, index) != "" || Owner(nil) != nil {
		t.Fatal("missing identity was persisted")
	}
	prog := ssa.NewProgram(token.NewFileSet(), 0)
	synthetic := prog.NewFunction("wrapper", types.NewSignatureType(nil, nil, nil, nil, nil, false), "test")
	if Owner(synthetic) != nil || FunctionID(synthetic, index) == "" {
		t.Fatal("unowned synthetic function identity changed")
	}
	if Position(prog.Fset, token.Pos(1000)) != "" {
		t.Fatal("position outside the source file set was accepted")
	}
}

func TestTypeGraphRejectsConflictingSignatureIdentity(t *testing.T) {
	index := NewTypes(nil)
	index.Add(types.Typ[types.Int])
	graph := &TypeGraph{Nodes: []TypeNode{{Kind: "func", Key: index.Key(types.Typ[types.Int])}}}
	importer := &TypeImporter{Graph: graph, Existing: index}
	if _, err := importer.Import(0); err == nil {
		t.Fatal("signature replaced an existing basic type identity")
	}
	if got, ok := index.Lookup(index.Key(types.Typ[types.Int])); !ok || got != types.Typ[types.Int] {
		t.Fatal("rejected signature corrupted existing type bindings")
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
