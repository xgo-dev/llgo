//go:build go1.26 && (amd64 || arm64)

package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

// This fixture executes the AMD64 owner's 64-bit mask/word algorithm. A real
// 386/ARM/WASM SDK has different PageWords and mask layouts, so injecting these
// bodies there would corrupt the fixture rather than provide portable evidence.
// AMD64 standard-Go CI owns the required actual Go+LLGo execution; ARM64 adds
// numerical portability, never an original AVX512 execution certificate.
// Execute the identical portable Go bodies on the LLVM host, even if it cannot
// execute AMD64 AVX512. The independent arithmetic oracle is in the program.
// This is not an execution certificate for the replaced SDK assembly; the
// complete actual AMD64 package/object is checked separately above.
func TestGCScanRuntimePortableBodiesExecute(t *testing.T) {
	// Go1.26 made this graph experimental; Go1.27 enables it by default.
	// The fixture explicitly imports the SDK owner: LLGo's alternate runtime
	// does not retain the stock runtime's complete GC dependency graph.
	t.Setenv("GOEXPERIMENT", "greenteagc")
	dir := t.TempDir()
	patchDir := filepath.Join(env.LLGoRuntimeDir(), "_patch", filepath.FromSlash(gcScanRuntimePackage))
	ownerFile := filepath.Join(patchDir, "scan_amd64.go")
	body, err := os.ReadFile(ownerFile)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcePatchBuildContext{goos: runtime.GOOS, goarch: runtime.GOARCH, goversion: runtime.Version()}
	changed, overlay, _, err := applySourcePatchForPkg(nil, nil, env.LLGoRuntimeDir(), runtime.GOROOT(), gcScanRuntimePackage, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if changed != (runtime.GOARCH == "amd64") {
		t.Fatal("the production gc scan owner has unexpected architecture selection", runtime.GOARCH, changed)
	}
	sourceDir := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(gcScanRuntimePackage))
	generatedOwner := filepath.Join(sourceDir, "z_llgo_patch_scan_amd64.go")
	if runtime.GOARCH != "amd64" {
		// Explicitly select the same bodies for portable algorithm execution.
		// Do not change the production architecture selection or CPU predicates.
		body = bytes.TrimPrefix(body, []byte("//go:build amd64 && go1.26\n"))
		ownerFile = filepath.Join(sourceDir, "z_llgo_gc_scan_execution.go")
		overlay = map[string][]byte{ownerFile: body}
	} else if len(overlay[generatedOwner]) == 0 {
		t.Fatal("the original Go oracle lacks its generated gc scan owner")
	}
	program, err := os.ReadFile("testdata/gc-scan-owner/main.go")
	if err != nil {
		t.Fatal(err)
	}
	// A virtual SDK child package may import the real internal owner without
	// modifying the installed SDK or copying its implementation into the test.
	mainPackage := gcScanRuntimePackage + "/llgotestmain"
	mainFile := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(mainPackage), "main.go")
	overlay[mainFile] = program
	replace := make(map[string]string)
	for name, source := range overlay {
		file := filepath.Join(dir, fmt.Sprintf("overlay-%d.go", len(replace)))
		if err := os.WriteFile(file, source, 0o600); err != nil {
			t.Fatal(err)
		}
		replace[name] = file
	}
	data, err := json.Marshal(struct{ Replace map[string]string }{replace})
	if err != nil {
		t.Fatal(err)
	}
	overlayFile := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlayFile, data, 0o600); err != nil {
		t.Fatal(err)
	}
	command := func(binary string, args ...string) string {
		t.Helper()
		var output bytes.Buffer
		if err := runRunnerCommand(commandEnv{environ: os.Environ()}, binary, args,
			runnerDetails{phase: "portable-gc-scan-owner", timeout: 5 * time.Minute}, &output, &output); err != nil {
			t.Fatalf("actual gc scan command: %v\n%s", err, output.String())
		}
		return strings.TrimSpace(output.String())
	}
	extension := ""
	if runtime.GOOS == "windows" {
		extension = ".exe"
	}
	original := filepath.Join(dir, "original"+extension)
	command(filepath.Join(runtime.GOROOT(), "bin", "go"+extension), "build", "-overlay="+overlayFile,
		"-ldflags=-checklinkname=0", "-o", original, mainPackage)
	want := command(original)
	if want != "gc scan owners 26 classes 104 patterns 19 partial tails" {
		t.Fatal("the independent Go execution denominator changed", want)
	}
	llgoOverlay := overlay
	if runtime.GOARCH == "amd64" {
		// Build selects the physical source patch itself. Only the stock-Go
		// oracle needs its generated copy in the public cmd/go overlay.
		llgoOverlay = map[string][]byte{mainFile: program}
	}
	conf := NewDefaultConf(ModeBuild)
	conf.GOEXPERIMENT = "greenteagc"
	conf.OptLevel, conf.LTO = optlevel.O0, lto.Off
	conf.ForceRebuild, conf.Overlay = true, llgoOverlay
	conf.SourceOverlay = make(map[string]SourceOverlayFile, len(llgoOverlay))
	for name, source := range llgoOverlay {
		conf.SourceOverlay[name] = SourceOverlayFile{BackingFile: replace[name], Contents: source}
	}
	conf.OutFile = filepath.Join(dir, "llgo"+extension)
	modules := 0
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath != gcScanRuntimePackage {
			return
		}
		modules++
		selectedOwners := 0
		for _, name := range pkg.CompiledGoFiles {
			if name == ownerFile {
				selectedOwners++
			}
			if runtime.GOARCH == "amd64" && name == generatedOwner {
				t.Fatal("LLGo selected the oracle's duplicate generated gc scan owner")
			}
		}
		if selectedOwners != 1 {
			t.Fatal("LLGo did not select the actual gc scan owner exactly once", ownerFile, selectedOwners)
		}
		module := pkg.LPkg.Module()
		for _, name := range []string{"ExpandAVX512", "scanSpanPackedAVX512", "FilterNilAVX512"} {
			fn := module.NamedFunction(gcScanRuntimePackage + "." + name)
			if fn.IsNil() || fn.IsDeclaration() {
				t.Fatal("executed portable SDK entry lacks its real body", name)
			}
		}
		if !module.NamedGlobal(gcScanRuntimePackage + ".gcExpandersAVX512").IsNil() {
			t.Fatal("portable owner manufactured a native PC table")
		}
		if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Do([]string{mainPackage}, conf); err != nil {
		t.Fatal("build actual portable gc scan bodies", err)
	}
	if modules != 1 {
		t.Fatal("the actual portable gc scan module was not observed exactly once", modules)
	}
	if got := command(conf.OutFile); got != want {
		t.Fatal("LLGo portable owner differs from the independent Go oracle", got, want)
	}
}
