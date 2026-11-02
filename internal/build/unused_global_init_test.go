package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestUnusedGlobalInitCache(t *testing.T) {
	pkg := buildWasmReflectTestProgram(t, `package main; var M = map[string]int{"key":42}; func main() {}`)
	ctx := &context{conf: &packages.Config{}, mode: ModeGen, buildConf: &Config{Goos: "wasip1", Goarch: "wasm"}, progSSA: pkg.Prog}
	fingerprint := func(preserve bool) (string, string) {
		ctx.frontendOptions.UnusedGlobalInits = cl.FindUnusedGlobalInits(pkg.Prog, func(*ssa.Global) bool { return preserve })
		ap := &aPackage{SSA: pkg, Package: &packages.Package{PkgPath: "main", ID: "main"}}
		if err := ctx.collectFingerprint(ap); err != nil {
			t.Fatal(err)
		}
		return ap.Fingerprint, ap.Manifest
	}
	pruned, manifest := fingerprint(false)
	kept, _ := fingerprint(true)
	if pruned == kept || !strings.Contains(manifest, "unused_global_inits:") {
		t.Fatal("program-specific initializer plan is absent from cache identity")
	}
	if repeated, _ := fingerprint(false); repeated != pruned {
		t.Fatal("unstable initializer cache identity")
	}
}

func TestUnusedGlobalInitAssemblyReferences(t *testing.T) {
	t.Setenv("LDFLAGS", "")
	pkg := buildWasmReflectTestProgram(t, `package main
var Kept int64 = 42
var Dead int64 = 27
func Read() int64
func main() { Read() }
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "read_wasm.s")
	assembly := []byte("TEXT ·Read(SB),NOSPLIT,$0-8\n\tMOVD ·Kept(SB), R0\n\tMOVD R0, ret+0(FP)\n\tRET\n")
	p := &packages.Package{ID: "main", PkgPath: "main", Dir: dir, Types: pkg.Pkg,
		TypesSizes: &types.StdSizes{WordSize: 4, MaxAlign: 8}, OtherFiles: []string{path, filepath.Join(dir, "textflag.h")}}
	prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
	defer prog.Dispose()
	ctx := &context{prog: prog, progSSA: pkg.Prog,
		buildConf: &Config{BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm", Overlay: map[string][]byte{path: assembly}},
		pkgByID:   map[string]*aPackage{"main": {Package: p, SSA: pkg}}}
	configureUnusedGlobalInits(ctx)
	if names := ctx.frontendOptions.UnusedGlobalInits.Names(pkg); len(names) != 1 || names[0] != pkg.Var("Dead").String() {
		t.Fatalf("pruned %v; assembly-referenced global must survive", names)
	}
	// Arbitrary native inputs cannot be inspected through Go SSA.
	p.OtherFiles = append(p.OtherFiles, filepath.Join(dir, "opaque.c"))
	ctx.frontendOptions.UnusedGlobalInits = nil
	configureUnusedGlobalInits(ctx)
	if ctx.frontendOptions.UnusedGlobalInits != nil || canPruneWasmTypeMetadata(ctx) {
		t.Fatal("pruning remained enabled despite opaque native inputs")
	}
}

func TestUnusedGlobalInitLinkAliases(t *testing.T) {
	t.Setenv("LDFLAGS", "")
	pkg := buildWasmReflectTestProgram(t, `package main; var M = map[string]int{"key":42}; func main() {}`)
	for _, alias := range []string{"", "source", "target", "thread local", "goroutine local"} {
		prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
		name := llssa.FullName(pkg.Pkg, "M")
		if alias == "source" {
			prog.SetLinkname(name, "C.map")
		}
		if alias == "target" {
			prog.SetLinkname("other.alias", name)
		}
		if alias == "thread local" {
			prog.SetLocalityInfo(name, llssa.LocalityInfo{Locality: llssa.ThreadLocal})
		}
		if alias == "goroutine local" {
			prog.SetLocalityInfo(name, llssa.LocalityInfo{Locality: llssa.GoroutineLocal})
		}
		ctx := &context{prog: prog, progSSA: pkg.Prog, buildConf: &Config{BuildMode: BuildModeExe, Goarch: "wasm"}}
		configureUnusedGlobalInits(ctx)
		if got := len(ctx.frontendOptions.UnusedGlobalInits.Names(pkg)) != 0; got != (alias == "") {
			t.Fatalf("alias %q: pruned = %v", alias, got)
		}
		prog.Dispose()
	}
}

func TestUnusedGlobalInitScope(t *testing.T) {
	pkg := buildWasmReflectTestProgram(t, `package main; var M = 42; func main() {}`)
	for _, tc := range []struct {
		name, arch, ldflags string
		mode                Mode
		buildMode           BuildMode
		level               optlevel.Level
		want                bool
	}{
		{"wasm", "wasm", "", ModeBuild, BuildModeExe, optlevel.Oz, true},
		{"native", "arm64", "", ModeBuild, BuildModeExe, optlevel.Oz, false},
		{"debug", "wasm", "", ModeBuild, BuildModeExe, optlevel.O0, false},
		{"gen", "wasm", "", ModeGen, BuildModeExe, optlevel.Oz, false},
		{"library", "wasm", "", ModeBuild, BuildModeCArchive, optlevel.Oz, false},
		{"native object", "wasm", "extra.o", ModeBuild, BuildModeExe, optlevel.Oz, false},
		{"link export", "wasm", "-Wl,--export=main.M", ModeBuild, BuildModeExe, optlevel.Oz, false},
		{"LTO diagnostics", "wasm", "-Wl,--save-temps -Wl,--lto-obj-path=out.o", ModeBuild, BuildModeExe, optlevel.Oz, true},
		{"ThinLTO cache", "wasm", "-Wl,--thinlto-cache-dir=cache", ModeBuild, BuildModeExe, optlevel.Oz, true},
		{"combined cache and root", "wasm", "-Wl,--thinlto-cache-dir=cache,--undefined=main.M", ModeBuild, BuildModeExe, optlevel.Oz, false},
		{"combined map and export", "wasm", "-Wl,-Map=out.map,--export=main.M", ModeBuild, BuildModeExe, optlevel.Oz, false},
		{"combined LTO output and root", "wasm", "-Wl,--lto-obj-path=out.o,--undefined=main.M", ModeBuild, BuildModeExe, optlevel.Oz, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LDFLAGS", tc.ldflags)
			prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: tc.arch})
			defer prog.Dispose()
			ctx := &context{prog: prog, progSSA: pkg.Prog, mode: tc.mode,
				buildConf: &Config{BuildMode: tc.buildMode, Goarch: tc.arch, OptLevel: tc.level}}
			configureUnusedGlobalInits(ctx)
			if got := canPruneWasmTypeMetadata(ctx); got != tc.want {
				t.Fatalf("metadata pruning=%v, want %v", got, tc.want)
			}
			if got := len(ctx.frontendOptions.UnusedGlobalInits.Names(pkg)) != 0; got != tc.want {
				t.Fatalf("pruned=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestUnusedGlobalInitExportAnnotation(t *testing.T) {
	t.Setenv("LDFLAGS", "")
	src := "package main\n//go:linkname\tM\nvar M = 42\nfunc main() {}"
	pkg := buildWasmReflectTestProgram(t, src)
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
	defer prog.Dispose()
	p := &packages.Package{ID: "main", PkgPath: "main", Types: pkg.Pkg, Syntax: []*ast.File{file}}
	ctx := &context{prog: prog, progSSA: pkg.Prog, buildConf: &Config{BuildMode: BuildModeExe, Goarch: "wasm"},
		pkgByID: map[string]*aPackage{"main": {Package: p, SSA: pkg}}}
	for _, alternate := range []bool{false, true} {
		if alternate {
			p.Syntax = nil
			ctx.pkgByID["main"].AltPkg = &packages.Cached{Types: pkg.Pkg, Syntax: []*ast.File{file}}
		}
		configureUnusedGlobalInits(ctx)
		if names := ctx.frontendOptions.UnusedGlobalInits.Names(pkg); len(names) != 0 {
			t.Fatalf("alternate=%v: single-name exported globals were pruned: %v", alternate, names)
		}
	}
}

func TestUnusedGlobalInitRewrite(t *testing.T) {
	t.Setenv("LDFLAGS", "")
	pkg := buildWasmReflectTestProgram(t, `package main; var Version = "dev"; func main() {}`)
	prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
	defer prog.Dispose()
	p := &packages.Package{ID: "main", PkgPath: "main", Types: pkg.Pkg}
	ctx := &context{prog: prog, progSSA: pkg.Prog, buildConf: &Config{BuildMode: BuildModeExe, Goarch: "wasm"},
		pkgByID: map[string]*aPackage{"main": {Package: p, SSA: pkg, rewriteVars: map[string]string{"Version": "release"}}}}
	configureUnusedGlobalInits(ctx)
	if names := ctx.frontendOptions.UnusedGlobalInits.Names(pkg); len(names) != 0 {
		t.Fatalf("rewritten initializer was pruned: %v", names)
	}
}

func TestUnusedGlobalInitNativeInputs(t *testing.T) {
	t.Setenv("LDFLAGS", "")
	t.Setenv(llgoPlan9ASMPkgs, "all")
	for _, tc := range []struct {
		name, declarations, assembly, native, dir string
		prune, asmDisabled                        bool
	}{
		{name: "native object", native: "extra.o"},
		{name: "LLGoFiles", declarations: `const LLGoFiles = "extra.c"`},
		{name: "linked package", declarations: `const LLGoPackage = "link: extra"`},
		{name: "C import", declarations: `import "C"`},
		{name: "raw C import", declarations: "import `C`"},
		{name: "cgo linker flag", declarations: "//go:cgo_ldflag \"-pthread\"\n"},
		{name: "cgo dynamic import", declarations: "//go:cgo_import_dynamic local remote \"lib.so\"\n"},
		{name: "missing assembly", native: "read_wasm.s"},
		{name: "invalid assembly", native: "read_wasm.s", assembly: "TEXT ·Read(SB),NOSPLIT,$0-8\n\tINVALID\n"},
		{name: "disabled assembly", native: "read_wasm.s", assembly: "TEXT ·Read(SB),NOSPLIT,$0-8\n\tRET\n", asmDisabled: true},
		{name: "empty assembly", native: "read_wasm.s", assembly: "// placeholder\n", prune: true},
		{name: "empty native declarations", declarations: `const LLGoFiles = ""; const LLGoPackage = "normal"`, prune: true},
		{name: "runtime provider", native: "extra.c", dir: "runtime", prune: true},
		{name: "library provider", native: "extra.c", dir: "lib", prune: true},
		{name: "outside runtime", native: "extra.c", dir: "runtime-other"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// C imports are loader metadata; keep them out of the ordinary
			// Go type checker while exercising the complete pruning gate.
			src := "package main\n" + tc.declarations + "\nvar M = 42; func Read() int64; func main() {}"
			ssaSrc := src
			if strings.HasPrefix(tc.declarations, "import") {
				ssaSrc = strings.Replace(src, tc.declarations, "", 1)
			}
			pkg := buildWasmReflectTestProgram(t, ssaSrc)
			file, err := parser.ParseFile(token.NewFileSet(), "main.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if tc.dir != "" {
				dir = filepath.Join(env.LLGoROOT(), tc.dir, "init-test")
			}
			p := &packages.Package{ID: "main", PkgPath: "main", Dir: dir, Types: pkg.Pkg,
				TypesSizes: &types.StdSizes{WordSize: 4, MaxAlign: 8}, Syntax: []*ast.File{file}}
			prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
			defer prog.Dispose()
			conf := &Config{BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm", Verbose: true}
			if tc.native != "" {
				path := filepath.Join(dir, tc.native)
				p.OtherFiles = []string{path}
				if tc.assembly != "" {
					conf.Overlay = map[string][]byte{path: []byte(tc.assembly)}
				}
			}
			ctx := &context{prog: prog, progSSA: pkg.Prog, buildConf: conf,
				pkgByID: map[string]*aPackage{"main": {Package: p, SSA: pkg}}}
			if tc.asmDisabled {
				t.Setenv(llgoPlan9ASMPkgs, "none")
			}
			configureUnusedGlobalInits(ctx)
			if got := canPruneWasmTypeMetadata(ctx); got != tc.prune {
				t.Fatalf("metadata pruning=%v, want %v", got, tc.prune)
			}
			if got := len(ctx.frontendOptions.UnusedGlobalInits.Names(pkg)) != 0; got != tc.prune {
				t.Fatalf("pruned=%v, want %v", got, tc.prune)
			}
		})
	}
}

func TestWasmMetadataEligibilityWithoutInitializerPlan(t *testing.T) {
	t.Setenv("LDFLAGS", "")
	pkg := buildWasmReflectTestProgram(t, "package main; func main() {}")
	prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
	defer prog.Dispose()
	ctx := &context{prog: prog, progSSA: pkg.Prog, mode: ModeBuild, buildConf: &Config{Goos: "wasip1", Goarch: "wasm", BuildMode: BuildModeExe, OptLevel: optlevel.O2}}
	configureUnusedGlobalInits(ctx)
	if !canPruneWasmTypeMetadata(ctx) {
		t.Fatal("closed optimized Wasm program was rejected")
	}
	// An absent initializer-removal plan is not a change in native visibility.
	ctx.frontendOptions.UnusedGlobalInits = nil
	if !canPruneWasmTypeMetadata(ctx) {
		t.Fatal("metadata eligibility depends on the initializer-removal plan")
	}
}
