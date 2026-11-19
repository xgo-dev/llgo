package jsonspecialize

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

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
			changed, err := Rewrite(closure, library)
			if err != nil {
				t.Fatal(err)
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
					f, _ := initial.TypesInfo.Uses[calleeIdent(c.Fun)].(*types.Func)
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
	`
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "types.go", strings.ReplaceAll(source, "~", "`"), 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{}).Check("example", files, []*ast.File{file}, nil)
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
	} {
		if got := planShape(pkg.Scope().Lookup(tc.name).Type(), &known) != nil; got != tc.want {
			t.Errorf("%s: eligible %v, want %v", tc.name, got, tc.want)
		}
	}
}
