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

// functionSources holds prepared function values. Source preparation publishes
// the AST and concrete-package indexes before any function wrapper is used.
// It contains no LLVM state and is shared by isolated package backends.
type functionSources struct {
	mu            sync.RWMutex
	functions     map[*ast.FuncDecl]*FunctionDirectives
	functionNames map[*types.Package]map[string]*FunctionDirectives
}

func newFunctionSources() *functionSources {
	return &functionSources{
		functions:     make(map[*ast.FuncDecl]*FunctionDirectives),
		functionNames: make(map[*types.Package]map[string]*FunctionDirectives),
	}
}

// set publishes one package's function declarations. names
// selects the effective declaration when source patches contain replacements.
// Both indexes point to the same immutable values; names is used only at the
// call-binding and types.Func boundaries, never to select a function body.
func (p *functionSources) set(pkg *types.Package, declarations map[*ast.FuncDecl]*FunctionDirectives, names map[string]*FunctionDirectives) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for decl, info := range declarations {
		p.functions[decl] = info
	}
	p.functionNames[pkg] = names
}

// declaration returns the exact source declaration's snapshot.
func (p *functionSources) declaration(decl *ast.FuncDecl) (FunctionDirectives, bool) {
	p.mu.RLock()
	info, ok := p.functions[decl]
	p.mu.RUnlock()
	if !ok {
		return FunctionDirectives{}, false
	}
	return *info, true
}

// binding returns the selected declaration within a concrete package
// view. The second result distinguishes unprepared packages from known absence.
func (p *functionSources) binding(pkg *types.Package, name string) (FunctionDirectives, bool) {
	p.mu.RLock()
	names, prepared := p.functionNames[pkg]
	info := names[name]
	p.mu.RUnlock()
	if info == nil {
		return FunctionDirectives{}, prepared
	}
	return *info, prepared
}

// Exports iterates prepared function exports for a concrete package view.
// Consumers do not need to revisit declarations to preserve these symbols.
func (p *Functions) Exports(pkg *types.Package) iter.Seq2[string, string] {
	p.sources.mu.RLock()
	names := p.sources.functionNames[p.effectivePackage(pkg)]
	p.sources.mu.RUnlock()
	return func(yield func(string, string) bool) {
		for name, info := range names {
			if info.ExportName != "" && !yield(name, info.ExportName) {
				return
			}
		}
	}
}
