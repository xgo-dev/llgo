package build

import (
	"go/ast"
	gobuild "go/build"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
	llruntime "github.com/xgo-dev/llgo/runtime"
)

func TestNativeRuntimeTimerRemainsReplaceAltPackage(t *testing.T) {
	if !llruntime.HasAltPkg("runtime") {
		t.Fatal("runtime must remain an LLGo alt package")
	}
	if llruntime.HasAdditiveAltPkg("runtime") {
		t.Fatal("runtime must remain replace-mode; additive mode would pull in the Go runtime sources")
	}

	runtimeDir := filepath.Join(env.LLGoRuntimeDir(), "internal", "lib", "runtime")
	runtimePatchDir := filepath.Join(env.LLGoRuntimeDir(), "_patch", "runtime")
	for _, target := range []struct {
		goos   string
		goarch string
	}{
		{goos: "darwin", goarch: "arm64"},
		{goos: "linux", goarch: "amd64"},
		{goos: "windows", goarch: "amd64"},
	} {
		t.Run(target.goos+"_"+target.goarch, func(t *testing.T) {
			ctx := gobuild.Default
			ctx.GOOS = target.goos
			ctx.GOARCH = target.goarch
			ctx.BuildTags = []string{"llgo"}
			pkg, err := ctx.ImportDir(runtimeDir, 0)
			if err != nil {
				t.Fatal(err)
			}

			selected := make(map[string]bool, len(pkg.GoFiles)+len(pkg.CgoFiles))
			for _, name := range append(pkg.GoFiles, pkg.CgoFiles...) {
				selected[name] = true
				file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(runtimeDir, name), nil, parser.ImportsOnly)
				if err != nil {
					t.Fatal(err)
				}
				for _, spec := range file.Imports {
					path, err := strconv.Unquote(spec.Path.Value)
					if err != nil {
						t.Fatal(err)
					}
					if path == "github.com/xgo-dev/llgo/runtime/internal/clite/libuv" {
						t.Fatalf("native runtime selected %s, which imports libuv", name)
					}
				}
			}
			if !selected["time_heap_llgo.go"] {
				t.Fatal("native runtime did not select the LLGo timer heap backend")
			}

			patchFiles, err := filepath.Glob(filepath.Join(runtimePatchDir, "*.go"))
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range patchFiles {
				name := filepath.Base(path)
				match, err := ctx.MatchFile(runtimePatchDir, name)
				if err != nil {
					t.Fatal(err)
				}
				if match {
					// Constants must reach the frontend before it checks importers.
					// Runtime implementations, including timers, still come from
					// the replace-mode alternative package.
					file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
					if err != nil {
						t.Fatal(err)
					}
					for _, decl := range file.Decls {
						gen, ok := decl.(*ast.GenDecl)
						if !ok || gen.Tok != token.CONST {
							t.Fatalf("native runtime source patch %s contains implementation declarations; timer reuse must stay in the replacement runtime", name)
						}
					}
				}
			}
		})
	}
}
