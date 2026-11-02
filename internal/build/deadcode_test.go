package build

import (
	"go/types"
	"reflect"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/meta"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestApplyDeadcodeDropOverridesWritesStrongTypeOverride(t *testing.T) {
	llssa.Initialize(llssa.InitAll)
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	defer ctx.prog.Dispose()

	// Isolated package workers and the synthetic entry module deliberately use
	// different LLVM contexts. Keep this test faithful to that production path.
	srcProg := llssa.NewProgram(nil)
	defer srcProg.Dispose()
	srcPkg := srcProg.NewPackage("pkg", "pkg")
	addMethodTypeGlobal(srcPkg.Module(), "_llgo_pkg.T")
	pkgMeta := buildDeadcodeMeta(t)
	defer pkgMeta.Close()
	srcAPkg := &aPackage{
		Package: &packages.Package{PkgPath: "pkg"},
		LPkg:    srcPkg,
		Meta:    pkgMeta,
	}
	entryPkg := genMainModule(ctx, llssa.PkgRuntime, &packages.Package{
		PkgPath:    "pkg",
		ExportFile: "pkg.a",
	}, &genConfig{})

	if err := applyDeadcodeDropOverrides([]Package{srcAPkg}, entryPkg, false, false, false); err != nil {
		t.Fatal(err)
	}

	out := entryPkg.LPkg.Module().String()
	if !strings.Contains(out, `@_llgo_pkg.T = constant`) {
		t.Fatalf("entry module missing strong type override:\n%s", out)
	}
	if !strings.Contains(out, `ptr @"pkg.(*T).M", ptr @pkg.T.M`) {
		t.Fatalf("live method slot was not preserved:\n%s", out)
	}
	if strings.Contains(out, `ptr @"pkg.(*T).N"`) || strings.Contains(out, `ptr @pkg.T.N`) {
		t.Fatalf("dead method slot still references N functions:\n%s", out)
	}
	if err := llvm.VerifyModule(entryPkg.LPkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("cross-context strong type override produced invalid entry module: %v\n%s", err, out)
	}
}

func TestDCEEntryRootCandidates(t *testing.T) {
	want := []string{"main.init", "main.main"}
	if got := dceEntryRootCandidates(nil, false); !reflect.DeepEqual(got, want) {
		t.Fatalf("dceEntryRootCandidates(false) = %v, want %v", got, want)
	}

	want = append(want, llssa.PkgRuntime+".init")
	if got := dceEntryRootCandidates(nil, true); !reflect.DeepEqual(got, want) {
		t.Fatalf("dceEntryRootCandidates(true) = %v, want %v", got, want)
	}
}

func TestApplyDeadcodeDropRejectsMissingMetadata(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{PkgPath: "example.com/missing"}}
	err := applyDeadcodeDropOverrides([]Package{pkg}, nil, false, false, false)
	if err == nil || !strings.Contains(err.Error(), "missing dead-code metadata for linked package example.com/missing") {
		t.Fatalf("applyDeadcodeDropOverrides() = %v, want missing metadata diagnostic", err)
	}
}

func TestDCEEntryRootCandidatesIncludesCExports(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	lpkg := prog.NewPackage("pkg", "pkg")
	lpkg.SetExport("main.Z", "Zed")
	lpkg.SetExport("main.A", "Add")
	lpkg.NewFunc("main.Z", llssa.NoArgsNoRet, llssa.GoDecl)
	lpkg.NewFunc("main.A", llssa.NoArgsNoRet, llssa.GoDecl)
	pkgs := []Package{&aPackage{LPkg: lpkg}}

	want := []string{"main.init", "main.main", "main.A", "main.Z"}
	if got := dceEntryRootCandidates(pkgs, false); !reflect.DeepEqual(got, want) {
		t.Fatalf("dceEntryRootCandidates() = %v, want %v", got, want)
	}
}

func TestLinkedCExportsIncludesDependencyAndExecutableCallbacks(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	newExport := func(path, name, goName, cName string) Package {
		lpkg := prog.NewPackage(path, path)
		lpkg.SetExport(goName, cName)
		lpkg.NewFunc(goName, llssa.NoArgsNoRet, llssa.GoDecl)
		return &aPackage{
			Package: &packages.Package{Name: name, PkgPath: path},
			LPkg:    lpkg,
		}
	}
	for _, goos := range []string{"darwin", "linux", "windows"} {
		for _, mode := range []BuildMode{BuildModeCShared, BuildModeCArchive, BuildModeExe} {
			t.Run(goos+"/"+string(mode), func(t *testing.T) {
				ctx := &context{buildConf: &Config{
					Goos:      goos,
					BuildMode: mode,
				}}
				exports, err := linkedCExports(ctx, []Package{
					newExport("main", "main", "main.Exported", "Exported"),
					newExport("example.com/dep", "dep", "example.com/dep.Callback", "Callback"),
					newExport("example.com/capi", "C", "example.com/capi.Xadd", "add"),
				})
				if err != nil {
					t.Fatal(err)
				}
				if len(exports) != 2 || exports[0].goName != "example.com/dep.Callback" || exports[1].goName != "main.Exported" {
					t.Fatalf("linked C exports = %+v, want dependency and command exports", exports)
				}
			})
		}
	}
}

func TestLinkedCExportsValidation(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	ctx := &context{buildConf: &Config{
		Goos:      "windows",
		BuildMode: BuildModeCShared,
	}}
	newExport := func(path, goName, cName string, sig *types.Signature, define bool) Package {
		lpkg := prog.NewPackage(path, path)
		lpkg.SetExport(goName, cName)
		if define {
			lpkg.NewFunc(goName, sig, llssa.GoDecl)
		}
		return &aPackage{
			Package: &packages.Package{Name: "main", PkgPath: path},
			LPkg:    lpkg,
		}
	}

	t.Run("empty C name", func(t *testing.T) {
		_, err := linkedCExports(ctx, []Package{
			newExport("main", "main.Exported", "", llssa.NoArgsNoRet, true),
		})
		if err == nil || !strings.Contains(err.Error(), "empty export name") {
			t.Fatalf("linkedCExports() error = %v, want empty export name", err)
		}
	})

	t.Run("Unicode C name", func(t *testing.T) {
		exports, err := linkedCExports(ctx, []Package{
			newExport("main", "main.函数", "函数", llssa.NoArgsNoRet, true),
		})
		if err != nil || len(exports) != 1 || exports[0].cName != "函数" {
			t.Fatalf("linkedCExports() = (%+v, %v), want Unicode name preserved", exports, err)
		}
	})

	t.Run("duplicate C name", func(t *testing.T) {
		_, err := linkedCExports(ctx, []Package{
			newExport("first", "first.Exported", "Exported", llssa.NoArgsNoRet, true),
			newExport("second", "second.Exported", "Exported", llssa.NoArgsNoRet, true),
		})
		if err == nil || !strings.Contains(err.Error(), "provided by both") {
			t.Fatalf("linkedCExports() error = %v, want duplicate C export", err)
		}
	})

	t.Run("duplicate mapping", func(t *testing.T) {
		exports, err := linkedCExports(ctx, []Package{
			newExport("main", "main.Exported", "Exported", llssa.NoArgsNoRet, true),
			newExport("main", "main.Exported", "Exported", llssa.NoArgsNoRet, true),
		})
		if err != nil || len(exports) != 1 {
			t.Fatalf("linkedCExports() = (%+v, %v), want one deduplicated export", exports, err)
		}
	})

	t.Run("missing implementation", func(t *testing.T) {
		_, err := linkedCExports(ctx, []Package{
			newExport("main", "main.Exported", "Exported", llssa.NoArgsNoRet, false),
		})
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("linkedCExports() error = %v, want missing implementation", err)
		}
	})

	t.Run("unsupported signature", func(t *testing.T) {
		sig := newSignature(nil, []types.Type{types.Typ[types.Int], types.Typ[types.Int]})
		_, err := linkedCExports(ctx, []Package{
			newExport("main", "main.Exported", "Exported", sig, true),
		})
		if err == nil || !strings.Contains(err.Error(), "unsupported signature") {
			t.Fatalf("linkedCExports() error = %v, want unsupported signature", err)
		}
	})

	t.Run("foreign implementation", func(t *testing.T) {
		exports, err := linkedCExports(ctx, []Package{
			newExport("main", "dependency.Exported", "Exported", llssa.NoArgsNoRet, false),
		})
		if err != nil || len(exports) != 0 {
			t.Fatalf("linkedCExports() = (%+v, %v), want no foreign export", exports, err)
		}
	})

	t.Run("sorted", func(t *testing.T) {
		exports, err := linkedCExports(ctx, []Package{
			newExport("zed", "zed.Exported", "Zed", llssa.NoArgsNoRet, true),
			newExport("add", "add.Exported", "Add", llssa.NoArgsNoRet, true),
		})
		if err != nil || len(exports) != 2 || exports[0].cName != "Add" || exports[1].cName != "Zed" {
			t.Fatalf("linkedCExports() = (%+v, %v), want exports sorted by C name", exports, err)
		}
	})
}

func TestCExportWrapperTargetPolicy(t *testing.T) {
	for _, test := range []struct {
		name, goos, target, path, pkgName string
		mode                              BuildMode
		want                              bool
	}{
		{name: "runtime bootstrap", goos: "linux", mode: BuildModeExe, path: llssa.PkgRuntime},
		{name: "runtime subtree", goos: "windows", mode: BuildModeCShared, path: llssa.PkgRuntime + "/signal"},
		{name: "standard runtime", goos: "darwin", mode: BuildModeExe, path: "runtime"},
		{name: "explicit target", goos: "linux", target: "wasi", mode: BuildModeExe, path: "example.com/callback"},
		{name: "package C direct exports", goos: "linux", mode: BuildModeExe, path: "example.com/capi", pkgName: "C"},
		{name: "unsupported build mode", goos: "linux", mode: "invalid", path: "example.com/callback"},
		{name: "other OS executable", goos: "freebsd", mode: BuildModeExe, path: "main", pkgName: "main"},
		{name: "other OS dependency", goos: "freebsd", mode: BuildModeCShared, path: "example.com/callback", pkgName: "callback"},
		{name: "other OS main shared", goos: "freebsd", mode: BuildModeCShared, path: "main", pkgName: "main", want: true},
		{name: "other OS main archive", goos: "freebsd", mode: BuildModeCArchive, path: "main", pkgName: "main", want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := &context{buildConf: &Config{Goos: test.goos, Target: test.target, BuildMode: test.mode}}
			pkg := &aPackage{Package: &packages.Package{PkgPath: test.path, Name: test.pkgName}}
			if got := needsCExportWrappers(ctx, pkg); got != test.want {
				t.Fatalf("C export wrapper policy = %v, want %v", got, test.want)
			}
		})
	}
	// No backend may exist for a package that contributes no native exports.
	ctx := &context{buildConf: &Config{Goos: "linux", BuildMode: BuildModeExe}}
	pkg := &aPackage{Package: &packages.Package{PkgPath: "example.com/empty"}}
	if exports, err := linkedCExports(ctx, []Package{nil, {}, pkg}); err != nil || len(exports) != 0 {
		t.Fatalf("packages without a backend contributed exports: %+v, %v", exports, err)
	}
}

func TestPlanMainLinkRejectsInvalidCExportSnapshot(t *testing.T) {
	app := snapshotPkg("example.com/main", "main.main")
	app.ID, app.ExportFile = app.PkgPath, "main.o"
	app.linkSnapshot.cExports = []cExport{{goName: "example.com/main.Missing", cName: "callback"}}
	ctx := &context{
		buildConf: &Config{Goos: "linux", BuildMode: BuildModeExe},
		pkgs:      map[*packages.Package]Package{app.Package: app},
	}
	if plan, err := planMainLink(ctx, app.Package, []*aPackage{app}); err == nil ||
		!strings.Contains(err.Error(), `C export implementation "example.com/main.Missing" not found`) || plan != nil {
		t.Fatalf("invalid export produced link plan %+v, error %v", plan, err)
	}
}

func buildDeadcodeMeta(t *testing.T) *meta.PackageMeta {
	t.Helper()
	b := meta.NewBuilder()
	main := b.Sym("main.main")
	use := b.Sym("pkg.use")
	typ := b.Sym("_llgo_pkg.T")
	iface := b.Sym("_llgo_iface$I")
	mtype := b.Sym("_llgo_func$X")

	b.AddOrdinaryEdge(mtype, mtype)
	b.AddIfaceMethod(iface, "M", mtype)
	b.AddMethodSlot(typ, "M", mtype, b.Sym("pkg.(*T).M"), b.Sym("pkg.T.M"))
	b.AddMethodSlot(typ, "N", mtype, b.Sym("pkg.(*T).N"), b.Sym("pkg.T.N"))
	b.AddOrdinaryEdge(main, use)
	b.AddOrdinaryEdge(main, typ)
	b.AddIfaceUse(main, typ)
	b.AddIfaceMethodUse(use, iface, 0)
	pm, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return pm
}

func addMethodTypeGlobal(mod llvm.Module, name string) {
	ctx := mod.Context()
	fnTy := llvm.FunctionType(ctx.VoidType(), nil, false)
	ptrTy := llvm.PointerType(fnTy, 0)
	stringTy := ctx.StructCreateNamed("runtime/internal/runtime.String")
	stringTy.StructSetBody([]llvm.Type{llvm.PointerType(ctx.Int8Type(), 0), ctx.Int64Type()}, false)
	methodTy := ctx.StructCreateNamed("github.com/xgo-dev/llgo/runtime/abi.Method")
	methodTy.StructSetBody([]llvm.Type{stringTy, ptrTy, ptrTy, ptrTy}, false)

	mtyp := llvm.AddGlobal(mod, ptrTy, "mtyp")
	ifnM := llvm.AddFunction(mod, "pkg.(*T).M", fnTy)
	tfnM := llvm.AddFunction(mod, "pkg.T.M", fnTy)
	ifnN := llvm.AddFunction(mod, "pkg.(*T).N", fnTy)
	tfnN := llvm.AddFunction(mod, "pkg.T.N", fnTy)
	methods := llvm.ConstArray(methodTy, []llvm.Value{
		llvm.ConstNamedStruct(methodTy, []llvm.Value{llvm.ConstNull(stringTy), mtyp, ifnM, tfnM}),
		llvm.ConstNamedStruct(methodTy, []llvm.Value{llvm.ConstNull(stringTy), mtyp, ifnN, tfnN}),
	})
	typeTy := ctx.StructCreateNamed("pkg.T.type")
	typeTy.StructSetBody([]llvm.Type{ctx.Int8Type(), methods.Type()}, false)
	typeDesc := llvm.AddGlobal(mod, typeTy, name)
	typeDesc.SetGlobalConstant(true)
	typeDesc.SetLinkage(llvm.WeakODRLinkage)
	typeDesc.SetInitializer(llvm.ConstNamedStruct(typeTy, []llvm.Value{
		llvm.ConstNull(ctx.Int8Type()), methods,
	}))
}

func TestInterfacePredicatesRequireClosedProgram(t *testing.T) {
	for _, closed := range []bool{false, true} {
		prog := llssa.NewProgram(nil)
		defer prog.Dispose()
		src := prog.NewPackage("source", "source")
		dst := prog.NewPackage("entry", "entry")
		name := "_llgo_iface_use._llgo_float64"
		llvm.AddFunction(src.Module(), name, llvm.FunctionType(src.Module().Context().Int1Type(), nil, false))
		builder := meta.NewBuilder()
		typ := builder.Sym("_llgo_float64")
		builder.AddOrdinaryEdge(typ, typ)
		pm, err := builder.Build()
		if err != nil {
			t.Fatal(err)
		}
		defer pm.Close()
		pkg := &aPackage{Package: &packages.Package{PkgPath: "source"}, LPkg: src, Meta: pm}
		entry := &aPackage{LPkg: dst}
		if err := applyDeadcodeDropOverrides([]Package{pkg}, entry, false, closed, false); err != nil {
			t.Fatal(err)
		}
		value := dst.Module().NamedFunction(name).FirstBasicBlock().FirstInstruction().Operand(0).ZExtValue()
		if (value == 0) != closed {
			t.Fatalf("closed=%v: predicate=%d", closed, value)
		}
	}
}
