package dcepass

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestDeadNoInlineFunctionsFromModulesRequiresNoInline(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("feedback")
	defer mod.Dispose()
	void := ctx.VoidType()
	fnType := llvm.FunctionType(void, nil, false)
	main := llvm.AddFunction(mod, "main", fnType)
	dead := llvm.AddFunction(mod, "candidate", fnType)
	llvm.AddBasicBlock(main, "entry")
	llvm.AddBasicBlock(dead, "entry")

	got := DeadNoInlineFunctionsFromModules([]llvm.Module{mod}, []string{"main"}, []string{"candidate"})
	if len(got) != 0 {
		t.Fatalf("feedback without noinline = %#v, want empty", got)
	}

	if got := MarkNoInlineFunctions(mod, []string{"candidate", "missing", "main"}); got != 2 {
		t.Fatalf("MarkNoInlineFunctions() = %d, want 2", got)
	}
	if got := MarkNoInlineFunctions(mod, []string{"candidate", "main"}); got != 0 {
		t.Fatalf("repeated MarkNoInlineFunctions() = %d, want 0", got)
	}
	got = DeadNoInlineFunctionsFromModules([]llvm.Module{mod}, []string{"main"}, []string{"candidate"})
	if _, ok := got["candidate"]; !ok {
		t.Fatalf("feedback with noinline = %#v, want candidate dead", got)
	}
}
