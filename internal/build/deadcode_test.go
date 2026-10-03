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

	if err := applyDeadcodeDropOverrides([]Package{srcAPkg}, entryPkg, false, false); err != nil {
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
	err := applyDeadcodeDropOverrides([]Package{pkg}, nil, false, false)
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
	lpkg.NewFunc("main.Z", llssa.NoArgsNoRet, llssa.InGo)
	lpkg.NewFunc("main.A", llssa.NoArgsNoRet, llssa.InGo)
	pkgs := []Package{&aPackage{LPkg: lpkg}}

	want := []string{"main.init", "main.main", "main.A", "main.Z"}
	if got := dceEntryRootCandidates(pkgs, false); !reflect.DeepEqual(got, want) {
		t.Fatalf("dceEntryRootCandidates() = %v, want %v", got, want)
	}
}

func TestLinkedCExportsIncludesOnlyLibraryMain(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	newExport := func(path, name, goName, cName string) Package {
		lpkg := prog.NewPackage(path, path)
		lpkg.SetExport(goName, cName)
		lpkg.NewFunc(goName, llssa.NoArgsNoRet, llssa.InGo)
		return &aPackage{
			Package: &packages.Package{Name: name, PkgPath: path},
			LPkg:    lpkg,
		}
	}
	for _, goos := range []string{"darwin", "linux", "windows"} {
		for _, mode := range []BuildMode{BuildModeCShared, BuildModeCArchive} {
			t.Run(goos+"/"+string(mode), func(t *testing.T) {
				ctx := &context{buildConf: &Config{
					Goos:      goos,
					BuildMode: mode,
				}}
				exports, err := linkedCExports(ctx, []Package{
					newExport("main", "main", "main.Exported", "Exported"),
					newExport("example.com/dep", "dep", "example.com/dep.Callback", "Callback"),
				})
				if err != nil {
					t.Fatal(err)
				}
				if len(exports) != 1 || exports[0].goName != "main.Exported" {
					t.Fatalf("linked C exports = %+v, want command-package export only", exports)
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
			lpkg.NewFunc(goName, sig, llssa.InGo)
		}
		return &aPackage{
			Package: &packages.Package{Name: "main", PkgPath: path},
			LPkg:    lpkg,
		}
	}

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
