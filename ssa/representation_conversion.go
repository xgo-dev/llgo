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

	"github.com/xgo-dev/llvm"
)

// CheckNativePointerConversion rejects a typed pointer view that would access
// different storage. Explicit conversions through unsafe.Pointer are outside
// this check; accepted ordinary conversions must preserve shared storage.
func (p Program) CheckNativePointerConversion(dst, src types.Type) error {
	if !p.packageSyntax.hasNativeTypeMetadata.Load() {
		return nil
	}
	d, dok := dst.Underlying().(*types.Pointer)
	s, sok := src.Underlying().(*types.Pointer)
	if !dok || !sok || !p.containsNativeAggregate(dst) && !p.containsNativeAggregate(src) && !p.containsNativeAggregate(d.Elem()) && !p.containsNativeAggregate(s.Elem()) {
		return nil
	}
	// Preserve pointer ownership too: a native pointer typedef can select a
	// narrower pointee view even when its element has an ordinary Go identity.
	if p.sharedStorageCompatible(p.Elem(p.Type(dst, GoDecl)), p.Elem(p.Type(src, GoDecl)), make(map[[2]Type]bool)) {
		return nil
	}
	return fmt.Errorf("cannot convert %s to %s: incompatible native storage", src, dst)
}

func (p Program) sharedStorageCompatible(a, b Type, seen map[[2]Type]bool) bool {
	// SIMD expressions use vector registers, but pointer views use the tagged
	// struct storage. A defined type can retain that storage without acquiring
	// the special register operations selected by the builtin vector's name.
	if a.kind == vkSIMD {
		a = p.rawType(a.RawType().Underlying())
	}
	if b.kind == vkSIMD {
		b = p.rawType(b.RawType().Underlying())
	}
	if a == b {
		return true
	}
	key := [2]Type{a, b}
	if seen[key] {
		return true
	}
	seen[key] = true
	if p.SizeOf(a) != p.SizeOf(b) || p.AlignOf(a) != p.AlignOf(b) || a.kind != b.kind {
		return false
	}
	switch u := a.RawType().Underlying().(type) {
	case *types.Struct:
		v, ok := b.RawType().Underlying().(*types.Struct)
		if !ok || u.NumFields() != v.NumFields() {
			return false
		}
		for i := 0; i < u.NumFields(); i++ {
			if p.OffsetOf(a, i) != p.OffsetOf(b, i) || !p.sharedStorageCompatible(p.Field(a, i), p.Field(b, i), seen) {
				return false
			}
		}
	case *types.Array:
		v, ok := b.RawType().Underlying().(*types.Array)
		return ok && u.Len() == v.Len() && p.sharedStorageCompatible(p.Index(a), p.Index(b), seen)
	case *types.Pointer:
		return p.sharedStorageCompatible(p.Elem(a), p.Elem(b), seen)
	case *types.Signature:
		if p.isStdcallType(a.RawType()) == p.isStdcallType(b.RawType()) {
			return true
		}
		return p.stdcallCallConv() == llvm.CCallConv
	}
	return true
}
