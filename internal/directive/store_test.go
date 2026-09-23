package directive

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func parseSource(t *testing.T, text string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "directives.go", text, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return fset, file
}

func TestGroupCompatibility(t *testing.T) {
	tests := []struct {
		name, comments        string
		noinline, nointerface bool
		bg                    string
		skip                  []string
	}{
		{name: "noinline accepts arguments", comments: "//go:noinline ignored", noinline: true},
		{name: "spaced go is not a function directive", comments: "// go:noinline"},
		{name: "nointerface trailing go annotations", comments: "//go:nointerface\n//go:other", nointerface: true},
		{name: "nointerface stops at ordinary comment", comments: "//go:nointerface\n// ordinary"},
		{name: "nointerface is exact", comments: "//go:nointerface "},
		{name: "type only last line", comments: "//llgo:type C\n// ordinary"},
		{name: "spaced type", comments: "// llgo:type stdcall", bg: "stdcall"},
		{name: "type separator stays space", comments: "//llgo:type\tC"},
		{name: "skip stops at ordinary comment", comments: "//llgo:skip old\n// ordinary\n// llgo:skip new one\n//go:other", skip: []string{"new", "one"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Construct comments directly to retain trailing whitespace that go/parser
			// may normalize; the source parser/CRLF path is covered separately below.
			doc := &ast.CommentGroup{}
			for _, line := range strings.Split(tt.comments, "\n") {
				doc.List = append(doc.List, &ast.Comment{Text: line})
			}
			g := new(Store).Group(doc)
			if g.Function.NoInline != tt.noinline || g.NoInterface != tt.nointerface || g.TypeBackground != tt.bg || !reflect.DeepEqual(g.Skip.Names, tt.skip) {
				t.Fatalf("group = %+v", g)
			}
		})
	}
}

func TestPackageAssociationAndFrozenSnapshots(t *testing.T) {
	_, file := parseSource(t, strings.ReplaceAll(`package p
import _ "unsafe"
//llgo:env
//go:noinline
//go:nosplit
//go:uintptrescapes
//go:wasmimport old name
//go:wasmimport malformed
func env() {}
func plain() {}
//go:linkname env shared.symbol
//go:linkname plain shared.symbol
//go:cgo_ldflag "-lm"
//go:cgo_import_dynamic local remote "libc"
`, "\n", "\r\n"))
	store := new(Store)
	f := store.File(file)
	p := Collect([]*File{f}, false, false)
	env := p.Functions[file.Decls[1].(*ast.FuncDecl)]
	plain := p.Functions[file.Decls[2].(*ast.FuncDecl)]
	if !env.ClosureEnv || !env.NoInline || !env.NoSplit || !env.UintptrEscapes || env.WasmImport != nil {
		t.Fatalf("env = %+v", env)
	}
	if plain.ClosureEnv || plain.NoInline || plain.Linkname != env.Linkname || env.Linkname != "shared.symbol" {
		t.Fatalf("aliased declarations lost identity: %+v / %+v", env, plain)
	}
	if !reflect.DeepEqual(f.Cgo.LDFlags, []string{"-lm"}) || !reflect.DeepEqual(f.Cgo.Imports, []DynamicImport{{"local", "remote"}}) {
		t.Fatalf("cgo = %+v", f.Cgo)
	}
	store.Freeze()
	// Mutating/removing the original comments must not change any subsequent
	// consumer. A consumer that reparses comments would observe the poison text.
	for _, g := range file.Comments {
		for _, c := range g.List {
			c.Text = "//ordinary"
		}
	}
	file.Comments = nil
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if store.File(file) != f || !store.Function(env.Source).NoInline || f.Cgo.Imports[0].Alias != "remote" {
					t.Error("snapshot changed")
				}
			}
		}()
	}
	wg.Wait()
}

func TestFreezeRejectsDiscovery(t *testing.T) {
	_, file := parseSource(t, "package p\n")
	store := new(Store)
	store.Freeze()
	defer func() {
		if recover() == nil {
			t.Error("late discovery accepted")
		}
	}()
	store.File(file)
}

func TestBindReceiverAliasAndPackageVariants(t *testing.T) {
	fset, file := parseSource(t, `package p
type T struct{}
type Alias = T
//llgo:link Alias.M C.m
//go:nointerface
func (Alias) M() {}
`)
	var variants []*Package
	var methods []*types.Func
	for i := 0; i < 2; i++ {
		info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
		pkg, err := new(types.Config).Check("example.com/p", fset, []*ast.File{file}, info)
		if err != nil {
			t.Fatal(err)
		}
		records := Collect(new(Store).Files([]*ast.File{file}), false, false)
		records.Bind(info)
		method := pkg.Scope().Lookup("T").Type().(*types.Named).Method(0)
		r, ok := records.Objects[method].(*FunctionDecl)
		if !ok || !r.NoInterface || r.Linkname != "C.m" {
			t.Fatalf("method record = %+v", r)
		}
		variants = append(variants, records)
		methods = append(methods, method)
	}
	if _, ok := variants[0].Objects[methods[1]]; ok {
		t.Fatal("test package variants share object identity")
	}
}

func TestEmbedAndCgoDialectsStayIndependent(t *testing.T) {
	_, file := parseSource(t, `package p
/*
 * go:cgo_import_dynamic local remote
 */
// go:embed "a b" c
var data string
// llgo:env
func f() {}
`)
	r := new(Store).File(file)
	g := r.Group(file.Decls[0].(*ast.GenDecl).Doc)
	if !g.Embed.Present || !reflect.DeepEqual(g.Embed.Patterns, []string{"a b", "c"}) || g.Has("go:embed") {
		t.Fatalf("embed dialect = %+v", g)
	}
	if len(r.Cgo.Imports) != 1 || !r.Functions[file.Decls[1].(*ast.FuncDecl)].ClosureEnv {
		t.Fatalf("file = %+v", r)
	}
}
