package dcepass

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestEmitStrongTypeOverridesPrunesDeadMethodSlots(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()

	src := ctx.NewModule("src")
	defer src.Dispose()
	dst := ctx.NewModule("dst")
	defer dst.Dispose()

	voidTy := ctx.VoidType()
	fnTy := llvm.FunctionType(voidTy, nil, false)
	ptrTy := llvm.PointerType(fnTy, 0)
	stringTy := ctx.StructCreateNamed("runtime/internal/runtime.String")
	stringTy.StructSetBody([]llvm.Type{llvm.PointerType(ctx.Int8Type(), 0), ctx.Int64Type()}, false)
	methodTy := ctx.StructCreateNamed("github.com/goplus/llgo/runtime/abi.Method")
	methodTy.StructSetBody([]llvm.Type{stringTy, ptrTy, ptrTy, ptrTy}, false)

	mtyp := llvm.AddGlobal(src, ptrTy, "mtyp")
	ifnM := llvm.AddFunction(src, "pkg.(*T).M", fnTy)
	tfnM := llvm.AddFunction(src, "pkg.T.M", fnTy)
	ifnN := llvm.AddFunction(src, "pkg.(*T).N", fnTy)
	tfnN := llvm.AddFunction(src, "pkg.T.N", fnTy)

	methods := llvm.ConstArray(methodTy, []llvm.Value{
		llvm.ConstNamedStruct(methodTy, []llvm.Value{
			llvm.ConstNull(stringTy),
			mtyp,
			ifnM,
			tfnM,
		}),
		llvm.ConstNamedStruct(methodTy, []llvm.Value{
			llvm.ConstNull(stringTy),
			mtyp,
			ifnN,
			tfnN,
		}),
	})
	typeTy := ctx.StructCreateNamed("pkg.T.type")
	typeTy.StructSetBody([]llvm.Type{ctx.Int8Type(), methods.Type()}, false)
	typeDesc := llvm.AddGlobal(src, typeTy, "_llgo_pkg.T")
	typeDesc.SetGlobalConstant(true)
	typeDesc.SetLinkage(llvm.LinkOnceODRLinkage)
	typeDesc.SetInitializer(llvm.ConstNamedStruct(typeTy, []llvm.Value{
		llvm.ConstNull(ctx.Int8Type()),
		methods,
	}))

	liveSlots := map[string][]int{"_llgo_pkg.T": {0}}
	if err := EmitStrongTypeOverrides(dst, []llvm.Module{src}, liveSlots); err != nil {
		t.Fatalf("EmitStrongTypeOverrides: %v", err)
	}

	out := dst.String()
	if !strings.Contains(out, `@_llgo_pkg.T = constant`) {
		t.Fatalf("override type global was not emitted as a strong constant:\n%s", out)
	}
	if !strings.Contains(out, `ptr @"pkg.(*T).M", ptr @pkg.T.M`) {
		t.Fatalf("live method slot was not preserved:\n%s", out)
	}
	if strings.Contains(out, `ptr @"pkg.(*T).N"`) || strings.Contains(out, `ptr @pkg.T.N`) {
		t.Fatalf("dead method slot still references N functions:\n%s", out)
	}
}
