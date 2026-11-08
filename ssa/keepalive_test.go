package ssa

import (
	"go/token"
	"go/types"
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
