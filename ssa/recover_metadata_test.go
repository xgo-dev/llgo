package ssa

import (
	"go/types"
	"strings"
	"testing"
)

func TestRecoverDeferClassificationFallbacks(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("p", "example.com/p")
	callee := pkg.NewFunc("callee", NoArgsNoRet, InGo)
	caller := pkg.NewFunc("caller", NoArgsNoRet, InGo)
	b := caller.MakeBody(1)

	fnPtr := b.ChangeType(prog.rawType(NoArgsNoRet), callee.Expr)
	closure := b.MakeClosure(callee.Expr, nil)
	ifaceMethod := Expr{Type: &aType{kind: vkIfaceMethod}}
	ordinary := prog.Val(1)

	for _, test := range []struct {
		name string
		fn   Expr
		want bool
	}{
		{name: "nil", fn: Nil, want: false},
		{name: "ordinary", fn: ordinary, want: false},
		{name: "function declaration", fn: callee.Expr, want: true},
		{name: "closure", fn: closure, want: true},
		{name: "function pointer", fn: fnPtr, want: true},
		{name: "interface method", fn: ifaceMethod, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := deferMayRecover(test.fn); got != test.want {
				t.Fatalf("deferMayRecover() = %v, want %v", got, test.want)
			}
		})
	}

	if token := b.recoverDeferToken(fnPtr, false); token.IsNil() {
		t.Fatal("function pointer did not produce a recover token")
	}
	if token := b.recoverDeferToken(ordinary, true); !token.IsNil() {
		t.Fatal("ordinary value unexpectedly produced a recover token")
	}

	if !isRecoverBuiltin(Builtin("recover")) {
		t.Fatal("recover builtin not recognized")
	}
	for _, value := range []Expr{
		Nil,
		ordinary,
		Builtin("panic"),
		{Type: &aType{raw: rawType{Type: types.Typ[types.Int]}, kind: vkBuiltin}},
	} {
		if isRecoverBuiltin(value) {
			t.Fatalf("%v unexpectedly recognized as the recover builtin", value.Type)
		}
	}

	b.Return()
	b.EndBuild()
}

func TestDefaultDeferConservativelyScopesRecoveringClosure(t *testing.T) {
	prog := NewProgram(nil)
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("p", "example.com/p")

	callee := pkg.NewFunc("recovering", NoArgsNoRet, InGo)
	cb := callee.MakeBody(1)
	cb.BindRecoverFrame()
	_ = cb.Recover()
	cb.Return()
	cb.EndBuild()

	caller := pkg.NewFunc("caller", NoArgsNoRet, InGo)
	b := caller.MakeBody(1)
	caller.SetRecover(caller.MakeBlock())
	closure := b.MakeClosure(callee.Expr, nil)
	b.Defer(DeferAlways, closure, Builder.Call)
	b.RunDefers()
	b.Return()
	b.EndBuild()

	ir := pkg.String()
	if !strings.Contains(ir, "StartRecoverFrame") {
		t.Fatalf("default closure defer did not open a recover scope:\n%s", ir)
	}
	if strings.Contains(ir, "llgo.may-recover") {
		t.Fatalf("recover classification leaked into an LLVM attribute:\n%s", ir)
	}
}
