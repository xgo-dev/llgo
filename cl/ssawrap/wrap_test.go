//go:build !llgo
// +build !llgo

package ssawrap

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Test source code: contains simple functions for wrapping
const testSrc = `
package demo

func Add(a, b int) int {
	return a + b
}

func Greet(name string) string {
	return "Hello, " + name
}

func NoReturn(a int) {
	_ = a
}
`

// buildTestProgram builds an SSA program for testing
func buildTestProgram(t *testing.T) (*ssa.Program, *ssa.Package) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "demo.go", testSrc, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	files := []*ast.File{f}

	pkg := types.NewPackage("demo", "")
	ssapkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset, pkg, files, ssa.SanityCheckFunctions,
	)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}

	return ssapkg.Prog, ssapkg
}

// TestMakeCallWrapper_Basic tests basic function wrapping
func TestMakeCallWrapper_Basic(t *testing.T) {
	prog, ssapkg := buildTestProgram(t)

	// Get the original Add function
	origFn := ssapkg.Func("Add")
	if origFn == nil {
		t.Fatal("Add function not found")
	}

	// Generate wrapper function
	wrapper := MakeCallWrapper(prog, origFn)
	if wrapper == nil {
		t.Fatal("MakeCallWrapper returned nil")
	}

	// Verify wrapper signature matches the original function
	if !types.Identical(wrapper.Signature, origFn.Signature) {
		t.Errorf("wrapper signature mismatch:\n got: %v\nwant: %v",
			wrapper.Signature, origFn.Signature)
	}

	// Verify wrapper has exactly 1 basic block
	if len(wrapper.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(wrapper.Blocks))
	}

	entry := wrapper.Blocks[0]

	// Verify basic block has 2 instructions: Call + Return
	if len(entry.Instrs) != 2 {
		t.Errorf("expected 2 instructions, got %d", len(entry.Instrs))
		for i, instr := range entry.Instrs {
			t.Logf("  instr[%d]: %T = %v", i, instr, instr)
		}
	}

	// Verify first instruction is Call
	call, ok := entry.Instrs[0].(*ssa.Call)
	if !ok {
		t.Fatalf("expected first instruction to be *ssa.Call, got %T", entry.Instrs[0])
	}

	// Verify Call invokes the original function
	if call.Call.Value != origFn {
		t.Errorf("call target mismatch: got %v, want %v", call.Call.Value, origFn)
	}

	// Verify Call arguments are wrapper parameters
	if len(call.Call.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(call.Call.Args))
	}
	for i, arg := range call.Call.Args {
		if arg != wrapper.Params[i] {
			t.Errorf("arg[%d] mismatch: got %v, want %v", i, arg, wrapper.Params[i])
		}
	}

	// Verify second instruction is Return with Call result
	ret, ok := entry.Instrs[1].(*ssa.Return)
	if !ok {
		t.Fatalf("expected second instruction to be *ssa.Return, got %T", entry.Instrs[1])
	}
	if len(ret.Results) != 1 || ret.Results[0] != call {
		t.Errorf("return results mismatch: got %v, want [%v]", ret.Results, call)
	}

	// Print SSA for readability verification
	var buf bytes.Buffer
	wrapper.WriteTo(&buf)
	ssastr := buf.String()
	t.Logf("Generated SSA:\n%s", ssastr)

	// Verify SSA text contains expected content
	if !strings.Contains(ssastr, "Add$wrapper") {
		t.Error("SSA output missing wrapper function name")
	}
	if !strings.Contains(ssastr, "demo.Add") {
		t.Error("SSA output missing original function call")
	}
	if !strings.Contains(ssastr, "return") {
		t.Error("SSA output missing return statement")
	}
}

// TestMakeCallWrapper_StringReturn tests wrapping a function returning string
func TestMakeCallWrapper_StringReturn(t *testing.T) {
	prog, ssapkg := buildTestProgram(t)

	origFn := ssapkg.Func("Greet")
	if origFn == nil {
		t.Fatal("Greet function not found")
	}

	wrapper := MakeCallWrapper(prog, origFn)
	if wrapper == nil {
		t.Fatal("MakeCallWrapper returned nil")
	}

	// Verify signature
	if !types.Identical(wrapper.Signature, origFn.Signature) {
		t.Errorf("signature mismatch:\n got: %v\nwant: %v",
			wrapper.Signature, origFn.Signature)
	}

	// Verify instruction sequence
	entry := wrapper.Blocks[0]
	if len(entry.Instrs) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(entry.Instrs))
	}

	call := entry.Instrs[0].(*ssa.Call)
	ret := entry.Instrs[1].(*ssa.Return)

	if call.Call.Value != origFn {
		t.Error("call target mismatch")
	}
	if len(ret.Results) != 1 || ret.Results[0] != call {
		t.Error("return value mismatch")
	}
}

// TestMakeCallWrapper_NoReturn tests wrapping a function with no return value
func TestMakeCallWrapper_NoReturn(t *testing.T) {
	prog, ssapkg := buildTestProgram(t)

	origFn := ssapkg.Func("NoReturn")
	if origFn == nil {
		t.Fatal("NoReturn function not found")
	}

	wrapper := MakeCallWrapper(prog, origFn)
	if wrapper == nil {
		t.Fatal("MakeCallWrapper returned nil")
	}

	// Verify signature
	if !types.Identical(wrapper.Signature, origFn.Signature) {
		t.Errorf("signature mismatch:\n got: %v\nwant: %v",
			wrapper.Signature, origFn.Signature)
	}

	// Verify instruction sequence
	entry := wrapper.Blocks[0]
	if len(entry.Instrs) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(entry.Instrs))
	}

	call := entry.Instrs[0].(*ssa.Call)
	ret := entry.Instrs[1].(*ssa.Return)

	if call.Call.Value != origFn {
		t.Error("call target mismatch")
	}
	// No-return function: Return should have no results
	if len(ret.Results) != 0 {
		t.Errorf("expected 0 return results, got %d", len(ret.Results))
	}
}

// TestMakeCallWrapper_NilFunction tests passing nil function
func TestMakeCallWrapper_NilFunction(t *testing.T) {
	prog, _ := buildTestProgram(t)

	// Passing nil should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil function, but got none")
		}
	}()

	MakeCallWrapper(prog, nil)
}

// TestMakeCallWrapper_Referrers verifies Value reference relationships are correct
func TestMakeCallWrapper_Referrers(t *testing.T) {
	prog, ssapkg := buildTestProgram(t)

	origFn := ssapkg.Func("Add")
	wrapper := MakeCallWrapper(prog, origFn)
	entry := wrapper.Blocks[0]
	call := entry.Instrs[0].(*ssa.Call)

	// Verify Call's Referrers include Return
	refs := call.Referrers()
	if refs == nil {
		t.Fatal("call.Referrers() is nil")
	}

	foundReturn := false
	for _, ref := range *refs {
		if _, ok := ref.(*ssa.Return); ok {
			foundReturn = true
			break
		}
	}
	if !foundReturn {
		t.Error("Return instruction not found in Call's referrers")
	}
}
