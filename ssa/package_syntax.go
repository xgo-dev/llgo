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
	"sort"
	"strings"
	"sync"

	"github.com/xgo-dev/llgo/internal/directive"
)

// packageSyntaxData is Go-owned metadata collected before LLVM package
// lowering. Build creates backend Programs only after this data is complete,
// so those Programs can share it directly for concurrent read-only access.
// One-shot compiler users keep the same Program-local mutation behavior.
type packageSyntaxData struct {
	mu                   sync.RWMutex
	source               directive.Index
	declarations         map[*types.Package]*directive.Package
	effective            map[*types.Package]*types.Package
	linknames            map[string]string
	wasmImports          map[string]wasmImport
	exports              map[string]string
	closureEnvDirectives map[closureEnvDirectiveKey]none
	parsedPackages       map[*types.Package]struct{}
	noInterface          map[string]none
	typeBackgrounds      map[string]Background
}

func newPackageSyntaxData() *packageSyntaxData {
	return &packageSyntaxData{
		declarations:         make(map[*types.Package]*directive.Package),
		effective:            make(map[*types.Package]*types.Package),
		linknames:            make(map[string]string),
		wasmImports:          make(map[string]wasmImport),
		exports:              make(map[string]string),
		closureEnvDirectives: make(map[closureEnvDirectiveKey]none),
		parsedPackages:       make(map[*types.Package]struct{}),
		noInterface:          make(map[string]none),
		typeBackgrounds:      make(map[string]Background),
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

func (p Program) packageTypeBackground(name string) (Background, bool) {
	p.packageSyntax.mu.RLock()
	background, ok := p.packageSyntax.typeBackgrounds[name]
	p.packageSyntax.mu.RUnlock()
	return background, ok
}

// Directives owns the source snapshots shared by coordinator and backends.
func (p Program) Directives() *directive.Index { return &p.packageSyntax.source }
func (p Program) SetPackageDirectives(pkg *types.Package, records *directive.Package) {
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	p.packageSyntax.declarations[pkg] = records
}
func (p Program) PackageDirectives(pkg *types.Package) *directive.Package {
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	return p.packageSyntax.declarations[pkg]
}

// SetDirectivePackage records the effective patch view for object-based type
// and symbol queries. Function source properties retain their source identity.
func (p Program) SetDirectivePackage(original, effective *types.Package) {
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	p.packageSyntax.effective[original] = effective
}
func (p Program) effectivePackageDirectives(pkg *types.Package) *directive.Package {
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	if effective := p.packageSyntax.effective[pkg]; effective != nil {
		pkg = effective
	}
	return p.packageSyntax.declarations[pkg]
}

// FunctionDeclaration supports source functions, imported objects and generic
// origins without deriving declaration identity from a linker symbol.
func (p Program) FunctionDeclaration(pkg *types.Package, obj *types.Func, syntax *ast.FuncDecl) *directive.FunctionDecl {
	if obj != nil {
		obj = obj.Origin()
	}
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	if r := p.packageSyntax.declarations[pkg]; r != nil {
		if d, ok := r.Objects[obj].(*directive.FunctionDecl); ok {
			return d
		}
		if d := r.Functions[syntax]; d != nil {
			return d
		}
	}
	// Patched functions may retain their original types.Object package.
	if obj != nil && obj.Pkg() != pkg {
		if r := p.packageSyntax.declarations[obj.Pkg()]; r != nil {
			if d, ok := r.Objects[obj].(*directive.FunctionDecl); ok {
				return d
			}
		}
	}
	return nil
}

// LinknameFor resolves a source declaration in its owning package instance.
// Package records are authoritative, including absent declarations and links:
// a global entry may belong to another package instance with the same path.
// Only packages without records fall back to the global name index.
func (p Program) LinknameFor(pkg *types.Package, obj types.Object, fullName string) (string, bool) {
	if fn, ok := obj.(*types.Func); ok {
		obj = fn.Origin()
	}
	if r := p.effectivePackageDirectives(pkg); r != nil {
		rec := r.Objects[obj]
		if rec == nil {
			rec = r.Names[strings.TrimPrefix(fullName, PathOf(pkg)+".")]
		}
		switch d := rec.(type) {
		case *directive.FunctionDecl:
			return d.Linkname, d.HasLinkname
		case *directive.VariableDecl:
			return d.Linkname, d.HasLinkname
		}
		return "", false
	}
	return p.Linkname(fullName)
}

// namedBackground uses the effective package's records, including an absent
// type or background. Only packages without records use typeBackgrounds, which
// cannot distinguish package instances sharing an import path.
func (p *packageSyntaxData) namedBackground(t *types.Named) (Background, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	obj := t.Origin().Obj()
	pkg := obj.Pkg()
	if effective := p.effective[pkg]; effective != nil {
		pkg = effective
	}
	if r := p.declarations[pkg]; r != nil {
		rec := r.Objects[obj]
		if rec == nil {
			rec = r.Names[obj.Name()]
		}
		if d, ok := rec.(*directive.TypeDecl); ok && d.Background != "" {
			switch d.Background {
			case "C":
				return InC, true
			case "stdcall":
				return InStdcall, true
			default:
				return InGo, true
			}
		}
		return InGo, false
	}
	bg, ok := p.typeBackgrounds[namedLinkname(t)]
	return bg, ok
}

// ValidateDirectiveContracts checks concrete signatures and known link aliases
// before workers start. It compares declarations without merging their records.
func (p Program) ValidateDirectiveContracts(fset *token.FileSet) error {
	type declaration struct {
		symbol string
		pos    token.Pos
		attrs  []directive.Attribute
		sig    *types.Signature
	}
	var declarations []declaration
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	for pkg, records := range p.packageSyntax.declarations {
		if effective := p.packageSyntax.effective[pkg]; effective != nil && effective != pkg {
			continue
		}
		for obj, raw := range records.Objects {
			fn, ok := obj.(*types.Func)
			if !ok {
				continue
			}
			rec, ok := raw.(*directive.FunctionDecl)
			if !ok {
				continue
			}
			props := rec.Function.WithPositions(fset)
			if props.ContractError != nil {
				return props.ContractError
			}
			symbol := FullName(pkg, rec.Name)
			if rec.HasLinkname {
				symbol = rec.Linkname
			}
			seen := make(map[string]bool)
			for !seen[symbol] {
				seen[symbol] = true
				next, ok := p.packageSyntax.linknames[symbol]
				if !ok {
					break
				}
				symbol = next
			}
			declarations = append(declarations, declaration{symbol, rec.Pos, props.Values, fn.Type().(*types.Signature)})
		}
	}
	sort.Slice(declarations, func(i, j int) bool {
		if declarations[i].symbol != declarations[j].symbol {
			return declarations[i].symbol < declarations[j].symbol
		}
		return declarations[i].pos < declarations[j].pos
	})
	known := make(map[string][]directive.Attribute)
	for _, d := range declarations {
		if err := directive.Validate(d.attrs, d.sig, p.Int().ll.IntTypeWidth(), true); err != nil {
			return err
		}
		merged, err := directive.Merge(known[d.symbol], d.attrs)
		if err != nil {
			return err
		}
		known[d.symbol] = merged
	}
	return nil
}
