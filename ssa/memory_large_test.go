package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	llabi "github.com/xgo-dev/llgo/internal/abi"
)

func TestAllocLargeLocalOnHeap(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	runtimePkg := types.NewPackage(PkgRuntime, "runtime")
	params := types.NewTuple(types.NewVar(token.NoPos, runtimePkg, "size", types.Typ[types.Uintptr]))
	results := types.NewTuple(types.NewVar(token.NoPos, runtimePkg, "", types.Typ[types.UnsafePointer]))
	allocAtomicU := types.NewFunc(token.NoPos, runtimePkg, "AllocAtomicU", types.NewSignatureType(nil, nil, nil, params, results, false))
	runtimePkg.Scope().Insert(allocAtomicU)
	prog.SetRuntime(runtimePkg)

	pkg := prog.NewPackage("main", "main")
	b := pkg.NewFunc("main", NoArgsNoRet, GoDecl).MakeBody(1)
	small := prog.Type(types.NewArray(types.Typ[types.Byte], int64(llabi.MaxStackVarSize)), GoDecl)
	large := prog.Type(types.NewArray(types.Typ[types.Byte], int64(llabi.MaxStackVarSize+1)), GoDecl)
	b.Alloc(small, false)
	b.Alloc(large, false)
	b.Return()
	b.EndBuild()

	ir := pkg.String()
	if !strings.Contains(ir, "alloca [131072 x i8]") {
		t.Fatalf("local at the explicit stack limit was not kept on the stack:\n%s", ir)
	}
	if strings.Contains(ir, "alloca [131073 x i8]") {
		t.Fatalf("local above the explicit stack limit remained on the stack:\n%s", ir)
	}
	if !strings.Contains(ir, `call ptr @"github.com/xgo-dev/llgo/runtime/internal/runtime.AllocAtomicU"(i64 131073)`) {
		t.Fatalf("large local was not allocated with runtime.AllocAtomicU:\n%s", ir)
	}
	if !pkg.NeedRuntime {
		t.Fatal("large local heap allocation did not mark the runtime as needed")
	}
}
