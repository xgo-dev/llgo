package jsonspecialize

import (
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/packages"
	gopackages "golang.org/x/tools/go/packages"
)

func TestRewriteJSONDecode(t *testing.T) {
	overlay := make(map[string][]byte)
	root := runtime.GOROOT()
	for _, path := range []string{"encoding/json", "encoding/json/v2"} {
		patch := filepath.Join("..", "..", "..", "runtime", "_patch", filepath.FromSlash(path), "typed_decode.go")
		data, err := os.ReadFile(patch)
		if err != nil {
			t.Fatal(err)
		}
		_, source, ok := strings.Cut(string(data), "\n")
		if !ok {
			t.Fatalf("missing build constraint in %s", patch)
		}
		overlay[filepath.Join(root, "src", filepath.FromSlash(path), "llgo_typed_decode.go")] = []byte(source)
	}
	pkgs, err := gopackages.Load(&gopackages.Config{
		Mode:    gopackages.LoadAllSyntax,
		Dir:     filepath.Join("..", "testdata", "json-typed-decode"),
		Env:     append(os.Environ(), "GOEXPERIMENT=jsonv2"),
		Overlay: overlay,
	}, "./main.go")
	if err != nil || len(pkgs) != 1 || gopackages.PrintErrors(pkgs) != 0 {
		t.Fatalf("load decoder fixture: %v", err)
	}
	initial := pkgs[0]
	syntax := initial.Syntax
	library := initial.Imports["encoding/json"]
	var closure []*packages.Package
	packages.Visit(pkgs, nil, func(p *packages.Package) { closure = append(closure, p) })
	changed, err := Rewrite(closure, library, initial)
	if err != nil || len(changed) != 1 || changed[0] != initial {
		t.Fatalf("rewrite decoder: %d packages, %v", len(changed), err)
	}
	var decoded int
	for _, file := range syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			fn, _ := typesynth.Callee(initial.TypesInfo, call.Fun)
			if fn == nil || len(call.Args) != 2 {
				return true
			}
			if fn.Pkg() == library.Types && fn.Name() == "Unmarshal" {
				t.Error("typed decode still calls Unmarshal")
			}
			if fn.Pkg() == initial.Types && strings.HasPrefix(fn.Name(), "__llgo_json_") {
				decoded++
				if _, ok := fn.Signature().Params().At(1).Type().(*types.Pointer); !ok {
					t.Error("decoder did not preserve the pointer type")
				}
			}
			return true
		})
	}
	if decoded != 2 {
		t.Fatalf("decoded %d call sites, want 2", decoded)
	}
}
