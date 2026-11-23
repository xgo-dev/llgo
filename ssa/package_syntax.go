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
	"sync"
	"sync/atomic"
)

// packageSyntaxData is Go-owned metadata collected before LLVM package
// lowering. Build creates backend Programs only after this data is complete,
// so those Programs can share it directly for concurrent read-only access.
// One-shot compiler users keep the same Program-local mutation behavior.
type packageSyntaxData struct {
	mu                    sync.RWMutex
	hasNativeTypeMetadata atomic.Bool
	linknames             map[string]string
	wasmImports           map[string]wasmImport
	exports               map[string]string
	closureEnvDirectives  map[closureEnvDirectiveKey]none
	parsedPackages        map[*types.Package]struct{}
	noInterface           map[string]none
	typeCallConvs         map[string]CallConv
	nativeTypes           []nativeTypeDecl
	nativeStructFields    map[token.Pos]bool
	nativeStructVars      map[*types.Var]bool
}

func newPackageSyntaxData() *packageSyntaxData {
	return &packageSyntaxData{
		linknames:            make(map[string]string),
		wasmImports:          make(map[string]wasmImport),
		exports:              make(map[string]string),
		closureEnvDirectives: make(map[closureEnvDirectiveKey]none),
		parsedPackages:       make(map[*types.Package]struct{}),
		noInterface:          make(map[string]none),
		typeCallConvs:        make(map[string]CallConv),
		nativeStructFields:   make(map[token.Pos]bool),
		nativeStructVars:     make(map[*types.Var]bool),
	}
}

type wasmImport struct {
	module string
	name   string
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

type nativeTypeDecl struct {
	pkg  *types.Package
	name string
}

// SetNativeTypeDecl records a source declaration before its underlying type is known.
func (p Program) SetNativeTypeDecl(pkg *types.Package, name string, cc CallConv) {
	syntax := p.packageSyntax
	syntax.mu.Lock()
	defer syntax.mu.Unlock()
	syntax.setTypeCallConv(pkg.Path()+"."+name, cc, nil)
	syntax.nativeTypes = append(syntax.nativeTypes, nativeTypeDecl{pkg, name})
}

// HasNativeFuncTypes reports whether a native callback can reach reflection.
// With executable roots, RTA identifies the named function types. An alias can
// share its Go identity with an ordinary signature, so keep aliases conservative.
func (p Program) HasNativeFuncTypes(visible map[*types.TypeName]bool) bool {
	syntax := p.packageSyntax
	syntax.mu.RLock()
	defer syntax.mu.RUnlock()
	for _, decl := range syntax.nativeTypes {
		obj, ok := decl.pkg.Scope().Lookup(decl.name).(*types.TypeName)
		if !ok || visible != nil && !obj.IsAlias() && !visible[obj] {
			continue
		}
		if _, ok := obj.Type().Underlying().(*types.Signature); ok {
			return true
		}
	}
	return false
}

// The caller holds p.mu while registering the declaration.
func (p *packageSyntaxData) setTypeCallConv(name string, cc CallConv, fields []*types.Var) {
	p.typeCallConvs[name] = cc
	if isNativeCallConv(cc) {
		if len(fields) != 0 {
			p.nativeStructVars[fields[0]] = true
		}
		p.hasNativeTypeMetadata.Store(true)
	}
}

func (p *packageSyntaxData) typeCallConv(name string) (CallConv, bool) {
	p.mu.RLock()
	background, ok := p.typeCallConvs[name]
	p.mu.RUnlock()
	return background, ok
}
