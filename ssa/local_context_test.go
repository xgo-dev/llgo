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
	"strings"
	"testing"
)

func TestBuildLocalPackageAccessor(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(localContextTestRuntime())
	pkg := prog.NewPackage("accessor", "example.com/accessor")
	cache := pkg.NewThreadLocalVar("example.com/accessor.cache", types.NewPointer(types.Typ[types.Uintptr]), InGo)
	cache.InitNil()
	field := types.NewField(token.NoPos, nil, "pointer", types.NewPointer(types.Typ[types.Int]), false)
	block := types.NewStruct([]*types.Var{field}, nil)
	blockType := prog.Type(block, InGo)
	result := types.NewPointer(block)
	results := types.NewTuple(types.NewVar(token.NoPos, nil, "", result))
	accessor := pkg.NewFunc("example.com/accessor.block", types.NewSignatureType(nil, nil, nil, nil, results, false), InGo)
	accessor.BuildLocalPackageAccessor(
		cache.Expr,
		prog.IntVal(prog.SizeOf(blockType), prog.Uintptr()),
		prog.IntVal(prog.AlignOf(blockType), prog.Uintptr()),
	)

	ir := pkg.String()
	if !strings.Contains(ir, `@"example.com/accessor.cache" = thread_local global i64 0`) {
		t.Fatalf("direct cache definition not found:\n%s", ir)
	}
	if got := strings.Count(ir, "icmp "); got != 1 {
		t.Fatalf("accessor comparisons = %d, want one cache check:\n%s", got, ir)
	}
	if !strings.Contains(ir, `call ptr @"`+PkgRuntime+`.LocalPackage"(ptr @"example.com/accessor.cache"`) {
		t.Fatalf("accessor has no cache-backed LocalPackage slow path:\n%s", ir)
	}
}

func TestBuildGoroutineLocalPackageAccessor(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(localContextTestRuntime())
	pkg := prog.NewPackage("accessor", "example.com/accessor")
	key := pkg.NewVar("example.com/accessor.key", types.NewPointer(types.Typ[types.Uintptr]), InGo)
	key.InitNil()
	field := types.NewField(token.NoPos, nil, "value", types.Typ[types.Int], false)
	block := types.NewStruct([]*types.Var{field}, nil)
	blockType := prog.Type(block, InGo)
	result := types.NewPointer(block)
	results := types.NewTuple(types.NewVar(token.NoPos, nil, "", result))
	accessor := pkg.NewFunc("example.com/accessor.gls", types.NewSignatureType(nil, nil, nil, nil, results, false), InGo)
	accessor.BuildGoroutineLocalPackageAccessor(
		key.Expr,
		prog.IntVal(prog.SizeOf(blockType), prog.Uintptr()),
		prog.IntVal(prog.AlignOf(blockType), prog.Uintptr()),
	)

	ir := pkg.String()
	if !strings.Contains(ir, `@"example.com/accessor.key" = global i64 0`) ||
		strings.Contains(ir, `@"example.com/accessor.key" = thread_local`) {
		t.Fatalf("logical package key is not process-global:\n%s", ir)
	}
	if !strings.Contains(ir, `call ptr @"`+PkgRuntime+`.GoroutineLocalPackage"(ptr @"example.com/accessor.key"`) {
		t.Fatalf("accessor has no logical-G package lookup:\n%s", ir)
	}
}

func localContextTestRuntime() *types.Package {
	pkg := types.NewPackage(PkgRuntime, "runtime")
	unsafePointer := types.Typ[types.UnsafePointer]

	params := types.NewTuple(
		types.NewVar(token.NoPos, pkg, "cache", types.NewPointer(types.Typ[types.Uintptr])),
		types.NewVar(token.NoPos, pkg, "size", types.Typ[types.Uintptr]),
		types.NewVar(token.NoPos, pkg, "align", types.Typ[types.Uintptr]),
	)
	results := types.NewTuple(types.NewVar(token.NoPos, pkg, "", unsafePointer))
	pkg.Scope().Insert(types.NewFunc(token.NoPos, pkg, "LocalPackage", types.NewSignatureType(nil, nil, nil, params, results, false)))
	pkg.Scope().Insert(types.NewFunc(token.NoPos, pkg, "GoroutineLocalPackage", types.NewSignatureType(nil, nil, nil, params, results, false)))
	return pkg
}
