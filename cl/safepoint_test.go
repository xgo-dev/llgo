//go:build !llgo

package cl_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/goplus/llgo/cl/cltest"
	llssa "github.com/goplus/llgo/ssa"
)

func TestCompileCooperativeSafepoints(t *testing.T) {
	const src = `package main

func leaf(p *int) *int {
	return p
}

func loop(p *int, n int) *int {
	for n > 0 {
		n--
	}
	return p
}

//go:nosplit
func noPoll(p *int) *int {
	return p
}
`
	ir := cltest.CompileIREx(t, src, "safepoint.go", false, func(prog llssa.Program) {
		prog.EnableGCRoots(true)
		prog.EnableCooperativeSafepoints(true)
	})

	leaf := findLLVMFunction(t, ir, "leaf")
	if got := strings.Count(leaf, "CooperativeSafepoint"); got != 1 {
		t.Fatalf("leaf has %d safepoints, want 1:\n%s", got, leaf)
	}
	if !strings.Contains(leaf, `[1 x ptr]`) {
		t.Fatalf("leaf parameter is not rooted across the entry safepoint:\n%s", leaf)
	}

	loop := findLLVMFunction(t, ir, "loop")
	if got := strings.Count(loop, "CooperativeSafepoint"); got != 2 {
		t.Fatalf("loop has %d safepoints, want entry plus backedge:\n%s", got, loop)
	}
	if !strings.Contains(loop, `[1 x ptr]`) {
		t.Fatalf("loop parameter is not rooted across safepoints:\n%s", loop)
	}

	noPoll := findLLVMFunction(t, ir, "noPoll")
	if strings.Contains(noPoll, "CooperativeSafepoint") ||
		strings.Contains(noPoll, "llvm_gc_root_chain") {
		t.Fatalf("//go:nosplit function contains a safepoint or root frame:\n%s", noPoll)
	}
}

func TestCompileCooperativeSafepointsDisabled(t *testing.T) {
	const src = `package main
func loop(n int) {
	for n > 0 {
		n--
	}
}
`
	ir := cltest.CompileIREx(t, src, "safepoint_disabled.go", false, nil)
	if strings.Contains(ir, "CooperativeSafepoint") {
		t.Fatalf("disabled cooperative safepoints changed ordinary code:\n%s", ir)
	}
}

func findLLVMFunction(t *testing.T, ir, name string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?ms)^define [^{]*\.` + regexp.QuoteMeta(name) + `"?\([^)]*\).*?^\}`)
	body := pattern.FindString(ir)
	if body == "" {
		t.Fatalf("LLVM function %s not found:\n%s", name, ir)
	}
	return body
}
