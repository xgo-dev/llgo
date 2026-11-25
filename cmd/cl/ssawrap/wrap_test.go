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

func Pair(a, b uint64) (uint64, bool) {
	return a * b, a != 0
}

func External(a, b uint64) (uint64, bool)

type Box struct{}
func (Box) Method(x int) int { return x }
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
	if wrapper.Name() != origFn.Name() || wrapper.Pkg != origFn.Pkg || wrapper.Synthetic != CallWrapper {
		t.Error("wrapper did not retain its source symbol and package")
	}
	if !strings.Contains(ssastr, "demo.Add") {
		t.Error("SSA output missing original function call")
	}
	for i, param := range wrapper.Params {
		if param == origFn.Params[i] || param.Parent() != wrapper {
			t.Error("wrapper reused the original function's parameter graph")
		}
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

func TestMakeCallWrapperTupleAndBodyless(t *testing.T) {
	prog, pkg := buildTestProgram(t)
	for _, name := range []string{"Pair", "External"} {
		t.Run(name, func(t *testing.T) {
			original := pkg.Func(name)
			wrapper := MakeCallWrapper(prog, original)
			if wrapper.Name() != original.Name() || wrapper.Pos() != original.Pos() || wrapper.Pkg != original.Pkg {
				t.Fatal("wrapper lost declaration identity or source position")
			}
			if len(wrapper.Params) != 2 {
				t.Fatal("wrapper did not derive parameters from the signature")
			}
			block := wrapper.Blocks[0]
			call := block.Instrs[0].(*ssa.Call)
			if !types.Identical(call.Type(), original.Signature.Results()) {
				t.Fatal("forwarded call lost its tuple type")
			}
			ret := block.Instrs[len(block.Instrs)-1].(*ssa.Return)
			if len(ret.Results) != 2 {
				t.Fatal("tuple results were not extracted")
			}
			for i, result := range ret.Results {
				extract := result.(*ssa.Extract)
				if extract.Tuple != call || extract.Index != i || extract.Parent() != wrapper ||
					!types.Identical(extract.Type(), original.Signature.Results().At(i).Type()) {
					t.Fatal("invalid typed tuple extraction")
				}
			}
			for i, param := range wrapper.Params {
				if param.Parent() != wrapper || call.Call.Args[i] != param || len(*param.Referrers()) != 1 {
					t.Fatal("wrapper parameter graph is not self-contained")
				}
			}
		})
	}
}

func TestMakeCallWrapperReceiver(t *testing.T) {
	prog, pkg := buildTestProgram(t)
	method := pkg.Type("Box").Type().(*types.Named).Method(0)
	original := prog.FuncValue(method)
	wrapper := MakeCallWrapper(prog, original)
	if len(wrapper.Params) != 2 || wrapper.Signature.Recv() != original.Signature.Recv() {
		t.Fatal("wrapper lost the receiver")
	}
	if !types.Identical(wrapper.Params[0].Type(), original.Signature.Recv().Type()) || wrapper.Params[0].Parent() != wrapper {
		t.Fatal("wrapper receiver is not its own parameter")
	}
}
