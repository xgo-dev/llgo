package build

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/pclnmap"
	"github.com/xgo-dev/llgo/internal/pclnpost"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestEmbeddedPCLNPruningScope(t *testing.T) {
	t.Setenv("LLGO_PCLNPOST", "1")
	for _, tc := range []struct {
		name                 string
		goos, goarch, target string
		mode                 PCLNMode
		buildMode            BuildMode
		sites, want          bool
	}{
		{"linux", "linux", "amd64", "", PCLNEmbedded, BuildModeExe, true, true},
		{"darwin", "darwin", "arm64", "", PCLNEmbedded, BuildModeExe, true, true},
		{"external", "linux", "amd64", "", PCLNExternal, BuildModeExe, true, false},
		{"none", "linux", "amd64", "", PCLNNone, BuildModeExe, true, false},
		{"shared", "linux", "amd64", "", PCLNEmbedded, BuildModeCShared, true, false},
		{"named target", "linux", "amd64", "custom", PCLNEmbedded, BuildModeExe, true, false},
		{"no sites", "linux", "amd64", "", PCLNEmbedded, BuildModeExe, false, false},
		{"32-bit", "linux", "386", "", PCLNEmbedded, BuildModeExe, true, false},
		{"windows", "windows", "amd64", "", PCLNEmbedded, BuildModeExe, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &context{buildConf: &Config{
				Goos: tc.goos, Goarch: tc.goarch, Target: tc.target,
				PCLNMode: tc.mode, BuildMode: tc.buildMode,
				LinkOptions: LinkOptions{DWARF: DWARFOmit},
			}, linkRuntimeSites: &tc.sites, crossCompile: crosscompile.Export{}}
			if got := canPruneEmbeddedPCLN(ctx); got != tc.want {
				t.Fatalf("eligible = %v, want %v", got, tc.want)
			}
			if tc.want {
				t.Setenv("LLGO_PCLNPOST", "0")
				if canPruneEmbeddedPCLN(ctx) {
					t.Fatal("PCLNPOST escape hatch was ignored")
				}
			}
		})
	}
}

func TestRewriteEmbeddedPCLNIRRemapsAllTables(t *testing.T) {
	ir, data := embeddedPCLNFixture(t)
	data.EntrySites = []pclnmap.Site{{PCOffset: 16, ID: funcInfoSymbolID("main.live")}}
	pruned, err := pclnmap.Prune(data)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "entry.ll")
	if err := os.WriteFile(path, []byte(ir), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := rewriteEmbeddedPCLNIR(path, pruned); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ir = string(raw)
	for _, forbidden := range []string{"DeadOnly2420", "/dead", ".unpruned"} {
		if strings.Contains(ir, forbidden) {
			t.Fatalf("rewritten entry retains %q", forbidden)
		}
	}
	for _, expected := range []string{
		"@__llgo_funcinfo_count = global i64 1",
		"@__llgo_pcline_count = global i64 0",
		"@__llgo_funcinfo_symbol_index_count = hidden global i64 1",
		`@"__llgo_funcinfo_table$data" = private unnamed_addr constant [1 x`,
		`@"__llgo_pcline_table$data" = private unnamed_addr constant [0 x`,
	} {
		if !strings.Contains(ir, expected) {
			t.Fatalf("rewritten entry is missing %q:\n%s", expected, ir)
		}
	}
}

func embeddedPCLNFixture(t *testing.T) (string, pclnmap.Data) {
	t.Helper()
	t.Setenv("LLGO_PCLNPOST", "1")
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	prog.EnableFuncInfoSites(true)
	pkg := prog.NewPackage("main", "main")
	defer pkg.Module().Dispose()
	ctx := &context{prog: prog, buildConf: &Config{
		Goos: "linux", Goarch: "amd64", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded,
	}}
	emitFuncInfoTable(ctx, pkg, []funcInfoRecord{
		{symbol: "main.dead", name: "main.DeadOnly2420", file: "/dead/file.go", line: 10},
		{symbol: "main.live", name: "main.Live", file: "/src/main.go", line: 20},
	}, []pcLineRecord{{id: 7, symbol: "main.dead", file: "/dead/site.go", line: 11}})
	return pkg.String(), *ctx.pclnEmbedded
}

func TestEmbeddedPCLNPublication(t *testing.T) {
	ir, original := embeddedPCLNFixture(t)
	initial := pclnpost.ExternalAnalysis{PointerSize: original.PointerSize,
		EntrySites: []pclnpost.ExternalSite{{PCOffset: 16, ID: funcInfoSymbolID("main.live")}}}
	for _, stage := range []string{"success", "disabled", "unchanged", "initial analysis", "initial join", "mkdir", "IR", "compile", "link", "final analysis", "final join", "liveness", "publish"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, "app")
			if err := os.WriteFile(output, []byte("original-valid-image"), 0o700); err != nil {
				t.Fatal(err)
			}
			plan := &mainLinkPlan{embeddedPCLN: &original, entryIR: ir, linkInputs: []string{"old-entry.o", "go-packages.a"}}
			if stage == "disabled" {
				plan.embeddedPCLN = nil
			}
			if stage == "IR" {
				plan.entryIR = "invalid IR"
			}
			if stage == "mkdir" {
				output = filepath.Join(dir, "absent", "app")
			}
			linked := false
			fail := errors.New("injected tool failure")
			linker := embeddedPCLNLinker{
				analyze: func(path string) (pclnpost.ExternalAnalysis, error) {
					a := initial
					if (path == output && stage == "initial analysis") || (path != output && stage == "final analysis") {
						return a, fail
					}
					if (path == output && stage == "initial join") || (path != output && stage == "final join") {
						a.EntrySites = nil
					}
					if stage == "unchanged" || (path != output && stage == "liveness") {
						a.EntrySites = append(slices.Clone(a.EntrySites), pclnpost.ExternalSite{PCOffset: 32, ID: funcInfoSymbolID("main.dead")})
						a.PCLineSites = []pclnpost.ExternalSite{{PCOffset: 33, ID: 7, OwnerSymbol: "main.dead"}}
					}
					return a, nil
				},
				compile: func(path, object string) error {
					if stage == "compile" {
						return fail
					}
					raw, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					if strings.Contains(string(raw), "DeadOnly2420") {
						t.Fatal("compiler received dead metadata")
					}
					return os.WriteFile(object, []byte("compact-entry"), 0o600)
				},
				link: func(candidate string, inputs []string) error {
					linked = true
					if len(inputs) != 2 || inputs[1] != "go-packages.a" || inputs[0] == "old-entry.o" {
						t.Fatalf("link inputs = %v", inputs)
					}
					if stage == "link" {
						return fail
					}
					return os.WriteFile(candidate, []byte("verified-compact-image"), 0o700)
				},
				publish: func(candidate, dest string) error {
					if stage == "publish" {
						return fail
					}
					return replaceFile(candidate, dest)
				},
			}
			err := pruneEmbeddedPCLNWith(plan, output, true, linker)
			wantError := stage != "success" && stage != "disabled" && stage != "unchanged"
			if (err != nil) != wantError {
				t.Fatalf("error = %v, want failure %v", err, wantError)
			}
			if stage == "mkdir" {
				return
			}
			raw, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			want := "original-valid-image"
			if stage == "success" {
				want = "verified-compact-image"
			}
			if string(raw) != want {
				t.Fatalf("published %q, want %q", raw, want)
			}
			if (stage == "disabled" || stage == "unchanged") && linked {
				t.Fatal("unnecessary second link")
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Fatalf("staged files leaked: %v, %v", entries, err)
			}
		})
	}
}

func TestEmbeddedPCLNRejectsMalformedIR(t *testing.T) {
	ir, data := embeddedPCLNFixture(t)
	for _, name := range []string{funcInfoDataSymbol, pcLineDataSymbol, funcInfoStringsDataSymbol, funcInfoStringOffsetsDataSymbol, funcInfoHashDataSymbol, funcInfoSymbolIndexDataSymbol, funcInfoCountSymbol, pcLineCountSymbol, funcInfoStringCountSymbol, funcInfoSymbolIndexCountSymbol, funcInfoHashMaskSymbol} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "entry.ll")
			raw := strings.ReplaceAll(ir, name, name+"_absent")
			if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := rewriteEmbeddedPCLNIR(path, data); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("error = %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != raw {
				t.Fatalf("failed rewrite changed input: %v", err)
			}
		})
	}
	if err := rewriteEmbeddedPCLNIR(filepath.Join(t.TempDir(), "missing.ll"), data); err == nil {
		t.Fatal("accepted missing IR")
	}
	analysis := pclnpost.ExternalAnalysis{PointerSize: 4}
	if _, err := linkedPCLNData(data, analysis); err == nil {
		t.Fatal("accepted wrong pointer size")
	}
}

func TestCompileEmbeddedEntryLTO(t *testing.T) {
	for _, mode := range []lto.Mode{lto.Off, lto.Full, lto.Thin} {
		t.Run(mode.String(), func(t *testing.T) {
			dir := t.TempDir()
			ir, object := filepath.Join(dir, "entry.ll"), filepath.Join(dir, "entry.o")
			if err := os.WriteFile(ir, []byte("define i32 @main() { ret i32 0 }"), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx := &context{buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, LTO: mode}}
			plan := &mainLinkPlan{}
			if err := compileEmbeddedEntry(ctx, plan, ir, object); err != nil {
				t.Fatal(err)
			}
			if info, err := os.Stat(object); err != nil || info.Size() == 0 {
				t.Fatalf("empty object: %v", err)
			}
			if err := compileEmbeddedEntry(ctx, plan, ir+"missing", object); err == nil {
				t.Fatal("accepted missing input")
			}
			if err := os.WriteFile(ir, []byte("invalid IR"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := compileEmbeddedEntry(ctx, plan, ir, object); err == nil {
				t.Fatal("accepted malformed IR")
			}
		})
	}
}
