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

package cl

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sync"

	llssa "github.com/xgo-dev/llgo/ssa"
)

// registerTypeDefinitions keeps the source definition edges that go/types
// deliberately omits from Named's public API. Pass the same Info being filled
// by the checker so layout queries can see an RHS as soon as it is bound.
// Declaration positions identify declarations within this package, not the
// owner of shared struct fields. RHS types always come from semantic binding.
//
// A one-shot SSA client may have discarded Info already. For that path, query
// only named RHS expressions in the completed package's original lexical scope.
func registerTypeDefinitions(prog llssa.Program, fset *token.FileSet, pkg *types.Package, files []*ast.File, info *types.Info) {
	if pkg == nil {
		return
	}
	rhs := make(map[token.Pos]ast.Expr)
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			if spec, ok := node.(*ast.TypeSpec); ok {
				switch ast.Unparen(spec.Type).(type) {
				case *ast.Ident, *ast.SelectorExpr, *ast.IndexExpr, *ast.IndexListExpr:
					rhs[spec.Name.Pos()] = spec.Type
				}
			}
			return true
		})
	}
	var mu sync.Mutex
	resolved := make(map[*types.TypeName]types.Type)
	prog.SetTypeDefinitionSource(pkg, func(obj *types.TypeName) types.Type {
		expr := rhs[obj.Pos()]
		if expr == nil {
			return nil
		}
		if info != nil {
			return info.TypeOf(expr)
		}
		if !pkg.Complete() {
			return nil
		}
		mu.Lock()
		defer mu.Unlock()
		if t, ok := resolved[obj]; ok {
			return t
		}
		bound := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
		if err := types.CheckExpr(fset, pkg, expr.Pos(), expr, bound); err != nil {
			// This is already checked source. Silently selecting Go storage
			// here would turn a lost scope/import into a representation change.
			panic(fmt.Errorf("resolve representation of %s: %w", obj, err))
		}
		t := bound.TypeOf(expr)
		resolved[obj] = t
		return t
	})
}
