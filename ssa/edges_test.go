package ssa

import (
	"testing"

	"github.com/xgo-dev/llgo/internal/meta"
	"github.com/xgo-dev/llvm"
)

func TestExtractOrdinaryEdgesFromFunctionAndGlobal(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("ordinary")
	defer mod.Dispose()

	voidTy := ctx.VoidType()
	fnTy := llvm.FunctionType(voidTy, nil, false)
	mainFn := llvm.AddFunction(mod, "pkg.main", fnTy)
	helperFn := llvm.AddFunction(mod, "pkg.helper", fnTy)

	b := ctx.NewBuilder()
	defer b.Dispose()
	entry := ctx.AddBasicBlock(mainFn, "entry")
	b.SetInsertPointAtEnd(entry)
	b.CreateCall(fnTy, helperFn, nil, "")
	b.CreateRetVoid()

	global := llvm.AddGlobal(mod, llvm.PointerType(fnTy, 0), "pkg.global")
	global.SetInitializer(helperFn)
	llvm.AddGlobal(mod, llvm.PointerType(fnTy, 0), "pkg.external")

	mb := meta.NewBuilder()
	extractOrdinaryEdges(mb, mod, nil)
	pm, _ := mb.Build()

	if !hasOrdinaryEdge(pm, "pkg.main", "pkg.helper") {
		t.Fatalf("missing ordinary edge pkg.main -> pkg.helper")
	}
	if !hasOrdinaryEdge(pm, "pkg.global", "pkg.helper") {
		t.Fatalf("missing ordinary edge pkg.global -> pkg.helper")
	}
	summary, err := meta.NewGlobalSummary([]*meta.PackageMeta{pm})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := summary.LookupSymbol("pkg.external"); ok {
		t.Fatal("external global declaration was recorded in metadata")
	}
}

func TestExtractOrdinaryEdgesUsesABITypeMarkers(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("ordinary")
	defer mod.Dispose()

	voidTy := ctx.VoidType()
	fnTy := llvm.FunctionType(voidTy, nil, false)
	ifn := llvm.AddFunction(mod, "pkg.(*T).M", fnTy)
	tfn := llvm.AddFunction(mod, "pkg.T.M", fnTy)

	i8ptrTy := llvm.PointerType(ctx.Int8Type(), 0)
	methodTy := ctx.StructCreateNamed("runtime/internal/runtime.Method")
	methodTy.StructSetBody([]llvm.Type{i8ptrTy, i8ptrTy, llvm.PointerType(fnTy, 0), llvm.PointerType(fnTy, 0)}, false)
	methods := llvm.ConstArray(methodTy, []llvm.Value{
		llvm.ConstNamedStruct(methodTy, []llvm.Value{
			llvm.ConstNull(i8ptrTy),
			llvm.ConstNull(i8ptrTy),
			ifn,
			tfn,
		}),
	})

	typeTy := ctx.StructCreateNamed("pkg.T.type")
	typeTy.StructSetBody([]llvm.Type{i8ptrTy, i8ptrTy, methods.Type()}, false)
	typeDesc := llvm.AddGlobal(mod, typeTy, "_llgo_pkg.T")
	typeDesc.SetInitializer(llvm.ConstNamedStruct(typeTy, []llvm.Value{
		llvm.ConstNull(i8ptrTy),
		llvm.ConstNull(i8ptrTy),
		methods,
	}))
	unmarkedTypeDesc := llvm.AddGlobal(mod, typeTy, "_llgo_pkg.Unmarked")
	unmarkedTypeDesc.SetInitializer(typeDesc.Initializer())

	mb := meta.NewBuilder()
	extractOrdinaryEdges(mb, mod, map[llvm.Value]struct{}{typeDesc: {}})
	pm, _ := mb.Build()

	if hasOrdinaryEdge(pm, "_llgo_pkg.T", "pkg.(*T).M") {
		t.Fatalf("method table IFn was recorded as an ordinary edge")
	}
	if hasOrdinaryEdge(pm, "_llgo_pkg.T", "pkg.T.M") {
		t.Fatalf("method table TFn was recorded as an ordinary edge")
	}
	if !hasOrdinaryEdge(pm, "_llgo_pkg.Unmarked", "pkg.(*T).M") {
		t.Fatalf("unmarked global method-shaped IFn edge was skipped")
	}
	if !hasOrdinaryEdge(pm, "_llgo_pkg.Unmarked", "pkg.T.M") {
		t.Fatalf("unmarked global method-shaped TFn edge was skipped")
	}
}

func TestFinishMetaCollectionDeduplicatesFunctionEdges(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackageEx("pkg", "pkg", true)

	callee := pkg.NewFunc("pkg.callee", NoArgsNoRet, InGo)
	callee.MakeBody(1).Return()
	caller := pkg.NewFunc("pkg.caller", NoArgsNoRet, InGo)
	b := caller.MakeBody(1)
	b.Call(caller.Expr)
	b.Call(callee.Expr)
	b.Call(callee.Expr)
	b.Return()

	if err := pkg.FinishMetaCollection(); err != nil {
		t.Fatal(err)
	}
	defer pkg.Meta.Close()
	summary, err := meta.NewGlobalSummary([]*meta.PackageMeta{pkg.Meta})
	if err != nil {
		t.Fatal(err)
	}
	callerSym, ok := summary.LookupSymbol("pkg.caller")
	if !ok {
		t.Fatal("pkg.caller is missing from metadata")
	}
	calleeSym, ok := summary.LookupSymbol("pkg.callee")
	if !ok {
		t.Fatal("pkg.callee is missing from metadata")
	}
	edges := summary.OrdinaryEdges(callerSym)
	if len(edges) != 1 || edges[0] != calleeSym {
		t.Fatalf("OrdinaryEdges(pkg.caller) = %v, want [%d]", edges, calleeSym)
	}
}

func hasOrdinaryEdge(pm *meta.PackageMeta, srcName, dstName string) bool {
	summary, err := meta.NewGlobalSummary([]*meta.PackageMeta{pm})
	if err != nil {
		return false
	}
	src, ok := summary.LookupSymbol(srcName)
	if !ok {
		return false
	}
	dst, ok := summary.LookupSymbol(dstName)
	if !ok {
		return false
	}
	for _, e := range summary.OrdinaryEdges(src) {
		if e == dst {
			return true
		}
	}
	return false
}
