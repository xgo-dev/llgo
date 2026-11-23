package build

import (
	"go/ast"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/buildenv"
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
		want    bool
	}{
		{name: "closed Go inputs", want: buildenv.Dev},
		{name: "Thin LTO", thin: true},
		{name: "source overlay", overlay: true},
		{name: "binary linker alias", alias: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := gopackages.Config{Mode: gopackages.LoadAllSyntax, Dir: filepath.Join("testdata", "binary-fixed")}
			dedup := packages.NewDeduper()
			initial, err := packages.LoadEx(dedup, nil, &cfg, ".")
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
			result, err := applySourceTemplates(&conf, cl.Options{}, runtime.GOROOT(), dedup, initial, nil, false)
			if err != nil {
				t.Fatal(err)
			}
			if (len(result.binary) != 0) != tc.want {
				t.Fatalf("rewritten = %d packages, want enabled %v", len(result.binary), tc.want)
			}
		})
	}
}
