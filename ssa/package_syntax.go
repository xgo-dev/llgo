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
	"sync"
	"sync/atomic"
)

// packageSyntaxData is Go-owned metadata collected before LLVM package
// lowering. Build creates backend Programs only after this data is complete,
// so those Programs can share it directly for concurrent read-only access.
// One-shot compiler users keep the same Program-local mutation behavior.
type packageSyntaxData struct {
	functions             map[*ast.FuncDecl]*FunctionDirectives
	functionNames         map[*types.Package]map[string]*FunctionDirectives
	mu                    sync.RWMutex
	hasNativeTypeMetadata atomic.Bool
	linknames             map[string]string
	exports               map[string]string
	parsedPackages        map[*types.Package]struct{}
	noInterface           map[string]none
	typeCallConvs         map[string]CallConv
	nativeStructFields    map[token.Pos]bool
	nativeStructVars      map[*types.Var]bool
}

func newPackageSyntaxData() *packageSyntaxData {
	return &packageSyntaxData{
		functions:          make(map[*ast.FuncDecl]*FunctionDirectives),
		functionNames:      make(map[*types.Package]map[string]*FunctionDirectives),
		linknames:          make(map[string]string),
		exports:            make(map[string]string),
		parsedPackages:     make(map[*types.Package]struct{}),
		noInterface:        make(map[string]none),
		typeCallConvs:      make(map[string]CallConv),
		nativeStructFields: make(map[token.Pos]bool),
		nativeStructVars:   make(map[*types.Var]bool),
	}
}

// SetPackageExport records an export directive before the LLVM Package that
// will later preserve the symbol exists.
func (p Program) SetPackageExport(name, export string) {
	p.packageSyntax.mu.Lock()
	p.packageSyntax.exports[name] = export
	p.packageSyntax.mu.Unlock()
}

// PackageExport returns the preloaded export name for name.
func (p Program) PackageExport(name string) (string, bool) {
	p.packageSyntax.mu.RLock()
	export, ok := p.packageSyntax.exports[name]
	p.packageSyntax.mu.RUnlock()
	return export, ok
}

func (p Program) packageSyntaxParsed(pkg *types.Package) bool {
	p.packageSyntax.mu.RLock()
	_, ok := p.packageSyntax.parsedPackages[pkg]
	p.packageSyntax.mu.RUnlock()
	return ok
}

func (p Program) markPackageSyntaxParsed(pkg *types.Package) {
	p.packageSyntax.mu.Lock()
	p.packageSyntax.parsedPackages[pkg] = struct{}{}
	p.packageSyntax.mu.Unlock()
}

func (p Program) packageTypeCallConv(name string) (CallConv, bool) {
	p.packageSyntax.mu.RLock()
	background, ok := p.packageSyntax.typeCallConvs[name]
	p.packageSyntax.mu.RUnlock()
	return background, ok
}

func (p *packageSyntaxData) typeCallConv(name string) (CallConv, bool) {
	p.mu.RLock()
	background, ok := p.typeCallConvs[name]
	p.mu.RUnlock()
	return background, ok
}
