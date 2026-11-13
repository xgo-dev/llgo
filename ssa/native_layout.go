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
func (p Program) SetNativeStructFields(positions []token.Pos) {
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	for _, pos := range positions {
		if pos.IsValid() {
			p.packageSyntax.nativeStructFields[pos] = true
			p.packageSyntax.hasNativeTypeMetadata.Store(true)
		}
	}
}

func (p Program) nativeFields(fields []*types.Var) bool {
	if len(fields) == 0 || !p.packageSyntax.hasNativeTypeMetadata.Load() {
		return false
	}
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	return p.packageSyntax.nativeStructVars[fields[0]] || p.packageSyntax.nativeStructFields[fields[0].Pos()]
}
