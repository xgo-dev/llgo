//go:build !llgo
// +build !llgo

package build

import (
	"go/constant"
	"go/token"
	"go/types"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"

	"github.com/goplus/llgo/internal/packages"
	llssa "github.com/goplus/llgo/ssa"
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
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	mod := genMainModule(ctx, llssa.PkgRuntime, pkg,
		&genConfig{rtInit: true, pyInit: true, packageInits: []string{"example.com/b.init", "example.com/z.init", "example.com/a.init"}})
	if mod.ExportFile != "foo.a-main" {
		t.Fatalf("unexpected export file: %s", mod.ExportFile)
	}
	ir := mod.LPkg.String()
	checks := []string{
		"define i32 @main(",
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
				"define internal void @__llgo_runtime_ctor()",
				"call void @\"github.com/goplus/llgo/runtime/internal/runtime.init\"()",
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
			if strings.Contains(ir, "define i32 @main") {
				t.Fatalf("library mode should not emit main function:\n%s", ir)
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
			if !strings.Contains(ir, "call void @\"github.com/goplus/llgo/runtime/internal/runtime.init\"()") {
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
		`call void @"example.com/foo.init"()`,
		`call void @"example.com/foo.main"()`,
		"LeaveLocalContext",
	)
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
