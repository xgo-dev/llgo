package build

import (
	"go/ast"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/buildenv"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	gopackages "golang.org/x/tools/go/packages"
)

func TestBinaryCodecSupport(t *testing.T) {
	base := Config{Mode: ModeBuild, BuildMode: BuildModeExe, OptLevel: optlevel.Oz,
		DeadcodeDrop: true, DisableGoGlobalDCE: true, LTO: lto.Full}
	for _, tc := range []struct {
		name    string
		thin    bool
		overlay bool
		alias   bool
		unbound bool
		want    bool
	}{
		{name: "closed Go inputs", want: buildenv.Dev},
		{name: "Thin LTO", thin: true},
		{name: "source overlay", overlay: true},
		{name: "binary linker alias", alias: true},
		{name: "unbound standard library", unbound: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, overlay, files, err := applySourcePatchForPkg(nil, nil, env.LLGoRuntimeDir(), runtime.GOROOT(), "encoding/binary", sourcePatchBuildContext{
				goos: runtime.GOOS, goarch: runtime.GOARCH, goversion: runtime.Version(),
				buildFlags: []string{"-tags=llgo,llgo.size.binary"},
			})
			if err != nil {
				t.Fatal(err)
			}
			cfg := gopackages.Config{Mode: gopackages.LoadAllSyntax, Dir: filepath.Join("testdata", "binary-fixed")}
			dedup := packages.NewDeduper()
			if !tc.unbound {
				dedup.SetLLGoFiles(map[string][]string{"encoding/binary": files})
			}
			initial, err := packages.LoadExWithSourceOverlay(dedup, nil, &cfg, "", overlay, ".")
			if err != nil || len(initial) != 1 || gopackages.PrintErrors(initial) != 0 {
				t.Fatalf("load binary fixture: %v", err)
			}
			conf := base
			if tc.thin {
				conf.LTO = lto.Thin
			}
			if tc.overlay {
				conf.SourceOverlay = map[string]SourceOverlayFile{"encoding/binary/binary.go": {}}
			}
			if tc.alias {
				initial[0].Syntax[0].Comments = append(initial[0].Syntax[0].Comments, &ast.CommentGroup{
					List: []*ast.Comment{{Text: "//go:linkname nativePacket encoding/binary.Encode"}},
				})
			}
			bindings, err := loadSourceBindings(&conf, cl.Options{}, runtime.GOROOT(), initial)
			if err != nil {
				t.Fatal(err)
			}
			result, err := applySourceTemplates(&conf, cl.Options{}, bindings, initial, nil, false)
			if err != nil {
				t.Fatal(err)
			}
			if (len(result.Binary) != 0) != tc.want {
				t.Fatalf("rewritten = %d packages, want enabled %v", len(result.Binary), tc.want)
			}
		})
	}
}
