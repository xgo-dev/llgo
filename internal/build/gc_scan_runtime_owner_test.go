//go:build go1.26

package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	xtoolllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
	"github.com/xgo-dev/llvm"
)

const gcScanRuntimePackage = "internal/runtime/gc/scan"

// These are the complete supported expansion factors, not a sample of the
// first failing private entry. The SDK's PCs carry AX input and Z1/Z2 output;
// they are not ordinary void(ptr) Go functions.
func gcScanRuntimeFactors() []int {
	return []int{
		1, 2, 3, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32,
		36, 40, 44, 48, 52, 56, 60, 64,
	}
}

func TestGCScanRuntimeAMD64CompleteOriginalFamily(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src", "internal", "runtime", "gc", "scan")
	var expanders []string
	for _, factor := range gcScanRuntimeFactors() {
		expanders = append(expanders, fmt.Sprintf("expandAVX512_%d<>", factor))
	}
	text := regexp.MustCompile(`(?m)^\s*TEXT\s+·([^\s(,]+)\(SB\),`)
	for _, owner := range []struct {
		file string
		want []string
	}{
		{"expand_amd64.s", expanders},
		{"scan_amd64.s", []string{"ExpandAVX512", "scanSpanPackedAVX512"}},
		{"filter_amd64.s", []string{"FilterNilAVX512"}},
	} {
		t.Run(owner.file, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(dir, owner.file))
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, match := range text.FindAllSubmatch(source, -1) {
				names = append(names, string(match[1]))
			}
			if !reflect.DeepEqual(names, owner.want) {
				t.Fatalf("complete original SDK owner changed: got %v, want %v", names, owner.want)
			}
			if owner.file == "expand_amd64.s" {
				pc := regexp.MustCompile(`\$·(expandAVX512_[0-9]+<>)\(SB\)`)
				var entries []string
				for _, match := range pc.FindAllSubmatch(source, -1) {
					entries = append(entries, string(match[1]))
				}
				if !reflect.DeepEqual(entries, expanders) {
					t.Fatal("the actual PC table does not own every private expander exactly once", entries)
				}
			}
		})
	}
}

// Compile the actual complete SDK package on every LLVM host. ModeBuild owns
// and releases its program; inspect the live module only inside ModuleHook.
// This is object/owner evidence, not execution of the original AVX512 source.
func TestGCScanRuntimeAMD64CompletePackage(t *testing.T) {
	conf := NewDefaultConf(ModeBuild)
	conf.Goos, conf.Goarch = runtime.GOOS, "amd64"
	// The package is cross-compiled on ARM64 hosts. Give the real IR compiler
	// the same AMD64 target, including for dependencies with x86 inline asm.
	t.Setenv("CCFLAGS", "--target="+xtoolllvm.GetTargetTriple(conf.Goos, conf.Goarch))
	conf.OptLevel, conf.ForceRebuild = optlevel.O0, true
	conf.BinPath = t.TempDir()
	bodies := map[string]bool{}
	modules := 0
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath != gcScanRuntimePackage {
			return
		}
		modules++
		module := pkg.LPkg.Module()
		for _, name := range []string{"ExpandAVX512", "scanSpanPackedAVX512", "FilterNilAVX512"} {
			function := module.NamedFunction(gcScanRuntimePackage + "." + name)
			bodies[name] = !function.IsNil() && !function.IsDeclaration()
		}
		if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
			t.Fatal("the complete selected SDK package has invalid LLVM IR", err)
		}
	}
	pkgs, err := Build(Invocation{Args: []string{gcScanRuntimePackage}, Config: conf})
	if err != nil {
		t.Fatal("compile the complete selected AMD64 SDK scan owner", err)
	}
	if modules != 1 {
		t.Fatal("the complete SDK scan package was not observed exactly once", modules)
	}
	for _, name := range []string{"ExpandAVX512", "scanSpanPackedAVX512", "FilterNilAVX512"} {
		if !bodies[name] {
			t.Fatal("the runtime replacement lacks an actual typed Go body", name)
		}
	}
	pkg := findPkg(pkgs, gcScanRuntimePackage)
	if pkg == nil || pkg.ArchiveFile == "" {
		t.Fatal("the complete SDK package did not publish its object archive")
	}
	info, err := os.Stat(pkg.ArchiveFile)
	if err != nil || info.Size() <= 8 {
		t.Fatal("the complete SDK package has no nonempty object archive", err)
	}
}

func TestGCScanRuntimeAMD64SourceOwnerBoundaries(t *testing.T) {
	ctx := sourcePatchBuildContext{goos: runtime.GOOS, goarch: "amd64", goversion: runtime.Version()}
	runtimeDir := env.LLGoRuntimeDir()
	dir := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(gcScanRuntimePackage))
	patch := filepath.Join(runtimeDir, "_patch", filepath.FromSlash(gcScanRuntimePackage), "scan_amd64.go")
	body, err := os.ReadFile(patch)
	if err != nil {
		t.Fatal(err)
	}
	apply := func(base map[string][]byte) (map[string][]byte, error) {
		changed, overlay, _, err := applySourcePatchForPkg(base, nil, runtimeDir, runtime.GOROOT(), gcScanRuntimePackage, ctx)
		if err == nil && !changed {
			return nil, fmt.Errorf("missing complete source owner")
		}
		return overlay, err
	}
	good, err := apply(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"expand_amd64.s", "scan_amd64.s", "filter_amd64.s"} {
		if string(good[filepath.Join(dir, name)]) != "// replaced by LLGo source patch\n" {
			t.Fatal("missing whole assembly owner", name)
		}
	}
	for _, name := range []string{"expand_amd64.go", "scan_amd64.go", "filter_amd64.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), name, good[filepath.Join(dir, name)], 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok && ident.Name == "gcExpandersAVX512" {
				t.Fatal("the selected Go view retains a fake or unowned PC table")
			}
			return true
		})
	}
	if _, err := apply(good); err != nil {
		t.Fatal("the complete owner cannot be safely reused", err)
	}
	parsed := token.NewFileSet()
	file, err := parser.ParseFile(parsed, patch, body, 0)
	if err != nil {
		t.Fatal(err)
	}
	var noBody []byte
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "ExpandAVX512" {
			start, end := parsed.Position(fn.Body.Pos()).Offset, parsed.Position(fn.Body.End()).Offset
			noBody = append(append([]byte{}, body[:start]...), body[end:]...)
		}
	}
	if len(noBody) == 0 {
		t.Fatal("missing-body negative did not modify the actual owner")
	}
	for _, test := range []struct {
		name string
		base map[string][]byte
	}{
		{"missing body", map[string][]byte{patch: bytes.Replace(body, []byte("func ExpandAVX512("), []byte("func MissingExpand("), 1)}},
		{"declaration only", map[string][]byte{patch: noBody}},
		{"wrong width", map[string][]byte{patch: bytes.Replace(body, []byte("n int32"), []byte("n int64"), 1)}},
		{"empty source", map[string][]byte{filepath.Join(dir, "expand_amd64.s"): nil}},
		{"new public TEXT", map[string][]byte{filepath.Join(dir, "scan_amd64.s"): []byte("TEXT ·Unexpected(SB),$0-0\nRET\n")}},
		{"remaining PC consumer", map[string][]byte{filepath.Join(dir, "scan_go.go"): []byte("package scan\nfunc usePC() uintptr { return gcExpandersAVX512[1] }\n")}},
		{"duplicate PC declaration", map[string][]byte{filepath.Join(dir, "scan_go.go"): []byte("package scan\nvar gcExpandersAVX512 [1]uintptr\n")}},
		{"partial marker", map[string][]byte{filepath.Join(dir, "scan_amd64.s"): good[filepath.Join(dir, "scan_amd64.s")]}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := apply(test.base); err == nil {
				t.Fatal("accepted an incomplete or changed gc scan owner")
			}
		})
	}
	t.Run("new assembly owner", func(t *testing.T) {
		goroot := t.TempDir()
		fixture := filepath.Join(goroot, "src", filepath.FromSlash(gcScanRuntimePackage))
		if err := os.MkdirAll(fixture, 0o700); err != nil {
			t.Fatal(err)
		}
		for name := range gcScanReviewedSources {
			source, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			assemblyLanguageTestWrite(t, fixture, name, string(source))
		}
		assemblyLanguageTestWrite(t, fixture, "new_amd64.s", "TEXT ·NewEntry(SB),$0-0\nRET\n")
		if _, _, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, gcScanRuntimePackage, ctx); err == nil {
			t.Fatal("accepted a new assembly owner outside the complete reviewed group")
		}
	})
	for _, arch := range []string{"arm64", "386"} {
		t.Run("retain "+arch, func(t *testing.T) {
			other := ctx
			other.goarch = arch
			changed, _, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, runtime.GOROOT(), gcScanRuntimePackage, other)
			if err != nil || changed {
				t.Fatal("AMD64 owner changed another architecture", err)
			}
		})
	}
	t.Run("retain older SDK", func(t *testing.T) {
		older := ctx
		older.goversion = "go1.25.14"
		changed, _, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, runtime.GOROOT(), gcScanRuntimePackage, older)
		if err != nil || changed {
			t.Fatal("the Go1.26+ owner changed an older SDK route", err)
		}
	})
}

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
	body, err := os.ReadFile(filepath.Join(patchDir, "scan_amd64.go"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcePatchBuildContext{goos: runtime.GOOS, goarch: runtime.GOARCH, goversion: runtime.Version()}
	_, overlay, _, err := applySourcePatchForPkg(nil, nil, env.LLGoRuntimeDir(), runtime.GOROOT(), gcScanRuntimePackage, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOARCH != "amd64" {
		// Explicitly select the same bodies for portable algorithm execution.
		// Do not change the production architecture selection or CPU predicates.
		body = bytes.TrimPrefix(body, []byte("//go:build amd64 && go1.26\n"))
		overlay = map[string][]byte{filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(gcScanRuntimePackage), "z_llgo_gc_scan_execution.go"): body}
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
	if want != "gc scan owners 26 classes 104 patterns" {
		t.Fatal("the independent Go execution denominator changed", want)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.GOEXPERIMENT = "greenteagc"
	conf.OptLevel, conf.LTO = optlevel.O0, lto.Off
	conf.ForceRebuild, conf.Overlay = true, overlay
	conf.SourceOverlay = make(map[string]SourceOverlayFile, len(replace))
	for name, backing := range replace {
		conf.SourceOverlay[name] = SourceOverlayFile{BackingFile: backing, Contents: overlay[name]}
	}
	conf.OutFile = filepath.Join(dir, "llgo"+extension)
	modules := 0
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath != gcScanRuntimePackage {
			return
		}
		modules++
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
