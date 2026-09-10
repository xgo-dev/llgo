//go:build !llgo
// +build !llgo

package build

import (
	"go/constant"
	"go/token"
	"go/types"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llvm"

	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func init() {
	llssa.Initialize(llssa.InitAll)
}

func TestGenMainModuleExecutable(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	ctx.prog.EnableFuncInfoMetadata(true)
	ctx.prog.EnableFuncInfoSites(true)
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	mod := genMainModule(ctx, llssa.PkgRuntime, pkg,
		&genConfig{rtInit: true, pyInit: true, packageInits: []string{"example.com/b.init", "example.com/z.init", "example.com/a.init"}})
	if mod.ExportFile != "foo.a-main" {
		t.Fatalf("unexpected export file: %s", mod.ExportFile)
	}
	ir := mod.LPkg.String()
	checks := []string{
		"define i32 @" + processEntrySymbol + "(",
		"define void @runtime.main()",
		".pushsection llgo_funcinfo_entry",
		".quad " + uint64Hex(funcInfoSymbolID(runtimeMainSymbol)),
		".quad " + uint64Hex(funcInfoSymbolID(processEntrySymbol)),
		"call void @Py_Initialize()",
		"call void @Py_Finalize()",
		"call void @\"example.com/foo.init\"()",
		`@"example.com/foo..inittask" = global { i32, i32 } zeroinitializer`,
		"define weak void @_start()",
	}
	for _, want := range checks {
		if !strings.Contains(ir, want) {
			t.Fatalf("main module IR missing %q:\n%s", want, ir)
		}
	}
	funcNames := make(map[string]string)
	for _, rec := range readFuncInfo(mod.LPkg.Module()) {
		funcNames[rec.symbol] = rec.name
	}
	for symbol, want := range map[string]string{
		processEntrySymbol: runtimeGoexitName,
		runtimeMainSymbol:  runtimeMainSymbol,
	} {
		if got := funcNames[symbol]; got != want {
			t.Fatalf("funcinfo name for %q = %q, want %q", symbol, got, want)
		}
	}
	assertInOrder(t, ir,
		"call void @Py_Initialize()",
		`call void @"example.com/b.init"()`,
		`call void @"example.com/z.init"()`,
		`call void @"example.com/a.init"()`,
		"call void @\"example.com/foo.init\"()",
		"call void @\"example.com/foo.main\"()",
		"call void @Py_Finalize()",
	)
}

func TestGenMainModuleWindowsExitsAfterMain(t *testing.T) {
	llvm.InitializeAllTargets()
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "windows",
			Goarch:    "arm64",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	mod := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{rtInit: true})
	ir := mod.LPkg.String()
	assertInOrder(t, ir,
		`call void @"example.com/foo.main"()`,
		`call void @runtime.exit(i32 0)`,
	)
}

func TestGenMainModuleWindowsStdioNobufUsesUCRTStreams(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "1")
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "windows",
			Goarch:    "arm64",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	ir := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{}).LPkg.String()
	for _, want := range []string{
		"call ptr @__acrt_iob_func(i32 1)",
		"call ptr @__acrt_iob_func(i32 2)",
		"call i32 @setvbuf(",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("Windows stdio setup IR missing %q:\n%s", want, ir)
		}
	}
	if got := strings.Count(ir, "i32 4, i64 0)"); got != 2 {
		t.Fatalf("Windows stdio setup used _IONBF=4 %d times, want 2:\n%s", got, ir)
	}
	for _, unwanted := range []string{"@stdout =", "@stderr ="} {
		if strings.Contains(ir, unwanted) {
			t.Fatalf("Windows stdio setup IR contains unavailable UCRT global %q:\n%s", unwanted, ir)
		}
	}
}

func TestPackageInitOrderUsesLexicalReadyPackage(t *testing.T) {
	newPackage := func(path string, imports ...*packages.Package) *packages.Package {
		pkg := &packages.Package{ID: path, PkgPath: path, Imports: make(map[string]*packages.Package)}
		for _, imported := range imports {
			pkg.Imports[imported.PkgPath] = imported
		}
		return pkg
	}
	z := newPackage("example.com/z")
	a := newPackage("example.com/a", z)
	b := newPackage("example.com/b")
	root := newPackage("example.com/main", a, b)

	order, err := packageInitOrder(root)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(order))
	for i, pkg := range order {
		got[i] = pkg.PkgPath
	}
	want := []string{"example.com/b", "example.com/z", "example.com/a", "example.com/main"}
	if !slices.Equal(got, want) {
		t.Fatalf("package init order = %v, want %v", got, want)
	}
}

func TestPackageInitOrderEdgeCases(t *testing.T) {
	if order, err := packageInitOrder(nil); err != nil || order != nil {
		t.Fatalf("nil root order = %v, %v, want nil, nil", order, err)
	}

	first := &packages.Package{ID: "first", PkgPath: "example.com/same"}
	second := &packages.Package{ID: "second", PkgPath: "example.com/same"}
	root := &packages.Package{
		ID:      "root",
		PkgPath: "example.com/root",
		Imports: map[string]*packages.Package{
			"alias/first": first,
			"first":       first,
			"nil":         nil,
			"second":      second,
		},
	}
	order, err := packageInitOrder(root)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(order))
	for i, pkg := range order {
		got[i] = pkg.ID
	}
	if want := []string{"first", "second", "root"}; !slices.Equal(got, want) {
		t.Fatalf("package init order = %v, want %v", got, want)
	}

	a := &packages.Package{ID: "a", PkgPath: "example.com/a", Imports: make(map[string]*packages.Package)}
	b := &packages.Package{ID: "b", PkgPath: "example.com/b", Imports: map[string]*packages.Package{"a": a}}
	a.Imports["b"] = b
	if _, err := packageInitOrder(a); err == nil || !strings.Contains(err.Error(), "contains a cycle") {
		t.Fatalf("cyclic package order error = %v, want cycle error", err)
	}
}

func TestLinkedPackageInitNamesFiltersUnavailablePackages(t *testing.T) {
	newPackage := func(id string) *packages.Package {
		path := "example.com/" + id
		return &packages.Package{ID: id, PkgPath: path, Types: types.NewPackage(path, id)}
	}
	root := newPackage("root")
	normal := newPackage("normal")
	noinit := newPackage("noinit")
	noinit.Types.Scope().Insert(types.NewConst(token.NoPos, noinit.Types, "LLGoPackage", types.Typ[types.String], constant.MakeString("noinit")))
	missingBuilt := newPackage("missing-built")
	missingSSA := newPackage("missing-ssa")
	missingInit := newPackage("missing-init")
	missingTypes := &packages.Package{ID: "missing-types", PkgPath: "example.com/missing-types"}
	root.Imports = map[string]*packages.Package{
		"normal":        normal,
		"noinit":        noinit,
		"missing-built": missingBuilt,
		"missing-ssa":   missingSSA,
		"missing-init":  missingInit,
		"missing-types": missingTypes,
	}

	ssaProg := ssa.NewProgram(token.NewFileSet(), 0)
	normalSSA := ssaProg.CreatePackage(normal.Types, nil, nil, true)
	noinitSSA := ssaProg.CreatePackage(noinit.Types, nil, nil, true)
	if normalSSA.Func("init") == nil || noinitSSA.Func("init") == nil {
		t.Fatal("test SSA packages are missing synthetic init functions")
	}
	missingInitSSA := &ssa.Package{Members: make(map[string]ssa.Member)}

	linked := []Package{
		nil,
		&aPackage{},
		{Package: normal, SSA: normalSSA},
		{Package: noinit, SSA: noinitSSA},
		{Package: missingSSA},
		{Package: missingInit, SSA: missingInitSSA},
		{Package: missingTypes, SSA: normalSSA},
	}
	names, err := linkedPackageInitNames(root, linked)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"example.com/normal.init"}; !slices.Equal(names, want) {
		t.Fatalf("linked init names = %v, want %v", names, want)
	}

	cycle := &packages.Package{ID: "cycle", PkgPath: "example.com/cycle", Imports: make(map[string]*packages.Package)}
	cycle.Imports["self"] = cycle
	if _, err := linkedPackageInitNames(cycle, nil); err == nil {
		t.Fatal("linkedPackageInitNames accepted an import cycle")
	}
}

func TestLinkMainPkgRejectsPackageInitCycle(t *testing.T) {
	cycle := &packages.Package{ID: "cycle", PkgPath: "example.com/cycle", Imports: make(map[string]*packages.Package)}
	cycle.Imports["self"] = cycle
	ctx := &context{
		buildConf: &Config{},
		pkgs:      make(map[*packages.Package]Package),
		pkgByID:   make(map[string]Package),
	}
	if err := linkMainPkg(ctx, cycle, nil, "", false); err == nil || !strings.Contains(err.Error(), "contains a cycle") {
		t.Fatalf("linkMainPkg cycle error = %v, want cycle error", err)
	}
}

func TestPrepareMainLinkBuildsEntryPlan(t *testing.T) {
	llvm.InitializeAllTargets()
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      runtime.GOOS,
			Goarch:    runtime.GOARCH,
		},
		pkgs:    make(map[*packages.Package]Package),
		pkgByID: make(map[string]Package),
	}
	pkg := &packages.Package{ID: "example.com/main", PkgPath: "example.com/main"}
	plan, err := prepareMainLink(ctx, pkg, nil, "main.out", false)
	if err != nil {
		t.Fatal(err)
	}
	defer removeFiles(plan.linkInputs)
	if plan.outputPath != "main.out" || len(plan.linkInputs) == 0 {
		t.Fatalf("link plan = %+v", plan)
	}
}

func TestGenMainModuleWASIAsyncifyEntry(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	prog := llssa.NewProgram(nil)
	installLocalContextTestRuntime(prog)
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "wasip1",
			Goarch:    "wasm",
			Target:    "wasi",
		},
		crossCompile: crosscompile.Export{
			WasmPostLink: crosscompile.WasmPostLink{Asyncify: true},
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	mod := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{
		packageInits: []string{"example.com/dependency.init"},
	})
	ir := mod.LPkg.String()
	checks := []string{
		`define hidden ptr @__llgo_wasm_main(ptr %0)`,
		`call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.init"()`,
		`call void @"example.com/dependency.init"()`,
		`call void @"example.com/foo.init"()`,
		`call void @"example.com/foo.main"()`,
		`call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.RunWasmMain"()`,
	}
	for _, want := range checks {
		if !strings.Contains(ir, want) {
			t.Fatalf("WASI main module IR missing %q:\n%s", want, ir)
		}
	}
	taskStart := strings.Index(ir, "define hidden ptr @__llgo_wasm_main(")
	task := ir[taskStart:]
	task = task[:strings.Index(task, "}\n")+2]
	assertInOrder(t, task,
		`call void @"example.com/dependency.init"()`,
		`call void @"example.com/foo.init"()`,
		`call void @"example.com/foo.main"()`,
	)
	entryStart := strings.Index(ir, "define hidden i32 @__main_argc_argv(")
	if entryStart < 0 {
		t.Fatalf("WASI main module missing host entry:\n%s", ir)
	}
	if strings.Contains(ir, "define weak void @_start()") {
		t.Fatalf("WASI main module replaced wasi-libc _start:\n%s", ir)
	}
	entry := ir[entryStart:]
	entry = entry[:strings.Index(entry, "}\n")+2]
	assertInOrder(t, entry,
		"EnterLocalContext",
		`call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.init"()`,
		`call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.RunWasmMain"()`,
		"LeaveLocalContext",
	)
	if strings.Contains(entry, `call void @"example.com/dependency.init"()`) ||
		strings.Contains(entry, `call void @"example.com/foo.init"()`) ||
		strings.Contains(entry, `call void @"example.com/foo.main"()`) {
		t.Fatalf("WASI system-stack entry calls package main directly:\n%s", entry)
	}
}

func TestNeedsWasmRuntimeScheduler(t *testing.T) {
	tests := []struct {
		name   string
		conf   Config
		target crosscompile.Export
		want   bool
	}{
		{
			name:   "emscripten",
			conf:   Config{BuildMode: BuildModeExe},
			target: crosscompile.Export{WasmProfile: crosscompile.WasmProfileJ32, WasmProvider: crosscompile.WasmProviderEmscripten},
			want:   true,
		},
		{
			name:   "emscripten_memory64",
			conf:   Config{BuildMode: BuildModeExe},
			target: crosscompile.Export{WasmProfile: crosscompile.WasmProfileJ64, WasmProvider: crosscompile.WasmProviderEmscripten},
			want:   true,
		},
		{
			name: "wasi_asyncify",
			conf: Config{BuildMode: BuildModeExe},
			target: crosscompile.Export{
				WasmPostLink: crosscompile.WasmPostLink{Asyncify: true},
			},
			want: true,
		},
		{
			name:   "emscripten_library",
			conf:   Config{BuildMode: BuildModeCArchive},
			target: crosscompile.Export{WasmProfile: crosscompile.WasmProfileJ32, WasmProvider: crosscompile.WasmProviderEmscripten},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := &context{buildConf: &test.conf, crossCompile: test.target}
			if got := needsWasmRuntimeScheduler(ctx); got != test.want {
				t.Fatalf("needsWasmRuntimeScheduler() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestGenMainModuleLibrary(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeCArchive,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	mod := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{})
	ir := mod.LPkg.String()
	if strings.Contains(ir, "define i32 @main") {
		t.Fatalf("library mode should not emit main function:\n%s", ir)
	}
	if !strings.Contains(ir, "@__llgo_argc = global i32 0") {
		t.Fatalf("library mode missing argc global:\n%s", ir)
	}
	if !strings.Contains(ir, "@llvm.global_ctors") {
		t.Fatalf("library mode missing constructor:\n%s", ir)
	}
	if strings.Contains(ir, "EnableForeignThreadRegistration") {
		t.Fatalf("library without C exports enabled foreign threads:\n%s", ir)
	}
}

func TestGenMainModuleLibraryInitializesRuntime(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	for _, mode := range []BuildMode{BuildModeCArchive, BuildModeCShared} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := &context{
				prog: llssa.NewProgram(nil),
				buildConf: &Config{
					BuildMode: mode,
					Goos:      "linux",
					Goarch:    "amd64",
				},
			}
			pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
			mod := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{
				rtInit:       true,
				packageInits: []string{"example.com/dep.init"},
			})
			ir := mod.LPkg.String()
			checks := []string{
				"define internal void @__llgo_runtime_ctor(i32 %0, ptr %1)",
				"store i32 %0, ptr @__llgo_argc",
				"store ptr %1, ptr @__llgo_argv",
				"call void @\"github.com/xgo-dev/llgo/runtime/internal/runtime.init\"()",
				"call void @\"example.com/dep.init\"()",
				"call void @\"example.com/foo.init\"()",
			}
			if mode == BuildModeCShared {
				checks = append(checks, `@__llgo_runtime_ctor_init = hidden constant ptr @__llgo_runtime_ctor, section ".init_array"`)
			} else {
				checks = append(checks, "@llvm.global_ctors = appending global")
			}
			for _, want := range checks {
				if !strings.Contains(ir, want) {
					t.Fatalf("library module IR missing %q:\n%s", want, ir)
				}
			}
			assertInOrder(t, ir,
				"store i32 %0, ptr @__llgo_argc",
				"store ptr %1, ptr @__llgo_argv",
				"call void @\"github.com/xgo-dev/llgo/runtime/internal/runtime.init\"()",
				"call void @\"example.com/dep.init\"()",
				"call void @\"example.com/foo.init\"()",
			)
			if strings.Contains(ir, "define i32 @main") {
				t.Fatalf("library mode should not emit main function:\n%s", ir)
			}
		})
	}
}

func TestGenMainModuleWindowsCSharedInitializesFromCExport(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeCShared,
			Goos:      "windows",
			Goarch:    "arm64",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	ir := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{
		rtInit:       true,
		packageInits: []string{"example.com/dep.init"},
		cExports: []cExport{{
			goName: "example.com/foo.Exported",
			cName:  "Exported",
			sig:    llssa.NoArgsNoRet,
		}, {
			goName: "example.com/foo.Value",
			cName:  "Value",
			sig:    newSignature([]types.Type{types.Typ[types.Int32]}, []types.Type{types.Typ[types.Int32]}),
		}},
	}).LPkg.String()

	for _, want := range []string{
		"define internal void @__llgo_runtime_initialize()",
		"declare dllimport i32 @InitOnceExecuteOnce(",
		"define hidden void @__llgo_runtime_ensure_initialized()",
		"define void @Exported()",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("Windows c-shared module IR missing %q:\n%s", want, ir)
		}
	}
	for _, unwanted := range []string{"@llvm.global_ctors", "@__llgo_runtime_ctor"} {
		if strings.Contains(ir, unwanted) {
			t.Fatalf("Windows c-shared module IR contains loader-lock constructor %q:\n%s", unwanted, ir)
		}
	}
	assertInOrder(t, ir,
		"define internal void @__llgo_runtime_initialize()",
		`call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.init"()`,
		`call void @"example.com/dep.init"()`,
		`call void @"example.com/foo.init"()`,
	)
	wrapper := ir[strings.Index(ir, "define void @Exported()"):]
	assertInOrder(t, wrapper,
		"call void @__llgo_runtime_ensure_initialized()",
		`call i1 @"github.com/xgo-dev/llgo/runtime/internal/runtime.EnterForeignThread"()`,
		`call void @"example.com/foo.Exported"()`,
		`call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.ExitForeignThread"(i1`,
	)
	valueWrapper := ir[strings.Index(ir, "define i32 @Value(i32 %0)"):]
	assertInOrder(t, valueWrapper,
		"call void @__llgo_runtime_ensure_initialized()",
		`call i1 @"github.com/xgo-dev/llgo/runtime/internal/runtime.EnterForeignThread"()`,
		`call i32 @"example.com/foo.Value"(i32 %0)`,
		`call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.ExitForeignThread"(i1`,
		"ret i32",
	)
}

func TestGenMainModuleCArchiveRegistersCExportThread(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	for _, test := range []struct {
		goos   string
		goarch string
	}{
		{goos: "darwin", goarch: "arm64"},
		{goos: "linux", goarch: "amd64"},
		{goos: "windows", goarch: "386"},
	} {
		t.Run(test.goos+"/"+test.goarch, func(t *testing.T) {
			ctx := &context{
				prog: llssa.NewProgram(nil),
				buildConf: &Config{
					BuildMode: BuildModeCArchive,
					Goos:      test.goos,
					Goarch:    test.goarch,
				},
			}
			pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
			ir := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{
				rtInit: true,
				cExports: []cExport{{
					goName: "example.com/foo.Exported",
					cName:  "Exported",
					sig:    llssa.NoArgsNoRet,
				}},
			}).LPkg.String()

			if !strings.Contains(ir, "@llvm.global_ctors") {
				t.Fatalf("%s c-archive module is missing its runtime constructor:\n%s", test.goos, ir)
			}
			if strings.Contains(ir, "@__llgo_runtime_ensure_initialized") {
				t.Fatalf("%s c-archive module unexpectedly uses lazy DLL initialization:\n%s", test.goos, ir)
			}
			enablesForeignThreads := strings.Contains(ir,
				`call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.EnableForeignThreadRegistration"()`)
			if want := test.goos != "windows"; enablesForeignThreads != want {
				t.Fatalf("%s c-archive foreign-thread enable = %v, want %v:\n%s",
					test.goos, enablesForeignThreads, want, ir)
			}
			wrapper := ir[strings.Index(ir, "define void @Exported()"):]
			assertInOrder(t, wrapper,
				`call i1 @"github.com/xgo-dev/llgo/runtime/internal/runtime.EnterForeignThread"()`,
				`call void @"example.com/foo.Exported"()`,
				`call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.ExitForeignThread"(i1`,
			)
		})
	}
}

func TestGenMainModuleWindowsCShared386UsesStdcallInitOnce(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeCShared,
			Goos:      "windows",
			Goarch:    "386",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	ir := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{rtInit: true}).LPkg.String()
	for _, want := range []string{
		"declare dllimport x86_stdcallcc i32 @InitOnceExecuteOnce(",
		"call x86_stdcallcc i32 @InitOnceExecuteOnce(",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("Windows/386 c-shared module IR missing %q:\n%s", want, ir)
		}
	}
}

func TestGenMainModuleLibraryConstructorArgsByPlatform(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	for _, test := range []struct {
		goos     string
		wantArgs bool
	}{
		{goos: "linux", wantArgs: true},
		{goos: "darwin", wantArgs: true},
		{goos: "windows", wantArgs: false},
	} {
		t.Run(test.goos, func(t *testing.T) {
			ctx := &context{
				prog: llssa.NewProgram(nil),
				buildConf: &Config{
					BuildMode: BuildModeCShared,
					Goos:      test.goos,
					Goarch:    "amd64",
				},
			}
			pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
			ir := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{}).LPkg.String()
			hasArgSignature := strings.Contains(ir, "define internal void @__llgo_runtime_ctor(i32 %0, ptr %1)")
			hasArgStores := strings.Contains(ir, "store i32 %0, ptr @__llgo_argc") &&
				strings.Contains(ir, "store ptr %1, ptr @__llgo_argv")
			if hasArgSignature != test.wantArgs || hasArgStores != test.wantArgs {
				t.Fatalf("constructor argument capture = (%v, %v), want %v:\n%s", hasArgSignature, hasArgStores, test.wantArgs, ir)
			}
		})
	}
}

func TestGenMainModuleTestLibraryDefersMainInit(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	for _, mode := range []BuildMode{BuildModeCArchive, BuildModeCShared} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := &context{
				prog: llssa.NewProgram(nil),
				mode: ModeTest,
				buildConf: &Config{
					Mode:      ModeTest,
					BuildMode: mode,
					Goos:      "linux",
					Goarch:    "amd64",
				},
			}
			pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
			mod := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{
				rtInit:       true,
				packageInits: []string{"example.com/dep.init"},
			})
			ir := mod.LPkg.String()
			if !strings.Contains(ir, "call void @\"github.com/xgo-dev/llgo/runtime/internal/runtime.init\"()") {
				t.Fatalf("test library constructor missing runtime init:\n%s", ir)
			}
			if strings.Contains(ir, "call void @\"example.com/foo.init\"()") {
				t.Fatalf("test library constructor initialized test main before the C runner supplied argc/argv:\n%s", ir)
			}
			if strings.Contains(ir, "call void @\"example.com/dep.init\"()") {
				t.Fatalf("test library constructor initialized a test dependency before the C runner supplied argc/argv:\n%s", ir)
			}
		})
	}
}

func TestGenMainModuleInstallsLocalContextWhenNeeded(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	prog := llssa.NewProgram(nil)
	installLocalContextTestRuntime(prog)
	prog.SetLocalityInfo("example.com/state.Value", llssa.LocalityInfo{Locality: llssa.GoroutineLocal})
	prog.SetLocalStorage("example.com/state.Value", llssa.LocalStoragePackage)
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	ir := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{}).LPkg.String()
	assertInOrder(t, ir,
		"EnterLocalContext",
		"call void @runtime.main()",
		"LeaveLocalContext",
	)
}

func installLocalContextTestRuntime(prog llssa.Program) {
	runtimePkg := types.NewPackage(llssa.PkgRuntime, "runtime")
	contextName := types.NewTypeName(token.NoPos, runtimePkg, "LocalContext", nil)
	contextType := types.NewNamed(contextName, types.NewStruct(nil, nil), nil)
	runtimePkg.Scope().Insert(contextName)
	contextPointer := types.NewPointer(contextType)
	enterParams := types.NewTuple(types.NewParam(token.NoPos, runtimePkg, "ctx", contextPointer))
	enterResults := types.NewTuple(types.NewParam(token.NoPos, runtimePkg, "previous", types.Typ[types.Uintptr]))
	runtimePkg.Scope().Insert(types.NewFunc(token.NoPos, runtimePkg, "EnterLocalContext", types.NewSignatureType(nil, nil, nil, enterParams, enterResults, false)))
	leaveParams := types.NewTuple(
		types.NewParam(token.NoPos, runtimePkg, "ctx", contextPointer),
		types.NewParam(token.NoPos, runtimePkg, "previous", types.Typ[types.Uintptr]),
	)
	runtimePkg.Scope().Insert(types.NewFunc(token.NoPos, runtimePkg, "LeaveLocalContext", types.NewSignatureType(nil, nil, nil, leaveParams, nil, false)))
	prog.SetRuntime(runtimePkg)
}

func assertInOrder(t *testing.T, s string, wants ...string) {
	t.Helper()
	offset := 0
	for _, want := range wants {
		i := strings.Index(s[offset:], want)
		if i < 0 {
			t.Fatalf("main module IR missing ordered entry %q after byte %d:\n%s", want, offset, s)
		}
		offset += i + len(want)
	}
}
