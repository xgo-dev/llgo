/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ssa_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
	"github.com/xgo-dev/llgo/types"
)

type testImporter func(string) (*types.Package, error)

func (f testImporter) Import(path string) (*types.Package, error) { return f(path) }

func buildSSAPkg(t *testing.T, path, src string, retainInit map[string]bool) (*ssa.Package, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Instances:  make(map[*ast.Ident]types.Instance),
	}
	cfg := &types.Config{
		Importer: testImporter(func(impPath string) (*types.Package, error) {
			if impPath == "unsafe" {
				return types.Unsafe, nil
			}
			return nil, types.Error{Msg: "unexpected import " + impPath}
		}),
	}
	pkg, err := cfg.Check(path, fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	prog := ssa.NewProgram(fset, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
	prog.CreatePackage(types.Unsafe, nil, nil, true)
	ssaPkg := prog.CreatePackage(pkg, []*ast.File{file}, info, true)
	ssaPkg.RetainInit = retainInit
	ssaPkg.Build()
	return ssaPkg, file
}

func TestSSALLGoMetadataCallKindAndIntrinsics(t *testing.T) {
	const src = `package demo

const LLGoPackage = "py.sys"

//llgo:skip OldSym
type skipMarker int

//llgo:link cPrintf C.printf
//go:noinline
//go:noescape
func cPrintf(fmt *byte, __llgo_va_list ...any) int32

//llgo:link winProc stdcall.WndProc
func winProc(x int32) int32

//llgo:link pyExit py.exit
func pyExit(code int32)

//llgo:link allocaT llgo.alloca
func allocaT[T any](n uintptr) *T { return nil }

//llgo:link llvmTrap llvm.trap
func llvmTrap()

//go:wasmimport env log_i32
func logI32(x int32)

//go:wasmexport tick
func tick() {}

//llgo:env
func envEntry() {}

//llgointernal:tls
//llgo:link tlsVar C.tls_var
var tlsVar uintptr

func caller() {
	_ = cPrintf(nil, 1)
	_ = winProc(1)
	pyExit(0)
	_ = allocaT[int](8)
	llvmTrap()
	logI32(42)
}
`
	ssaPkg, _ := buildSSAPkg(t, "example.com/demo", src, nil)

	if ssaPkg.Kind != directive.PkgPyModule || ssaPkg.PyModule() != "sys" || !ssaPkg.Skipped("OldSym") || ssaPkg.SkipsInit() {
		t.Fatalf("pkg metadata = (Kind=%v Py=%q Skipped=%v)", ssaPkg.Kind, ssaPkg.PyModule(), ssaPkg.Skipped("OldSym"))
	}

	cPrintf := ssaPkg.Func("cPrintf")
	if cPrintf.CallKind() != ssa.CallC || cPrintf.CallConv() != types.CDecl || cPrintf.LinkSymbol() != "printf" || !cPrintf.NoInline || !cPrintf.NoEscape {
		t.Fatalf("cPrintf = (Kind=%v Conv=%v Sym=%q NoInline=%v NoEscape=%v)",
			cPrintf.CallKind(), cPrintf.CallConv(), cPrintf.LinkSymbol(), cPrintf.NoInline, cPrintf.NoEscape)
	}
	if winProc := ssaPkg.Func("winProc"); winProc.CallKind() != ssa.CallStdcall || winProc.LinkSymbol() != "WndProc" {
		t.Fatalf("winProc = (%v, %q)", winProc.CallKind(), winProc.LinkSymbol())
	}
	if pyExit := ssaPkg.Func("pyExit"); pyExit.CallKind() != ssa.CallPy || pyExit.LinkSymbol() != "exit" {
		t.Fatalf("pyExit = (%v, %q)", pyExit.CallKind(), pyExit.LinkSymbol())
	}
	if llvmTrap := ssaPkg.Func("llvmTrap"); llvmTrap.CallKind() != ssa.CallLLVMIntrinsic || llvmTrap.LLVMIntrinsic() != "llvm.trap" {
		t.Fatalf("llvmTrap = (%v, %q)", llvmTrap.CallKind(), llvmTrap.LLVMIntrinsic())
	}
	if logI32 := ssaPkg.Func("logI32"); logI32.CallKind() != ssa.CallWasm || logI32.WasmImport == nil || logI32.WasmImport.Name != "log_i32" {
		t.Fatalf("logI32 = (%v, %+v)", logI32.CallKind(), logI32.WasmImport)
	}
	if tick := ssaPkg.Func("tick"); tick.WasmExport != "tick" {
		t.Fatalf("tick.WasmExport = %q", tick.WasmExport)
	}
	if envEntry := ssaPkg.Func("envEntry"); !envEntry.HasClosureEnv {
		t.Fatal("envEntry.HasClosureEnv = false")
	}
	if tlsVar := ssaPkg.Var("tlsVar"); tlsVar.Locality != directive.LocalityThread || tlsVar.LinkName != "C.tls_var" {
		t.Fatalf("tlsVar = (Locality=%v LinkName=%q)", tlsVar.Locality, tlsVar.LinkName)
	} else if ns, sym := tlsVar.LinkTarget(); ns != directive.LinkC || sym != "tls_var" {
		t.Fatalf("tlsVar.LinkTarget() = (%v, %q)", ns, sym)
	}

	// Verify CallCommon.CallKind / Intrinsic / IsCVariadic and generic intrinsic propagation.
	var sawCVariadic, sawGenericAlloca, sawLLVM, sawWasm bool
	for _, blk := range ssaPkg.Func("caller").Blocks {
		for _, instr := range blk.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			cc := &call.Call
			switch cc.CallKind() {
			case ssa.CallC:
				if cc.IsCVariadic() {
					sawCVariadic = true
				}
			case ssa.CallLLGoIntrinsic:
				if cc.Intrinsic() == ssa.IntrinsicAlloca {
					sawGenericAlloca = true
				}
			case ssa.CallLLVMIntrinsic:
				sawLLVM = true
			case ssa.CallWasm:
				sawWasm = true
			}
		}
	}
	if !sawCVariadic || !sawGenericAlloca || !sawLLVM || !sawWasm {
		t.Fatalf("call kinds missing: cVariadic=%v alloca=%v llvm=%v wasm=%v",
			sawCVariadic, sawGenericAlloca, sawLLVM, sawWasm)
	}
}

func TestSSARecvNilCheckAndVerify(t *testing.T) {
	const src = `package demo

type Scalar struct { A int; B float64 }
type HasPtr struct { P *int }

type T struct { X int }
func (*T) M(int) {}
type OuterVal struct { Pad int; T }
type OuterPtr struct { *T }

func allocs(s []int) (*Scalar, *HasPtr, []int, map[int]int) {
	sc := new(Scalar)
	p := new(HasPtr)
	return sc, p, s[:], make(map[int]int)
}

func calls(t *T, v *OuterVal, p *OuterPtr) func(int) {
	(*t).M(1)
	v.M(2)
	p.M(3)
	return (*t).M
}
`
	ssaPkg, _ := buildSSAPkg(t, "example.com/demo", src, nil)
	if err := ssa.VerifyPackage(ssaPkg); err != nil {
		t.Fatalf("VerifyPackage failed on real package with optional nil operands: %v", err)
	}

	var sawPtrFree, sawPtrBearing bool
	for _, blk := range ssaPkg.Func("allocs").Blocks {
		for _, instr := range blk.Instrs {
			alloc, ok := instr.(*ssa.Alloc)
			if !ok {
				continue
			}
			elem := alloc.Type().(*types.Pointer).Elem()
			if elem.HasPtrData() {
				sawPtrBearing = true
			} else {
				sawPtrFree = true
			}
		}
	}
	if !sawPtrFree || !sawPtrBearing {
		t.Fatalf("HasPtrData classification incomplete: ptrFree=%v ptrBearing=%v", sawPtrFree, sawPtrBearing)
	}

	var callChecks, addrChecks, closureChecks int
	for _, blk := range ssaPkg.Func("calls").Blocks {
		for _, instr := range blk.Instrs {
			switch ins := instr.(type) {
			case *ssa.Call:
				if ins.Call.RecvNilCheck != nil {
					callChecks++
					if ins.Call.RecvNilCheck.Address {
						addrChecks++
					}
				}
			case *ssa.MakeClosure:
				if ins.RecvNilCheck != nil {
					closureChecks++
					if !ins.RecvNilCheck.Address {
						t.Fatal("(*t).M bound method expected Address=true")
					}
				}
			}
		}
	}
	if callChecks != 3 || addrChecks != 2 || closureChecks != 1 {
		t.Fatalf("RecvNilCheck counts = (calls=%d addr=%d closures=%d), want (3, 2, 1)",
			callChecks, addrChecks, closureChecks)
	}
}

func TestSSAStaticBytesNoASTMutationAndRetainInit(t *testing.T) {
	const n = 4096
	src := fmt.Sprintf(`package demo

import _ "unsafe"

var Big = [%d]byte{0: 7, %s %d: 9}
var Ellipsis = [...]byte{%s}

//llgointernal:gls
var GlsBig = [%d]byte{%s}

//go:linkname LinkBig other.LinkBig
var LinkBig = [%d]byte{%s}
`, n, strings.Repeat("1, ", n-2), n-1, strings.Repeat("2, ", n), n, strings.Repeat("3, ", n), n, strings.Repeat("4, ", n))

	ssaPkg, file := buildSSAPkg(t, "example.com/demo", src, nil)
	big := ssaPkg.Var("Big")
	if len(big.StaticBytes) != n || big.StaticBytes[0] != 7 || big.StaticBytes[n-1] != 9 {
		t.Fatalf("Big.StaticBytes len=%d [0]=%d [%d]=%d", len(big.StaticBytes), big.StaticBytes[0], n-1, big.StaticBytes[n-1])
	}
	ellipsis := ssaPkg.Var("Ellipsis")
	if len(ellipsis.StaticBytes) != n || ellipsis.StaticBytes[0] != 2 || ellipsis.StaticBytes[n-1] != 2 {
		t.Fatalf("Ellipsis.StaticBytes len=%d", len(ellipsis.StaticBytes))
	}
	if len(ssaPkg.Var("GlsBig").StaticBytes) != 0 {
		t.Fatal("GlsBig with locality must not fold into StaticBytes")
	}
	if len(ssaPkg.Var("LinkBig").StaticBytes) != 0 {
		t.Fatal("LinkBig with linkname must not fold into StaticBytes")
	}
	// Verify AST was NOT mutated.
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		vs := gd.Specs[0].(*ast.ValueSpec)
		if len(vs.Values) != 1 {
			t.Fatalf("AST ValueSpec %s was mutated (Values=%d)", vs.Names[0].Name, len(vs.Values))
		}
	}

	// Verify RetainInit keeps executable stores instead of folding StaticBytes.
	retainedPkg, _ := buildSSAPkg(t, "example.com/demo", src, map[string]bool{"Big": true})
	if len(retainedPkg.Var("Big").StaticBytes) != 0 {
		t.Fatal("RetainInit global Big unexpectedly folded into StaticBytes")
	}
	if len(retainedPkg.Var("Ellipsis").StaticBytes) != n {
		t.Fatalf("non-retained Ellipsis lost StaticBytes: len=%d", len(retainedPkg.Var("Ellipsis").StaticBytes))
	}
}

func TestSSASIMDIntrinsicExcludesGoBodyHelpers(t *testing.T) {
	const src = `package archsimd

type X86Features struct{}
func (X86Features) AVX() bool { return true }
func ClearAVXUpperBits()
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "archsimd.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	ssaPkg, _, err := ssautil.BuildPackage(&types.Config{}, fset, types.NewPackage("simd/archsimd", "archsimd"), []*ast.File{file}, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	x86 := ssaPkg.Type("X86Features").Type().(*types.Named)
	avx := ssaPkg.Prog.FuncValue(x86.Method(0))
	if avx.Intrinsic() != ssa.IntrinsicNone {
		t.Fatalf("X86Features.AVX() with Go body misclassified as %v", avx.Intrinsic())
	}
	if clearFn := ssaPkg.Func("ClearAVXUpperBits"); clearFn.Intrinsic() != ssa.IntrinsicSIMD {
		t.Fatalf("ClearAVXUpperBits.Intrinsic() = %v, want IntrinsicSIMD", clearFn.Intrinsic())
	}
}
