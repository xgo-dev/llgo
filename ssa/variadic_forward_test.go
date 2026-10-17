package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestCVariadicForwardingRejectsResultConversion(t *testing.T) {
	// Hosted wasm32 stores Go uintptr in 64 bits but returns C uintptr in
	// 32 bits. Even matching native prototypes cannot tail-forward through
	// Call's result widening. Check this independently of the cl target gate.
	prog := NewProgram(&Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"})
	defer prog.Dispose()
	pkg := prog.NewPackage("forward", "forward")
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(VArg()),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Uintptr])), true)
	callee := pkg.NewFunc("native", sig, CDecl)
	caller := pkg.NewFunc("wrapper", sig, CDecl)
	if callee.ll != caller.ll || callee.ll.ReturnType() == prog.retType(sig).ll {
		t.Fatal("fixture must have matching prototypes but require result conversion")
	}
	b := caller.MakeBody(1)
	if b.CanForwardVarargs(callee.Expr) {
		t.Fatal("result conversion must prevent variadic tail forwarding")
	}
	defer func() {
		message := fmt.Sprint(recover())
		if !strings.Contains(message, "from wrapper to native: incompatible ABI") {
			t.Fatalf("unexpected diagnostic: %s", message)
		}
		if !caller.impl.EntryBasicBlock().FirstInstruction().IsNil() {
			t.Fatal("ABI rejection must happen before emitting the call or conversion")
		}
		b.Unreachable()
		if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
	}()
	b.CallForwardVarargs(callee.Expr)
}

func TestCVariadicForwardingRejectsDifferentPrototype(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg := prog.NewPackage("forward", "forward")
	signature := func(fixed types.Type) *types.Signature {
		return types.NewSignatureType(nil, nil, nil, types.NewTuple(
			types.NewVar(token.NoPos, nil, "fixed", fixed), VArg(),
		), nil, true)
	}
	callee := pkg.NewFunc("native", signature(types.Typ[types.Int32]), CDecl)
	caller := pkg.NewFunc("wrapper", signature(types.Typ[types.Float64]), CDecl)
	b := caller.MakeBody(1)
	if b.CanForwardVarargs(callee.Expr) {
		t.Fatal("different fixed-argument ABIs must prevent variadic forwarding")
	}
	b.Return()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}
