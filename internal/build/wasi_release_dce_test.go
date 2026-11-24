package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/env"
	llssa "github.com/xgo-dev/llgo/internal/llvmgen"
	"github.com/xgo-dev/llgo/internal/meta"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llvm"
)

func TestWasiSizePruningAssembly(t *testing.T) {
	goroot := t.TempDir()
	dir := filepath.Join(goroot, "src", "internal", "bytealg")
	path := filepath.Join(dir, "compare_wasm.s")
	direct := parseWasmAggregateIR(t, `
@input = external global i32
@unused = external global i32
declare i32 @callGo(i32)
declare void @unusedGo()
define i32 @asmEntry() {
  %input = load i32, ptr @input
  %result = call i32 @callGo(i32 %input)
  ret i32 %result
}
`)
	indirect := parseWasmAggregateIR(t, `
define void @asmEntry(ptr %fn) {
  call void %fn()
  ret void
}
`)
	standard := &packages.Package{PkgPath: "internal/bytealg", Dir: dir}
	thirdParty := &packages.Package{PkgPath: "example.com/bytealg", Dir: t.TempDir()}

	known := &wasiSizePruningInputs{goroot: goroot}
	known.collectAsmReferences(standard, path, &Config{}, direct)
	if known.opaqueAsmPkg != "" || !slices.Equal(known.roots, []string{"callGo", "input"}) {
		t.Fatalf("standard assembly roots: %+v", known)
	}
	assembly := []struct {
		name string
		pkg  *packages.Package
		conf *Config
	}{
		{"outside GOROOT", thirdParty, &Config{}},
		{"overlaid", standard, &Config{Overlay: map[string][]byte{path: []byte("replacement")}}},
	}
	for _, test := range assembly {
		t.Run(test.name, func(t *testing.T) {
			in := &wasiSizePruningInputs{goroot: goroot}
			in.collectAsmReferences(test.pkg, path, test.conf, direct)
			if in.opaqueAsmPkg != test.pkg.PkgPath || len(in.roots) != 0 {
				t.Fatalf("opaque assembly modeled as ordinary roots: %+v", in)
			}
		})
	}
	in := &wasiSizePruningInputs{goroot: goroot}
	in.collectAsmReferences(standard, path, &Config{}, indirect)
	if in.opaqueAsmPkg != standard.PkgPath {
		t.Fatalf("indirect assembly call treated as closed: %+v", in)
	}
}

func TestWasiSizePruningEligibility(t *testing.T) {
	t.Setenv("LDFLAGS", "")
	t.Setenv(llgoPlan9ASMPkgs, "all")
	src := "package main\nfunc main() {}"
	ssaPkg := buildWasmReflectTestProgram(t, src)
	for _, test := range []struct {
		name      string
		flags     string
		decl      string
		arch      crosscompile.WasmProfile
		proof     bool
		noRoot    bool
		nativeAsm string
	}{
		{name: "closed", arch: crosscompile.WasmProfileW32, proof: true},
		{name: "known C export", decl: "//export main\n", arch: crosscompile.WasmProfileW32, proof: true},
		{name: "unknown export", decl: "//go:wasmexport hostMain\n", arch: crosscompile.WasmProfileW32},
		{name: "explicit linker root", flags: "-Wl,--export=hostMain", arch: crosscompile.WasmProfileW32},
		{name: "JS", arch: crosscompile.WasmProfileJ32},
		{name: "no source roots", arch: crosscompile.WasmProfileW32, noRoot: true},
		{name: "bundled assembly", arch: crosscompile.WasmProfileW32, nativeAsm: "bundled"},
		{name: "alternate assembly", arch: crosscompile.WasmProfileW32, nativeAsm: "alternate"},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
			defer prog.Dispose()
			file, err := parser.ParseFile(token.NewFileSet(), "main.go",
				"package main\n"+test.decl+"func main() {}", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			pkg := &packages.Package{PkgPath: "main", Name: "main", Types: ssaPkg.Pkg, Syntax: []*ast.File{file}}
			conf := &Config{
				Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm",
				OptLevel: optlevel.Oz, WasmOpt: WasmOptSize, PrintCommands: true,
				LinkOptions: LinkOptions{ExternalLinkerFlags: test.flags},
			}
			ctx := &context{
				mode: ModeBuild, prog: prog, progSSA: ssaPkg.Prog, buildConf: conf,
				initial: []*packages.Package{pkg},
				pkgByID: map[string]*aPackage{"main": {Package: pkg}},
				crossCompile: crosscompile.Export{
					WasmProfile: test.arch, WasmProvider: crosscompile.WasmProviderWASI,
				},
			}
			if test.noRoot {
				ctx.initial = nil
			}
			if test.nativeAsm != "" {
				dir := t.TempDir()
				if test.nativeAsm == "bundled" {
					dir = filepath.Join(env.LLGoROOT(), "runtime")
				}
				native := &packages.Package{ID: "runtime-test", PkgPath: "example.com/runtime-test", Dir: dir,
					OtherFiles: []string{filepath.Join(dir, "read_wasm.s")}}
				if test.nativeAsm == "bundled" {
					ctx.pkgByID[native.ID] = &aPackage{Package: native}
				} else {
					ctx.pkgByID["main"].AltPkg = &packages.Cached{Package: native}
				}
			}
			inputs := ctx.wasiSizePruningInputs(t.TempDir())
			configureUnusedGlobalInits(ctx, inputs)
			configureWasiReleaseDCE(ctx, inputs)
			if got := conf.deadcodeDropEnabled(); got != test.proof {
				t.Fatalf("WASI -Oz pruning = %v, want %v", got, test.proof)
			}
		})
	}
}

func TestWasiSizePruningRetainsEntryAndAssemblyRoots(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"})
	defer prog.Dispose()
	installLocalContextTestRuntime(prog)
	src := prog.NewPackage("example.com/dep", "dep")
	builder := meta.NewBuilder()
	const namePrefix = "_llgo_iface_use."
	for typ, symbol := range map[string]string{
		"_llgo_float64": "example.com/dep.init",
		"_llgo_int64":   "example.com/dep.fromAsm",
		"_llgo_uint64":  "example.com/dep.unreachable",
	} {
		mod := src.Module()
		llvm.AddFunction(mod, namePrefix+typ, llvm.FunctionType(mod.Context().Int1Type(), nil, false))
		typeSymbol := builder.Sym(typ)
		builder.AddOrdinaryEdge(typeSymbol, typeSymbol)
		builder.AddIfaceUse(builder.Sym(symbol), typeSymbol)
	}
	packageMeta, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	defer packageMeta.Close()
	dep := &aPackage{Package: &packages.Package{PkgPath: src.Path()}, LPkg: src, Meta: packageMeta}
	app := &packages.Package{ID: "app", PkgPath: "example.com/app", ExportFile: filepath.Join(t.TempDir(), "app")}
	ctx := &context{
		prog: prog, wasmClosedProgram: true, wasiSizeRoots: []string{"example.com/dep.fromAsm"},
		buildConf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm",
			OptLevel: optlevel.Oz, GenLL: true, wasiReleaseDCE: true},
		crossCompile: crosscompile.Export{CC: "clang", CCFLAGS: []string{"-target", "wasm32-unknown-wasip1"}},
	}
	preparation := &mainLinkPreparation{
		linkedOrder: []*aPackage{dep},
		gen:         genConfig{packageInits: []string{"example.com/dep.init"}},
	}
	plan, err := buildMainLink(ctx, app, preparation, "", false)
	if err != nil {
		t.Fatal(err)
	}
	defer removeFiles(plan.linkInputs)
	raw, err := os.ReadFile(app.ExportFile + "-main.ll")
	if err != nil {
		t.Fatal(err)
	}
	module := parseWasmAggregateIR(t, string(raw))
	for typ, want := range map[string]uint64{"_llgo_float64": 1, "_llgo_int64": 1, "_llgo_uint64": 0} {
		fn := module.NamedFunction(namePrefix + typ)
		if fn.IsNil() || fn.IsDeclaration() {
			t.Errorf("missing interface predicate for %s", typ)
			continue
		}
		value := fn.FirstBasicBlock().FirstInstruction().Operand(0)
		if got := value.ZExtValue(); got != want {
			t.Errorf("interface predicate for %s returned %d, want %d", typ, got, want)
		}
	}
}
