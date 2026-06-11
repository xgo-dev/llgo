//go:build !llgo
// +build !llgo

package cl

import (
	"go/types"
	"strings"
	"testing"

	gossa "golang.org/x/tools/go/ssa"
)

func withMayMoreStackHook(t *testing.T, hook string) {
	t.Helper()
	old := mayMoreStackHook
	SetMayMoreStackHook(hook)
	t.Cleanup(func() {
		SetMayMoreStackHook(old)
	})
}

func TestMayMoreStackHookNameBranches(t *testing.T) {
	pkg := types.NewPackage("example.com/foo", "foo")

	withMayMoreStackHook(t, "")
	if got := (&context{goTyps: pkg}).mayMoreStackHookName(); got != "" {
		t.Fatalf("empty global hook resolved to %q", got)
	}
	if got := (&context{maymorestack: "cached"}).mayMoreStackHookName(); got != "cached" {
		t.Fatalf("cached hook = %q, want cached", got)
	}

	withMayMoreStackHook(t, " \t ")
	if got := (&context{goTyps: pkg}).mayMoreStackHookName(); got != "" {
		t.Fatalf("blank global hook resolved to %q", got)
	}
	if got := (&context{}).mayMoreStackHookName(); got != "" {
		t.Fatalf("nil package resolved to %q", got)
	}

	withMayMoreStackHook(t, " foo.mayMoreStack ")
	if got := (&context{goTyps: pkg}).mayMoreStackHookName(); got != "example.com/foo.mayMoreStack" {
		t.Fatalf("package-name hook = %q", got)
	}

	withMayMoreStackHook(t, "example.com/foo.mayMoreStack")
	if got := (&context{goTyps: pkg}).mayMoreStackHookName(); got != "example.com/foo.mayMoreStack" {
		t.Fatalf("package-path hook = %q", got)
	}

	withMayMoreStackHook(t, "bar.mayMoreStack")
	if got := (&context{goTyps: pkg}).mayMoreStackHookName(); got != "" {
		t.Fatalf("foreign short hook resolved to %q", got)
	}

	withMayMoreStackHook(t, "main.mayMoreStack")
	mainPkg := types.NewPackage("cmd/example", "main")
	if got := (&context{goTyps: mainPkg}).mayMoreStackHookName(); got != "cmd/example.mayMoreStack" {
		t.Fatalf("main hook = %q", got)
	}
}

func TestEmitMayMoreStackHookSkipsMissingAndSyntheticFunctions(t *testing.T) {
	withMayMoreStackHook(t, "foo.mayMoreStack")
	(&context{}).emitMayMoreStackHook(nil)
	(&context{goFn: &gossa.Function{Synthetic: "package initializer"}}).emitMayMoreStackHook(nil)
}

func TestMayMoreStackHookIsEmittedForOrdinaryFunctions(t *testing.T) {
	withMayMoreStackHook(t, "main.mayMoreStack")

	ssaPkg, _, files := buildGoSSAPkg(t, `package main

func mayMoreStack() {}

func helper() {}

func main() {
	helper()
}
`)
	prog := newLLSSAProg(t)
	pkg, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.String()
	call := "call void @main.mayMoreStack()"
	if got := strings.Count(ir, call); got != 2 {
		t.Fatalf("mayMoreStack hook call count = %d, want 2 for main and helper:\n%s", got, ir)
	}
	if body := functionIR(t, ir, "main.mayMoreStack"); strings.Contains(body, call) {
		t.Fatalf("mayMoreStack hook should not call itself:\n%s", body)
	}

	fn := pkg.FuncOf("main.mayMoreStack")
	if fn == nil {
		t.Fatal("missing compiled mayMoreStack function")
	}
}

func TestMayMoreStackHookDeclarationIsCreated(t *testing.T) {
	withMayMoreStackHook(t, "main.missingHook")

	ssaPkg, _, files := buildGoSSAPkg(t, `package main

func helper() {}

func main() {
	helper()
}
`)
	prog := newLLSSAProg(t)
	pkg, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.String()
	call := "call void @main.missingHook()"
	if got := strings.Count(ir, call); got != 2 {
		t.Fatalf("missingHook call count = %d, want 2 for main and helper:\n%s", got, ir)
	}
	if fn := pkg.FuncOf("main.missingHook"); fn == nil {
		t.Fatal("missing synthesized maymorestack hook declaration")
	}
}

func functionIR(t *testing.T, ir, name string) string {
	t.Helper()
	start := strings.Index(ir, "define void @"+name+"(")
	if start < 0 {
		t.Fatalf("missing function %q in IR:\n%s", name, ir)
	}
	rest := ir[start:]
	end := strings.Index(rest, "\n}")
	if end < 0 {
		t.Fatalf("unterminated function %q in IR:\n%s", name, rest)
	}
	return rest[:end+2]
}
