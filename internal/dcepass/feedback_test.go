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
	if got := UnmarkNoInlineFunctions(mod, []string{"candidate", "missing", "main"}); got != 2 {
		t.Fatalf("UnmarkNoInlineFunctions() = %d, want 2", got)
	}
	if got := UnmarkNoInlineFunctions(mod, []string{"candidate", "main"}); got != 0 {
		t.Fatalf("repeated UnmarkNoInlineFunctions() = %d, want 0", got)
	}
	got = DeadNoInlineFunctionsFromModules([]llvm.Module{mod}, []string{"main"}, []string{"candidate"})
	if len(got) != 0 {
		t.Fatalf("feedback after removing noinline = %#v, want empty", got)
	}
}

func TestDeadNoInlineFunctionsFromModulesReportsDeletedKnownDefinition(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("feedback")
	defer mod.Dispose()
	void := ctx.VoidType()
	fnType := llvm.FunctionType(void, nil, false)
	main := llvm.AddFunction(mod, "main", fnType)
	llvm.AddBasicBlock(main, "entry")

	got := DeadNoInlineFunctionsFromModulesWithDefinitions(
		[]llvm.Module{mod},
		[]string{"main"},
		[]string{"candidate"},
		map[string]struct{}{"candidate": {}},
	)
	if _, ok := got["candidate"]; !ok {
		t.Fatalf("deleted known definition feedback = %#v, want candidate dead", got)
	}

	// Without an explicit pre-link definition fact, an absent candidate remains
	// unknown rather than being treated as dead.
	got = DeadNoInlineFunctionsFromModulesWithDefinitions([]llvm.Module{mod}, []string{"main"}, []string{"candidate"}, nil)
	if len(got) != 0 {
		t.Fatalf("unknown absent candidate feedback = %#v, want empty", got)
	}
}
