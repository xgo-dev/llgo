package build

import (
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

func TestDCEEntryRootCandidatesIncludesCExports(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	lpkg := prog.NewPackage("pkg", "pkg")
	lpkg.SetExport("main.Z", "Zed")
	lpkg.SetExport("main.A", "Add")
	pkgs := []Package{&aPackage{LPkg: lpkg}}

	want := []string{"main.init", "main.main", "Add", "Zed"}
	if got := dceEntryRootCandidates(pkgs, false); !reflect.DeepEqual(got, want) {
		t.Fatalf("dceEntryRootCandidates() = %v, want %v", got, want)
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
