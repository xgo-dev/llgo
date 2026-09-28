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

package ssa

import (
	"go/token"
	"go/types"

	"github.com/xgo-dev/llvm"
)

// commaOk is the SSA tuple (T, bool) used by map lookup, recv, and type assert.
// The bool stays i1, matching a scalar result, not a C struct field.
func (p Program) commaOk(val Type) Type {
	return p.resultTuple(val, p.Bool())
}

func (p Program) resultTuple(ts ...Type) Type {
	vars := make([]*types.Var, len(ts))
	for i, t := range ts {
		vars[i] = types.NewVar(token.NoPos, nil, "", t.RawType())
	}
	return p.rawType(types.NewTuple(vars...))
}

func isLLVMInt1(t llvm.Type) bool {
	return t.TypeKind() == llvm.IntegerTypeKind && t.IntTypeWidth() == 1
}

// boolI1 converts a Go bool to i1 for br/select. SSA bools are already i1;
// values extracted from memory or aggregates are i8 and need trunc.
func (b Builder) boolI1(cond Expr) llvm.Value {
	if isLLVMInt1(cond.impl.Type()) {
		return cond.impl
	}
	return b.fromMemory(cond.impl, b.Prog.Bool()).impl
}

// llvmMemType is Clang's ConvertTypeForMem: a bool is i1 as an SSA value
// and i8 in memory, arrays, and struct fields.
func (p Program) llvmMemType(t Type) llvm.Type {
	if t.kind == vkBool {
		return p.tyInt8()
	}
	return t.ll
}

func (p Program) boolToMemConst(v llvm.Value) llvm.Value {
	if v.Type() == p.tyInt8() {
		return v
	}
	var n uint64
	if c := v.IsAConstantInt(); !c.IsNil() && v.ZExtValue() != 0 {
		n = 1
	}
	return llvm.ConstInt(p.tyInt8(), n, false)
}

// toMemory converts a Go bool SSA value (i1) to its in-memory i8 form.
func (b Builder) toMemory(v Expr) llvm.Value {
	if v.kind != vkBool || v.impl.Type() == b.Prog.tyInt8() {
		return v.impl
	}
	if !v.impl.IsAConstant().IsNil() {
		return b.Prog.boolToMemConst(v.impl)
	}
	return llvm.CreateZExt(b.impl, v.impl, b.Prog.tyInt8())
}

// fromMemory converts a loaded/extracted in-memory bool (i8) to an i1 SSA value.
func (b Builder) fromMemory(v llvm.Value, t Type) Expr {
	if t.kind != vkBool || isLLVMInt1(v.Type()) {
		return Expr{v, t}
	}
	if !v.IsAConstant().IsNil() {
		var n uint64
		if c := v.IsAConstantInt(); !c.IsNil() && v.ZExtValue() != 0 {
			n = 1
		}
		return Expr{llvm.ConstInt(t.ll, n, false), t}
	}
	return Expr{llvm.CreateTrunc(b.impl, v, t.ll), t}
}
