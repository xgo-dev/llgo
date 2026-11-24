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

package types

import gotypes "go/types"

// IsHostLayoutMarker recognizes the private identity retained by aliases and
// defined types of structs.HostLayout. HostLayout deliberately does not
// propagate across nested struct boundaries.
func IsHostLayoutMarker(t gotypes.Type) bool {
	if t == nil {
		return false
	}
	st, ok := t.Underlying().(*gotypes.Struct)
	if !ok || st.NumFields() != 1 || st.Field(0).Name() != "_" {
		return false
	}
	n, ok := unaliasType(st.Field(0).Type()).(*gotypes.Named)
	if !ok || n.Obj().Pkg() == nil ||
		(n.Obj().Pkg().Path() != "structs" && n.Obj().Pkg().Path() != "github.com/xgo-dev/llgo/runtime/internal/structs") ||
		n.Obj().Name() != "hostLayout" {
		return false
	}
	body, ok := n.Underlying().(*gotypes.Struct)
	return ok && body.NumFields() == 0
}

// HasHostLayout reports whether t is a struct directly containing a
// structs.HostLayout marker field.
func HasHostLayout(t gotypes.Type) bool {
	if t == nil {
		return false
	}
	st, ok := t.Underlying().(*gotypes.Struct)
	if !ok {
		return false
	}
	for i := 0; i < st.NumFields(); i++ {
		if IsHostLayoutMarker(st.Field(i).Type()) {
			return true
		}
	}
	return false
}

// ContainsHostLayout reports whether t is or transitively embeds/contains a
// structs.HostLayout marker in structs or arrays.
func ContainsHostLayout(t gotypes.Type) bool {
	if t == nil {
		return false
	}
	switch u := t.Underlying().(type) {
	case *gotypes.Struct:
		for i := 0; i < u.NumFields(); i++ {
			f := u.Field(i).Type()
			if IsHostLayoutMarker(f) || ContainsHostLayout(f) {
				return true
			}
		}
	case *gotypes.Array:
		return ContainsHostLayout(u.Elem())
	}
	return false
}
