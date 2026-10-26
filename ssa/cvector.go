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
	"strings"

	"github.com/xgo-dev/llvm"
)

// Keep registry keys split so ordinary named queries never allocate a linkname.
type cVectorTypeName struct{ pkgPath, name string }

// SetCVectorType restores the vector identity that cmd/cgo's Go array spelling
// cannot express. The build driver obtains it from Clang, never from an array's
// size or its typedef name. This first boundary supports naturally aligned,
// 128-bit integer and floating vectors.
func (p Program) SetCVectorType(name string, lanes int) {
	p.packageSyntax.mu.Lock()
	key := cVectorTypeName{name: name}
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		key.pkgPath, key.name = name[:i], name[i+1:]
	}
	p.packageSyntax.cVectors[key] = lanes
	p.packageSyntax.hasCVectorTypes.Store(true)
	p.packageSyntax.hasNativeTypeMetadata.Store(true)
	p.packageSyntax.mu.Unlock()
}

func (p Program) cVectorArray(raw types.Type) (*types.Array, bool) {
	if !p.packageSyntax.hasCVectorTypes.Load() {
		return nil, false
	}
	named, ok := types.Unalias(raw).(*types.Named)
	if !ok {
		return nil, false
	}
	p.packageSyntax.mu.RLock()
	key := cVectorTypeName{name: named.Obj().Name()}
	if pkg := named.Obj().Pkg(); pkg != nil {
		key.pkgPath = pkg.Path()
	}
	n, ok := p.packageSyntax.cVectors[key]
	p.packageSyntax.mu.RUnlock()
	if !ok {
		return nil, false
	}
	array, ok := named.Underlying().(*types.Array)
	if !ok || array.Len() != int64(n) {
		panic(fmt.Sprintf("C vector %s has an incompatible Go representation", raw))
	}
	elem, ok := array.Elem().Underlying().(*types.Basic)
	if !ok || elem.Info()&(types.IsInteger|types.IsFloat) == 0 ||
		elem.Kind() == types.Int || elem.Kind() == types.Uint || elem.Kind() == types.Uintptr ||
		p.sizes.Sizeof(elem)*array.Len() != 16 {
		panic(fmt.Sprintf("C vector %s requires 128 bits of integer or floating lanes", raw))
	}
	return array, true
}

func (b Builder) arrayElement(value llvm.Value, index int) llvm.Value {
	if value.Type().TypeKind() == llvm.VectorTypeKind {
		return b.impl.CreateExtractElement(value, llvm.ConstInt(b.Prog.tyInt32(), uint64(index), false), "")
	}
	return b.impl.CreateExtractValue(value, index, "")
}
