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

// StoragePointerBytes is a backend metadata query. Unlike source size queries,
// it runs after checking, when LLVM storage and native field views are complete.
// Keep the owner through recursion: a native memory32 pointer slot can be four
// bytes even though a standalone Go pointer occupies an eight-byte slot.
func (p *goProgram) StoragePointerBytes(t types.Type) (uintptr, bool) {
	prog := Program(p)
	if !prog.containsNativeAggregate(t) {
		return 0, false
	}
	return uintptr(prog.storagePointerBytes(prog.rawType(t))), true
}

func (p Program) storagePointerBytes(t Type) uint64 {
	switch u := t.RawType().Underlying().(type) {
	case *types.Pointer, *types.Map, *types.Chan, *types.Signature:
		return p.SizeOf(t)
	case *types.Basic:
		switch u.Kind() {
		case types.UnsafePointer:
			return p.SizeOf(t)
		case types.String:
			return uint64(p.PointerSize())
		}
	case *types.Slice:
		return uint64(p.PointerSize())
	case *types.Interface:
		return p.SizeOf(t)
	case *types.Struct:
		for i := u.NumFields() - 1; i >= 0; i-- {
			if bytes := p.storagePointerBytes(p.Field(t, i)); bytes != 0 {
				return p.OffsetOf(t, i) + bytes
			}
		}
	case *types.Array:
		if u.Len() != 0 {
			if bytes := p.storagePointerBytes(p.Index(t)); bytes != 0 {
				stride := p.td.TypeAllocSize(p.storageType(t).ElementType())
				return uint64(u.Len()-1)*stride + bytes
			}
		}
	}
	return 0
}
