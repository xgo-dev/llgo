package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
)

func TestCgoPreambleImportOwnership(t *testing.T) {
	for _, test := range []struct {
		name       string
		importPath string
		source     string
		want       int
	}{
		{"original-single", "C", "/* int value; */\nimport \"C\"", 1},
		{"original-group", "C", "import (\n/* int value; */\n\"C\"\n\"unsafe\"\n)", 1},
		{"rewritten-single", "unsafe", "/* int value; */\nimport _ \"unsafe\"", 1},
		{"ordinary-group", "unsafe", "import (\n// \"internal/race\"\n// \"runtime\"\n\"unsafe\"\n)", 0},
		{"ordinary-multiple", "unsafe", "import (\n\"runtime\"\n// Needed for go:linkname.\n_ \"unsafe\"\n)", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "fixture.go", "package fixture\n"+test.source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			preambles, _, err := parseCgoFilePreambles(commandEnv{}, nil, fset, []*ast.File{file}, test.importPath)
			if err != nil {
				t.Fatal(err)
			}
			if len(preambles) != test.want {
				t.Fatalf("preambles = %#v, want %d", preambles, test.want)
			}
			if test.want != 0 && !strings.Contains(preambles[0].src, "int value;") {
				t.Fatalf("lost original C preamble: %q", preambles[0].src)
			}
		})
	}
}

func TestSourcePatchedSyncHasNoCgoPreamble(t *testing.T) {
	for _, target := range []struct{ goos, goarch string }{
		{"wasip1", "wasm"},
		{"js", "wasm"},
		{"linux", "amd64"},
		{"darwin", "arm64"},
	} {
		t.Run(target.goos+"-"+target.goarch, func(t *testing.T) {
			_, overlay, _, err := applySourcePatchForPkg(nil, nil, env.LLGoRuntimeDir(), runtime.GOROOT(), "sync", sourcePatchBuildContext{
				goos: target.goos, goarch: target.goarch, goversion: runtime.Version(),
			})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(runtime.GOROOT(), "src", "sync", "pool.go")
			source, ok := overlay[path]
			if !ok {
				t.Fatal("expected selected SDK Pool source overlay")
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			preambles, _, err := parseCgoFilePreambles(commandEnv{}, nil, fset, []*ast.File{file}, "unsafe")
			if err != nil {
				t.Fatal(err)
			}
			if len(preambles) != 0 {
				t.Fatalf("ordinary SDK imports became C code: %#v", preambles)
			}
		})
	}
}
