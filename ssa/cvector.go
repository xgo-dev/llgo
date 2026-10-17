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
	"go/token"
	"go/types"

	"github.com/xgo-dev/llvm"
)

// SetCVectorType restores the vector identity that cmd/cgo's Go array spelling
// cannot express. The build driver obtains it from Clang, never from an array's
// size or its typedef name. This first boundary supports naturally aligned,
// 128-bit integer and floating vectors.
func (p Program) SetCVectorType(name string, lanes int) {
	p.packageSyntax.mu.Lock()
	p.packageSyntax.cVectors[name] = lanes
	p.packageSyntax.mu.Unlock()
}

func (p Program) cVectorArray(raw types.Type) (*types.Array, bool) {
	named, ok := types.Unalias(raw).(*types.Named)
	if !ok {
		return nil, false
	}
	p.packageSyntax.mu.RLock()
	n, ok := p.packageSyntax.cVectors[namedLinkname(named)]
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

// SetNativeStructFields supplies the context missing from types.Sizes.Offsetsof:
// its argument contains fields, but not their enclosing named struct. Positions
// come from the same parsed declarations as the type calling convention.
func (p Program) SetNativeStructFields(positions []token.Pos) {
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	for _, pos := range positions {
		if pos.IsValid() {
			p.packageSyntax.nativeStructFields[pos] = true
		}
	}
}

func (p Program) nativeFields(fields []*types.Var) bool {
	if len(fields) == 0 {
		return false
	}
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	return p.packageSyntax.nativeStructFields[fields[0].Pos()]
}

func (b Builder) arrayElement(value llvm.Value, index int) llvm.Value {
	if value.Type().TypeKind() == llvm.VectorTypeKind {
		return b.impl.CreateExtractElement(value, llvm.ConstInt(b.Prog.tyInt32(), uint64(index), false), "")
	}
	return b.impl.CreateExtractValue(value, index, "")
}
