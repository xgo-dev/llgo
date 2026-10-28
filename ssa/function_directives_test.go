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
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
)

func TestNoInterfaceUsesConcretePackageView(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	newMethod := func() (*types.Package, *types.Func) {
		pkg := types.NewPackage("example.com/p", "p")
		named := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "T", nil), types.NewStruct(nil, nil), nil)
		method := types.NewFunc(token.NoPos, pkg, "M", types.NewSignatureType(types.NewVar(token.NoPos, pkg, "", named), nil, nil, nil, nil, false))
		named.AddMethod(method)
		return pkg, method
	}
	original, hidden := newMethod()
	replacement, visible := newMethod()
	oldDecl, newDecl := new(ast.FuncDecl), new(ast.FuncDecl)
	oldInfo, newInfo := &directive.FunctionDirectives{NoInterface: true}, new(directive.FunctionDirectives)
	prog.FunctionSources().Set(original, map[*ast.FuncDecl]*directive.FunctionDirectives{oldDecl: oldInfo}, map[string]*directive.FunctionDirectives{"example.com/p.T.M": oldInfo})
	prog.FunctionSources().Set(replacement, map[*ast.FuncDecl]*directive.FunctionDirectives{newDecl: newInfo}, map[string]*directive.FunctionDirectives{"example.com/p.T.M": newInfo})
	prog.SetNoInterfaceMethod("example.com/p.T.M")
	if !prog.isNoInterfaceMethod(hidden) || prog.isNoInterfaceMethod(visible) {
		t.Fatal("same-path package views shared nointerface state")
	}
	prog.FunctionSources().UseBindings(original, replacement)
	if prog.isNoInterfaceMethod(hidden) {
		t.Fatal("types.Func consumer did not select the prepared patch view")
	}
	if source, ok := prog.FunctionSources().Declaration(oldDecl); !ok || !source.NoInterface {
		t.Fatal("selecting a patch changed the source declaration")
	}
}
