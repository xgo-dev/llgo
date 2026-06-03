//go:build !llgo

/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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
	"go/constant"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/goplus/llgo/ssa"
	"github.com/goplus/llgo/ssa/ssatest"
)

func TestDeferInLoopIR(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	callee := pkg.NewFunc("callee", ssa.NoArgsNoRet, ssa.InGo)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())

	// Ensure entry block has a terminator like real codegen
	b.Return()
	b.SetBlockEx(fn.Block(0), ssa.BeforeLast, true)

	b.Defer(ssa.DeferInLoop, callee.Expr, ssa.Builder.Call)
	b.EndBuild()

	ir := pkg.Module().String()
	if !strings.Contains(ir, "icmp ne ptr") {
		t.Fatalf("expected loop defer condition in IR, got:\n%s", ir)
	}
	// Loop defers must record each execution (even with no args) so the drain loop
	// can run deferred calls the correct number of times.
	if !strings.Contains(ir, "FreeDeferNode") {
		t.Fatalf("expected loop defer node free in IR, got:\n%s", ir)
	}
}

func TestDeferAtomicInLoopIR(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	int64PtrType := types.NewPointer(types.Typ[types.Int64])
	params := types.NewTuple(types.NewParam(token.NoPos, nil, "", int64PtrType))
	IntPtrArgSig := types.NewSignatureType(nil, nil, nil, params, nil, false)
	fn := pkg.NewFunc("main", IntPtrArgSig, ssa.InGo)
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())

	// Ensure entry block has a terminator like real codegen
	b.Return()
	b.SetBlockEx(fn.Block(0), ssa.BeforeLast, true)

	ptr := fn.Param(0)
	val := b.Const(constant.MakeInt64(1), prog.Int64())
	b.Defer(ssa.DeferInLoop, ssa.Nil, func(b ssa.Builder, _ ssa.Expr, args ...ssa.Expr) ssa.Expr {
		return b.Store(ptr, val).SetOrdering(ssa.OrderingSeqConsistent)
	})
	b.EndBuild()

	ir := pkg.Module().String()
	atomicIdx := strings.Index(ir, "store atomic i64 1, ptr %0 seq_cst")
	if atomicIdx == -1 {
		t.Fatalf("expected store atomic in IR, got:\n%s", ir)
	}
	// Loop defers must record each execution (even with no args) so the drain loop
	// can run deferred calls the correct number of times.
	if !strings.Contains(ir[atomicIdx:], "FreeDeferNode") {
		t.Fatalf("expected loop defer node free in IR, got:\n%s", ir)
	}
}
