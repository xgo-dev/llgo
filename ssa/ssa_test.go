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
	"bytes"
	"go/constant"
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

func TestProgramPackageFunctionDominatorsAndVerify(t *testing.T) {
	fset := token.NewFileSet()
	prog := ssa.NewProgram(fset, ssa.SanityCheckFunctions|ssa.GlobalDebug)
	goPkg := types.NewPackage("example.com/demo", "demo")
	pkg := prog.CreatePackage(goPkg, nil, nil, true)
	if prog.Package(goPkg) != pkg || prog.ImportedPackage("example.com/demo") != pkg {
		t.Fatal("Program package lookup failed")
	}

	gVar := types.NewVar(0, goPkg, "Counter", types.Typ[types.Int])
	g := pkg.AddGlobal("Counter", gVar, types.NewPointer(types.Typ[types.Int]))
	if pkg.Var("Counter") != g || g.String() != "example.com/demo.Counter" {
		t.Fatalf("Global lookup mismatch: %v", g)
	}

	// func choose(cond bool, a, b int) int
	sig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(0, goPkg, "cond", types.Typ[types.Bool]),
			types.NewVar(0, goPkg, "a", types.Typ[types.Int]),
			types.NewVar(0, goPkg, "b", types.Typ[types.Int]),
		),
		types.NewTuple(types.NewVar(0, goPkg, "", types.Typ[types.Int])),
		false,
	)
	fnObj := types.NewFunc(0, goPkg, "choose", sig)
	fn := prog.NewFunction("choose", sig, "")
	pkg.AddFunction(fn, fnObj)
	pCond := fn.AddParam("cond", nil, types.Typ[types.Bool])
	pA := fn.AddParam("a", nil, types.Typ[types.Int])
	pB := fn.AddParam("b", nil, types.Typ[types.Int])

	entry := fn.NewBasicBlock("entry")
	thenBlk := fn.NewBasicBlock("then")
	elseBlk := fn.NewBasicBlock("else")
	mergeBlk := fn.NewBasicBlock("merge")

	entry.Emit(&ssa.If{Cond: pCond}, nil, token.Pos(1))
	entry.AddEdge(thenBlk)
	entry.AddEdge(elseBlk)

	one := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
	addA := &ssa.BinOp{Op: token.ADD, X: pA, Y: one}
	thenBlk.Emit(addA, types.Typ[types.Int], token.Pos(2))
	thenBlk.Emit(&ssa.Jump{}, nil, token.Pos(3))
	thenBlk.AddEdge(mergeBlk)

	elseBlk.Emit(&ssa.Store{Addr: g, Val: pB}, nil, token.Pos(4))
	elseBlk.Emit(&ssa.Jump{}, nil, token.Pos(5))
	elseBlk.AddEdge(mergeBlk)

	phi := &ssa.Phi{Comment: "res", Edges: []ssa.Value{addA, pB}}
	mergeBlk.Emit(phi, types.Typ[types.Int], token.Pos(6))
	mergeBlk.Emit(&ssa.Return{Results: []ssa.Value{phi}}, nil, token.Pos(7))

	prog.Build()

	if err := ssa.VerifyPackage(pkg); err != nil {
		t.Fatalf("VerifyPackage: %v", err)
	}
	if !entry.Dominates(mergeBlk) || thenBlk.Dominates(mergeBlk) || mergeBlk.Idom() != entry {
		t.Fatalf("unexpected dominator tree: merge.Idom()=%v", mergeBlk.Idom())
	}
	if len(fn.DomPreorder()) != 4 || len(fn.DomPostorder()) != 4 {
		t.Fatalf("unexpected dom traversal lengths: %d %d", len(fn.DomPreorder()), len(fn.DomPostorder()))
	}
	if refs := pA.Referrers(); refs == nil || len(*refs) != 1 || (*refs)[0] != addA {
		t.Fatalf("pA.Referrers() = %v, want [%v]", refs, addA)
	}
	if prog.FuncValue(fnObj) != fn || pkg.Func("choose") != fn {
		t.Fatal("Func lookup failed")
	}

	var buf bytes.Buffer
	ssa.WritePackage(&buf, pkg)
	ssa.WriteFunction(&buf, fn)
	if !strings.Contains(buf.String(), "phi [1: t0, 2: b]") {
		t.Fatalf("unexpected disassembly:\n%s", buf.String())
	}
}

func TestVerifyCatchesErrors(t *testing.T) {
	prog := ssa.NewProgram(token.NewFileSet(), 0)
	sig := types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])), false)
	fn := prog.NewFunction("bad", sig, "")
	b := fn.NewBasicBlock("entry")
	// Alloc.NoScan on pointer-bearing type must fail verification.
	b.Emit(&ssa.Alloc{Heap: true, NoScan: true}, types.NewPointer(types.NewPointer(types.Typ[types.Int])), token.NoPos)
	b.Emit(&ssa.Return{Results: []ssa.Value{ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])}}, nil, token.NoPos)
	if err := ssa.VerifyFunction(fn); err == nil || !strings.Contains(err.Error(), "noscan") {
		t.Fatalf("expected noscan verification error, got %v", err)
	}

	// BuilderMode flag round-trip (matching x/tools/go/ssa letter order DPFSCNLIG).
	var mode ssa.BuilderMode
	if err := mode.Set("DPFSCNLIG"); err != nil {
		t.Fatalf("BuilderMode.Set: %v", err)
	}
	if mode.String() != "DPFSCNLIG" {
		t.Fatalf("BuilderMode.String() = %q", mode.String())
	}
	if err := mode.Set("Z"); err == nil {
		t.Fatal("expected error for unknown BuilderMode flag")
	}
}
