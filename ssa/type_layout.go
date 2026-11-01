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

// LayoutStorageWrapperPrefix marks field storage that must survive native ABI
// normalization. Its padding and alignment are part of the declared layout.
const LayoutStorageWrapperPrefix = "$llgo.layout.storage"

type structLayout struct {
	wrapped []bool
}

func (p Program) hasNativeStructLayout(raw *types.Named) bool {
	if p.packageSyntax.layout.TypeAttrs(raw).Native {
		return true
	}
	if !p.packageSyntax.hasNativeTypeMetadata.Load() {
		return false
	}
	background, ok := p.packageTypeCallConv(namedLinkname(raw))
	return ok && isNativeCallConv(background)
}

func (p Program) setStructLayout(typ types.Type, layout *structLayout) {
	if layout != nil {
		p.structLayouts.Set(typ, *layout)
		p.hasStructLayouts = true
	}
}

func (p Program) structLayout(typ Type) (structLayout, bool) {
	if !p.hasStructLayouts {
		return structLayout{}, false
	}
	layout := p.structLayouts.At(typ.raw.Type)
	if layout == nil {
		return structLayout{}, false
	}
	return layout.(structLayout), true
}

// toLLVMStructBody materializes the shared source layout, including explicit
// padding and pack directives. Extra bytes belong to the preceding field's
// storage wrapper; the next field still starts at its source offset. Thus Go
// field indexes and addresses need no synthetic addressable padding fields.
func (p Program) toLLVMStructBody(raw *types.Struct, native bool) ([]llvm.Type, *structLayout) {
	p.packageSyntax.layout.RecordType(raw)
	if err := p.sourceSizes.Validate(raw); err != nil {
		panic(err)
	}
	goFields := make([]*types.Var, raw.NumFields())
	for i := range goFields {
		f := raw.Field(i)
		goFields[i] = f
		if patched := p.patch(f.Type()); patched != f.Type() {
			goFields[i] = types.NewField(f.Pos(), f.Pkg(), f.Name(), patched, f.Embedded())
		}
	}
	attrs := p.packageSyntax.layout.StructAttrs
	if len(goFields) != 0 {
		native = native || attrs(goFields[0]).Native || p.nativeFields(goFields)
	}
	fields := p.toLLVMFields(raw, native)
	if len(fields) == 0 {
		return fields, nil
	}

	// Go function values have already been converted to closure structs here;
	// remaining raw signatures denote native code pointers. Do not let source
	// function-value sizing expand those pointers for a second time.
	sizes := *p.sourceSizes
	nativeTypeHook := sizes.NativeType
	sizes.NativeType = func(t types.Type) bool {
		if _, ok := t.(*types.Signature); ok {
			return true
		}
		return nativeTypeHook != nil && nativeTypeHook(t)
	}
	storage := make(map[types.Type][2]int64, len(fields))
	for i, f := range goFields {
		a := sizes.Alignof(f.Type())
		if p.fieldNativeStorage(f.Type(), native) {
			a = int64(p.td.ABITypeAlignment(fields[i]))
		}
		storage[f.Type()] = [2]int64{int64(p.td.TypeAllocSize(fields[i])), a}
	}
	sizes.StorageLayout = func(t types.Type) (int64, int64, bool) {
		layout, ok := storage[t]
		return layout[0], layout[1], ok
	}
	layout, offsets := sizes.FieldsLayout(goFields, native)
	goSize, goAlign := layout[0], layout[1]
	if goSize < 0 {
		panic("struct layout exceeds addressable storage: " + raw.String())
	}
	alignments := make([]int64, len(fields))
	pack := attrs(goFields[0]).Pack
	for i, field := range goFields {
		a := sizes.Alignof(field.Type())
		if p.fieldNativeStorage(field.Type(), native) {
			a = int64(p.td.ABITypeAlignment(fields[i]))
		}
		if pack > 0 && a > pack {
			a = pack
		}
		alignments[i] = a
	}

	nativeType := p.ctx.StructType(fields, false)
	if int64(p.td.TypeAllocSize(nativeType)) == goSize &&
		int64(p.td.ABITypeAlignment(nativeType)) == goAlign {
		equal := true
		for i, offset := range offsets {
			if int64(p.td.ElementOffset(nativeType, i)) != offset {
				equal = false
				break
			}
		}
		if equal {
			return fields, nil
		}
	}

	wrapped := make([]bool, len(fields))
	body := make([]llvm.Type, 0, len(fields)+1)
	for i, field := range fields {
		end := goSize
		if i+1 < len(offsets) {
			end = offsets[i+1]
		}
		fieldSize := int64(p.td.TypeAllocSize(field))
		padding := end - offsets[i] - fieldSize
		fieldAlign := int64(p.td.ABITypeAlignment(field))
		goFieldAlign := alignments[i]
		if padding < 0 {
			panic(fmt.Sprintf("Go field layout is smaller than its LLVM representation: %s field %d", raw, i))
		}
		if padding != 0 || fieldAlign > goFieldAlign {
			parts := []llvm.Type{field}
			if padding != 0 {
				parts = append(parts, llvm.ArrayType(p.tyInt8(), int(padding)))
			}
			field = p.ctx.StructType(parts, true)
			if p.target.effectiveGOOS() == "windows" && p.target.effectiveGOARCH() == "386" &&
				(pack != 0 || fieldAlign <= goFieldAlign) {
				// Win32's legacy ABI adapter unwraps ordinary Go/386 scalar
				// alignment adjustments. Packing and explicit storage padding
				// must retain their bytes and offsets at the call boundary.
				// Reuse each physical shape: repeated tuple/metadata lowering
				// must not introduce distinct LLVM identities for equal layouts.
				if p.layoutStorageWrappers == nil {
					p.layoutStorageWrappers = make(map[llvm.Type]llvm.Type)
				}
				wrapper, ok := p.layoutStorageWrappers[field]
				if !ok {
					wrapper = p.ctx.StructCreateNamed(LayoutStorageWrapperPrefix)
					wrapper.StructSetBody(parts, true)
					p.layoutStorageWrappers[field] = wrapper
				}
				field = wrapper
			}
			wrapped[i] = true
		}
		body = append(body, field)
	}

	probe := p.ctx.StructType(body, false)
	if int64(p.td.ABITypeAlignment(probe)) < goAlign {
		// Integers carry ordinary alignment; vector carriers are needed for
		// over-aligned SIMD storage, since i128 need not be 16-byte aligned.
		alignType := p.ctx.IntType(int(goAlign * 8))
		if goAlign > int64(p.GoWordSize()) {
			alignType = llvm.VectorType(p.tyInt8(), int(goAlign))
		}
		body = append(body, llvm.ArrayType(alignType, 0))
		probe = p.ctx.StructType(body, false)
	}
	if int64(p.td.TypeAllocSize(probe)) != goSize {
		panic(fmt.Sprintf("invalid Go LLVM struct size: got %d, want %d for %s", p.td.TypeAllocSize(probe), goSize, raw))
	}
	for i, offset := range offsets {
		if got := int64(p.td.ElementOffset(probe, i)); got != offset {
			panic(fmt.Sprintf("invalid Go LLVM field offset: got %d, want %d for %s field %d", got, offset, raw, i))
		}
	}
	return body, &structLayout{wrapped: wrapped}
}

func (p Program) hasNativeTypeLayout(raw types.Type) bool {
	if p.packageSyntax.layout.TypeAttrs(raw).Native {
		return true
	}
	switch t := types.Unalias(raw).(type) {
	case *types.Named:
		return p.hasNativeStructLayout(t)
	case *types.Array:
		return p.hasNativeTypeLayout(t.Elem())
	}
	return false
}

func (p Program) fieldNativeStorage(t types.Type, parentNative bool) bool {
	if p.hasNativeTypeLayout(t) {
		return true
	}
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		return false
	case *types.Array:
		return p.fieldNativeStorage(t.Elem(), parentNative)
	}
	return parentNative
}

func (b Builder) checkPointerLayoutConversion(to, from Type) {
	x, ok := from.RawType().Underlying().(*types.Pointer)
	if !ok {
		return
	}
	y, ok := to.RawType().Underlying().(*types.Pointer)
	if !ok || types.Identical(x.Elem(), y.Elem()) {
		return
	}
	// unsafe.Pointer remains the explicit escape hatch. Ordinary pointer
	// conversions cannot perform the field-by-field copy used for values.
	if !b.Prog.sameStorageShape(b.Prog.rawType(x.Elem()), b.Prog.rawType(y.Elem()), make(map[[2]types.Type]bool)) {
		panic(fmt.Errorf("cannot convert pointers between different LLGo layouts: %s to %s", from.RawType(), to.RawType()))
	}
}

func (p Program) sameStorageShape(x, y Type, seen map[[2]types.Type]bool) bool {
	key := [2]types.Type{x.RawType(), y.RawType()}
	if seen[key] {
		return true
	}
	seen[key] = true
	if p.SizeOf(x) != p.SizeOf(y) || p.AlignOf(x) != p.AlignOf(y) {
		return false
	}
	switch xt := x.RawType().Underlying().(type) {
	case *types.Struct:
		yt, ok := y.RawType().Underlying().(*types.Struct)
		if !ok || xt.NumFields() != yt.NumFields() {
			return false
		}
		for i := 0; i < xt.NumFields(); i++ {
			if p.OffsetOf(x, i) != p.OffsetOf(y, i) || !p.sameStorageShape(p.Field(x, i), p.Field(y, i), seen) {
				return false
			}
		}
	case *types.Array:
		yt, ok := y.RawType().Underlying().(*types.Array)
		return ok && xt.Len() == yt.Len() && p.sameStorageShape(p.Index(x), p.Index(y), seen)
	case *types.Pointer:
		if yt, ok := y.RawType().Underlying().(*types.Pointer); ok {
			return p.sameStorageShape(p.rawType(xt.Elem()), p.rawType(yt.Elem()), seen)
		}
	}
	return true
}
