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
	"sort"
	"strings"
	"sync"

	"github.com/goplus/llgo/internal/locality"
	localitylayout "github.com/goplus/llgo/internal/locality/layout"
)

type Locality = locality.Kind

const (
	LocalityNone   = locality.None
	ThreadLocal    = locality.Thread
	GoroutineLocal = locality.Goroutine
)

type LocalityInfo = locality.Info
type LocalStorage = localitylayout.Storage

const (
	LocalStorageUnknown   = localitylayout.StorageUnknown
	LocalStorageNativeTLS = localitylayout.StorageNativeTLS
	LocalStoragePackage   = localitylayout.StoragePackage
)

// VariableLocality is the locality metadata attached to one package variable.
type VariableLocality struct {
	LocalStorage LocalStorage
	locality.Info
}

type localityInfos struct {
	mu     sync.RWMutex
	frozen bool
	// entries and ownerlessEntries retain the canonical-only compatibility
	// view. Production declaration handling uses declarationEntries instead.
	entries            map[string]VariableLocality
	ownerlessEntries   map[string]VariableLocality
	declarationEntries map[string]map[string]VariableLocality
	activePackages     map[string]struct{}
}

// LocalityState is an immutable handle to Go-side locality metadata. It
// contains no LLVM objects and can be shared by independent backend Programs.
type LocalityState struct {
	infos *localityInfos
}

func newLocalityInfos() *localityInfos {
	return &localityInfos{
		entries:            make(map[string]VariableLocality),
		ownerlessEntries:   make(map[string]VariableLocality),
		declarationEntries: make(map[string]map[string]VariableLocality),
		activePackages:     make(map[string]struct{}),
	}
}

// FreezeLocalityState freezes p's locality metadata and returns it without
// copying. Package syntax parse markers remain local to each Program.
func (p Program) FreezeLocalityState() LocalityState {
	p.localities.mu.Lock()
	p.localities.frozen = true
	p.localities.mu.Unlock()
	return LocalityState{infos: p.localities}
}

// UseLocalityState replaces p's locality metadata with shared immutable state.
func (p Program) UseLocalityState(state LocalityState) {
	if state.infos == nil {
		state = LocalityState{infos: newLocalityInfos()}
		state.infos.frozen = true
	}
	p.localities = state.infos
}

func (p *localityInfos) update(name string, update func(*VariableLocality)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.assertMutable()
	info := p.entries[name]
	update(&info)
	p.entries[name] = info
	ownerless := p.ownerlessEntries[name]
	update(&ownerless)
	p.ownerlessEntries[name] = ownerless
}

func (p *localityInfos) updateFor(pkg *types.Package, name string, update func(*VariableLocality)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.assertMutable()
	entries := p.declarationEntries[name]
	if entries == nil {
		entries = make(map[string]VariableLocality)
		p.declarationEntries[name] = entries
	}
	owner := pkg.Path()
	info := entries[owner]
	update(&info)
	entries[owner] = info
	p.entries[name] = info
}

func (p *localityInfos) assertMutable() {
	if p.frozen {
		panic("cannot modify frozen locality state")
	}
}

func (p Program) SetLocalityInfo(name string, info LocalityInfo) {
	p.localities.update(name, func(current *VariableLocality) { current.Info = info })
}

// SetLocalityInfoFor updates locality metadata for one concrete package
// declaration. It preserves distinct metadata for standard and alternate
// packages whose canonical symbol names are identical.
func (p Program) SetLocalityInfoFor(pkg *types.Package, name string, info LocalityInfo) {
	p.localities.updateFor(pkg, name, func(current *VariableLocality) { current.Info = info })
}

// DeclareLocality records locality metadata found on a declaration in pkg.
// Alternate packages can share canonical symbol names with the packages they
// replace, so both ownership and metadata retain the raw import path. Repeated
// loads of the same raw path preserve metadata enriched during preparation.
func (p Program) DeclareLocality(pkg *types.Package, name string, info LocalityInfo) {
	fullName := FullName(pkg, name)
	owner := pkg.Path()
	p.localities.mu.Lock()
	defer p.localities.mu.Unlock()
	entries := p.localities.declarationEntries[fullName]
	if entries != nil {
		if _, exists := entries[owner]; exists {
			return
		}
	}
	p.localities.assertMutable()
	if entries == nil {
		entries = make(map[string]VariableLocality)
		p.localities.declarationEntries[fullName] = entries
	}
	if _, exists := entries[owner]; !exists {
		current := VariableLocality{Info: info}
		entries[owner] = current
		p.localities.entries[fullName] = current
	}
}

func (p Program) SetLocalStorage(name string, storage LocalStorage) {
	p.localities.update(name, func(info *VariableLocality) { info.LocalStorage = storage })
}

// SetLocalStorageFor records the selected storage for one concrete package
// declaration.
func (p Program) SetLocalStorageFor(pkg *types.Package, name string, storage LocalStorage) {
	p.localities.updateFor(pkg, name, func(info *VariableLocality) { info.LocalStorage = storage })
}

// ActivateLocalitiesFor marks pkg's concrete import path as part of the
// effective build graph. Alternate packages are scanned and prepared before
// link reachability is known; their declarations must not require a
// LocalContext merely because metadata exists.
func (p Program) ActivateLocalitiesFor(pkg *types.Package) {
	if pkg == nil {
		return
	}
	p.localities.mu.Lock()
	defer p.localities.mu.Unlock()
	p.localities.assertMutable()
	p.localities.activePackages[pkg.Path()] = struct{}{}
}

// VariableLocality returns the legacy canonical-only metadata view. Its result
// is unspecified when multiple declaration owners share name; use
// VariableLocalityFor in owner-aware code.
func (p Program) VariableLocality(name string) (VariableLocality, bool) {
	p.localities.mu.RLock()
	info, ok := p.localities.entries[name]
	p.localities.mu.RUnlock()
	return info, ok
}

// VariableLocalityFor returns metadata for a concrete declaration, falling
// back to owner-less preloaded metadata by canonical symbol name.
func (p Program) VariableLocalityFor(pkg *types.Package, name string) (VariableLocality, bool) {
	p.localities.mu.RLock()
	if entries := p.localities.declarationEntries[name]; entries != nil {
		if info, ok := entries[pkg.Path()]; ok {
			p.localities.mu.RUnlock()
			return info, true
		}
	}
	info, ok := p.localities.ownerlessEntries[name]
	p.localities.mu.RUnlock()
	return info, ok
}

// ResolveLocality returns locality metadata for one declaration. Locality
// variables cannot participate in go:linkname alias chains.
func (p Program) ResolveLocality(name string) (string, VariableLocality, bool, error) {
	lookup := func(name string) (VariableLocality, bool) {
		p.localities.mu.RLock()
		info, ok := p.localities.entries[name]
		p.localities.mu.RUnlock()
		return info, ok
	}
	return resolveLocality(lookup, p.Linkname, name)
}

// ResolveLocalityFor resolves locality metadata using the concrete package's
// declaration when the canonical package path has multiple owners. Owner-less
// preloaded metadata remains a canonical-name compatibility fallback.
func (p Program) ResolveLocalityFor(pkg *types.Package, name string) (string, VariableLocality, bool, error) {
	prefix := PathOf(pkg) + "."
	lookup := func(name string) (VariableLocality, bool) {
		p.localities.mu.RLock()
		defer p.localities.mu.RUnlock()
		if strings.HasPrefix(name, prefix) {
			if entries := p.localities.declarationEntries[name]; entries != nil {
				if info, ok := entries[pkg.Path()]; ok {
					return info, true
				}
			}
			info, ok := p.localities.ownerlessEntries[name]
			return info, ok
		}
		info, ok := p.localities.entries[name]
		return info, ok
	}
	return resolveLocality(lookup, p.Linkname, name)
}

func resolveLocality(lookup func(string) (VariableLocality, bool), linkname func(string) (string, bool), name string) (string, VariableLocality, bool, error) {
	result, ok := lookup(name)
	if !ok {
		result = VariableLocality{}
	}
	var seen map[string]bool
	current := name
	for {
		target, hasLink := linkname(current)
		target = strings.TrimPrefix(target, "go:")
		if !hasLink || target == "" {
			return current, result, ok, nil
		}
		if seen == nil {
			seen = make(map[string]bool)
		}
		if seen[current] {
			return "", VariableLocality{}, false, fmt.Errorf("declaration linkname cycle involving %s", current)
		}
		seen[current] = true
		if currentInfo, exists := lookup(current); exists && currentInfo.Locality != locality.None {
			return "", VariableLocality{}, false, fmt.Errorf("local variable %s cannot use go:linkname", current)
		}
		if targetInfo, exists := lookup(target); exists && targetInfo.Locality != locality.None {
			return "", VariableLocality{}, false, fmt.Errorf("go:linkname alias %s cannot reference local variable %s", name, target)
		}
		if target == current {
			return current, result, ok, nil
		}
		current = target
	}
}

func hasInitialization(info locality.Info) bool {
	return info.HasInitializer || info.InitFunc != "" || info.InitOrder != 0
}

// ValidateLocalities validates the legacy canonical-only metadata view. Its
// result is unspecified when multiple declaration owners share a canonical
// path; use ValidateLocalitiesFor in owner-aware code.
func (p Program) ValidateLocalities(pkgPath string) error {
	return p.validateLocalities(pkgPath, p.PackageLocalities(pkgPath), p.ResolveLocality)
}

// ValidateLocalitiesFor validates only declarations that belong to pkg, plus
// owner-less preloaded metadata for its canonical package path.
func (p Program) ValidateLocalitiesFor(pkg *types.Package) error {
	return p.validateLocalities(PathOf(pkg), p.PackageLocalitiesFor(pkg), func(name string) (string, VariableLocality, bool, error) {
		return p.ResolveLocalityFor(pkg, name)
	})
}

func (p Program) validateLocalities(pkgPath string, packageEntries map[string]VariableLocality, resolve func(string) (string, VariableLocality, bool, error)) error {
	prefix := pkgPath + "."
	p.localities.mu.RLock()
	if len(p.localities.entries) == 0 && len(p.localities.declarationEntries) == 0 {
		p.localities.mu.RUnlock()
		return nil
	}
	nameSet := make(map[string]bool)
	localNames := make(map[string]bool)
	for name, info := range p.localities.entries {
		if info.Locality != locality.None {
			localNames[name] = true
		}
	}
	for name, entries := range p.localities.declarationEntries {
		for _, info := range entries {
			if info.Locality != locality.None {
				localNames[name] = true
				break
			}
		}
	}
	p.localities.mu.RUnlock()
	for name := range packageEntries {
		nameSet[name] = true
	}
	if len(localNames) == 0 {
		return nil
	}
	p.linknameMu.RLock()
	links := make(map[string]string, len(p.linkname))
	for name, target := range p.linkname {
		links[name] = strings.TrimPrefix(target, "go:")
	}
	p.linknameMu.RUnlock()
	for name := range links {
		if strings.HasPrefix(name, prefix) && linknameReachesLocal(name, links, localNames) {
			nameSet[name] = true
		}
	}
	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, _, _, err := resolve(name); err != nil {
			return err
		}
	}
	return nil
}

func linknameReachesLocal(name string, links map[string]string, localNames map[string]bool) bool {
	seen := make(map[string]bool)
	for name != "" && !seen[name] {
		if localNames[name] {
			return true
		}
		seen[name] = true
		name = links[name]
	}
	return false
}

func (p Program) PackageSyntaxParsed(pkg *types.Package) bool {
	p.parsedPackagesMu.RLock()
	_, ok := p.parsedPackages[pkg]
	p.parsedPackagesMu.RUnlock()
	return ok
}

func (p Program) MarkPackageSyntaxParsed(pkg *types.Package) {
	p.parsedPackagesMu.Lock()
	p.parsedPackages[pkg] = struct{}{}
	p.parsedPackagesMu.Unlock()
}

// PackageLocalities returns the legacy canonical-only metadata view. Its
// result is unspecified when a canonical path has multiple declaration
// owners; use PackageLocalitiesFor in owner-aware code.
func (p Program) PackageLocalities(pkgPath string) map[string]VariableLocality {
	prefix := pkgPath + "."
	ret := make(map[string]VariableLocality)
	p.localities.mu.RLock()
	for name, info := range p.localities.entries {
		if info.Locality != locality.None && strings.HasPrefix(name, prefix) {
			ret[name] = info
		}
	}
	p.localities.mu.RUnlock()
	return ret
}

// PackageLocalitiesFor returns locality metadata applicable to the concrete
// package. Declaration entries retain both their owner and owner-specific
// metadata. Entries with no declaration owner remain applicable by canonical
// package path for compatibility with preloaded metadata.
func (p Program) PackageLocalitiesFor(pkg *types.Package) map[string]VariableLocality {
	prefix := PathOf(pkg) + "."
	ret := make(map[string]VariableLocality)
	p.localities.mu.RLock()
	for name, info := range p.localities.ownerlessEntries {
		if info.Locality == locality.None || !strings.HasPrefix(name, prefix) {
			continue
		}
		ret[name] = info
	}
	for name, entries := range p.localities.declarationEntries {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if info, ok := entries[pkg.Path()]; ok && info.Locality != locality.None {
			ret[name] = info
		}
	}
	p.localities.mu.RUnlock()
	return ret
}

func (p Program) NeedsLocalContext() bool {
	p.localities.mu.RLock()
	defer p.localities.mu.RUnlock()
	needsContext := func(info VariableLocality) bool {
		return info.Locality != locality.None && (info.LocalStorage != LocalStorageNativeTLS || hasInitialization(info.Info))
	}
	for _, info := range p.localities.ownerlessEntries {
		if needsContext(info) {
			return true
		}
	}
	for _, entries := range p.localities.declarationEntries {
		for owner, info := range entries {
			if _, active := p.localities.activePackages[owner]; active && needsContext(info) {
				return true
			}
		}
	}
	for name, info := range p.localities.entries {
		if _, ownerless := p.localities.ownerlessEntries[name]; ownerless {
			continue
		}
		if _, declared := p.localities.declarationEntries[name]; declared {
			continue
		}
		if needsContext(info) {
			return true
		}
	}
	return false
}
