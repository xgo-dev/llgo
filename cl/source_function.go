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
	"go/ast"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// sourceFunction keeps the concrete SSA body/signature with its source record.
// Generic instances share their origin's declaration. Patch selection belongs
// to call target resolution and never changes this association.
type sourceFunction struct {
	SSA  *ssa.Function
	Decl *directive.FunctionDecl
}

func (p *context) sourceFunction(fn *ssa.Function) sourceFunction {
	if source, ok := p.sourceFunctions[fn]; ok {
		return source
	}
	source := sourceFunction{SSA: fn}
	if fn != nil {
		if origin := fn.Origin(); origin != nil {
			fn = origin
		}
		syntax, _ := fn.Syntax().(*ast.FuncDecl)
		obj, _ := fn.Object().(*types.Func)
		if syntax != nil || fn.Synthetic == "" || obj != nil && fn.Prog.FuncValue(obj) == fn {
			var pkg *types.Package
			if fn.Pkg != nil {
				pkg = fn.Pkg.Pkg
			} else if obj != nil {
				pkg = obj.Pkg()
			}
			source.Decl = p.prog.FunctionDeclaration(pkg, obj, syntax)
			if source.Decl == nil && !p.options.PreloadedSyntax {
				source.Decl = p.prog.Directives().FunctionDeclaration(syntax)
			}
		}
	}
	if p.sourceFunctions == nil {
		p.sourceFunctions = make(map[*ssa.Function]sourceFunction)
	}
	p.sourceFunctions[source.SSA] = source
	return source
}

// callableDeclaration selects a patch record only for packages with patches.
// Backend-created entries also use this selection without constructing Go SSA.
func (p *context) callableDeclaration(pkg *types.Package, obj *types.Func, name string, source *directive.FunctionDecl) *directive.FunctionDecl {
	if patch, ok := p.patches[llssa.PathOf(pkg)]; ok {
		if records := p.prog.PackageDirectives(patch.Types); records != nil {
			if obj != nil {
				if decl, ok := records.Objects[obj.Origin()].(*directive.FunctionDecl); ok {
					return decl
				}
			}
			if name == "" && obj != nil {
				fullName := llssa.FuncName(pkg, obj.Name(), obj.Type().(*types.Signature).Recv(), true)
				name = strings.TrimPrefix(fullName, llssa.PathOf(pkg)+".")
			}
			if decl, ok := records.Names[name].(*directive.FunctionDecl); ok {
				return decl
			}
		}
	}
	return source
}

// applyFunctionAttributes consumes the declaration already selected with the
// callable symbol. It does not resolve source identities or patches again.
func (p *context) applyFunctionAttributes(fn llssa.Function, signature *types.Signature, decl *directive.FunctionDecl) {
	if decl == nil {
		return
	}
	if decl.Cold {
		fn.SetCold()
	}
	if decl.NoReturn {
		fn.SetNoReturn()
	}
	properties := decl.Function.WithPositions(p.fset)
	if properties.ContractError != nil {
		panic(properties.ContractError)
	}
	fn.ApplyValueAttributes(signature, properties.Values)
}

// Backend-created entries consume prepared records without constructing Go SSA.
func (p *context) initFunctionAttributes(fn llssa.Function, obj *types.Func, signature *types.Signature) {
	decl := p.prog.FunctionDeclaration(obj.Pkg(), obj, nil)
	decl = p.callableDeclaration(obj.Pkg(), obj, "", decl)
	source := obj.Type().(*types.Signature)
	if decl != nil && source.Recv() != nil && signature.Recv() != nil && !types.Identical(source.Recv().Type(), signature.Recv().Type()) {
		// Receiver promises describe the loaded value, not the address received by
		// an ABI wrapper. Filter a local copy without changing the source record.
		adjusted := *decl
		adjusted.Values = nil
		for _, attr := range decl.Values {
			if attr.Target.Scope != directive.Receiver {
				adjusted.Values = append(adjusted.Values, attr)
			}
		}
		decl = &adjusted
	}
	p.applyFunctionAttributes(fn, signature, decl)
}
