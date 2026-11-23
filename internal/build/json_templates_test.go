package build

import (
	"go/ast"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	gopackages "golang.org/x/tools/go/packages"
)

func TestRewriteJSONTemplates(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		count   int
	}{
		{"json-static", 1},
		{"json-typed-decode", 2},
		{"json-mixed", 0},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			const library = "encoding/json"
			t.Setenv("GOEXPERIMENT", "jsonv2")
			goroot := runtime.GOROOT()
			cfg := sourcePatchBuildContext{
				goos: runtime.GOOS, goarch: runtime.GOARCH, goversion: runtime.Version(),
				buildFlags: []string{"-tags=llgo,llgo.size.json,goexperiment.jsonv2"},
			}
			deps := make(map[string][]string)
			var overlay map[string][]byte
			for _, path := range []string{library, library + "/v2"} {
				_, next, files, err := applySourcePatchForPkg(nil, overlay, env.LLGoRuntimeDir(), goroot, path, cfg)
				if err != nil {
					t.Fatal(err)
				}
				deps[path], overlay = files, next
			}
			dedup := packages.NewDeduper()
			dedup.SetLLGoFiles(deps)
			load := gopackages.Config{Mode: gopackages.LoadAllSyntax, Dir: filepath.Join("testdata", tc.fixture)}
			initial, err := packages.LoadExWithSourceOverlay(dedup, nil, &load, "", overlay, ".")
			if err != nil || len(initial) != 1 || gopackages.PrintErrors(initial) != 0 {
				t.Fatalf("load fixture: %v", err)
			}
			conf := Config{Mode: ModeBuild, BuildMode: BuildModeExe, OptLevel: optlevel.Oz,
				DeadcodeDrop: true, DisableGoGlobalDCE: true, LTO: lto.Full}
			bindings, err := loadSourceBindings(&conf, cl.Options{}, goroot, initial)
			if err != nil {
				t.Fatal(err)
			}
			entry := initial[0]
			original := entry.Syntax
			result, err := applySourceTemplates(&conf, cl.Options{}, bindings, initial, nil, false)
			if err != nil || len(result.Changes) != tc.count {
				t.Fatalf("applied %d calls, want %d: %v", len(result.Changes), tc.count, err)
			}
			var specialized int
			for _, file := range original {
				ast.Inspect(file, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					fn, _ := typesynth.Callee(entry.TypesInfo, call.Fun)
					if fn != nil && fn.Pkg() == entry.Types && strings.HasPrefix(fn.Name(), "__llgo_json_") {
						specialized++
					}
					return true
				})
			}
			if specialized != tc.count {
				t.Fatalf("bound %d generated calls, want %d", specialized, tc.count)
			}
		})
	}
}
