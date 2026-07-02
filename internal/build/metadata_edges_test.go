package build

import (
	"testing"

	"github.com/goplus/llgo/internal/meta"
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

	mb := meta.NewBuilder()
	extractOrdinaryEdges(mb, mod)
	pm, _ := mb.Build()

	if !hasOrdinaryEdge(pm, "pkg.main", "pkg.helper") {
		t.Fatalf("missing ordinary edge pkg.main -> pkg.helper")
	}
	if !hasOrdinaryEdge(pm, "pkg.global", "pkg.helper") {
		t.Fatalf("missing ordinary edge pkg.global -> pkg.helper")
	}
}

func TestExtractOrdinaryEdgesSkipsUncommonMethodTable(t *testing.T) {
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

	mb := meta.NewBuilder()
	extractOrdinaryEdges(mb, mod)
	pm, _ := mb.Build()

	if hasOrdinaryEdge(pm, "_llgo_pkg.T", "pkg.(*T).M") {
		t.Fatalf("method table IFn was recorded as an ordinary edge")
	}
	if hasOrdinaryEdge(pm, "_llgo_pkg.T", "pkg.T.M") {
		t.Fatalf("method table TFn was recorded as an ordinary edge")
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
	for _, dst := range summary.OrdinaryEdges(src) {
		if summary.SymbolName(dst) == dstName {
			return true
		}
	}
	return false
}
