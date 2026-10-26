package midway

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/goembed"

	"github.com/xgo-dev/llgo/internal/directive"
)

func TestRewriteEmbeddedGlobals(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "data.txt"), []byte("embedded data"), 0600); err != nil {
		t.Fatal(err)
	}
	source := `package client
import (_ "embed"; "simd")
type Text string
type Bytes []byte
func (Text) Width() int { var v simd.Uint32s; return v.Len() }
func (Bytes) Width() int { var v simd.Uint32s; return v.Len() }
//go:embed data.txt
var text Text
var (
 //go:embed data.txt
 bytes Bytes
)
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(dir, "main.go"), source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := testInfo()
	cfg := types.Config{Importer: testSIMDImports(t)}
	pkg, err := cfg.Check("client", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	index := new(directive.Index)
	if _, _, err = Rewrite(fset, pkg, info, []*ast.File{file}, index); err != nil {
		t.Fatal(err)
	}
	if _, err = cfg.Check("client", fset, []*ast.File{file}, testInfo()); err != nil {
		t.Fatal(err)
	}
	index.Invalidate([]*ast.File{file})
	records := index.Files([]*ast.File{file})
	index.Freeze()
	embedded, err := goembed.LoadRecords(fset, records)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"text", "bytes"} {
		for _, suffix := range []string{"", suffix(0), suffix(128)} {
			data, ok := embedded[name+suffix]
			if !ok || len(data.Files) != 1 || string(data.Files[0].Data) != "embedded data" {
				t.Errorf("missing embedded contents for %s: %+v", name+suffix, data)
			}
		}
	}
}

func TestRewriteRejectsGlobalBindings(t *testing.T) {
	for _, tc := range []struct {
		name, decl, directive string
		reject                bool
	}{
		{"vector alias", "var Value simd.Uint32s", "//go:linkname Value client.Target", true},
		{"scalar method dependency", "type Number int; func (Number) Width() int {var v simd.Uint32s;return v.Len()}; var Value Number", "//go:linkname Value client.Target", true},
		{"exported binding", "var Value simd.Uint32s", "//go:linkname Value", true},
		{"external binding", "var Value simd.Uint32s", "//go:linkname Value external.Target", true},
		{"llgo link", "var Value simd.Uint32s", "//llgo:link Value client.Target", true},
		{"spaced llgo link", "var Value simd.Uint32s", "// llgo:link Value client.Target", true},
		{"ordinary scalar", "var Value int;var Vector simd.Uint32s", "//go:linkname Value client.Target", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			// Linkname need not be attached to the declaration and may live in a
			// different file. Inspect package comments, not only declaration docs.
			decl, link, position := tc.decl, tc.directive, "link.go:3"
			if strings.Contains(tc.directive, "llgo:link") {
				// LLGo links, unlike go:linkname, must be attached to the variable.
				decl, link, position = tc.directive+"\n"+decl, "", "decl.go:3"
			}
			first, err := parser.ParseFile(fset, "decl.go", "package client\nimport \"simd\"\n"+decl, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			second, err := parser.ParseFile(fset, "link.go", "package client\nimport _ \"unsafe\"\n"+link+"\n", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			files := []*ast.File{first, second}
			cfg := types.Config{Importer: testSIMDImports(t)}
			info := testInfo()
			pkg, err := cfg.Check("client", fset, files, info)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = Rewrite(fset, pkg, info, files, new(directive.Index))
			if tc.reject {
				if err == nil || !strings.Contains(err.Error(), "binding for SIMD-dependent global Value") || !strings.Contains(err.Error(), position) {
					t.Fatalf("unexpected diagnostic: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRewriteRejectsImportedGlobalLinknameTarget(t *testing.T) {
	imports := testSIMDImports(t)
	testCheck(t, imports, "foreign", `package foreign
import "simd"
type Number int
func (Number) Width() int {var v simd.Uint32s;return v.Len()}
var Value Number = 7
`, true)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "client.go", `package client
import (_ "foreign"; _ "unsafe")
//go:linkname Alias foreign.Value
var Alias int
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	cfg := types.Config{Importer: imports}
	info := testInfo()
	pkg, err := cfg.Check("client", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = Rewrite(fset, pkg, info, []*ast.File{file}, new(directive.Index))
	if err == nil || !strings.Contains(err.Error(), "target SIMD-dependent global foreign.Value") {
		t.Fatalf("unexpected diagnostic: %v", err)
	}
}
