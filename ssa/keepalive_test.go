package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestKeepAliveInterfaceWords(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("p", "example.com/keepalive")
	params := types.NewTuple(types.NewVar(token.NoPos, nil, "x", types.NewInterfaceType(nil, nil).Complete()))
	fn := pkg.NewFunc("keep", types.NewSignatureType(nil, nil, nil, params, nil, false), GoDecl)
	b := fn.MakeBody(1)
	x := fn.Param(0)
	call := b.KeepAlive(x).impl
	if call.Type().TypeKind() != llvm.VoidTypeKind || call.ArgOperandsCount() != 2 {
		t.Fatalf("expected a void use of both interface words: %s", call.String())
	}
	for i := 0; i < 2; i++ {
		word := call.Operand(i)
		if word.InstructionOpcode() != llvm.ExtractValue || word.Operand(0) != x.impl {
			t.Fatalf("operand %d does not retain the input interface: %s", i, word.String())
		}
		if indices := word.Indices(); len(indices) != 1 || indices[0] != uint32(i) {
			t.Fatalf("operand %d retains the wrong interface word: %v", i, indices)
		}
	}
	b.Return()
	b.EndBuild()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestKeepAliveRequiresInterface(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "example.com/keepalive")
	b := pkg.NewFunc("keep", NoArgsNoRet, GoDecl).MakeBody(1)
	defer func() {
		if got := recover(); got != "keepAlive(any): invalid argument type" {
			t.Fatalf("unexpected diagnostic: %v", got)
		}
	}()
	b.KeepAlive(prog.IntVal(1, prog.Int()))
}

func TestKeepAliveStackBox(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("p", "example.com/keepalive")
	holder := types.NewStruct([]*types.Var{
		types.NewVar(token.NoPos, nil, "child", types.NewPointer(types.Typ[types.Int])),
		types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int]),
	}, nil)
	params := types.NewTuple(types.NewVar(token.NoPos, nil, "x", holder))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
	fn := pkg.NewFunc("keep", sig, GoDecl)
	b := fn.MakeBody(1)
	anyType := prog.Type(types.NewInterfaceType(nil, nil).Complete(), GoDecl)
	b.KeepAlive(b.MakeInterfaceDataOnlyEx(anyType, fn.Param(0), false))
	b.Return()
	b.EndBuild()
	heapFn := pkg.NewFunc("keepHeap", sig, GoDecl)
	heapBuilder := heapFn.MakeBody(1)
	heapBuilder.KeepAlive(heapBuilder.MakeInterfaceDataOnly(anyType, heapFn.Param(0)))
	heapBuilder.Return()
	heapBuilder.EndBuild()
	options := llvm.NewPassBuilderOptions()
	defer options.Dispose()
	if err := pkg.Module().RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	ir := fn.impl.String()
	if strings.Contains(ir, "runtime.Alloc") || !strings.Contains(ir, "alloca ") || !strings.Contains(ir, "store ") ||
		!strings.Contains(ir, `asm sideeffect "", "X,X,~{memory}"`) {
		t.Fatalf("optimized KeepAlive lost its materialized stack box or allocated on the heap:\n%s", ir)
	}
	if ir := heapFn.impl.String(); !strings.Contains(ir, "runtime.AllocU") {
		t.Fatalf("independently owned interface box became stack-local:\n%s", ir)
	}
}
