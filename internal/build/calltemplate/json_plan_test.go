package calltemplate

import (
	"errors"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	gopackages "golang.org/x/tools/go/packages"
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
			addCodecPatch(t, library, "templates_llgo.go")
			registry := templateRegistry(t, library)
			if v2 := library.Imports["encoding/json/v2"]; v2 != nil {
				initial.Imports[v2.PkgPath] = v2
				if used, err := registry.RewriteStructural(pkgs, nil); used.Providers[library.PkgPath] || err != nil {
					t.Fatalf("direct JSON v2 import rewrote calls = %v, %v", used, err)
				}
				delete(initial.Imports, v2.PkgPath)
			}
			used, err := registry.RewriteStructural(pkgs, nil)
			changed := used.Providers[library.PkgPath]
			if err != nil {
				t.Fatal(err)
			}
			if changed != tc.rewrite {
				t.Fatalf("rewrite = %t, want %t", changed, tc.rewrite)
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
	initial := loadCodecInput(t, filepath.Join("..", "testdata", "json-static"))
	library := initial.Imports["encoding/json"]
	addCodecPatch(t, library, "templates_llgo.go")
	rule := templateRegistry(t, library).structural[0].entries["Marshal"].body
	deps := templateDeps(initial)
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"basic", true}, {"named", true}, {"record", true}, {"array", true}, {"pointer", true},
		{"text", false}, {"graph", false}, {"bytes", false}, {"duplicate", false},
		{"tagged", false}, {"embedded", false}, {"floating", false}, {"dynamic", false}, {"wrongKey", false},
		{"standardNumber", false}, {"unsupportedField", false}, {"invalidTag", false}, {"omittedOptionalFields", true},
	} {
		gen := newExpansion(initial, deps, "test")
		ctx := expansionContext{g: gen, rule: rule}
		_, err := ctx.Emit("encode", pkg.Scope().Lookup(tc.name).Type())
		if err != nil && !errors.Is(err, errExpansionRejected) {
			t.Fatal(err)
		}
		if got := err == nil; got != tc.want {
			t.Errorf("%s: eligible %v, want %v", tc.name, got, tc.want)
		}
	}
}
