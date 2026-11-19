package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/buildenv"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/meta"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
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

	known := &linkPruningInputs{goroot: goroot}
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
			in := &linkPruningInputs{goroot: goroot}
			in.collectAsmReferences(test.pkg, path, test.conf, direct)
			if in.opaqueAsmPkg != test.pkg.PkgPath || len(in.roots) != 0 {
				t.Fatalf("opaque assembly modeled as ordinary roots: %+v", in)
			}
		})
	}
	in := &linkPruningInputs{goroot: goroot}
	in.collectAsmReferences(standard, path, &Config{}, indirect)
	if in.opaqueAsmPkg != standard.PkgPath {
		t.Fatalf("indirect assembly call treated as closed: %+v", in)
	}
}

func TestWasiSizePruningAssemblySystemRegisters(t *testing.T) {
	goroot := t.TempDir()
	dir := filepath.Join(goroot, "src", "internal", "cpu")
	pkg := &packages.Package{PkgPath: "internal/cpu", Dir: dir}
	path := filepath.Join(dir, "cpu_arm64.s")
	const read = `%n = call i64 asm sideeffect "mrs $0, ID_AA64ISAR0_EL1", "=r"()`
	for _, tc := range []struct {
		name, call   string
		native, want bool
	}{
		{name: "read", call: read, native: true, want: true},
		{name: "write", call: `call void asm sideeffect "msr S3_3_C4_C2_5, $0", "r,~{memory}"(i64 1)`, native: true, want: true},
		{name: "wasm", call: read},
		{name: "escape", call: strings.Replace(read, "EL1", "EL1; bl external", 1), native: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mod := parseWasmAggregateIR(t, "define void @asmEntry() { "+tc.call+" ret void }")
			in := &linkPruningInputs{goroot: goroot, nativeFormat: tc.native}
			in.collectAsmReferences(pkg, path, &Config{}, mod)
			if got := in.opaqueAsmPkg == ""; got != tc.want {
				t.Fatalf("register instruction modeled as closed = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWasiSizePruningEligibility(t *testing.T) {
	t.Setenv("LDFLAGS", "")
	t.Setenv(llgoPlan9ASMPkgs, "all")
	src := "package main\nfunc main() {}"
	ssaPkg := buildWasmReflectTestProgram(t, src)
	for _, test := range []struct {
		name        string
		flags       string
		envFlags    string
		wantLog     string
		decl        string
		arch        crosscompile.WasmProfile
		proof       bool
		manualDCE   bool
		forFormat   bool
		noRoot      bool
		runtimeFree bool
		nativeAsm   string
		asmSource   string
		noAsm       bool
		lto         lto.Mode
		plugin      string
	}{
		{name: "closed", arch: crosscompile.WasmProfileW32, proof: true},
		{name: "full LTO", arch: crosscompile.WasmProfileW32, lto: lto.Full, proof: true},
		{name: "manual DCE", arch: crosscompile.WasmProfileW32, manualDCE: true},
		{name: "manual DCE and formatter", arch: crosscompile.WasmProfileW32, lto: lto.Full, manualDCE: true, forFormat: true},
		{name: "full LTO with plugin", arch: crosscompile.WasmProfileW32, lto: lto.Full, plugin: "unknown-pass.so", wantLog: "LTO pass plugin"},
		{name: "full LTO with bundled assembly", arch: crosscompile.WasmProfileW32, lto: lto.Full, nativeAsm: "bundled"},
		{name: "thin LTO", arch: crosscompile.WasmProfileW32, lto: lto.Thin, wantLog: "retain WASI code and metadata: ThinLTO"},
		{name: "link map", arch: crosscompile.WasmProfileW32, lto: lto.Full, envFlags: "-Wl,--print-map", proof: true},
		{name: "environment adds root", arch: crosscompile.WasmProfileW32, lto: lto.Full, envFlags: "-Wl,--print-map,--undefined=main.hostMain", wantLog: "linker flags"},
		{name: "bundled placeholder", arch: crosscompile.WasmProfileW32, lto: lto.Full, nativeAsm: "bundled", asmSource: "// linker placeholder\n", proof: true},
		{name: "alternate placeholder", arch: crosscompile.WasmProfileW32, lto: lto.Full, nativeAsm: "alternate", asmSource: "// linker placeholder\n", proof: true},
		{name: "known C export", decl: "//export main\n", arch: crosscompile.WasmProfileW32, proof: true},
		{name: "unknown export", decl: "//go:wasmexport hostMain\n", arch: crosscompile.WasmProfileW32, wantLog: "untracked host export in main"},
		{name: "explicit linker root", flags: "-Wl,--export=hostMain", arch: crosscompile.WasmProfileW32},
		{name: "JS", arch: crosscompile.WasmProfileJ32},
		{name: "runtime-free before formatter proof", arch: crosscompile.WasmProfileW32, runtimeFree: true},
		{name: "no source roots", arch: crosscompile.WasmProfileW32, noRoot: true},
		{name: "bundled assembly", arch: crosscompile.WasmProfileW32, nativeAsm: "bundled",
			asmSource: "TEXT ·Read(SB),NOSPLIT,$0-0\n\tRET\n", wantLog: "opaque assembly in example.com/runtime-test"},
		{name: "unreadable alternate assembly", arch: crosscompile.WasmProfileW32, nativeAsm: "alternate",
			noAsm: true, wantLog: "opaque assembly in example.com/runtime-test"},
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
				DeadcodeDrop: test.manualDCE,
				LTO:          test.lto, DisableGoGlobalDCE: true, LTOPlugin: lto.PassPlugin{Path: test.plugin},
				LinkOptions: LinkOptions{ExternalLinkerFlags: test.flags},
			}
			ctx := &context{
				mode: ModeBuild, prog: prog, progSSA: ssaPkg.Prog, buildConf: conf,
				commands: commandEnv{environ: []string{"LDFLAGS=" + test.envFlags}},
				initial:  []*packages.Package{pkg},
				pkgByID:  map[string]*aPackage{"main": {Package: pkg}},
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
				path := filepath.Join(dir, "read_wasm.s")
				native := &packages.Package{ID: "runtime-test", PkgPath: "example.com/runtime-test", Dir: dir,
					Types: types.NewPackage("example.com/runtime-test", "runtimetest"), OtherFiles: []string{path},
					TypesSizes: &types.StdSizes{WordSize: 4, MaxAlign: 8}}
				read := types.NewFunc(token.NoPos, native.Types, "Read", types.NewSignatureType(nil, nil, nil, nil, nil, false))
				native.Types.Scope().Insert(read)
				if test.noAsm {
					t.Setenv(llgoPlan9ASMPkgs, "none")
				}
				if test.asmSource != "" {
					conf.Overlay = map[string][]byte{path: []byte(test.asmSource)}
				}
				if test.nativeAsm == "bundled" {
					ctx.pkgByID[native.ID] = &aPackage{Package: native}
				} else {
					ctx.pkgByID["main"].AltPkg = &packages.Cached{Package: native}
				}
			}
			readStderr := captureStderr(t)
			ctx.wasmRuntimeFree = test.runtimeFree
			inputs := ctx.wasiSizePruningInputs(t.TempDir(), test.runtimeFree || test.forFormat)
			if test.runtimeFree && inputs != nil {
				t.Fatal("the formatter prepass must skip runtime-free programs")
			}
			manual := test.manualDCE && buildenv.Dev
			if wantInputs := !manual || test.forFormat; test.manualDCE && (inputs != nil) != wantInputs {
				t.Fatalf("manual DCE collects formatter inputs = %v, want %v", inputs != nil, wantInputs)
			}
			configureUnusedGlobalInits(ctx, inputs)
			configureWasiReleaseDCE(ctx, inputs)
			log := readStderr()
			if test.wantLog != "" && !strings.Contains(log, test.wantLog) {
				t.Fatalf("WASI -Oz diagnostics = %q, want %q", log, test.wantLog)
			}
			wantAutomatic := test.proof || test.manualDCE && !manual
			if conf.wasiReleaseDCE != wantAutomatic || conf.deadcodeDropEnabled() != (manual || wantAutomatic) {
				t.Fatalf("WASI -Oz automatic pruning = %v, want %v (manual DCE = %v)", conf.wasiReleaseDCE, wantAutomatic, manual)
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
