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

import (
	gotypes "go/types"
	"math"

	"github.com/xgo-dev/llgo/target"
)

// LayoutPair holds an overflow-aware (Size, Align) pair computed without LLVM
// TargetData. Size == -1 signals int64 overflow matching go/types.Sizes.
type LayoutPair struct {
	Size  int64
	Align int64
}

// AlignSize rounds size up to a multiple of alignment, returning -1 on int64
// overflow or when size is already negative.
func AlignSize(size, alignment int64) int64 {
	if alignment <= 1 {
		return size
	}
	if size < 0 || size > math.MaxInt64-(alignment-1) {
		return -1
	}
	return (size + alignment - 1) &^ (alignment - 1)
}

// ArrayLayout computes the overflow-aware layout of [n]elem.
func ArrayLayout(elem LayoutPair, n int64) LayoutPair {
	size := int64(0)
	if n > 0 {
		size = -1
		if elem.Size >= 0 && (elem.Size == 0 || n <= math.MaxInt64/elem.Size) {
			size = elem.Size * n
		}
	}
	return LayoutPair{Size: size, Align: elem.Align}
}

// AggregateLayout computes struct/tuple size, alignment, and per-field offsets
// with overflow protection and optional Go trailing zero-sized field padding.
func AggregateLayout(numFields int, fieldLayout func(int) LayoutPair, padTail bool) (LayoutPair, []int64) {
	result := LayoutPair{Align: 1}
	offsets := make([]int64, numFields)
	for i := 0; i < numFields; i++ {
		field := fieldLayout(i)
		if field.Align > result.Align {
			result.Align = field.Align
		}
		result.Size = AlignSize(result.Size, field.Align)
		offsets[i] = result.Size
		if result.Size < 0 || field.Size < 0 || field.Size > math.MaxInt64-result.Size {
			result.Size = -1
		} else {
			result.Size += field.Size
			if padTail && i == numFields-1 && field.Size == 0 && result.Size > 0 {
				if result.Size == math.MaxInt64 {
					result.Size = -1
				} else {
					result.Size++ // Keep a final zero-sized Go field inside the object.
				}
			}
		}
	}
	result.Size = AlignSize(result.Size, result.Align)
	return result, offsets
}

// FieldLayout records the physical placement of a single struct or tuple field.
type FieldLayout struct {
	Offset       int64
	Size         int64
	Align        int64
	PaddingAfter int64
	WasmWideSlot bool
}

// Layout is the immutable physical, ABI, and GC metadata computed once per
// canonical LLGo Type.
type Layout struct {
	Size             int64
	Align            int64
	PtrBytes         int64
	PtrOffsets       []int64
	GCBitmap         []byte
	Fields           []FieldLayout
	IsRegularMemory  bool
	HasWasmWideSlots bool
	HasHostLayout    bool
	HasSIMDStorage   bool
}

// SourceSizesConfig configures SourceSizes, the unified pure-Go layout engine
// that implements go/types.Sizes across standard Go types, 2-word closures,
// structs.HostLayout, Wasm32 wide Go vs native C storage, and simd/archsimd
// vector alignment.
type SourceSizesConfig struct {
	Spec target.Spec

	// BaseSizes optionally supplies the underlying go/types.Sizes (evaluated
	// dynamically so callers that swap sizes at runtime remain consistent).
	// When nil or returning nil, a target-accurate StdSizes is used.
	BaseSizes func() gotypes.Sizes

	// HostBasicLayouts overrides default host C basic scalar layouts when
	// HostBasicLayouts[gotypes.Bool].Size != 0.
	HostBasicLayouts [gotypes.UnsafePointer + 1]LayoutPair

	// SIMDAlignments overrides v128/v256/v512 alignments when non-zero.
	SIMDAlignments [3]int64

	// CallConvLookup optionally resolves directive calling conventions by
	// qualified type name ("pkg.Name"). Used when NativeTypeDirective is nil.
	CallConvLookup CallConvLookup

	// Optional fine-grained hooks used by compiler frontends/backends that
	// carry per-package syntax metadata or legacy synthetic closure structs.
	HasNativeMetadata         func() bool
	NativeTypeDirective       func(gotypes.Type) CallConv
	HasLegacyNativeTypeLayout func(gotypes.Type) bool
	NativeFields              func([]*gotypes.Var) bool
	IsClosureStruct           func(*gotypes.Struct) bool
	IsClosureFields           func([]*gotypes.Var) bool
}

// SourceSizes is LLGo's canonical pure-Go implementation of go/types.Sizes.
// It computes target-accurate sizes, alignments, and field offsets without
// materializing LLVM types or mutating go/types nodes, and is safe for
// concurrent reads once constructed.
type SourceSizes struct {
	spec             target.Spec
	baseSizesFn      func() gotypes.Sizes
	defaultBaseSizes gotypes.Sizes
	hostBasicLayouts [gotypes.UnsafePointer + 1]LayoutPair
	simdAlignments   [3]int64

	hasNativeMetadata         func() bool
	nativeTypeDirective       func(gotypes.Type) CallConv
	hasLegacyNativeTypeLayout func(gotypes.Type) bool
	nativeFields              func([]*gotypes.Var) bool
	isClosureStruct           func(*gotypes.Struct) bool
	isClosureFields           func([]*gotypes.Var) bool
}

// NewSourceSizes constructs a SourceSizes engine from cfg.
func NewSourceSizes(cfg SourceSizesConfig) *SourceSizes {
	spec := cfg.Spec
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	hostLayouts := cfg.HostBasicLayouts
	if hostLayouts[gotypes.Bool].Size == 0 {
		hostLayouts = DefaultHostBasicLayouts(spec)
	}
	simdAligns := cfg.SIMDAlignments
	if simdAligns[0] == 0 {
		simdAligns = DefaultSIMDAlignments(spec)
	}
	maxAlign := int64(spec.MaxAlign)
	if maxAlign == 0 {
		maxAlign = int64(spec.WordSize)
	}
	s := &SourceSizes{
		spec:                      spec,
		baseSizesFn:               cfg.BaseSizes,
		defaultBaseSizes:          &gotypes.StdSizes{WordSize: int64(spec.WordSize), MaxAlign: maxAlign},
		hostBasicLayouts:          hostLayouts,
		simdAlignments:            simdAligns,
		hasNativeMetadata:         cfg.HasNativeMetadata,
		nativeTypeDirective:       cfg.NativeTypeDirective,
		hasLegacyNativeTypeLayout: cfg.HasLegacyNativeTypeLayout,
		nativeFields:              cfg.NativeFields,
		isClosureStruct:           cfg.IsClosureStruct,
		isClosureFields:           cfg.IsClosureFields,
	}
	if s.nativeTypeDirective == nil && cfg.CallConvLookup != nil {
		lookup := cfg.CallConvLookup
		s.nativeTypeDirective = func(t gotypes.Type) CallConv {
			return lookupNativeCallConv(t, lookup)
		}
		if s.hasNativeMetadata == nil {
			s.hasNativeMetadata = func() bool { return true }
		}
		if s.hasLegacyNativeTypeLayout == nil {
			s.hasLegacyNativeTypeLayout = func(raw gotypes.Type) bool {
				return hasLegacyNativeLayoutByLookup(raw, lookup)
			}
		}
	}
	return s
}

func lookupNativeCallConv(t gotypes.Type, lookup CallConvLookup) CallConv {
	if lookup == nil {
		return CallConvUnknown
	}
	for {
		var obj *gotypes.TypeName
		var next gotypes.Type
		var isAlias bool
		if n, ok := t.(*gotypes.Named); ok {
			obj = n.Obj()
		} else if aObj, rhs, ok := aliasObjAndRhs(t); ok {
			obj = aObj
			next = rhs
			isAlias = true
		}
		if obj != nil {
			if cc, ok := lookup(typeNameKey(obj)); ok && cc.IsNative() {
				return cc
			}
		}
		if !isAlias || next == t {
			return CallConvUnknown
		}
		t = next
	}
}

func hasLegacyNativeLayoutByLookup(raw gotypes.Type, lookup CallConvLookup) bool {
	if lookup == nil {
		return false
	}
	switch t := unaliasType(raw).(type) {
	case *gotypes.Named:
		cc, ok := lookup(typeNameKey(t.Obj()))
		return ok && cc.IsNative()
	case *gotypes.Array:
		return hasLegacyNativeLayoutByLookup(t.Elem(), lookup)
	}
	return false
}

func (s *SourceSizes) baseSizes() gotypes.Sizes {
	if s.baseSizesFn != nil {
		if b := s.baseSizesFn(); b != nil {
			return b
		}
	}
	return s.defaultBaseSizes
}

// Spec returns the resolved target specification of s.
func (s *SourceSizes) Spec() target.Spec {
	return s.spec
}

// HostBasicLayout returns the target host C ABI (size, align) for basic kind.
func (s *SourceSizes) HostBasicLayout(kind gotypes.BasicKind) LayoutPair {
	if kind >= 0 && int(kind) < len(s.hostBasicLayouts) {
		return s.hostBasicLayouts[kind]
	}
	return LayoutPair{}
}

// SIMDAlignment returns the target ABI alignment for a simd/archsimd tag index
// (0=v128, 1=v256, 2=v512).
func (s *SourceSizes) SIMDAlignment(index int) int64 {
	if index >= 0 && index < len(s.simdAlignments) {
		return s.simdAlignments[index]
	}
	return 1
}

func (s *SourceSizes) hasNativeMeta() bool {
	return s.hasNativeMetadata != nil && s.hasNativeMetadata()
}

func (s *SourceSizes) nativeDirective(t gotypes.Type) CallConv {
	if s.nativeTypeDirective == nil {
		return CallConvUnknown
	}
	return s.nativeTypeDirective(t)
}

func (s *SourceSizes) hasNativeDirective(t gotypes.Type) bool {
	return s.nativeDirective(t).IsNative()
}

func (s *SourceSizes) hasLegacyNativeLayout(raw gotypes.Type) bool {
	if s.hasLegacyNativeTypeLayout == nil {
		return false
	}
	return s.hasLegacyNativeTypeLayout(raw)
}

func (s *SourceSizes) hasNativeStructLayout(raw *gotypes.Named) bool {
	return HasHostLayout(raw) || s.hasLegacyNativeLayout(raw)
}

func (s *SourceSizes) hasNativeFields(fields []*gotypes.Var) bool {
	return s.nativeFields != nil && s.nativeFields(fields)
}

func (s *SourceSizes) isClosure(st *gotypes.Struct) bool {
	return s.isClosureStruct != nil && s.isClosureStruct(st)
}

func (s *SourceSizes) isClosureFieldSlice(fields []*gotypes.Var) bool {
	return s.isClosureFields != nil && s.isClosureFields(fields)
}

// HostUsesNativeGoLayout reports whether ordinary Go aggregates on this target
// already use native C alignment for 64-bit scalars (e.g. 32-bit ARM/MIPS/RISC-V).
func (s *SourceSizes) HostUsesNativeGoLayout() bool {
	arch := s.spec.GOARCH
	return arch != "386" && arch != "wasm" && s.spec.WordSize == 4 && s.hostBasicLayouts[gotypes.Int64].Align > 4
}

// Alignof implements go/types.Sizes.
func (s *SourceSizes) Alignof(T gotypes.Type) int64 {
	if T == nil {
		return 1
	}
	if layout, ok := s.WasmNativeLayout(T); ok {
		return layout.Align
	}
	return s.goAlignof(T)
}

func (s *SourceSizes) goAlignof(T gotypes.Type) int64 {
	if layout, ok := s.HostLayout(T); ok {
		return layout.Align
	}
	if layout, ok := s.VectorLayout(T, false); ok {
		return layout.Align
	}
	return s.baseSizes().Alignof(T)
}

// Sizeof implements go/types.Sizes.
func (s *SourceSizes) Sizeof(T gotypes.Type) int64 {
	if T == nil {
		return 0
	}
	if layout, ok := s.WasmNativeLayout(T); ok {
		return layout.Size
	}
	return s.goSizeof(T)
}

func (s *SourceSizes) goSizeof(T gotypes.Type) int64 {
	if layout, ok := s.HostLayout(T); ok {
		return layout.Size
	}
	if layout, ok := s.VectorLayout(T, false); ok {
		return layout.Size
	}
	base := s.baseSizes()
	word := int64(s.spec.WordSize)
	baseSize := base.Sizeof(T) + s.extraSize(T, word)
	switch T.Underlying().(type) {
	case *gotypes.Struct, *gotypes.Array:
		return alignUp(baseSize, base.Alignof(T))
	}
	return baseSize
}

// Offsetsof implements go/types.Sizes.
func (s *SourceSizes) Offsetsof(fields []*gotypes.Var) []int64 {
	native := s.hasNativeFields(fields)
	if s.spec.UsesWideGoStorage() {
		if native {
			_, offsets := s.WasmNativeFieldsLayout(len(fields), func(i int) *gotypes.Var { return fields[i] })
			return offsets
		}
		if _, offsets, ok := s.WasmGoFieldsLayout(len(fields), func(i int) *gotypes.Var { return fields[i] }); ok {
			return offsets
		}
	}
	if !native {
		for _, f := range fields {
			if IsHostLayoutMarker(f.Type()) || ContainsHostLayout(f.Type()) {
				_, offsets := s.HostFieldsLayout(fields)
				return offsets
			}
		}
	}
	word := int64(s.spec.WordSize)
	if s.isClosureFieldSlice(fields) {
		return []int64{0, word}
	}
	if _, offsets, ok := s.VectorStructLayout(fields, native); ok {
		return offsets
	}
	extra := int64(0)
	ret := s.baseSizes().Offsetsof(fields)
	for i, f := range fields {
		ret[i] += extra
		extra += s.extraSize(f.Type(), word)
	}
	return ret
}

func (s *SourceSizes) extraSize(typ gotypes.Type, word int64) (ret int64) {
retry:
	if _, rhs, ok := aliasObjAndRhs(typ); ok {
		if s.hasNativeDirective(typ) {
			return 0
		}
		typ = rhs
		goto retry
	}
	switch t := typ.(type) {
	case *gotypes.Named:
		if s.hasNativeDirective(t) {
			return 0
		}
		typ = t.Underlying()
		goto retry
	case *gotypes.Signature:
		return word
	case *gotypes.Struct:
		if s.isClosure(t) {
			return 0
		}
		n := t.NumFields()
		for i := 0; i < n; i++ {
			ret += s.extraSize(t.Field(i).Type(), word)
		}
		return ret
	case *gotypes.Array:
		return s.extraSize(t.Elem(), word) * t.Len()
	}
	return 0
}

// HostLayout computes the target layout of a struct or array containing a
// structs.HostLayout marker without materializing LLVM types.
func (s *SourceSizes) HostLayout(t gotypes.Type) (LayoutPair, bool) {
	raw := t
	switch u := t.Underlying().(type) {
	case *gotypes.Struct:
		if !ContainsHostLayout(u) {
			return LayoutPair{}, false
		}
		if s.hasNativeMeta() && s.hasNativeDirective(raw) {
			return LayoutPair{}, false
		}
		fields := make([]*gotypes.Var, u.NumFields())
		for i := range fields {
			fields[i] = u.Field(i)
		}
		if s.hasNativeFields(fields) {
			return LayoutPair{}, false
		}
		layout, _ := s.HostFieldsLayout(fields)
		return layout, true
	case *gotypes.Array:
		if elem, ok := s.HostLayout(u.Elem()); ok {
			if s.hasNativeMeta() && s.hasNativeDirective(raw) {
				return LayoutPair{}, false
			}
			return ArrayLayout(elem, u.Len()), true
		}
	}
	return LayoutPair{}, false
}

// HostFieldsLayout computes the aggregate layout and field offsets for a struct
// field slice containing or directly carrying structs.HostLayout.
func (s *SourceSizes) HostFieldsLayout(fields []*gotypes.Var) (LayoutPair, []int64) {
	host := false
	for _, f := range fields {
		if IsHostLayoutMarker(f.Type()) {
			host = true
			break
		}
	}
	fieldLayout := func(i int) LayoutPair {
		t := fields[i].Type()
		if host {
			return s.HostFieldLayout(t, false)
		}
		return s.hostNestedLayout(t)
	}
	return AggregateLayout(len(fields), fieldLayout, host || !s.HostUsesNativeGoLayout())
}

// HostFieldLayout computes the target layout of a direct field inside a
// structs.HostLayout struct.
func (s *SourceSizes) HostFieldLayout(t gotypes.Type, native bool) LayoutPair {
	native = native || s.hasNativeDirective(t)
	word := int64(s.spec.WordSize)
	switch u := t.Underlying().(type) {
	case *gotypes.Basic:
		if u.Kind() == gotypes.Uintptr || u.Kind() == gotypes.UnsafePointer {
			return LayoutPair{Size: word, Align: word}
		}
		if u.Kind() != gotypes.String {
			return s.HostBasicLayout(u.Kind())
		}
	case *gotypes.Pointer, *gotypes.Map, *gotypes.Chan:
		return LayoutPair{Size: word, Align: word}
	case *gotypes.Signature:
		if native {
			return LayoutPair{Size: word, Align: word}
		}
	case *gotypes.Array:
		return ArrayLayout(s.HostFieldLayout(u.Elem(), native), u.Len())
	}
	return s.hostNestedLayout(t)
}

func (s *SourceSizes) hostNestedLayout(t gotypes.Type) LayoutPair {
	if !s.HostUsesNativeGoLayout() {
		return LayoutPair{Size: s.Sizeof(t), Align: s.Alignof(t)}
	}
	return s.hostNativeLayout(t, false)
}

func (s *SourceSizes) hostNativeLayout(t gotypes.Type, native bool) LayoutPair {
	native = native || s.hasNativeDirective(t)
	if l, ok := s.HostLayout(t); ok {
		return l
	}
	if l, ok := s.VectorLayout(t, native); ok {
		return l
	}
	word := int64(s.spec.WordSize)
	switch u := t.Underlying().(type) {
	case *gotypes.Basic:
		if u.Kind() == gotypes.String {
			return LayoutPair{Size: 2 * word, Align: word}
		}
		if u.Kind() > gotypes.Invalid && u.Kind() <= gotypes.UnsafePointer {
			return s.HostBasicLayout(u.Kind())
		}
	case *gotypes.Pointer, *gotypes.Chan, *gotypes.Map:
		return LayoutPair{Size: word, Align: word}
	case *gotypes.Interface:
		return LayoutPair{Size: 2 * word, Align: word}
	case *gotypes.Slice:
		return LayoutPair{Size: 3 * word, Align: word}
	case *gotypes.Signature:
		if native {
			return LayoutPair{Size: word, Align: word}
		}
		return LayoutPair{Size: 2 * word, Align: word}
	case *gotypes.Array:
		return ArrayLayout(s.hostNativeLayout(u.Elem(), native), u.Len())
	case *gotypes.Struct:
		native = native || s.isClosure(u)
		fields := make([]*gotypes.Var, u.NumFields())
		for i := range fields {
			fields[i] = u.Field(i)
		}
		native = native || s.hasNativeFields(fields)
		l, _ := AggregateLayout(len(fields), func(i int) LayoutPair { return s.hostNativeLayout(fields[i].Type(), native) }, false)
		return l
	}
	base := s.baseSizes()
	return LayoutPair{Size: base.Sizeof(t), Align: base.Alignof(t)}
}

// WasmNativeLayout computes explicit C/native storage layout on targets where
// Go word size exceeds physical pointer size (e.g. Wasm32 Memory32).
func (s *SourceSizes) WasmNativeLayout(t gotypes.Type) (LayoutPair, bool) {
	if !s.spec.UsesWideGoStorage() || !s.hasNativeMeta() {
		return LayoutPair{}, false
	}
	switch u := t.Underlying().(type) {
	case *gotypes.Array:
		if _, named := unaliasType(t).(*gotypes.Named); named && s.hasLegacyNativeLayout(t) {
			return ArrayLayout(s.wasmNativeFieldLayout(u.Elem()), u.Len()), true
		}
		if elem, ok := s.WasmNativeLayout(u.Elem()); ok {
			return ArrayLayout(elem, u.Len()), true
		}
	case *gotypes.Struct:
		native := s.hasLegacyNativeLayout(t)
		if !native && u.NumFields() > 0 {
			native = s.hasNativeFields([]*gotypes.Var{u.Field(0)})
		}
		if native {
			layout, _ := s.WasmNativeFieldsLayout(u.NumFields(), u.Field)
			return layout, true
		}
		layout, _, ok := s.WasmGoFieldsLayout(u.NumFields(), u.Field)
		return layout, ok
	}
	return LayoutPair{}, false
}

// WasmNativeFieldsLayout lays out fields of a native C struct on Wasm32.
func (s *SourceSizes) WasmNativeFieldsLayout(n int, field func(int) *gotypes.Var) (LayoutPair, []int64) {
	return AggregateLayout(n, func(i int) LayoutPair {
		return s.wasmNativeFieldLayout(field(i).Type())
	}, false)
}

func (s *SourceSizes) wasmGoFieldLayout(t gotypes.Type) LayoutPair {
	if layout, ok := s.WasmNativeLayout(t); ok {
		return layout
	}
	return LayoutPair{Size: s.goSizeof(t), Align: s.goAlignof(t)}
}

func (s *SourceSizes) wasmNativeFieldLayout(t gotypes.Type) LayoutPair {
	ptr := int64(s.spec.PtrSize)
	switch u := t.Underlying().(type) {
	case *gotypes.Basic:
		if u.Kind() == gotypes.Uintptr || u.Kind() == gotypes.UnsafePointer {
			return LayoutPair{Size: ptr, Align: ptr}
		}
	case *gotypes.Pointer, *gotypes.Signature, *gotypes.Map, *gotypes.Chan:
		return LayoutPair{Size: ptr, Align: ptr}
	case *gotypes.Array:
		return ArrayLayout(s.wasmNativeFieldLayout(u.Elem()), u.Len())
	}
	return s.wasmGoFieldLayout(t)
}

// WasmGoFieldsLayout lays out an ordinary Go struct containing at least one
// native-storage field on Wasm32.
func (s *SourceSizes) WasmGoFieldsLayout(n int, field func(int) *gotypes.Var) (LayoutPair, []int64, bool) {
	for i := 0; i < n; i++ {
		if known, ok := s.WasmNativeLayout(field(i).Type()); ok {
			layout, offsets := AggregateLayout(n, func(j int) LayoutPair {
				if j == i {
					return known
				}
				return s.wasmGoFieldLayout(field(j).Type())
			}, true)
			return layout, offsets, true
		}
	}
	return LayoutPair{}, nil, false
}

// VectorLayout computes target-aligned layout for simd/archsimd vector tags,
// structs, and arrays without materializing LLVM types.
func (s *SourceSizes) VectorLayout(t gotypes.Type, native bool) (LayoutPair, bool) {
	if index, ok := SIMDTagIndex(t); ok {
		return LayoutPair{Size: 0, Align: s.simdAlignments[index]}, true
	}
	switch u := unaliasType(t).(type) {
	case *gotypes.Named:
		if !ContainsVectorStorage(u) {
			return LayoutPair{}, false
		}
		return s.VectorLayout(u.Underlying(), s.hasNativeStructLayout(u))
	case *gotypes.Array:
		elem, changed := s.VectorLayout(u.Elem(), false)
		if !changed {
			return LayoutPair{}, false
		}
		return ArrayLayout(elem, u.Len()), true
	case *gotypes.Struct:
		if !ContainsVectorStorage(u) {
			return LayoutPair{}, false
		}
		fields := make([]*gotypes.Var, u.NumFields())
		for i := range fields {
			fields[i] = u.Field(i)
		}
		layout, _ := s.vectorFieldsLayout(fields, native)
		return layout, true
	}
	return LayoutPair{}, false
}

// VectorStructLayout computes layout and offsets for a struct field slice when
// at least one field contains simd/archsimd vector storage.
func (s *SourceSizes) VectorStructLayout(fields []*gotypes.Var, native bool) (LayoutPair, []int64, bool) {
	changed := false
	for _, field := range fields {
		if ContainsVectorStorage(field.Type()) {
			changed = true
			break
		}
	}
	if !changed {
		return LayoutPair{}, nil, false
	}
	layout, offsets := s.vectorFieldsLayout(fields, native)
	return layout, offsets, true
}

func (s *SourceSizes) vectorFieldsLayout(fields []*gotypes.Var, native bool) (LayoutPair, []int64) {
	layouts := make([]LayoutPair, len(fields))
	changedFields := make([]bool, len(fields))
	for i, field := range fields {
		layout, ok := s.VectorLayout(field.Type(), native)
		layouts[i], changedFields[i] = layout, ok
	}
	ptr := int64(s.spec.PtrSize)
	for i, field := range fields {
		if !changedFields[i] {
			layout := LayoutPair{Size: s.Sizeof(field.Type()), Align: s.Alignof(field.Type())}
			if native {
				if _, ok := field.Type().Underlying().(*gotypes.Signature); ok {
					layout.Size, layout.Align = ptr, ptr
				}
			}
			layouts[i] = layout
		}
	}
	return AggregateLayout(len(layouts), func(i int) LayoutPair { return layouts[i] }, !native)
}
