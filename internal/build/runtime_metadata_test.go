package build

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/internal/llvmgen"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/packages"
)

func TestPlanMainLinkRuntimeMetadata(t *testing.T) {
	for _, test := range []struct {
		name                                  string
		runtime, python, wasm, local, cExport bool
		wantRuntime                           bool
	}{
		{name: "runtime-free native"},
		{name: "runtime required", runtime: true, wantRuntime: true},
		{name: "python required", python: true, wantRuntime: true},
		{name: "wasm scheduler", wasm: true, wantRuntime: true},
		{name: "local context startup", local: true, wantRuntime: true},
		{name: "leaf C export", cExport: true, wantRuntime: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := llssa.NewProgram(nil)
			defer prog.Dispose()
			if test.local {
				prog.SetLocalityInfo("example.com/p.value", llssa.LocalityInfo{Locality: llssa.GoroutineLocal})
			}
			app := snapshotPkg("example.com/main", "main.main")
			rt := snapshotPkg(llssa.PkgRuntime, llssa.PkgRuntime+".unusedHelper")
			app.NeedRt, app.NeedPyInit = test.runtime, test.python
			if test.cExport {
				app.linkSnapshot.cExports = []cExport{{
					goName: "example.com/main.Callback", cName: "callback", sig: llssa.NoArgsNoRet,
				}}
			}
			for _, pkg := range []Package{app, rt} {
				pkg.ID = pkg.PkgPath
				pkg.ExportFile = pkg.PkgPath + ".o"
				pkg.ArchiveFile = pkg.PkgPath + ".a"
			}
			ctx := &context{
				prog:      prog,
				buildConf: &Config{BuildMode: BuildModeExe, Goos: "linux", PCLNMode: PCLNEmbedded},
				pkgs:      map[*packages.Package]Package{app.Package: app, rt.Package: rt},
			}
			ctx.crossCompile.WasmPostLink.Asyncify = test.wasm
			plan, err := planMainLink(ctx, app.Package, []*aPackage{app, rt})
			if err != nil {
				t.Fatal(err)
			}
			if test.cExport && (!plan.gen.rtInit || len(plan.gen.cExports) != 1) {
				t.Fatal("leaf C export lost its runtime initialization or entry wrapper")
			}
			// The host archive policy is unchanged, even when metadata is omitted.
			if !slices.Contains(plan.archiveInputs, rt.ArchiveFile) {
				t.Fatal("native link lost its runtime archive")
			}
			wantCount := 1
			if test.wantRuntime {
				wantCount++
			}
			if len(plan.gen.funcInfo) != wantCount || len(plan.gen.pcLineInfo) != wantCount {
				t.Fatalf("metadata counts = %d/%d, want %d", len(plan.gen.funcInfo), len(plan.gen.pcLineInfo), wantCount)
			}
		})
	}
}

func TestRuntimeFreeNativeMetadata(t *testing.T) {
	t.Setenv(llgoBuildCache, "off")
	for _, mode := range []lto.Mode{lto.Off, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			// Inspect the retained entry-site IDs. Darwin DWARF deliberately
			// omits these sites, allowing unused symbol-index data to be stripped
			// from this runtime-free program. The site policy is tested separately.
			conf.LinkOptions.DWARF = DWARFOmit
			if runtime.GOOS == "windows" {
				// An explicit -o file name is exact; Windows execution needs the
				// suffix before Build resolves its private configuration copy.
				conf.AppExt = ".exe"
			}
			conf.LTO = mode
			conf.PCLNMode, conf.PCLNModeSet = PCLNEmbedded, true
			conf.OutFile = filepath.Join(t.TempDir(), "cprintf"+conf.AppExt)
			var runtimeInfo []funcInfoRecord
			conf.ModuleHook = func(pkg Package) {
				if pkg.PkgPath == llssa.PkgRuntime {
					runtimeInfo = readFuncInfo(pkg.LPkg.Module())
				}
			}
			_, err := Do([]string{"../../benchmark/binary_size/cprintf"}, conf)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(conf.OutFile)
			if err != nil {
				t.Fatal(err)
			}
			containsID := func(symbol string) bool {
				var id [8]byte
				binary.NativeEndian.PutUint64(id[:], funcInfoSymbolID(symbol))
				return bytes.Contains(raw, id[:])
			}
			// Verify real runtime metadata was available to the planner, but none
			// of these unused helpers' symbol IDs reached the final executable.
			checked := 0
			for _, rec := range runtimeInfo {
				if rec.symbol != llssa.PkgRuntime+".Alloc" && rec.symbol != llssa.PkgRuntime+".NewProc" {
					continue
				}
				checked++
				if containsID(rec.symbol) {
					t.Errorf("unused runtime metadata survived the link: %s", rec.symbol)
				}
			}
			if checked == 0 {
				t.Fatal("runtime metadata was not prepared; regression was not exercised")
			}
			if !containsID(runtimeMainSymbol) || !containsID(processEntrySymbol) {
				t.Fatal("generated runtime.main/runtime.goexit metadata is missing")
			}
			output, err := exec.Command(conf.OutFile).CombinedOutput()
			if err != nil || strings.TrimSpace(string(output)) != "Hello, world" {
				t.Fatalf("run cprintf: %v\n%s", err, output)
			}
		})
	}
}
