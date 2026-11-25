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
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/types"
)

func TestDebugRefMoveHelpers(t *testing.T) {
	movedValue := &BinOp{}
	instrs := []Instruction{
		movedValue,
		&DebugRef{X: movedValue},
		&Return{Results: []Value{movedValue}},
	}

	moved := orderMovedValuesForIndices(instrs, map[int]struct{}{
		-1:          {},
		0:           {},
		2:           {},
		len(instrs): {},
	})
	if len(moved) != 1 {
		t.Fatalf("moved values = %d, want 1", len(moved))
	}
	if _, ok := moved[movedValue]; !ok {
		t.Fatal("moved values do not contain the selected SSA value")
	}

	move := map[int]struct{}{0: {}}
	orderIncludeDebugRefsForMovedValues(instrs, move, moved, -1, len(instrs)+1)
	if _, ok := move[1]; !ok {
		t.Fatal("DebugRef for the moved value was not included")
	}
}

func TestMoveInstrsAfter(t *testing.T) {
	const src = `package p
func f() {
	println(1)
	println(2)
	println(3)
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
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
	pkg, err := (&types.Config{}).Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	prog := NewProgram(fset, SanityCheckFunctions|InstantiateGenerics)
	ssapkg := prog.CreatePackage(pkg, []*ast.File{file}, info, true)
	ssapkg.Build()
	fn := ssapkg.Func("f")
	instrs := fn.Blocks[0].Instrs
	if len(instrs) < 4 {
		t.Fatalf("instructions = %d, want at least 4", len(instrs))
	}

	assertOrder := func(t *testing.T, got, want []Instruction) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("instruction count = %d, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("instruction %d = %v, want %v", i, got[i], want[i])
			}
		}
	}

	t.Run("empty", func(t *testing.T) {
		assertOrder(t, orderMoveInstrsAfter(instrs, nil, instrs[1]), instrs)
	})
	t.Run("nil-anchor", func(t *testing.T) {
		moving := map[Instruction]struct{}{instrs[0]: {}}
		assertOrder(t, orderMoveInstrsAfter(instrs, moving, nil), instrs)
	})
	t.Run("missing-anchor", func(t *testing.T) {
		moving := map[Instruction]struct{}{instrs[0]: {}}
		assertOrder(t, orderMoveInstrsAfter(instrs, moving, &Return{}), instrs)
	})
	t.Run("anchor-in-moving", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("orderMoveInstrsAfter did not reject an anchor in the moving set")
			}
		}()
		moving := map[Instruction]struct{}{instrs[1]: {}}
		orderMoveInstrsAfter(instrs, moving, instrs[1])
	})
	t.Run("stable", func(t *testing.T) {
		moving := map[Instruction]struct{}{
			instrs[0]: {},
			instrs[2]: {},
		}
		want := []Instruction{instrs[1], instrs[0], instrs[2]}
		want = append(want, instrs[3:]...)
		assertOrder(t, orderMoveInstrsAfter(instrs, moving, instrs[1]), want)
	})
}
