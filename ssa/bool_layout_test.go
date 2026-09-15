//go:build !llgo

package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llvm"
)

func TestBoolClangMemoryLayout(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()

	boolTy := prog.Bool()
	if got, want := boolTy.ll.IntTypeWidth(), 1; got != want {
		t.Fatalf("Go bool SSA type width = %d, want i1", got)
	}
	if got, want := prog.llvmMemType(boolTy).IntTypeWidth(), 8; got != want {
		t.Fatalf("Go bool memory type width = %d, want i8", got)
	}
	if unsafe.Sizeof(true) != 1 {
		t.Fatalf("unsafe.Sizeof(true) = %d, want 1", unsafe.Sizeof(true))
	}

	arr := prog.rawType(types.NewArray(types.Typ[types.Bool], 256))
	if got, want := prog.td.TypeAllocSize(arr.ll), uint64(256); got != want {
		t.Fatalf("[256]bool LLVM alloc size = %d, want 256 (C _Bool[256] / Go [256]bool)", got)
	}

	fields := []*types.Var{
		types.NewVar(token.NoPos, nil, "a", types.Typ[types.Bool]),
		types.NewVar(token.NoPos, nil, "b", types.Typ[types.Uint8]),
		types.NewVar(token.NoPos, nil, "c", types.Typ[types.Bool]),
	}
	st := prog.rawType(types.NewStruct(fields, nil))
	if got, want := prog.td.TypeAllocSize(st.ll), uint64(3); got != want {
		t.Fatalf("struct{bool; uint8; bool} alloc size = %d, want 3", got)
	}

	pkg := prog.NewPackage("abi", "abi")
	sig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "b", types.Typ[types.Bool])),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])),
		false)
	fn := pkg.NewFunc("Neg", sig, InGo)
	b := fn.MakeBody(1)
	b.Return(b.UnOp(token.NOT, fn.Param(0)))
	ir := pkg.String()
	if !strings.Contains(ir, "define i1 @Neg(i1 %0)") {
		t.Fatalf("scalar bool function should stay i1 like Clang:\n%s", ir)
	}

	g := pkg.NewVarEx("abi.flag", prog.Pointer(prog.Bool()))
	g.Init(prog.BoolVal(true))
	ir = pkg.String()
	if !strings.Contains(ir, "@abi.flag = global i8 1") {
		t.Fatalf("bool global should be i8 in memory:\n%s", ir)
	}

	arrTy := prog.rawType(types.NewArray(types.Typ[types.Bool], 4))
	elems := []Expr{prog.BoolVal(true), prog.BoolVal(false), prog.BoolVal(true), prog.BoolVal(false)}
	init := prog.ConstArray(arrTy, elems)
	if init.impl.Type() != arrTy.ll {
		t.Fatalf("[4]bool const type = %s, want %s", init.impl.Type(), arrTy.ll)
	}
	table := pkg.NewVarEx("abi.table", prog.Pointer(arrTy))
	table.Init(init)
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("bool array global failed verify: %v\n%s", err, pkg.String())
	}
}
