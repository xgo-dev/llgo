package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/pclnpost"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/packages"
)

func TestDarwinSymbolEntriesPreserveFunctionBodies(t *testing.T) {
	t.Setenv("LLGO_PCLNPOST", "1")
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(true)
	pkg := prog.NewPackage("example.com/p", "example.com/p")
	pkg.EmitFuncInfo("example.com/p.live", "example.com/p.Live", "live.go", 17, 3)
	pkg.NewFunc("example.com/p.live", llssa.NoArgsNoRet, llssa.CDecl).MakeBody(1).Return()
	c := &context{prog: prog, buildConf: &Config{
		Goos: "darwin", Goarch: "arm64", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded, LTO: lto.Full,
	}}
	before := pkg.String()
	emitFuncInfoEntrySites(c, pkg)
	if pkg.String() != before {
		t.Fatal("symbol entry collection changed package function IR")
	}
	t.Setenv("LLGO_PCLNPOST", "0")
	emitFuncInfoEntrySites(c, pkg)
	if pkg.String() != before {
		t.Fatal("post-link opt-out changed the legacy DWARF package IR")
	}
	t.Setenv("LLGO_PCLNPOST", "1")
	entry := genMainModule(c, llssa.PkgRuntime, &packages.Package{
		PkgPath: "example.com/main", ExportFile: "main.a",
	}, &genConfig{funcInfo: collectFuncInfo([]Package{{LPkg: pkg}})})
	ir := entry.LPkg.String()
	if !strings.Contains(ir, "@__llgo_funcinfo_entry_reserve = private constant") ||
		!strings.Contains(ir, "ptr @__llgo_funcinfo_entry_reserve") {
		t.Fatalf("missing retained data-only entry carrier: %s", ir)
	}
	if strings.Contains(ir, `@"example.com/p.live"`) {
		t.Fatal("entry catalog introduced a function reference that could root dead code")
	}
}

func TestSymbolEntryReservationGrowth(t *testing.T) {
	const source = `@unrelated = constant [32 x i8] zeroinitializer
@__llgo_funcinfo_entry_reserve = private constant { i64, ptr, [32 x i8] } { i64 1, ptr @__llgo_funcinfo_entry_reserve, [32 x i8] zeroinitializer }
`
	// A large foreign-code gap needs many buckets despite only two Go entries.
	analysis := pclnpost.ExternalAnalysis{TextStart: 0x1000, TextEnd: 0x4001000,
		EntrySites: []pclnpost.ExternalSite{{PCOffset: 0x1000}, {PCOffset: 0x4000000}}}
	grown, changed, err := growSymbolEntryReserve(source, analysis)
	if err != nil || !changed {
		t.Fatalf("grow reservation = %v, %v", changed, err)
	}
	if !strings.HasPrefix(grown, "@unrelated = constant [32 x i8] zeroinitializer\n") || strings.Count(grown, "[32 x i8]") != 1 {
		t.Fatalf("reservation growth changed unrelated data or left a stale type: %s", grown)
	}
	if next, changed, err := growSymbolEntryReserve(grown, analysis); err != nil || changed || next != grown {
		t.Fatalf("grown reservation is not sufficient: changed=%v, err=%v", changed, err)
	}
	if _, changed, err := growSymbolEntryReserve("@ordinary = global i32 0", analysis); err != nil || changed {
		t.Fatalf("ordinary entry IR = %v, %v", changed, err)
	}
	if _, _, err := growSymbolEntryReserve("@__llgo_funcinfo_entry_reserve = private constant i64 0", analysis); err == nil {
		t.Fatal("unexpected reservation declaration accepted")
	}
	if _, _, err := growSymbolEntryReserve(source, pclnpost.ExternalAnalysis{}); err == nil {
		t.Fatal("invalid linked text range accepted")
	}
	if _, _, err := growSymbolEntryReserve(source, pclnpost.ExternalAnalysis{TextStart: 1, TextEnd: ^uint64(0)}); err == nil {
		t.Fatal("text offsets exceeding the runtime's 32-bit format accepted")
	}
}

func TestDarwinSymbolEntryRewriteRequired(t *testing.T) {
	t.Setenv("LLGO_PCLNPOST", "1")
	sites := true
	c := &context{linkRuntimeSites: &sites, buildConf: &Config{
		Goos: "darwin", Goarch: "arm64", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded, LTO: lto.Full,
	}}
	path := filepath.Join(t.TempDir(), "app")
	const original = "not an executable"
	if err := os.WriteFile(path, []byte(original), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := rewritePrebuiltFuncTab(c, path, false); err == nil || !strings.Contains(err.Error(), "build native symbol index") {
		t.Fatalf("required symbol rewrite failure was lost: %v", err)
	}
	c.buildConf.Goos = "linux"
	if err := rewritePrebuiltFuncTab(c, path, false); err != nil {
		t.Fatalf("raw-site first-use fallback became mandatory: %v", err)
	}
	if raw, err := os.ReadFile(path); err != nil || string(raw) != original {
		t.Fatalf("rewrite failure changed output: %q, %v", raw, err)
	}
}

func TestDarwinSymbolEntriesAfterProgramDisposal(t *testing.T) {
	t.Setenv("LLGO_PCLNPOST", "")
	sites := true
	c := &context{linkRuntimeSites: &sites, buildConf: &Config{
		Goos: "darwin", Goarch: "arm64", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded, LTO: lto.Full,
	}}
	if !darwinSymbolEntries(c) || !shouldEmitRuntimeEntrySites(c) {
		t.Fatal("link execution lost the symbol-entry policy after disposing its Program")
	}
	t.Setenv("LLGO_PCLNPOST", "0")
	if darwinSymbolEntries(c) || shouldEmitRuntimeEntrySites(c) {
		t.Fatal("post-link opt-out must retain the legacy DWARF/dlsym policy")
	}
	t.Setenv("LLGO_PCLNPOST", "1")
	c.buildConf.LTO = lto.Off
	if darwinSymbolEntries(c) || shouldEmitRuntimeEntrySites(c) {
		t.Fatal("ordinary DWARF builds must not gain an extra metadata relink")
	}
}

func TestSymbolEntryReservationPublication(t *testing.T) {
	ir, data := embeddedPCLNFixture(t)
	ir += "\n@__llgo_funcinfo_entry_reserve = private constant [32 x i8] zeroinitializer\n"
	initial := pclnpost.ExternalAnalysis{PointerSize: data.PointerSize, TextStart: 0x1000, TextEnd: 0x4001000,
		EntrySites: []pclnpost.ExternalSite{{PCOffset: 16, ID: funcInfoSymbolID("main.live")},
			{PCOffset: 32, ID: funcInfoSymbolID("main.dead")}},
		PCLineSites: []pclnpost.ExternalSite{{PCOffset: 33, ID: 7, OwnerSymbol: "main.dead"}}}
	for _, outgrows := range []bool{false, true} {
		t.Run(map[bool]string{false: "growth-only", true: "outgrows-again"}[outgrows], func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "app")
			if err := os.WriteFile(output, []byte("original"), 0o700); err != nil {
				t.Fatal(err)
			}
			plan := &mainLinkPlan{entryIR: ir, embeddedPCLN: &data, linkInputs: []string{"entry.o", "unchanged-code.a"}}
			compiled, linked, published := false, false, false
			linker := embeddedPCLNLinker{
				analyze: func(path string) (pclnpost.ExternalAnalysis, error) {
					a := initial
					if path != output && outgrows {
						a.TextEnd += 0x4000000
					}
					return a, nil
				},
				compile: func(path, object string) error {
					compiled = true
					raw, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					if _, grow, err := growSymbolEntryReserve(string(raw), initial); err != nil || grow {
						t.Fatalf("compiler received insufficient reservation: %v, %v", grow, err)
					}
					return os.WriteFile(object, []byte("entry"), 0o600)
				},
				link: func(candidate string, inputs []string) error {
					linked = true
					if len(inputs) != 2 || inputs[1] != "unchanged-code.a" {
						t.Fatalf("relink changed program inputs: %v", inputs)
					}
					return os.WriteFile(candidate, []byte("grown"), 0o700)
				},
				publish: func(candidate, dest string) error {
					published = true
					return replaceFile(candidate, dest)
				},
			}
			err := pruneEmbeddedPCLNWith(plan, output, false, linker)
			if (err != nil) != outgrows || !compiled || !linked || published == outgrows {
				t.Fatalf("reservation transaction: error=%v, compiled=%v, linked=%v, published=%v", err, compiled, linked, published)
			}
			if outgrows {
				raw, err := os.ReadFile(output)
				if err != nil || string(raw) != "original" {
					t.Fatalf("failed verification changed the output: %s, %v", raw, err)
				}
			}
		})
	}
}
