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
)

// StorageRepresentation describes how a source type stores values. It is
// independent of the calling convention of a function accepting those values.
type StorageRepresentation uint8

const (
	GoRepresentation StorageRepresentation = iota
	NativeRepresentation
)

// TypeRepresentation retains the declaration that selected a native
// representation. NativeABI describes native function values, not the entry
// ABI of ordinary Go functions whose parameters contain this type.
type TypeRepresentation struct {
	Storage   StorageRepresentation
	NativeABI CallConv
	Origin    *types.TypeName
}

// TypeRepresentationChecker supplies provisional attributes exclusively to source
// layout queries while go/types is checking a package. It is never shared with
// backend Programs. The frontend must publish a complete snapshot afterwards.
type TypeRepresentationChecker func(*types.TypeName) TypeRepresentation

// localTypeDeclaration identifies a checked declaration, independently of the
// fresh TypeName allocated by SSA for each generic function instantiation.
// Package identity separates rechecks; position separates shadowing scopes.
// No source syntax or definition edges are retained in this index.
type localTypeDeclaration struct {
	pkg  *types.Package
	pos  token.Pos
	name string
}

func localDeclaration(obj *types.TypeName) localTypeDeclaration {
	return localTypeDeclaration{obj.Pkg(), obj.Pos(), obj.Name()}
}

func representationObject(t types.Type) *types.TypeName {
	switch t := t.(type) {
	case *types.Named:
		return t.Origin().Obj()
	case *types.Alias:
		return t.Origin().Obj()
	}
	return nil
}

// DeclaredTypeRepresentation reads only the legacy native root directive. The
// frontend uses it when resolving attributes; derived types do not acquire a
// calling-convention registration of their own.
func (p Program) DeclaredTypeRepresentation(obj *types.TypeName) TypeRepresentation {
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	return p.packageSyntax.declaredRepresentation(obj)
}

func (p *packageSyntaxData) declaredRepresentation(obj *types.TypeName) TypeRepresentation {
	if obj != nil && (obj.Parent() == nil || obj.Pkg() == nil || obj.Parent() == obj.Pkg().Scope()) {
		if cc := p.typeCallConvs[typeNameLinkname(obj)]; isNativeCallConv(cc) {
			return TypeRepresentation{NativeRepresentation, cc, obj}
		}
	}
	return TypeRepresentation{}
}

// SetTypeRepresentation publishes a resolved attribute for a programmatic SSA
// client. All attributes must be supplied before lowering; changing a published
// attribute would invalidate layouts already derived from it.
func (p Program) SetTypeRepresentation(obj *types.TypeName, rep TypeRepresentation) {
	validateTypeRepresentation(obj, rep)
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	p.packageSyntax.setTypeRepresentation(obj, rep)
}

func validateTypeRepresentation(obj *types.TypeName, rep TypeRepresentation) {
	if obj == nil || rep.Storage != GoRepresentation && rep.Storage != NativeRepresentation ||
		rep.Storage == GoRepresentation && rep != (TypeRepresentation{}) ||
		rep.Storage == NativeRepresentation && (rep.Origin == nil || !isNativeCallConv(rep.NativeABI)) {
		panic("ssa: invalid type representation")
	}
}

func (p *packageSyntaxData) setTypeRepresentation(obj *types.TypeName, rep TypeRepresentation) {
	if old, ok := p.typeRepresentations[obj]; ok && old != rep {
		panic("ssa: cannot change a published type representation")
	}
	p.typeRepresentations[obj] = rep
	if obj.Pkg() != nil && obj.Pos().IsValid() && obj.Parent() != nil && obj.Parent() != obj.Pkg().Scope() {
		p.localTypeRepresentations[localDeclaration(obj)] = rep
	}
	if rep.Storage == NativeRepresentation {
		p.hasNativeTypeMetadata.Store(true)
	}
}

// SetTypeRepresentationChecker starts the source-checking phase for pkg.
// Source sizes can consult checker, but LLVM lowering cannot. Completion must
// call PublishTypeRepresentations, even if the package has no native types.
func (p Program) SetTypeRepresentationChecker(pkg *types.Package, checker TypeRepresentationChecker) {
	p.checkingRepresentationsMu.Lock()
	defer p.checkingRepresentationsMu.Unlock()
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	if _, complete := p.packageSyntax.completedTypePackages[pkg]; complete {
		panic("ssa: cannot restart checking a completed type snapshot")
	}
	if p.checkingRepresentations == nil {
		p.checkingRepresentations = make(map[*types.Package]TypeRepresentationChecker)
	}
	p.checkingRepresentations[pkg] = checker
	p.packageSyntax.checkingTypePackages[pkg] = struct{}{}
}

// PublishTypeRepresentations atomically freezes frontend attributes and drops
// the checker callback. Copy the map so the frontend cannot mutate the snapshot.
func (p Program) PublishTypeRepresentations(pkg *types.Package, reps map[*types.TypeName]TypeRepresentation) {
	for obj, rep := range reps {
		validateTypeRepresentation(obj, rep)
		if obj.Pkg() != pkg {
			panic("ssa: type representation belongs to another package")
		}
	}
	p.checkingRepresentationsMu.Lock()
	defer p.checkingRepresentationsMu.Unlock()
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	for obj, rep := range reps {
		if old, ok := p.packageSyntax.typeRepresentations[obj]; ok && old != rep {
			panic("ssa: cannot change a published type representation")
		}
	}
	for obj, rep := range reps {
		p.packageSyntax.setTypeRepresentation(obj, rep)
	}
	p.packageSyntax.completedTypePackages[pkg] = struct{}{}
	delete(p.packageSyntax.checkingTypePackages, pkg)
	delete(p.checkingRepresentations, pkg)
}

// RepresentationOf reads resolved type attributes. It never traverses a
// definition chain or queries syntax, types.Info, or a type checker. Generic
// instances read their origin declaration's attribute. Programmatic clients
// may still register native roots using SetTypeCallConv.
func (p Program) RepresentationOf(t types.Type) TypeRepresentation {
	return p.gocvt.representationOf(t)
}

// CopyTypeRepresentation transfers a resolved attribute to a fresh backend
// type adapter (for example, a positionless generic-local ABI name). The copy
// belongs to this Program's lowering state, not the shared frontend snapshot.
func (p Program) CopyTypeRepresentation(dst *types.TypeName, src types.Type) {
	rep := p.RepresentationOf(src)
	validateTypeRepresentation(dst, rep)
	if old, ok := p.gocvt.representations[dst]; ok && old != rep {
		panic("ssa: cannot change a lowered type representation")
	}
	p.gocvt.representations[dst] = rep
}

func (p goTypes) representationOf(t types.Type) TypeRepresentation {
	syntax := p.packageSyntax
	for {
		obj := representationObject(t)
		if obj == nil {
			return TypeRepresentation{}
		}
		if rep, found := p.representations[obj]; found {
			return rep
		}
		syntax.mu.RLock()
		_, checking := syntax.checkingTypePackages[obj.Pkg()]
		rep, found := syntax.typeRepresentations[obj]
		if !found && obj.Parent() == nil && obj.Pos().IsValid() {
			// x/tools/go/ssa substitutes local Named/Alias types using a fresh
			// object with the same declaration coordinates and no scope. Its
			// Origin is fresh too, so pointer identity alone is insufficient.
			rep, found = syntax.localTypeRepresentations[localDeclaration(obj)]
		}
		if !found {
			rep = syntax.declaredRepresentation(obj)
		}
		syntax.mu.RUnlock()
		if checking {
			panic("ssa: type representation requested before frontend completion")
		}
		if found || rep.Storage == NativeRepresentation {
			return rep
		}
		// Programmatic aliases retain identity even without a frontend snapshot.
		// This is alias unwrapping, not reconstruction of named definitions.
		if alias, ok := t.(*types.Alias); ok {
			t = alias.Rhs()
			continue
		}
		return rep
	}
}

// sourceRepresentationOf is the only bridge to provisional checker state.
// Source layout must work during checking (e.g. array bounds using Sizeof).
// Backend Programs have no checkingRepresentations and read the snapshot.
func (p Program) sourceRepresentationOf(t types.Type) TypeRepresentation {
	obj := representationObject(t)
	if obj == nil {
		return TypeRepresentation{}
	}
	p.checkingRepresentationsMu.RLock()
	checker := p.checkingRepresentations[obj.Pkg()]
	p.checkingRepresentationsMu.RUnlock()
	if checker != nil {
		return checker(obj)
	}
	return p.RepresentationOf(t)
}

func (p Program) sourceHasNativeTypeLayout(t types.Type) bool {
	if p.sourceRepresentationOf(t).Storage == NativeRepresentation {
		return true
	}
	if a, ok := types.Unalias(t).(*types.Array); ok {
		return p.sourceHasNativeTypeLayout(a.Elem())
	}
	return false
}

// TypeRepresentationsPending reports whether a frontend still owns mutable
// bindings for pkg. A pending package cannot be used by backend lowering.
func (p Program) TypeRepresentationsPending(pkg *types.Package) bool {
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	_, pending := p.packageSyntax.checkingTypePackages[pkg]
	return pending
}

// TypeRepresentationsComplete also covers validated opaque imports. Such
// packages have no native declarations under the current export contract and
// do not need to reconstruct source bindings to establish Go representation.
func (p Program) TypeRepresentationsComplete(pkg *types.Package) bool {
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	_, complete := p.packageSyntax.completedTypePackages[pkg]
	return complete
}
