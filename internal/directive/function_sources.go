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

package directive

import (
	"go/ast"
	"go/types"
	"iter"
	"sync"
)

// FunctionSources holds prepared function values. Source preparation publishes
// the AST and concrete-package indexes before any function wrapper is used.
// It contains no LLVM state and is shared by isolated package backends.
type FunctionSources struct {
	mu            sync.RWMutex
	functions     map[*ast.FuncDecl]*FunctionDirectives
	functionNames map[*types.Package]map[string]*FunctionDirectives
}

func NewFunctionSources() *FunctionSources {
	return &FunctionSources{
		functions:     make(map[*ast.FuncDecl]*FunctionDirectives),
		functionNames: make(map[*types.Package]map[string]*FunctionDirectives),
	}
}

// Set publishes one package's function declarations. names
// selects the effective declaration when source patches contain replacements.
// Both indexes point to the same immutable values; names is used only at the
// call-binding and types.Func boundaries, never to select a function body.
func (p *FunctionSources) Set(pkg *types.Package, declarations map[*ast.FuncDecl]*FunctionDirectives, names map[string]*FunctionDirectives) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for decl, info := range declarations {
		p.functions[decl] = info
	}
	p.functionNames[pkg] = names
}

// Declaration returns the exact source declaration's snapshot.
func (p *FunctionSources) Declaration(decl *ast.FuncDecl) (FunctionDirectives, bool) {
	p.mu.RLock()
	info, ok := p.functions[decl]
	p.mu.RUnlock()
	if !ok {
		return FunctionDirectives{}, false
	}
	return *info, true
}

// Binding returns the selected declaration within a concrete package
// view. The second result distinguishes unprepared packages from known absence.
func (p *FunctionSources) Binding(pkg *types.Package, name string) (FunctionDirectives, bool) {
	p.mu.RLock()
	names, prepared := p.functionNames[pkg]
	info := names[name]
	p.mu.RUnlock()
	if info == nil {
		return FunctionDirectives{}, prepared
	}
	return *info, prepared
}

// UseBindings selects a prepared replacement view for types.Func
// consumers. Source body snapshots remain associated with their original AST.
func (p *FunctionSources) UseBindings(pkg, effective *types.Package) {
	p.mu.Lock()
	defer p.mu.Unlock()
	names, ok := p.functionNames[effective]
	if !ok {
		panic("function bindings not prepared for replacement package")
	}
	p.functionNames[pkg] = names
}

// RegisterGenerated records the known absence of directives on a
// compiler-created declaration. Call this where the AST helper is created,
// before SSA construction, rather than treating lookup misses as empty facts.
func (p *FunctionSources) RegisterGenerated(decl *ast.FuncDecl) {
	if decl.Doc != nil {
		panic("generated function unexpectedly has directives")
	}
	p.mu.Lock()
	p.functions[decl] = &FunctionDirectives{}
	p.mu.Unlock()
}

// Exports iterates prepared function exports for a concrete package view.
// Consumers do not need to revisit declarations to preserve these symbols.
func (p *FunctionSources) Exports(pkg *types.Package) iter.Seq2[string, string] {
	p.mu.RLock()
	names := p.functionNames[pkg]
	p.mu.RUnlock()
	return func(yield func(string, string) bool) {
		for name, info := range names {
			if info.ExportName != "" && !yield(name, info.ExportName) {
				return
			}
		}
	}
}
