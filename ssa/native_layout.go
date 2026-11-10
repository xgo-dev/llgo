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

// SetNativeStructFields retains the legacy field registration used by vector
// source layout. Positions must be valid identifiers from the parsed fields,
// including the first field. Empty structs need no positions.
//
// This cannot identify a unique enclosing type: go/types shares fields and
// positions across definitions. Definition provenance is recorded separately;
// replacing this registry requires an owner-aware source layout interface.
func (p Program) SetNativeStructFields(positions []token.Pos, anonymous ...bool) {
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	for _, pos := range positions {
		if pos.IsValid() {
			p.packageSyntax.nativeStructFields[pos] = true
			if len(anonymous) != 0 && anonymous[0] {
				p.packageSyntax.nativeAnonymousFields[pos] = true
			}
			p.packageSyntax.hasNativeTypeMetadata.Store(true)
		}
	}
}

// nativeAnonymousStruct identifies an explicitly annotated anonymous RHS, not
// an inferred definition edge between named types sharing fields.
func (p *packageSyntaxData) nativeAnonymousStruct(st *types.Struct) bool {
	if !p.hasNativeTypeMetadata.Load() {
		return false
	}
	p.mu.RLock()
	native := p.nativeAnonTypes[st]
	if !native && st.NumFields() != 0 {
		native = p.nativeAnonymousFields[st.Field(0).Pos()]
	}
	p.mu.RUnlock()
	return native
}

// SetNativeAnonymousLayout restores a checked layout attribute independently of
// source positions. Importers must validate the attribute before publishing it.
func (p Program) SetNativeAnonymousLayout(st *types.Struct) {
	p.packageSyntax.mu.Lock()
	p.packageSyntax.nativeAnonTypes[st] = true
	if st.NumFields() != 0 {
		p.packageSyntax.nativeStructVars[st.Field(0)] = true
	}
	p.packageSyntax.hasNativeTypeMetadata.Store(true)
	p.packageSyntax.mu.Unlock()
}

// LayoutIdentity supplies target-independent anonymous layout attributes for
// persistent type keys. Named identities already retain their declaration name.
func (p Program) LayoutIdentity(t types.Type) string {
	if st, ok := t.(*types.Struct); ok && p.packageSyntax.nativeAnonymousStruct(st) {
		return "native"
	}
	return ""
}

func (p *goProgram) NativeStruct(st *types.Struct) bool {
	return p.packageSyntax.nativeAnonymousStruct(st)
}

// Anonymous native layout participates in backend identity. gc identity alone
// would merge it with an ordinary same-shape struct (or a container of one).
func (p Program) containsNativeAnonymous(t types.Type) bool {
	if !p.packageSyntax.hasNativeTypeMetadata.Load() {
		return false
	}
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		return false
	case *types.Struct:
		if p.packageSyntax.nativeAnonymousStruct(t) {
			return true
		}
		for i := 0; i < t.NumFields(); i++ {
			if p.containsNativeAnonymous(t.Field(i).Type()) {
				return true
			}
		}
	case *types.Array:
		return p.containsNativeAnonymous(t.Elem())
	case *types.Pointer:
		return p.containsNativeAnonymous(t.Elem())
	case *types.Slice:
		return p.containsNativeAnonymous(t.Elem())
	case *types.Map:
		return p.containsNativeAnonymous(t.Key()) || p.containsNativeAnonymous(t.Elem())
	case *types.Chan:
		return p.containsNativeAnonymous(t.Elem())
	}
	return false
}

func (p Program) nativeFields(fields []*types.Var) bool {
	if len(fields) == 0 || !p.packageSyntax.hasNativeTypeMetadata.Load() {
		return false
	}
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	return p.packageSyntax.nativeStructVars[fields[0]] || p.packageSyntax.nativeStructFields[fields[0].Pos()]
}
