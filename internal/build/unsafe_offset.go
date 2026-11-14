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

package build

import (
	"go/ast"
	"go/types"

	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

type offsetOwnerSizes struct {
	types.Sizes
	prog  llssa.Program
	owner func([]*types.Var) types.Type
}

func (s *offsetOwnerSizes) Offsetsof(fields []*types.Var) []int64 {
	if owner := s.owner(fields); owner != nil {
		return s.prog.SourceOffsets(owner)
	}
	return s.Sizes.Offsetsof(fields)
}

func withUnsafeOffsetOwner(prog llssa.Program, pkg *packages.Package) types.Sizes {
	base := pkg.TypesSizes
	if previous, ok := base.(*offsetOwnerSizes); ok {
		base = previous.Sizes
	}
	if owner := unsafeOffsetOwner(pkg); owner != nil {
		return &offsetOwnerSizes{base, prog, owner}
	}
	return base
}

// unsafeOffsetOwner bridges the context omitted by types.Sizes.Offsetsof.
//
// Identity: for `type G struct{F func(); Tail uintptr}; type C G`, G and C may
// share every field object, while cdecl C stores F in one word instead of two.
// Selection.Recv retains the actual owner of unsafe.Offsetof(c.Tail).
//
// Timing: go/types records a fresh Selection before each Offsetof evaluation.
// A repeated const initializer reuses its AST, so track Selection identity,
// not whether the expression already has a constant in Info.Types. Promoted
// fields issue one callback for each struct along Selection.Index.
//
// Lifetime: this state belongs to one checker and is rebuilt on typed rewrite.
// It does not change the AST, field objects, or shared Program type metadata.
func unsafeOffsetOwner(pkg *packages.Package) func([]*types.Var) types.Type {
	type query struct {
		selector  *ast.SelectorExpr
		callee    *ast.Ident
		selection *types.Selection
		next      int
	}
	var queries []query
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			selector, ok := ast.Unparen(call.Args[0]).(*ast.SelectorExpr)
			if !ok {
				return true
			}
			var callee *ast.Ident
			switch fun := ast.Unparen(call.Fun).(type) {
			case *ast.SelectorExpr:
				callee = fun.Sel
			case *ast.Ident:
				callee = fun
			}
			if callee != nil && callee.Name == "Offsetof" {
				queries = append(queries, query{selector: selector, callee: callee})
			}
			return true
		})
	}
	if len(queries) == 0 {
		return nil
	}
	info, owner := pkg.TypesInfo, pkg.Types
	builtin := types.Unsafe.Scope().Lookup("Offsetof")
	return func(fields []*types.Var) types.Type {
		if owner.Complete() {
			return nil
		}
		match := -1
		var result types.Type
		for i := range queries {
			q := &queries[i]
			selection := info.Selections[q.selector]
			if selection == nil || info.ObjectOf(q.callee) != builtin {
				continue
			}
			if selection != q.selection {
				q.selection, q.next = selection, 0
			}
			index := selection.Index()
			if q.next == len(index) {
				continue
			}
			current := selection.Recv()
			for _, field := range index[:q.next] {
				current = current.Underlying().(*types.Struct).Field(field).Type()
			}
			st := current.Underlying().(*types.Struct)
			if st.NumFields() != len(fields) {
				continue
			}
			same := true
			for j, field := range fields {
				if st.Field(j) != field {
					same = false
					break
				}
			}
			if !same {
				continue
			}
			if match != -1 {
				panic("unsafe.Offsetof: multiple active selections for the same fields")
			}
			match, result = i, current
		}
		if match != -1 {
			queries[match].next++
		}
		return result
	}
}
