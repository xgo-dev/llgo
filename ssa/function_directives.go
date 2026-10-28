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
	"go/types"
)

// FunctionDirectives is the prepared value attached to a source function.
// Empty strings mean no binding; valid bindings always have a nonempty name.
// No AST, object, package, or derived symbol identity is duplicated here.
type FunctionDirectives struct {
	NoInline       bool
	NoSplit        bool
	UintptrEscapes bool
	ClosureEnv     bool
	NoInterface    bool
	WasmModule     string
	WasmName       string
	Linkname       string
	ExportName     string
}

// SetFunctionDirectives publishes one package's function declarations. names
// selects the effective declaration when source patches contain replacements.
// Both indexes point to the same immutable values; names is used only at the
// call-binding and types.Func boundaries, never to select a function body.
func (p Program) SetFunctionDirectives(pkg *types.Package, declarations map[*ast.FuncDecl]*FunctionDirectives, names map[string]*FunctionDirectives) {
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	for decl, info := range declarations {
		p.packageSyntax.functions[decl] = info
	}
	p.packageSyntax.functionNames[pkg] = names
}

// FunctionDirectives returns the exact source declaration's snapshot.
func (p Program) FunctionDirectives(decl *ast.FuncDecl) (FunctionDirectives, bool) {
	p.packageSyntax.mu.RLock()
	info, ok := p.packageSyntax.functions[decl]
	p.packageSyntax.mu.RUnlock()
	if !ok {
		return FunctionDirectives{}, false
	}
	return *info, true
}

// FunctionBinding returns the selected declaration within a concrete package
// view. The second result distinguishes unprepared packages from known absence.
func (p Program) FunctionBinding(pkg *types.Package, name string) (FunctionDirectives, bool) {
	p.packageSyntax.mu.RLock()
	names, prepared := p.packageSyntax.functionNames[pkg]
	info := names[name]
	p.packageSyntax.mu.RUnlock()
	if info == nil {
		return FunctionDirectives{}, prepared
	}
	return *info, prepared
}

// UseFunctionBindings selects a prepared replacement view for types.Func
// consumers. Source body snapshots remain associated with their original AST.
func (p Program) UseFunctionBindings(pkg, effective *types.Package) {
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	names, ok := p.packageSyntax.functionNames[effective]
	if !ok {
		panic("function bindings not prepared for replacement package")
	}
	p.packageSyntax.functionNames[pkg] = names
}

// RegisterGeneratedFunction records the known absence of directives on a
// compiler-created declaration. Call this where the AST helper is created,
// before SSA construction, rather than treating lookup misses as empty facts.
func (p Program) RegisterGeneratedFunction(decl *ast.FuncDecl) {
	if decl.Doc != nil {
		panic("generated function unexpectedly has directives")
	}
	p.packageSyntax.mu.Lock()
	p.packageSyntax.functions[decl] = &FunctionDirectives{}
	p.packageSyntax.mu.Unlock()
}
