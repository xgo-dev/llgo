package build

import (
	"bytes"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/fmtspecialize"

	"github.com/xgo-dev/llgo/cl"
	buildenv "github.com/xgo-dev/llgo/internal/buildenv"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestHostedValueFormatSupport(t *testing.T) {
	native := Config{
		Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "darwin", Goarch: "arm64",
		LTO: lto.Full, OptLevel: optlevel.Oz, DisableGoGlobalDCE: true, DeadcodeDrop: true,
	}
	for _, tc := range []struct {
		name   string
		change func(*Config, *crosscompile.Export)
		want   bool
	}{
		{name: "darwin arm64", want: true},
		{name: "linux amd64", want: true, change: func(c *Config, _ *crosscompile.Export) { c.Goos, c.Goarch = "linux", "amd64" }},
		{name: "linux arm64", want: true, change: func(c *Config, _ *crosscompile.Export) { c.Goos = "linux" }},
		{name: "disabled specialization", change: func(c *Config, _ *crosscompile.Export) { c.disableSizeSpecialization = true }},
		{name: "shared library", change: func(c *Config, _ *crosscompile.Export) { c.BuildMode = BuildModeCShared }},
		{name: "ordinary build", change: func(c *Config, _ *crosscompile.Export) { c.DeadcodeDrop = false }},
		{name: "thin LTO", change: func(c *Config, _ *crosscompile.Export) { c.LTO = lto.Thin }},
		{name: "extra link files", change: func(_ *Config, x *crosscompile.Export) { x.ExtraFiles = []string{"bridge.o"} }},
		{name: "emscripten wasm32", want: true, change: func(c *Config, x *crosscompile.Export) {
			c.Goos, c.Goarch, c.WasmOpt = "js", "wasm", WasmOptSize
			x.WasmProfile, x.WasmProvider = crosscompile.WasmProfileJ32, crosscompile.WasmProviderEmscripten
		}},
		{name: "emscripten memory64", want: true, change: func(c *Config, x *crosscompile.Export) {
			c.Goos, c.Goarch, c.WasmOpt = "js", "wasm", WasmOptSize
			x.WasmProfile, x.WasmProvider = crosscompile.WasmProfileJ64, crosscompile.WasmProviderEmscripten
		}},
		{name: "raw js wasm", change: func(c *Config, x *crosscompile.Export) {
			c.Goos, c.Goarch, c.WasmOpt = "js", "wasm", WasmOptSize
			x.WasmProfile = crosscompile.WasmProfileJ32
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf, target := native, crosscompile.Export{}
			if tc.change != nil {
				tc.change(&conf, &target)
			}
			if got := hostedValueFormatSupport(&conf, target, cl.Options{}); got != (tc.want && buildenv.Dev) {
				t.Fatalf("hosted formatter enabled = %v", got)
			}
		})
	}
}

func TestNativeFormatImports(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name, pkg, lib, symbol, flag string
		replaced                     bool
		want                         bool
	}{
		{name: "Darwin runtime", pkg: "runtime", lib: "/usr/lib/libSystem.B.dylib", symbol: "libc_write", want: true},
		{name: "Darwin syscall alias", pkg: "syscall", symbol: "libc_write", want: true},
		{name: "resolver", pkg: "internal/syscall/unix", lib: "/usr/lib/libresolv.9.dylib", symbol: "libresolv_res_init", flag: "-lresolv", want: true},
		{name: "unexpected flag", pkg: "syscall", lib: "/usr/lib/libSystem.B.dylib", symbol: "libc_write", flag: "-ldylib"},
		{name: "unexpected import", pkg: "syscall", lib: "/opt/vendor/libfoo.dylib", symbol: "libc_write"},
		{name: "user package", pkg: "user", lib: "/usr/lib/libSystem.B.dylib", symbol: "libc_write"},
		{name: "source overlay", pkg: "runtime", lib: "/usr/lib/libSystem.B.dylib", symbol: "libc_write", replaced: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(root, "src", tc.pkg)
			path := filepath.Join(dir, "sys.go")
			pkg := &packages.Package{PkgPath: tc.pkg, Dir: dir, GoFiles: []string{path}}
			ctx := &context{buildConf: &Config{Goos: "darwin"}}
			if tc.replaced {
				ctx.buildConf.SourceOverlay = map[string]SourceOverlayFile{path: {}}
			}
			var flags []string
			if tc.flag != "" {
				flags = []string{tc.flag}
			}
			imports := []cgoImportDynamicDecl{{local: tc.symbol, library: tc.lib}}
			if got := allowsNativeFormatImports(ctx, &linkPruningInputs{goroot: root, nativeFormat: true}, pkg, flags, imports); got != tc.want {
				t.Fatalf("allowsNativeFormatImports = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestClosedLinkExportRoots(t *testing.T) {
	for _, tc := range []struct {
		name, declarations    string
		trusted, root, opaque bool
	}{
		{name: "exported function", declarations: "//export foreignMain\nfunc Entry() {}", root: true},
		{name: "exported method", declarations: "type T int\n//export foreignMain\nfunc (T) Entry() {}", opaque: true},
		{name: "trusted link", declarations: "//llgo:link C.entry\nfunc Entry() {}", trusted: true},
		{name: "foreign link", declarations: "//llgo:link C.entry\nfunc Entry() {}", opaque: true},
		{name: "wasm export", declarations: "//go:wasmexport entry\nfunc Entry() {}", opaque: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			pkg := checkFormatTestPackage(t, prog.Fset, "dep", "package dep\n"+tc.declarations+"\n", types.Config{})
			goroot := t.TempDir()
			if tc.trusted {
				pkg.Dir = filepath.Join(goroot, "src", pkg.PkgPath)
			}
			dep := prog.CreatePackage(pkg.Types, pkg.Syntax, pkg.TypesInfo, true)
			prog.Build()
			roots, opaque := formatExportRoots(prog, nil, pkg, goroot)
			if opaque != tc.opaque || (len(roots) != 0) != tc.root || tc.root && (len(roots) != 1 || roots[0] != dep.Func("Entry")) {
				t.Fatalf("roots=%v opaque=%v", roots, opaque)
			}
			if tc.root {
				// Other linker inputs may contain external symbols without a Go package.
				got := fmtspecialize.AppendSymbolRoots(prog, nil, []string{"C.external", "bare", "dep.Entry", "dep.absent"})
				if len(got) != 1 || got[0] != roots[0] {
					t.Fatalf("assembly roots=%v", got)
				}
			}
		})
	}
}

func TestClosedLinkDarwinSourceOverlay(t *testing.T) {
	root, patches := t.TempDir(), t.TempDir()
	dir := filepath.Join(root, "src", "os")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "executable_darwin.go")
	source := []byte("package os\nvar initCwd, initCwdErr = Getwd()\n")
	if err := os.WriteFile(path, source, 0644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, goos string
		tags       []string
		base       map[string][]byte
		rewrite    bool
	}{
		{"release", "darwin", []string{"llgo.size.native"}, nil, true},
		{"other overlay", "darwin", []string{"llgo.size.native"}, map[string][]byte{"keep.go": []byte("package keep")}, true},
		{"source replaced", "darwin", []string{"llgo.size.native"}, map[string][]byte{path: []byte("package replacement")}, false},
		{"other OS", "linux", []string{"llgo.size.native"}, nil, false},
		{"regular build", "darwin", nil, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, _, err := buildSourcePatchOverlayForGOROOT(tc.base, patches, root, sourcePatchBuildContext{goos: tc.goos, targetTags: tc.tags})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(out[path]), darwinExecutableCwdInitializer) != tc.rewrite {
				t.Fatalf("overlay: %s", out[path])
			}
			for name, data := range tc.base {
				if !bytes.Equal(out[name], data) {
					t.Fatalf("input overlay %s was overwritten", name)
				}
			}
			if tc.rewrite && tc.base[path] != nil {
				t.Fatal("mutated the input overlay")
			}
		})
	}
}

func TestClosedLinkPruningBoundaries(t *testing.T) {
	if !buildenv.Dev {
		t.Skip("closed-link pruning requires DCE")
	}
	t.Setenv("LDFLAGS", "")
	for _, tc := range []struct {
		name, target                         string
		foreignObject, altExport, opaqueAsm  bool
		wantInputs, wantNative, wantEmbedded bool
	}{
		{name: "closed native", wantInputs: true, wantNative: true},
		{name: "foreign object", foreignObject: true},
		{name: "alternate source exports an entry", altExport: true, wantInputs: true},
		{name: "closed firmware", target: "esp32c3-basic", wantInputs: true, wantEmbedded: true},
		{name: "unmodeled firmware assembly", target: "esp32c3-basic", opaqueAsm: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			goroot := t.TempDir()
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			pkg := checkFormatTestPackage(t, prog.Fset, "main", "package main;func main() {}", types.Config{})
			pkg.Dir = t.TempDir()
			prog.CreatePackage(pkg.Types, pkg.Syntax, pkg.TypesInfo, true)
			entry := &aPackage{Package: pkg}
			if tc.foreignObject {
				pkg.OtherFiles = []string{filepath.Join(pkg.Dir, "foreign.o")}
			}
			if tc.altExport {
				alt := checkFormatTestPackage(t, prog.Fset, "alt", "package alt\n//go:wasmexport start\nfunc Start() {}", types.Config{})
				alt.Dir = t.TempDir()
				prog.CreatePackage(alt.Types, alt.Syntax, alt.TypesInfo, true)
				entry.AltPkg = &packages.Cached{Package: alt, Types: alt.Types, Syntax: alt.Syntax}
			}
			prog.Build()
			conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "linux", Goarch: "amd64",
				LTO: lto.Full, OptLevel: optlevel.Oz, DeadcodeDrop: true, DisableGoGlobalDCE: true, Verbose: true, Target: tc.target}
			if tc.target != "" {
				conf.Goarch = "riscv32"
			}
			backend := llssa.NewProgram(&llssa.Target{GOOS: conf.Goos, GOARCH: conf.Goarch})
			defer backend.Dispose()
			ctx := &context{progSSA: prog, prog: backend, buildConf: conf, initial: []*packages.Package{pkg}, pkgByID: map[string]*aPackage{"main": entry}}
			inputs := ctx.sizePruningInputs(goroot, false)
			if tc.opaqueAsm {
				inputs.opaqueAsmPkg = "main"
			}
			got := configureFormattingAndInits(ctx, goroot, inputs)
			if (got != nil) != tc.wantInputs || ctx.nativeClosedProgram != tc.wantNative || ctx.embeddedClosedProgram != tc.wantEmbedded {
				t.Fatalf("inputs=%v native=%t embedded=%t", got, ctx.nativeClosedProgram, ctx.embeddedClosedProgram)
			}
		})
	}
}
