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

import "go/types"

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

// TypeDefinitionSource returns the bound declaration RHS of an object. A nil
// result means that the RHS is not available yet, or is not a type definition.
// During source checking it may read the checker's partial types.Info; after
// checking it must support concurrent read-only queries from backend Programs.
// It must not infer a declaration edge from shared underlying types or fields.
type TypeDefinitionSource func(*types.TypeName) types.Type

// SetTypeDefinition records an actual definition edge, for importers and
// programmatic SSA clients. Register all edges before lowering any types.
// This does not change Go type identity or copy the RHS type's method set.
func (p Program) SetTypeDefinition(obj *types.TypeName, rhs types.Type) {
	if obj == nil || rhs == nil {
		panic("ssa: type definition requires an object and a RHS")
	}
	p.packageSyntax.mu.Lock()
	p.packageSyntax.typeDefinitions[obj] = rhs
	p.packageSyntax.mu.Unlock()
}

// SetTypeDefinitionSource supplies declaration provenance before type checking
// has populated the package scope. Unknown RHS queries are never cached: a
// later query must observe the checker completing the declaration.
func (p Program) SetTypeDefinitionSource(pkg *types.Package, source TypeDefinitionSource) {
	p.packageSyntax.mu.Lock()
	p.packageSyntax.typeDefinitionSources[pkg] = source
	p.packageSyntax.mu.Unlock()
}

// RepresentationOf follows definition edges, not Underlying(). In particular,
// an independent structurally identical declaration does not inherit native
// storage. Generic instances use their origin declaration's representation.
func (p Program) RepresentationOf(t types.Type) TypeRepresentation {
	return p.packageSyntax.representationOf(t)
}

func (p *packageSyntaxData) representationOf(t types.Type) TypeRepresentation {
	if !p.hasNativeTypeMetadata.Load() {
		return TypeRepresentation{}
	}
	var visited [8]*types.TypeName
	seen := visited[:0]
	for {
		var obj *types.TypeName
		var aliasRHS types.Type
		switch t := t.(type) {
		case *types.Named:
			obj = t.Origin().Obj()
		case *types.Alias:
			obj, aliasRHS = t.Origin().Obj(), t.Rhs()
		default:
			return TypeRepresentation{}
		}
		for _, previous := range seen {
			if previous == obj {
				// Invalid source cycles are diagnosed by the checker. Also keep
				// incomplete or cyclic programmatic metadata queries finite.
				return TypeRepresentation{}
			}
		}
		seen = append(seen, obj)

		p.mu.RLock()
		rhs := p.typeDefinitions[obj]
		source := p.typeDefinitionSources[obj.Pkg()]
		var cc CallConv
		// The legacy string API identifies package-level declarations only.
		// A local type with the same spelling must not inherit that directive.
		if obj.Parent() == nil || obj.Pkg() == nil || obj.Parent() == obj.Pkg().Scope() {
			cc = p.typeCallConvs[typeNameLinkname(obj)]
		}
		p.mu.RUnlock()
		if isNativeCallConv(cc) {
			return TypeRepresentation{NativeRepresentation, cc, obj}
		}
		if rhs == nil && source != nil {
			rhs = source(obj)
		}
		if rhs == nil {
			rhs = aliasRHS
		}
		t = rhs
	}
}
