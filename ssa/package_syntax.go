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
	"iter"
	"maps"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/xgo-dev/llgo/internal/directive"
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
	closureEnvDirectives  map[closureEnvDirectiveKey]none
	functions             directive.FunctionIndex
	preparedPackages      map[*types.Package]preparedPackage
	noInterface           map[string]none
	typeCallConvs         map[string]CallConv
	nativeStructFields    map[token.Pos]bool
	nativeStructVars      map[*types.Var]bool
}

func newPackageSyntaxData() *packageSyntaxData {
	return &packageSyntaxData{
		linknames:            make(map[string]string),
		wasmImports:          make(map[string]wasmImport),
		closureEnvDirectives: make(map[closureEnvDirectiveKey]none),
		preparedPackages:     make(map[*types.Package]preparedPackage),
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

// preparedPackage is an immutable snapshot published after syntax preparation.
// Its presence in preparedPackages also records packages with no exports.
type preparedPackage struct {
	exports []exportBinding
}

type exportBinding struct {
	name   string
	export string
}

// PackageSyntaxParsed reports whether this concrete package has prepared metadata.
func (p Program) PackageSyntaxParsed(pkg *types.Package) bool {
	p.packageSyntax.mu.RLock()
	_, ok := p.packageSyntax.preparedPackages[pkg]
	p.packageSyntax.mu.RUnlock()
	return ok
}

// MarkPackageSyntaxParsed publishes the package's completed source preparation.
// It sorts the caller's bindings into a deterministic snapshot independent of
// the caller's map. Duplicate names have already been resolved by the caller.
func (p Program) MarkPackageSyntaxParsed(pkg *types.Package, exports map[string]string) {
	prepared := preparedPackage{}
	if len(exports) != 0 {
		prepared.exports = make([]exportBinding, 0, len(exports))
		for _, name := range slices.Sorted(maps.Keys(exports)) {
			prepared.exports = append(prepared.exports, exportBinding{name, exports[name]})
		}
	}
	p.packageSyntax.mu.Lock()
	p.packageSyntax.preparedPackages[pkg] = prepared
	p.packageSyntax.mu.Unlock()
}

// PackageExports iterates the prepared exports of this concrete package.
// The snapshot distinguishes packages with the same path, including an
// explicitly prepared package with no exports.
func (p Program) PackageExports(pkg *types.Package) iter.Seq2[string, string] {
	p.packageSyntax.mu.RLock()
	exports := p.packageSyntax.preparedPackages[pkg].exports
	p.packageSyntax.mu.RUnlock()
	return func(yield func(string, string) bool) {
		for _, binding := range exports {
			if !yield(binding.name, binding.export) {
				return
			}
		}
	}
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
