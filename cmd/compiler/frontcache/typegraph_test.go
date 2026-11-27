package frontcache

import (
	"encoding/json"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
	"go/ast"
	"go/parser"
	"go/token"
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

func TestTypeGraphAnonymousInterfaceWithMethods(t *testing.T) {
	// Anonymous interfaces with explicit methods carry a synthetic self-receiver
	// on each method signature (matching crypto/hpke type assertions). Round-tripping
	// through TypeGraph must not trip the anonymous cycle guard.
	prog, pkg, index := fixture(t, `package p
func F(x any) ([]byte, bool) {
	if v, ok := x.(interface {
		Bytes() []byte
		Equal(other any) bool
	}); ok {
		return v.Bytes(), v.Equal(x)
	}
	return nil, false
}
`, true)
	var anon *types.Interface
	for _, tv := range pkg.Pkg.Scope().Names() {
		_ = tv
	}
	// Construct the anonymous interface via NewInterfaceType (same as type-checked AST).
	m1 := types.NewFunc(token.NoPos, pkg.Pkg, "Bytes", types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, pkg.Pkg, "", types.NewSlice(types.Typ[types.Byte]))), false))
	m2 := types.NewFunc(token.NoPos, pkg.Pkg, "Equal", types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, pkg.Pkg, "other", types.Universe.Lookup("any").Type())),
		types.NewTuple(types.NewParam(token.NoPos, pkg.Pkg, "", types.Typ[types.Bool])), false))
	anon = types.NewInterfaceType([]*types.Func{m1, m2}, nil).Complete()
	if anon.ExplicitMethod(0).Type().(*types.Signature).Recv().Type() != anon {
		t.Fatal("expected NewInterfaceType to bind self-receiver on explicit method")
	}

	graph := NewTypeGraph(index, prog.Fset)
	id := graph.Add(anon)
	data, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	var decoded TypeGraph
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	fresh := types.NewPackage(pkg.Pkg.Path(), pkg.Pkg.Name())
	importer := &TypeImporter{
		Graph:    &decoded,
		Packages: map[string]*types.Package{fresh.Path(): fresh},
		Existing: NewTypes(map[*types.Package]string{fresh: fresh.Path()}, prog.Fset),
	}
	got, err := importer.Import(id)
	if err != nil {
		t.Fatalf("importing anonymous interface with explicit methods failed: %v", err)
	}
	gotIface, ok := got.(*types.Interface)
	if !ok || !types.Identical(gotIface, anon) {
		t.Fatalf("imported type = %v, want %v", got, anon)
	}
	if recv := gotIface.ExplicitMethod(0).Type().(*types.Signature).Recv(); recv == nil || recv.Type() != gotIface {
		t.Fatalf("imported explicit method lost self-receiver binding: %v", recv)
	}
}

func TestOwnerPromotedWrapperUsesReceiverPackage(t *testing.T) {
	fset := token.NewFileSet()
	fileA, err := parser.ParseFile(fset, "a.go", `package a
type Writer interface { Write([]byte) (int, error) }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	infoA := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object), Selections: make(map[*ast.SelectorExpr]*types.Selection)}
	pkgA, err := new(types.Config).Check("example/a", fset, []*ast.File{fileA}, infoA)
	if err != nil {
		t.Fatal(err)
	}
	fileB, err := parser.ParseFile(fset, "b.go", `package b
import "example/a"
type writerOnly struct { a.Writer }
func Wrap(w a.Writer) a.Writer { return writerOnly{w} }
func Thunk() func(*writerOnly, []byte) (int, error) { return (*writerOnly).Write }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	infoB := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object), Selections: make(map[*ast.SelectorExpr]*types.Selection)}
	pkgB, err := (&types.Config{Importer: staticImporter{pkgA.Path(): pkgA}}).Check("example/b", fset, []*ast.File{fileB}, infoB)
	if err != nil {
		t.Fatal(err)
	}
	prog := ssa.NewProgram(fset, ssa.InstantiateGenerics)
	spA := prog.CreatePackage(pkgA, []*ast.File{fileA}, infoA, true)
	spB := prog.CreatePackage(pkgB, []*ast.File{fileB}, infoB, true)
	spA.Build()
	spB.Build()

	woType := pkgB.Scope().Lookup("writerOnly").Type()
	mset := types.NewMethodSet(types.NewPointer(woType))
	sel := mset.Lookup(pkgB, "Write")
	if sel == nil {
		t.Fatal("missing promoted Write method on *writerOnly")
	}
	wrapper := prog.MethodValue(sel)
	if got := Owner(wrapper); got != pkgB {
		t.Fatalf("Owner(%s) = %v, want %v", wrapper, got, pkgB)
	}
	thunkFn := spB.Func("Thunk")
	var foundThunk bool
	for _, blk := range thunkFn.Blocks {
		for _, instr := range blk.Instrs {
			if ret, ok := instr.(*ssa.Return); ok && len(ret.Results) == 1 {
				if fn, ok := ret.Results[0].(*ssa.Function); ok {
					foundThunk = true
					if got := Owner(fn); got != pkgB {
						t.Fatalf("Owner(%s) = %v, want %v", fn, got, pkgB)
					}
				}
			}
		}
	}
	if !foundThunk {
		t.Fatal("failed to locate method-expression thunk in Thunk()")
	}

	// Local struct embedding an imported interface (like mime/multipart.writerOnly):
	// FunctionID must remain identical after round-tripping through TypeGraph
	// even when neither package's source files are present in the warm FileSet.
	fileC, err := parser.ParseFile(fset, "c.go", `package c
import "example/a"
func WrapLocal(w a.Writer) a.Writer {
	type localWriter struct { a.Writer }
	return localWriter{w}
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	infoC := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object), Selections: make(map[*ast.SelectorExpr]*types.Selection)}
	pkgC, err := (&types.Config{Importer: staticImporter{pkgA.Path(): pkgA}}).Check("example/c", fset, []*ast.File{fileC}, infoC)
	if err != nil {
		t.Fatal(err)
	}
	spC := prog.CreatePackage(pkgC, []*ast.File{fileC}, infoC, true)
	spC.Build()

	var localPtrType types.Type
	for _, rt := range prog.RuntimeTypes() {
		if ptr, ok := rt.(*types.Pointer); ok {
			if named, ok := ptr.Elem().(*types.Named); ok && named.Obj().Name() == "localWriter" {
				localPtrType = ptr
				break
			}
		}
	}
	if localPtrType == nil {
		t.Fatal("missing *localWriter in RuntimeTypes")
	}
	coldSel := types.NewMethodSet(localPtrType).Lookup(pkgC, "Write")
	coldWrapper := prog.MethodValue(coldSel)
	coldTypes := NewTypes(map[*types.Package]string{pkgA: pkgA.Path(), pkgC: pkgC.Path()}, fset)
	coldTypes.AddPackage(pkgA, infoA)
	coldTypes.AddPackage(pkgC, infoC)
	coldID := FunctionID(coldWrapper, coldTypes)
	if got := Owner(coldWrapper); got != pkgC {
		t.Fatalf("cold Owner(localWriter.Write) = %v, want %v", got, pkgC)
	}

	graph := NewTypeGraph(coldTypes, fset)
	ref := graph.Add(localPtrType)
	rawJSON, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	var decoded TypeGraph
	if err := json.Unmarshal(rawJSON, &decoded); err != nil {
		t.Fatal(err)
	}
	// Simulate warm build with an empty FileSet (source files skipped parse).
	warmFset := token.NewFileSet()
	warmPkgA := types.NewPackage(pkgA.Path(), pkgA.Name())
	warmPkgC := types.NewPackage(pkgC.Path(), pkgC.Name())
	warmTypes := NewTypes(map[*types.Package]string{warmPkgA: warmPkgA.Path(), warmPkgC: warmPkgC.Path()}, warmFset)
	importer := &TypeImporter{
		Graph:    &decoded,
		Packages: map[string]*types.Package{warmPkgA.Path(): warmPkgA, warmPkgC.Path(): warmPkgC},
		Existing: warmTypes,
	}
	restored, err := importer.Import(ref)
	if err != nil {
		t.Fatalf("import *localWriter failed: %v", err)
	}
	warmProg := ssa.NewProgram(warmFset, ssa.InstantiateGenerics)
	warmProg.CreatePackage(warmPkgA, nil, nil, true)
	warmProg.CreatePackage(warmPkgC, nil, nil, true)
	warmSel := types.NewMethodSet(restored).Lookup(warmPkgC, "Write")
	warmWrapper := warmProg.MethodValue(warmSel)
	warmID := FunctionID(warmWrapper, warmTypes)
	if warmID != coldID {
		t.Fatalf("warm FunctionID mismatch:\n  warm: %s\n  cold: %s", warmID, coldID)
	}
}

type staticImporter map[string]*types.Package

func (m staticImporter) Import(path string) (*types.Package, error) {
	return m[path], nil
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
