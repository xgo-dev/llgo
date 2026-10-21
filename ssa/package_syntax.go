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
	"go/types"
	"sync"
)

// packageSyntaxData is Go-owned metadata collected before LLVM package
// lowering. Build creates backend Programs only after this data is complete,
// so those Programs can share it directly for concurrent read-only access.
// One-shot compiler users keep the same Program-local mutation behavior.
type packageSyntaxData struct {
	mu                   sync.RWMutex
	linknames            map[string]string
	linknameTargets      map[string]int
	linkDirectives       map[string]string
	nativeExports        map[string]NativeExportInfo
	externalImports      map[string]none
	wasmImports          map[string]wasmImport
	exports              map[string]string
	closureEnvDirectives map[closureEnvDirectiveKey]none
	parsedPackages       map[*types.Package]struct{}
	noInterface          map[string]none
	typeCallConvs        map[string]CallConv
}

func newPackageSyntaxData() *packageSyntaxData {
	return &packageSyntaxData{
		linknames:            make(map[string]string),
		linknameTargets:      make(map[string]int),
		linkDirectives:       make(map[string]string),
		nativeExports:        make(map[string]NativeExportInfo),
		externalImports:      make(map[string]none),
		wasmImports:          make(map[string]wasmImport),
		exports:              make(map[string]string),
		closureEnvDirectives: make(map[closureEnvDirectiveKey]none),
		parsedPackages:       make(map[*types.Package]struct{}),
		noInterface:          make(map[string]none),
		typeCallConvs:        make(map[string]CallConv),
	}
}

type wasmImport struct {
	module string
	name   string
}

// SetExternalImport records the literal local symbol of a cgo import pragma.
// Imports can name ordinary package variables without any linkname directive.
// The namespace is build-wide: gc also looks up cgo locals by literal linker
// symbol (version 0), so an import can replace eligible BSS in another package.
func (p Program) SetExternalImport(name string) {
	p.packageSyntax.mu.Lock()
	p.packageSyntax.externalImports[name] = none{}
	p.packageSyntax.mu.Unlock()
}

func (p Program) IsExternalImport(name string) bool {
	p.packageSyntax.mu.RLock()
	_, ok := p.packageSyntax.externalImports[name]
	p.packageSyntax.mu.RUnlock()
	return ok
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

// SetLinknameDirective retains the source form: llgo:link bodies are binding
// placeholders, whereas go:linkname bodies are actual definitions.
func (p Program) SetLinknameDirective(name, target, directive string) {
	p.packageSyntax.mu.Lock()
	p.packageSyntax.setLinkname(name, target)
	p.packageSyntax.linkDirectives[name] = directive
	p.packageSyntax.mu.Unlock()
}

func (p Program) LinknameDirective(name string) string {
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	return p.packageSyntax.linkDirectives[name]
}

type NativeExportInfo struct {
	CallConv CallConv
}

func (p Program) SetNativeExport(name string, info NativeExportInfo) {
	p.packageSyntax.mu.Lock()
	p.packageSyntax.nativeExports[name] = info
	p.packageSyntax.mu.Unlock()
}

// NativeExport reports an explicit native go:linkname definition's entry ABI.
func (p Program) NativeExport(name string) (NativeExportInfo, bool) {
	p.packageSyntax.mu.RLock()
	cc, ok := p.packageSyntax.nativeExports[name]
	p.packageSyntax.mu.RUnlock()
	return cc, ok
}

// setLinkname updates both indexes while the caller holds mu. Counts preserve a
// target shared by multiple declarations when one declaration is rebound.
func (p *packageSyntaxData) setLinkname(name, target string) {
	if previous, ok := p.linknames[name]; ok {
		p.linknameTargets[previous]--
		if p.linknameTargets[previous] == 0 {
			delete(p.linknameTargets, previous)
		}
	}
	p.linknames[name] = target
	p.linknameTargets[target]++
}
