package jsonspecialize

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/packages"
	gopackages "golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
)

func TestRewriteJSONMarshal(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		rewrite bool
	}{{"json-static", true}, {"json-mixed", false}} {
		t.Run(tc.fixture, func(t *testing.T) {
			pkgs, err := gopackages.Load(&gopackages.Config{
				Mode: gopackages.LoadAllSyntax,
				Dir:  filepath.Join("..", "testdata", tc.fixture),
			}, ".")
			if err != nil || len(pkgs) != 1 || gopackages.PrintErrors(pkgs) != 0 {
				t.Fatalf("load fixture: %v", err)
			}
			initial := pkgs[0]
			library := initial.Imports["encoding/json"]
			var closure []*packages.Package
			packages.Visit(pkgs, nil, func(p *packages.Package) { closure = append(closure, p) })
			for _, exclude := range []func(*packages.Package) bool{
				func(p *packages.Package) bool { return p.PkgPath == "strconv" },
				func(p *packages.Package) bool { return p.PkgPath == "sort" || p.PkgPath == "slices" },
				func(p *packages.Package) bool { return p == initial },
			} {
				incomplete := slices.DeleteFunc(slices.Clone(closure), exclude)
				if changed, err := Rewrite(incomplete, library, initial); len(changed) != 0 || err != nil {
					t.Fatalf("incomplete dependencies produced %d rewritten packages, %v", len(changed), err)
				}
			}
			if v2 := library.Imports["encoding/json/v2"]; v2 != nil {
				initial.Imports[v2.PkgPath] = v2
				if changed, err := Rewrite(closure, library, initial); len(changed) != 0 || err != nil {
					t.Fatalf("direct JSON v2 import produced %d rewritten packages, %v", len(changed), err)
				}
				delete(initial.Imports, v2.PkgPath)
			}
			if changed, err := Rewrite(closure, library, nil); len(changed) != 0 || err != nil {
				t.Fatalf("dependency produced %d rewritten packages, %v", len(changed), err)
			}
			changed, err := Rewrite(closure, library, initial)
			if err != nil {
				t.Fatal(err)
			}
			if len(changed) > 1 || len(changed) == 1 && changed[0] != initial {
				t.Fatal("only the executable package can be rewritten")
			}
			if (len(changed) != 0) != tc.rewrite {
				t.Fatalf("changed %d packages, rewrite expected %v", len(changed), tc.rewrite)
			}
			var generic, typed int
			for _, file := range initial.Syntax {
				ast.Inspect(file, func(n ast.Node) bool {
					c, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					f, _ := typesynth.Callee(initial.TypesInfo, c.Fun)
					if f == nil {
						return true
					}
					if f.Pkg() == library.Types && f.Name() == "Marshal" {
						generic++
					}
					if strings.HasPrefix(f.Name(), "__llgo_json_") {
						typed++
						sig := f.Signature()
						if _, boxed := sig.Params().At(0).Type().Underlying().(*types.Interface); boxed {
							t.Error("generated encoder uses an interface parameter")
						}
					}
					return true
				})
			}
			if tc.rewrite && (typed == 0 || generic != 0) || !tc.rewrite && (typed != 0 || generic == 0) {
				t.Fatalf("generic calls %d, generated calls %d", generic, typed)
			}
		})
	}
}

func TestMarshalShapeEligibility(t *testing.T) {
	const source = `package example
	import "encoding/json"
	type Marshaler string
	func (*Marshaler) MarshalText() ([]byte, error) { return nil, nil }
	type Graph map[string]Graph
	type Embedded struct{ Value int }
	var basic int32
	var named bool
	var record struct { Key []int; Value map[string]bool }
	var array [3]uint8
	var pointer *struct{ Value uint }
	var text Marshaler
	var graph Graph
	var bytes []byte
	var duplicate struct { X int ~json:"same"~; Y int ~json:"same"~ }
	var tagged struct { Value int ~json:",string"~ }
	var embedded struct { Embedded }
	var floating float64
	var dynamic any
	var wrongKey map[int]int
	var standardNumber json.Number
	var unsupportedField struct { Value any }
	var invalidTag struct { Value int ~json:"x\\y"~ }
	var omittedOptionalFields struct {
		Pointer *int ~json:",omitempty"~
		Flag bool ~json:",omitempty"~
		Text string ~json:",omitempty"~
		Number int ~json:",omitempty"~
		Object struct{} ~json:",omitempty"~
	}
	`
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "types.go", strings.ReplaceAll(source, "~", "`"), 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("example", files, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var known typeutil.Map
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"basic", true}, {"named", true}, {"record", true}, {"array", true}, {"pointer", true},
		{"text", false}, {"graph", false}, {"bytes", false}, {"duplicate", false},
		{"tagged", false}, {"embedded", false}, {"floating", false}, {"dynamic", false}, {"wrongKey", false},
		{"standardNumber", false}, {"unsupportedField", false}, {"invalidTag", false},
	} {
		if got := planShape(pkg.Scope().Lookup(tc.name).Type(), &known) != nil; got != tc.want {
			t.Errorf("%s: eligible %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestPlanJSONFallback(t *testing.T) {
	const common = `package p
		import "encoding/json"
	`
	for _, tc := range []struct {
		name, source string
		want         bool
	}{
		{"typed", "func encode() { _, _ = json.Marshal(struct{N int}{1}) }", true},
		{"unknown type parameter", "func encode[T any](v T) { _, _ = json.Marshal(v) }", false},
		{"shadowed builtin", "var append int; func encode() { _, _ = json.Marshal(1) }", false},
		{"function value", "var encode = json.Marshal", false},
		{"streaming", "var _ = json.NewEncoder; var _ = json.NewDecoder", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := token.NewFileSet()
			file, err := parser.ParseFile(files, "p.go", common+tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Uses: make(map[*ast.Ident]types.Object)}
			pkg, err := (&types.Config{Importer: importer.Default()}).Check("example/p", files, []*ast.File{file}, info)
			if err != nil {
				t.Fatal(err)
			}
			dep := pkg.Imports()[0]
			planned := &packages.Package{Types: pkg, TypesInfo: info, Syntax: []*ast.File{file}, Imports: map[string]*packages.Package{"encoding/json": {Types: dep}}}
			_, got := planPackage(planned, dep, false)
			if got != tc.want {
				t.Fatalf("accepted = %v, want %v", got, tc.want)
			}
			if _, ok := planPackage(planned, types.NewPackage(dep.Path(), dep.Name()), false); ok {
				t.Fatal("accepted a different type universe for encoding/json")
			}
		})
	}
}
