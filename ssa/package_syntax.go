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
	"fmt"
	"go/types"
	"sync"
)

// PackageSyntaxState is immutable Go-side metadata collected while package
// syntax is preloaded. It contains no LLVM objects and can be shared directly
// by Programs with independent LLVM contexts.
type PackageSyntaxState struct {
	data *packageSyntaxData
}

type packageSyntaxData struct {
	mu                   sync.RWMutex
	frozen               bool
	linknames            map[string]string
	exports              map[string]string
	closureEnvDirectives map[closureEnvDirectiveKey]none
	parsedPackages       map[*types.Package]struct{}
	noInterface          map[string]none
	typeBackgrounds      map[string]Background
}

func newPackageSyntaxData() *packageSyntaxData {
	return &packageSyntaxData{
		linknames:            make(map[string]string),
		exports:              make(map[string]string),
		closureEnvDirectives: make(map[closureEnvDirectiveKey]none),
		parsedPackages:       make(map[*types.Package]struct{}),
		noInterface:          make(map[string]none),
		typeBackgrounds:      make(map[string]Background),
	}
}

// FreezePackageSyntaxState freezes p's preloaded package metadata and returns
// the same immutable state for workers to share without copying.
func (p Program) FreezePackageSyntaxState() PackageSyntaxState {
	p.packageSyntax.mu.Lock()
	p.packageSyntax.frozen = true
	p.packageSyntax.mu.Unlock()
	return PackageSyntaxState{data: p.packageSyntax}
}

// UsePackageSyntaxState replaces p's package metadata with shared immutable
// state. A worker must not discover syntax that the coordinator did not preload.
func (p Program) UsePackageSyntaxState(state PackageSyntaxState) {
	if state.data == nil {
		state.data = newPackageSyntaxData()
		state.data.frozen = true
	}
	p.packageSyntax = state.data
	p.gocvt.packageSyntax = state.data
}

// SetPackageExport records an export directive independently of the LLVM
// Package on which it will later be applied.
func (p Program) SetPackageExport(name, export string) {
	p.packageSyntax.setString(p.packageSyntax.exports, name, export, "package export")
}

// PackageExport returns the export directive collected for name.
func (p Program) PackageExport(name string) (string, bool) {
	p.packageSyntax.mu.RLock()
	export, ok := p.packageSyntax.exports[name]
	p.packageSyntax.mu.RUnlock()
	return export, ok
}

func (p *packageSyntaxData) setLinkname(name, link string) {
	p.setString(p.linknames, name, link, "linkname")
}

func (p *packageSyntaxData) linknameOf(name string) (string, bool) {
	p.mu.RLock()
	link, ok := p.linknames[name]
	p.mu.RUnlock()
	return link, ok
}

func (p *packageSyntaxData) linknamesSnapshot() map[string]string {
	p.mu.RLock()
	links := make(map[string]string, len(p.linknames))
	copyMap(links, p.linknames)
	p.mu.RUnlock()
	return links
}

func (p *packageSyntaxData) setNoInterface(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.noInterface[name]; ok {
		return
	}
	p.assertMutable("nointerface", name)
	p.noInterface[name] = none{}
}

func (p *packageSyntaxData) noInterfaceMethod(name string) (none, bool) {
	p.mu.RLock()
	value, ok := p.noInterface[name]
	p.mu.RUnlock()
	return value, ok
}

func (p *packageSyntaxData) setClosureEnvDirective(key closureEnvDirectiveKey) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.closureEnvDirectives[key]; ok {
		return
	}
	p.assertMutable("closure environment directive", key.name)
	p.closureEnvDirectives[key] = none{}
}

func (p *packageSyntaxData) hasClosureEnvDirective(key closureEnvDirectiveKey) bool {
	p.mu.RLock()
	_, ok := p.closureEnvDirectives[key]
	p.mu.RUnlock()
	return ok
}

func (p *packageSyntaxData) markPackageParsed(pkg *types.Package) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.parsedPackages[pkg]; ok {
		return
	}
	p.assertMutable("parsed package", pkg.Path())
	p.parsedPackages[pkg] = struct{}{}
}

func (p *packageSyntaxData) packageParsed(pkg *types.Package) bool {
	p.mu.RLock()
	_, ok := p.parsedPackages[pkg]
	p.mu.RUnlock()
	return ok
}

func (p *packageSyntaxData) setTypeBackground(name string, bg Background) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if current, ok := p.typeBackgrounds[name]; ok && current == bg {
		return
	}
	p.assertMutable("type background", name)
	p.typeBackgrounds[name] = bg
}

func (p *packageSyntaxData) typeBackground(name string) (Background, bool) {
	p.mu.RLock()
	bg, ok := p.typeBackgrounds[name]
	p.mu.RUnlock()
	return bg, ok
}

func (p *packageSyntaxData) setString(dst map[string]string, key, value, kind string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if current, ok := dst[key]; ok && current == value {
		return
	}
	p.assertMutable(kind, key)
	dst[key] = value
}

func (p *packageSyntaxData) assertMutable(kind, name string) {
	if p.frozen {
		panic(fmt.Sprintf("cannot add %s %q to frozen package syntax state", kind, name))
	}
}

func copyMap[K comparable, V any](dst, src map[K]V) {
	for key, value := range src {
		dst[key] = value
	}
}
