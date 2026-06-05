//go:build !llgo
// +build !llgo

package build

import (
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/metadata"
	"github.com/goplus/llgo/internal/packages"
	llssa "github.com/goplus/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestApplyDCEOverridesWritesStrongTypeOverride(t *testing.T) {
	llssa.Initialize(llssa.InitAll)
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
			DCE:       true,
		},
	}

	srcPkg := ctx.prog.NewPackage("pkg", "pkg")
	srcMod := srcPkg.Module()
	addMethodTypeGlobal(t, srcMod, "_llgo_pkg.T")

	pkgMeta := buildDCEMeta()
	srcAPkg := &aPackage{
		Package: &packages.Package{PkgPath: "pkg"},
		LPkg:    srcPkg,
		Meta:    pkgMeta,
	}
	entryPkg := genMainModule(ctx, llssa.PkgRuntime, &packages.Package{
		PkgPath:    "pkg",
		ExportFile: "pkg.a",
	}, &genConfig{})

	if err := applyDCEOverrides(ctx, &packages.Package{PkgPath: "pkg"}, []Package{srcAPkg}, entryPkg, false, false); err != nil {
		t.Fatalf("applyDCEOverrides: %v", err)
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
}

func TestDCEEntryRootCandidatesIncludesRuntimeWhenNeeded(t *testing.T) {
	roots := dceEntryRootCandidates(&packages.Package{PkgPath: "pkg"}, true)
	want := []string{"pkg.init", "pkg.main", llssa.PkgRuntime + ".init"}
	if strings.Join(roots, "\n") != strings.Join(want, "\n") {
		t.Fatalf("roots mismatch:\ngot  %q\nwant %q", roots, want)
	}
}

func TestDCEEntryRootCandidatesSkipsRuntimeWhenNotNeeded(t *testing.T) {
	roots := dceEntryRootCandidates(&packages.Package{PkgPath: "pkg"}, false)
	want := []string{"pkg.init", "pkg.main"}
	if strings.Join(roots, "\n") != strings.Join(want, "\n") {
		t.Fatalf("roots mismatch:\ngot  %q\nwant %q", roots, want)
	}
}

func buildDCEMeta() *metadata.PackageMeta {
	b := metadata.NewBuilder()
	main := b.Symbol("pkg.main")
	use := b.Symbol("pkg.use")
	typ := b.Symbol("_llgo_pkg.T")
	iface := b.Symbol("_llgo_iface$I")
	mSig := metadata.MethodSig{Name: b.Name("M"), MType: b.Symbol("_llgo_func$X")}
	nSig := metadata.MethodSig{Name: b.Name("N"), MType: b.Symbol("_llgo_func$X")}

	b.AddIfaceEntry(iface, []metadata.MethodSig{mSig})
	b.AddMethodInfo(typ, []metadata.MethodSlot{
		{Sig: mSig, IFn: b.Symbol("pkg.(*T).M"), TFn: b.Symbol("pkg.T.M")},
		{Sig: nSig, IFn: b.Symbol("pkg.(*T).N"), TFn: b.Symbol("pkg.T.N")},
	})
	b.AddEdge(main, use)
	b.AddEdge(main, typ)
	b.AddUseIface(main, []metadata.Symbol{typ})
	b.AddUseIfaceMethod(use, []metadata.IfaceMethodDemand{{Target: iface, Sig: mSig}})
	return b.Build()
}

func addMethodTypeGlobal(t *testing.T, mod llvm.Module, name string) {
	t.Helper()
	ctx := mod.Context()
	voidTy := ctx.VoidType()
	fnTy := llvm.FunctionType(voidTy, nil, false)
	ptrTy := llvm.PointerType(fnTy, 0)
	stringTy := ctx.StructCreateNamed("runtime/internal/runtime.String")
	stringTy.StructSetBody([]llvm.Type{llvm.PointerType(ctx.Int8Type(), 0), ctx.Int64Type()}, false)
	methodTy := ctx.StructCreateNamed("github.com/goplus/llgo/runtime/abi.Method")
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
	typeDesc.SetLinkage(llvm.LinkOnceODRLinkage)
	typeDesc.SetInitializer(llvm.ConstNamedStruct(typeTy, []llvm.Value{
		llvm.ConstNull(ctx.Int8Type()),
		methods,
	}))
}
