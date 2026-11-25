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
	"sync"

	"github.com/xgo-dev/llgo/target"
)

var cachedSizes sync.Map // map[string]Sizes

// Sizes specifies the sizing functions for LLGo Types (mirroring go/types.Sizes).
type Sizes interface {
	Alignof(T Type) int64
	Offsetsof(fields []*Var) []int64
	Sizeof(T Type) int64
}

// archSizes is an immutable, stateless Sizes implementation returned by
// SizesFor("gc"|"llgo", arch). Keeping it stateless avoids retaining per-package
// type/layout caches across independent compilations while preserving pointer
// identity (SizesFor("gc", arch) == SizesFor("gc", arch)).
type archSizes struct {
	spec             target.Spec
	hostBasicLayouts [UnsafePointer + 1]LayoutPair
	simdAlignments   [3]int64
}

func newArchSizes(spec target.Spec) *archSizes {
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	if spec.MaxAlign == 0 {
		spec.MaxAlign = spec.WordSize
	}
	return &archSizes{
		spec:             spec,
		hostBasicLayouts: defaultHostBasicLayouts(spec),
		simdAlignments:   defaultSIMDAlignments(spec),
	}
}

func (s *archSizes) statelessContext() Context {
	return Context{
		spec:             s.spec,
		hostBasicLayouts: s.hostBasicLayouts,
		simdAlignments:   s.simdAlignments,
	}
}

func (s *archSizes) Sizeof(T Type) int64 {
	if T == nil {
		return 0
	}
	ctx := s.statelessContext()
	return ctx.layoutInMode(T, modeGo).Size
}

func (s *archSizes) Alignof(T Type) int64 {
	if T == nil {
		return 1
	}
	ctx := s.statelessContext()
	return ctx.layoutInMode(T, modeGo).Align
}

func (s *archSizes) Offsetsof(fields []*Var) []int64 {
	if len(fields) == 0 {
		return nil
	}
	ctx := s.statelessContext()
	return ctx.Offsetsof(fields)
}

// isSyncAtomicAlign64 reports whether t is sync/atomic.align64 or
// internal/runtime/atomic.align64, the empty marker struct recognized by the
// Go compiler and go/types as requiring 8-byte alignment even on 32-bit targets.
func isSyncAtomicAlign64(t Type) bool {
	named, ok := Unalias(t).(*Named)
	if !ok || named == nil {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Name() != "align64" || obj.Pkg() == nil {
		return false
	}
	path := obj.Pkg().Path()
	if path != "sync/atomic" && path != "internal/runtime/atomic" {
		return false
	}
	st, ok := named.Underlying().(*Struct)
	return ok && len(st.fields) == 0
}

// SizesFor returns a stateless Sizes implementation for the given
// compiler ("llgo" or "gc") and GOARCH (mirroring go/types.SizesFor).
func SizesFor(compiler, arch string) Sizes {
	if compiler == "gccgo" {
		if gs := gotypes.SizesFor(compiler, arch); gs != nil {
			key := compiler + ":" + arch
			if v, ok := cachedSizes.Load(key); ok {
				return v.(Sizes)
			}
			var s Sizes
			if std, ok := gs.(*gotypes.StdSizes); ok {
				s = &StdSizes{WordSize: std.WordSize, MaxAlign: std.MaxAlign}
			} else {
				s = &StdSizes{
					WordSize: gs.Sizeof(gotypes.Typ[gotypes.Int]),
					MaxAlign: gs.Alignof(gotypes.Typ[gotypes.Complex128]),
				}
			}
			actual, _ := cachedSizes.LoadOrStore(key, s)
			return actual.(Sizes)
		}
		return nil
	}
	if compiler != "llgo" && compiler != "gc" {
		return nil
	}
	key := compiler + ":" + arch
	if v, ok := cachedSizes.Load(key); ok {
		return v.(Sizes)
	}
	spec := target.ResolveSpec(target.Spec{GOARCH: arch})
	if spec.WordSize == 0 {
		return nil
	}
	if compiler == "gc" {
		if gs := gotypes.SizesFor("gc", arch); gs != nil {
			spec.WordSize = int(gs.Sizeof(gotypes.Typ[gotypes.Int]))
			spec.MaxAlign = int(gs.Alignof(gotypes.Typ[gotypes.Complex128]))
		}
	}
	s := newArchSizes(spec)
	actual, _ := cachedSizes.LoadOrStore(key, s)
	return actual.(Sizes)
}

// StdSizes is a convenience type for creating commonly used Sizes
// (mirroring go/types.StdSizes).
type StdSizes struct {
	WordSize int64 // word size in bytes - must be >= 4 (32bits)
	MaxAlign int64 // maximum alignment in bytes - must be >= 1
}

// Alignof implements Sizes on *StdSizes.
func (s *StdSizes) Alignof(T Type) int64 {
	if T == nil {
		return 1
	}
	switch t := T.Underlying().(type) {
	case *Array:
		return s.Alignof(t.elem)
	case *Struct:
		if len(t.fields) == 0 && isSyncAtomicAlign64(T) {
			return 8
		}
		max := int64(1)
		for _, f := range t.fields {
			if f != nil {
				if a := s.Alignof(f.typ); a > max {
					max = a
				}
			}
		}
		return max
	case *Slice, *Interface:
		return s.WordSize
	case *Basic:
		if t.info&IsString != 0 {
			return s.WordSize
		}
	}
	a := s.Sizeof(T)
	if a < 1 {
		return 1
	}
	if b, ok := T.Underlying().(*Basic); ok && b.info&IsComplex != 0 {
		a /= 2
	}
	if a > s.MaxAlign {
		return s.MaxAlign
	}
	return a
}

// Offsetsof implements Sizes on *StdSizes.
func (s *StdSizes) Offsetsof(fields []*Var) []int64 {
	offsets := make([]int64, len(fields))
	var o int64
	for i, f := range fields {
		a := s.Alignof(f.typ)
		o = alignSize(o, a)
		offsets[i] = o
		sz := s.Sizeof(f.typ)
		if o < 0 || sz < 0 || sz > math.MaxInt64-o {
			o = -1
		} else {
			o += sz
		}
	}
	return offsets
}

// Sizeof implements Sizes on *StdSizes.
func (s *StdSizes) Sizeof(T Type) int64 {
	if T == nil {
		return 0
	}
	switch t := T.Underlying().(type) {
	case *Basic:
		switch t.kind {
		case Bool, Int8, Uint8:
			return 1
		case Int16, Uint16:
			return 2
		case Int32, Uint32, Float32:
			return 4
		case Int64, Uint64, Float64, Complex64:
			return 8
		case Complex128:
			return 16
		case Int, Uint, Uintptr, UnsafePointer:
			return s.WordSize
		case String:
			return 2 * s.WordSize
		default:
			return 0
		}
	case *Array:
		n := t.length
		if n <= 0 {
			return 0
		}
		esize := s.Sizeof(t.elem)
		if esize < 0 {
			return -1
		}
		if esize == 0 {
			return 0
		}
		a := s.Alignof(t.elem)
		esize = alignSize(esize, a)
		if esize < 0 || n > math.MaxInt64/esize {
			return -1
		}
		return esize * n
	case *Slice:
		return 3 * s.WordSize
	case *Struct:
		n := len(t.fields)
		if n == 0 {
			return 0
		}
		offsets := s.Offsetsof(t.fields)
		last := offsets[n-1]
		if last < 0 {
			return -1
		}
		lastSz := s.Sizeof(t.fields[n-1].typ)
		if lastSz < 0 || lastSz > math.MaxInt64-last {
			return -1
		}
		size := last + lastSz
		if lastSz == 0 && size > 0 {
			if size == math.MaxInt64 {
				return -1
			}
			size++
		}
		return alignSize(size, s.Alignof(t))
	case *Interface:
		return 2 * s.WordSize
	case *Signature:
		if t.callConv.IsClosure() {
			return 2 * s.WordSize
		}
		return s.WordSize
	case *SIMDVec:
		return int64(basicBits(t.elem)/8) * int64(t.length)
	default:
		return s.WordSize
	}
}

// LayoutPair holds an overflow-aware (Size, Align) pair computed without LLVM
// TargetData. Size == -1 signals int64 overflow matching go/types.Sizes.
type LayoutPair struct {
	Size  int64
	Align int64
}

// alignSize rounds size up to a multiple of alignment, returning -1 on int64
// overflow or when size is already negative.
func alignSize(size, alignment int64) int64 {
	if alignment <= 1 {
		return size
	}
	if size < 0 || size > math.MaxInt64-(alignment-1) {
		return -1
	}
	return (size + alignment - 1) &^ (alignment - 1)
}

// arrayLayout computes the overflow-aware layout of [n]elem.
func arrayLayout(elem LayoutPair, n int64) LayoutPair {
	size := int64(0)
	if n > 0 {
		size = -1
		if elem.Size >= 0 && (elem.Size == 0 || n <= math.MaxInt64/elem.Size) {
			size = elem.Size * n
		}
	}
	return LayoutPair{Size: size, Align: elem.Align}
}

// aggregateLayout computes struct/tuple size, alignment, and per-field offsets
// with overflow protection and optional Go trailing zero-sized field padding.
func aggregateLayout(numFields int, fieldLayout func(int) LayoutPair, padTail bool) (LayoutPair, []int64) {
	result := LayoutPair{Align: 1}
	offsets := make([]int64, numFields)
	for i := 0; i < numFields; i++ {
		field := fieldLayout(i)
		if field.Align > result.Align {
			result.Align = field.Align
		}
		result.Size = alignSize(result.Size, field.Align)
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
	result.Size = alignSize(result.Size, result.Align)
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

type layoutMode uint8

const (
	modeGo layoutMode = iota
	modeHostField
	modeHostNative
	modeWasmNative
)

// Sizeof implements Sizes on *Context.
func (c *Context) Sizeof(T Type) int64 { return c.Layout(T).Size }

// Alignof implements Sizes on *Context.
func (c *Context) Alignof(T Type) int64 { return c.Layout(T).Align }

// Offsetsof implements Sizes on *Context.
func (c *Context) Offsetsof(fields []*Var) []int64 {
	if len(fields) == 0 {
		return nil
	}
	st := NewStruct(fields, nil)
	if c.hasNativeLLFields(fields) {
		st.nativeLayout = true
	}
	lay := c.computeStructLayout(st, modeGo)
	out := make([]int64, len(lay.Fields))
	for i, f := range lay.Fields {
		out[i] = f.Offset
	}
	return out
}

// HostUsesNativeGoLayout reports whether ordinary Go aggregates on this target
// already use native C alignment for 64-bit scalars (e.g. 32-bit ARM/MIPS/RISC-V).
func (c *Context) HostUsesNativeGoLayout() bool {
	arch := c.spec.GOARCH
	return arch != "386" && arch != "wasm" && c.spec.WordSize == 4 && c.hostBasicLayouts[Int64].Align > 4
}

// HostBasicLayout returns the target host C ABI (size, align) for basic kind.
func (c *Context) HostBasicLayout(kind BasicKind) LayoutPair {
	if kind >= 0 && int(kind) < len(c.hostBasicLayouts) {
		return c.hostBasicLayouts[kind]
	}
	return LayoutPair{}
}

// SIMDAlignment returns the target ABI alignment for a simd/archsimd tag index
// (0=v128, 1=v256, 2=v512).
func (c *Context) SIMDAlignment(index int) int64 {
	if index >= 0 && index < len(c.simdAlignments) {
		return c.simdAlignments[index]
	}
	return 1
}

// HostFieldLayout computes the target layout of a direct field inside a
// structs.HostLayout struct using the single layout engine.
func (c *Context) HostFieldLayout(t Type) LayoutPair {
	l := c.layoutHostField(t)
	return LayoutPair{Size: l.Size, Align: l.Align}
}

// Layout computes and caches the full immutable Layout for t.
func (c *Context) Layout(t Type) *Layout {
	if t == nil {
		return &Layout{Align: 1, IsRegularMemory: true}
	}
	c.mu.RLock()
	needCheck := len(c.incompleteNamed) > 0 || len(c.incompleteLLNamed) > 0
	if !needCheck {
		if l, ok := c.layouts[t]; ok {
			c.mu.RUnlock()
			return l
		}
	}
	c.mu.RUnlock()

	if needCheck {
		c.mu.Lock()
		c.checkIncompleteNamedLocked()
		if l, ok := c.layouts[t]; ok {
			c.mu.Unlock()
			return l
		}
		c.mu.Unlock()
	}

	l := c.layoutInMode(t, modeGo)

	c.mu.Lock()
	if existing, ok := c.layouts[t]; ok {
		c.mu.Unlock()
		return existing
	}
	c.layouts[t] = l
	c.mu.Unlock()
	return l
}

func (c *Context) clampAlign(a int64) int64 {
	maxAlign := int64(c.spec.MaxAlign)
	if maxAlign <= 0 {
		maxAlign = int64(c.spec.WordSize)
	}
	if a > maxAlign {
		return maxAlign
	}
	if a < 1 {
		return 1
	}
	return a
}

func (c *Context) basicLayoutGo(kind BasicKind) LayoutPair {
	word := int64(c.spec.WordSize)
	switch kind {
	case Bool, Int8, Uint8:
		return LayoutPair{Size: 1, Align: 1}
	case Int16, Uint16:
		return LayoutPair{Size: 2, Align: c.clampAlign(2)}
	case Int32, Uint32, Float32:
		return LayoutPair{Size: 4, Align: c.clampAlign(4)}
	case Int64, Uint64, Float64:
		return LayoutPair{Size: 8, Align: c.clampAlign(8)}
	case Complex64:
		return LayoutPair{Size: 8, Align: c.clampAlign(4)}
	case Complex128:
		return LayoutPair{Size: 16, Align: c.clampAlign(8)}
	case Int, Uint, Uintptr, UnsafePointer:
		return LayoutPair{Size: word, Align: c.clampAlign(word)}
	case String:
		return LayoutPair{Size: 2 * word, Align: c.clampAlign(word)}
	default:
		return LayoutPair{Size: 0, Align: 1}
	}
}

func (c *Context) layoutInMode(t Type, mode layoutMode) *Layout {
	if t == nil {
		return &Layout{Align: 1, IsRegularMemory: true}
	}
	word := int64(c.spec.WordSize)
	ptr := int64(c.spec.PtrSize)

	switch u := t.(type) {
	case *Alias:
		if (u.hasCallConv && u.callConv.IsNative()) || c.isNativeType(u) {
			if _, ok := u.Underlying().(*Signature); ok {
				sz := word
				if mode == modeWasmNative {
					sz = ptr
				}
				return &Layout{
					Size:            sz,
					Align:           c.alignForMode(sz, mode),
					IsRegularMemory: false,
				}
			}
		}
		if u.rhs == nil || u.rhs == u {
			return &Layout{Align: 1, IsRegularMemory: true}
		}
		return c.layoutInMode(u.rhs, mode)

	case *Named:
		u.resolve()
		if u.goNamed == nil {
			if u.underlying == nil {
				c.trackIncompleteLLNamed(u)
			} else if st, ok := u.underlying.(*Struct); ok && len(st.fields) == 0 {
				c.trackIncompleteLLNamed(u)
			}
		}
		if u.simdTagIndex >= 0 {
			return &Layout{
				Size:            0,
				Align:           c.SIMDAlignment(u.simdTagIndex),
				IsRegularMemory: true,
				HasSIMDStorage:  true,
			}
		}
		if isSyncAtomicAlign64(u) {
			return &Layout{
				Size:            0,
				Align:           8,
				IsRegularMemory: true,
			}
		}
		nativeNamed := u.hasCallConv && u.callConv.IsNative()
		if !nativeNamed && c.hasNativeMetadata.Load() {
			nativeNamed = c.isNativeType(u)
		}
		if nativeNamed {
			if _, ok := u.Underlying().(*Signature); ok {
				sz := word
				if mode == modeWasmNative {
					sz = ptr
				}
				return &Layout{
					Size:            sz,
					Align:           c.alignForMode(sz, mode),
					IsRegularMemory: false,
				}
			}
		}
		isNativeLayout := u.nativeLayout || nativeNamed
		if isNativeLayout {
			if st, ok := u.underlying.(*Struct); ok {
				return c.computeStructLayoutWithNative(st, mode, true)
			}
			if arr, ok := u.underlying.(*Array); ok && c.spec.UsesWideGoStorage() {
				return c.layoutInMode(arr, modeWasmNative)
			}
		}
		if u.underlying == nil || u.underlying == u {
			return &Layout{Align: 1, IsRegularMemory: true}
		}
		return c.layoutInMode(u.underlying, mode)

	case *Basic:
		if mode == modeWasmNative && (u.kind == Uintptr || u.kind == UnsafePointer) {
			l := &Layout{
				Size:            ptr,
				Align:           ptr,
				IsRegularMemory: true,
			}
			if u.kind == UnsafePointer {
				l.PtrOffsets = []int64{0}
				l.PtrBytes = ptr
				l.GCBitmap = gcBitmap(l.PtrOffsets, l.PtrBytes, ptr)
			}
			return l
		}
		if mode == modeHostField {
			if u.kind == Uintptr || u.kind == UnsafePointer {
				l := &Layout{Size: word, Align: word, IsRegularMemory: true}
				if u.kind == UnsafePointer {
					l.PtrOffsets = []int64{0}
					l.PtrBytes = word
					l.GCBitmap = gcBitmap(l.PtrOffsets, l.PtrBytes, word)
				}
				return l
			}
			if u.kind != String && u.kind > Invalid && u.kind <= UnsafePointer {
				p := c.HostBasicLayout(u.kind)
				return &Layout{
					Size:            p.Size,
					Align:           p.Align,
					IsRegularMemory: isRegularBasic(u.kind),
				}
			}
		}
		if mode == modeHostNative && u.kind != String && u.kind > Invalid && u.kind <= UnsafePointer {
			p := c.HostBasicLayout(u.kind)
			l := &Layout{
				Size:            p.Size,
				Align:           p.Align,
				IsRegularMemory: isRegularBasic(u.kind),
			}
			if u.kind == UnsafePointer {
				l.PtrOffsets = []int64{0}
				l.PtrBytes = p.Size
				l.GCBitmap = gcBitmap(l.PtrOffsets, l.PtrBytes, word)
			}
			return l
		}
		p := c.basicLayoutGo(u.kind)
		l := &Layout{
			Size:            p.Size,
			Align:           p.Align,
			IsRegularMemory: isRegularBasic(u.kind),
		}
		if u.kind == String || u.kind == UnsafePointer {
			l.PtrOffsets = []int64{0}
			l.PtrBytes = word
			l.GCBitmap = gcBitmap(l.PtrOffsets, l.PtrBytes, word)
		}
		return l

	case *Pointer:
		sz := word
		if mode == modeWasmNative {
			sz = ptr
		}
		l := &Layout{
			Size:            sz,
			Align:           c.alignForMode(sz, mode),
			IsRegularMemory: true,
		}
		if u.kind != CPtr {
			l.PtrOffsets = []int64{0}
			l.PtrBytes = sz
			l.GCBitmap = gcBitmap(l.PtrOffsets, l.PtrBytes, sz)
		}
		return l

	case *Map, *Chan:
		sz := word
		if mode == modeWasmNative {
			sz = ptr
		}
		_, isChan := u.(*Chan)
		l := &Layout{
			Size:            sz,
			Align:           c.alignForMode(sz, mode),
			PtrOffsets:      []int64{0},
			PtrBytes:        sz,
			IsRegularMemory: isChan,
		}
		l.GCBitmap = gcBitmap(l.PtrOffsets, l.PtrBytes, sz)
		return l

	case *Slice:
		l := &Layout{
			Size:            3 * word,
			Align:           c.alignForMode(word, mode),
			PtrOffsets:      []int64{0},
			PtrBytes:        word,
			IsRegularMemory: false,
		}
		l.GCBitmap = gcBitmap(l.PtrOffsets, l.PtrBytes, word)
		return l

	case *Interface:
		l := &Layout{
			Size:            2 * word,
			Align:           c.alignForMode(word, mode),
			PtrOffsets:      []int64{0, word},
			PtrBytes:        2 * word,
			IsRegularMemory: false,
		}
		l.GCBitmap = gcBitmap(l.PtrOffsets, l.PtrBytes, word)
		return l

	case *Signature:
		if mode == modeWasmNative {
			return &Layout{
				Size:            ptr,
				Align:           ptr,
				IsRegularMemory: false,
			}
		}
		if u.callConv.IsClosure() {
			l := &Layout{
				Size:            2 * word,
				Align:           c.alignForMode(word, mode),
				PtrOffsets:      []int64{word},
				PtrBytes:        2 * word,
				IsRegularMemory: false,
			}
			l.GCBitmap = gcBitmap(l.PtrOffsets, l.PtrBytes, word)
			return l
		}
		return &Layout{
			Size:            word,
			Align:           c.alignForMode(word, mode),
			IsRegularMemory: false,
		}

	case *SIMDVec:
		bits := int64(basicBits(u.elem))
		sz := (bits / 8) * int64(u.length)
		return &Layout{
			Size:            sz,
			Align:           c.SIMDAlignment(u.tagIndex),
			IsRegularMemory: isRegularBasic(u.elem.kind),
			HasSIMDStorage:  true,
		}

	case *Array:
		elemMode := mode
		if u.nativeStorage && c.spec.UsesWideGoStorage() {
			elemMode = modeWasmNative
		}
		elemL := c.layoutInMode(u.elem, elemMode)
		pair := arrayLayout(LayoutPair{Size: elemL.Size, Align: elemL.Align}, u.length)
		l := &Layout{
			Size:             pair.Size,
			Align:            pair.Align,
			HasWasmWideSlots: elemL.HasWasmWideSlots,
			HasHostLayout:    elemL.HasHostLayout,
			HasSIMDStorage:   elemL.HasSIMDStorage,
			IsRegularMemory:  elemL.IsRegularMemory || u.length == 0,
		}
		if u.length > 0 && elemL.PtrBytes > 0 && elemL.Size >= 0 && pair.Size >= 0 {
			l.PtrBytes = (u.length-1)*elemL.Size + elemL.PtrBytes
			if len(elemL.PtrOffsets) > 0 && u.length <= 1024 {
				offsets := make([]int64, 0, int(u.length)*len(elemL.PtrOffsets))
				for i := int64(0); i < u.length; i++ {
					base := i * elemL.Size
					for _, off := range elemL.PtrOffsets {
						offsets = append(offsets, base+off)
					}
				}
				l.PtrOffsets = offsets
				l.GCBitmap = gcBitmap(offsets, l.PtrBytes, word)
			}
		}
		return l

	case *Struct:
		return c.computeStructLayout(u, mode)

	case *Tuple:
		n := u.Len()
		fieldLayouts := make([]*Layout, n)
		for i := 0; i < n; i++ {
			fieldLayouts[i] = c.layoutInMode(u.At(i).typ, mode)
		}
		return c.assembleAggregate(fieldLayouts, nil, false, false, false)

	case *Union:
		return &Layout{Align: 1, IsRegularMemory: true}

	case *TypeParam:
		return &Layout{
			Size:            word,
			Align:           c.clampAlign(word),
			PtrOffsets:      []int64{0},
			PtrBytes:        word,
			IsRegularMemory: false,
		}
	}

	return &Layout{Align: 1, IsRegularMemory: true}
}

func (c *Context) alignForMode(a int64, mode layoutMode) int64 {
	if mode == modeWasmNative || mode == modeHostField || mode == modeHostNative {
		if a < 1 {
			return 1
		}
		return a
	}
	return c.clampAlign(a)
}

func (c *Context) isNativeType(t Type) bool {
	if !c.hasNativeMetadata.Load() {
		return false
	}
	cc, ok := c.NativeTypeCallConv(t)
	return ok && cc.IsNative()
}

func (c *Context) computeStructLayout(st *Struct, mode layoutMode) *Layout {
	return c.computeStructLayoutWithNative(st, mode, false)
}

func (c *Context) computeStructLayoutWithNative(st *Struct, mode layoutMode, forceNative bool) *Layout {
	if st.simdTagIndex >= 0 {
		return &Layout{
			Size:            0,
			Align:           c.SIMDAlignment(st.simdTagIndex),
			IsRegularMemory: true,
			HasSIMDStorage:  true,
		}
	}

	// Legacy synthetic closure struct { $f func(...); $data unsafe.Pointer }:
	// physical layout is always 2 words {codePtr, envPtr} without needing a
	// synthetic non-Go CallConv.
	if st.isClosureStruct {
		word := int64(c.spec.WordSize)
		al := c.alignForMode(word, mode)
		ptrOff := []int64{word}
		return &Layout{
			Size:            2 * word,
			Align:           al,
			PtrBytes:        2 * word,
			PtrOffsets:      ptrOff,
			GCBitmap:        gcBitmap(ptrOff, 2*word, word),
			IsRegularMemory: false,
			Fields: []FieldLayout{
				{Offset: 0, Size: word, Align: al, WasmWideSlot: c.spec.UsesWideGoStorage()},
				{Offset: word, Size: word, Align: al, WasmWideSlot: c.spec.UsesWideGoStorage()},
			},
			HasWasmWideSlots: c.spec.UsesWideGoStorage(),
		}
	}

	native := forceNative || st.nativeLayout || c.hasNativeLLFields(st.fields)

	// Determine effective field layout mode and tail zero-sized field padding.
	effectiveMode := mode
	hasHost := st.hostLayout
	padTail := true

	switch {
	case native && c.spec.UsesWideGoStorage():
		effectiveMode = modeWasmNative
		padTail = false
	case mode == modeWasmNative:
		if !native {
			effectiveMode = modeGo
			padTail = true
		} else {
			padTail = false
		}
	case st.hostLayout && !native:
		effectiveMode = modeHostField
		padTail = true
	case mode == modeHostNative:
		effectiveMode = modeHostNative
		padTail = false
	case !native && c.HostUsesNativeGoLayout() && st.ContainsHostLayout():
		effectiveMode = modeHostNative
		padTail = false
	default:
		if native && st.ContainsSIMDStorage() {
			padTail = false
		}
	}

	n := len(st.fields)
	fieldLayouts := make([]*Layout, n)
	wasmWide := false
	hasSIMD := false
	for i := 0; i < n; i++ {
		ft := st.fields[i].typ
		if native {
			if sig, ok := Unalias(ft).(*Signature); ok && sig.callConv == Go {
				ft = sig.WithCallConv(CDecl)
			}
		}
		var fl *Layout
		switch effectiveMode {
		case modeHostField:
			fl = c.layoutHostField(ft)
		case modeWasmNative:
			fl = c.layoutInMode(ft, modeWasmNative)
		case modeHostNative:
			fl = c.layoutInMode(ft, modeHostNative)
		default:
			fl = c.layoutInMode(ft, modeGo)
			if c.spec.UsesWideGoStorage() && isWasmWideGoSlot(ft) {
				wasmWide = true
			}
		}
		fieldLayouts[i] = fl
		if fl.HasHostLayout {
			hasHost = true
		}
		if fl.HasSIMDStorage {
			hasSIMD = true
		}
		if fl.HasWasmWideSlots {
			wasmWide = true
		}
	}

	if native && hasSIMD {
		padTail = false
	}

	return c.assembleAggregate(fieldLayouts, st.fields, padTail, hasHost, hasSIMD, wasmWide)
}

func (c *Context) layoutHostField(t Type) *Layout {
	if t == nil {
		return &Layout{Align: 1, IsRegularMemory: true}
	}
	if c.isNativeType(t) {
		if _, isSig := t.Underlying().(*Signature); isSig {
			return c.layoutInMode(t, modeHostField)
		}
	}
	switch u := Unalias(t).(type) {
	case *Named:
		u.resolve()
		if u.simdTagIndex >= 0 || (u.hasCallConv && u.callConv.IsNative()) || c.isNativeType(u) {
			if _, isArr := u.Underlying().(*Array); isArr {
				return c.layoutHostField(u.underlying)
			}
			if _, isSt := u.Underlying().(*Struct); isSt {
				return c.layoutHostNested(t)
			}
			return c.layoutInMode(t, modeHostField)
		}
		if _, ok := u.Underlying().(*Struct); ok {
			return c.layoutHostNested(t)
		}
		if _, ok := u.Underlying().(*Array); ok {
			return c.layoutHostField(u.underlying)
		}
		return c.layoutInMode(t, modeHostField)
	case *Array:
		elemL := c.layoutHostField(u.elem)
		pair := arrayLayout(LayoutPair{Size: elemL.Size, Align: elemL.Align}, u.length)
		return &Layout{
			Size:            pair.Size,
			Align:           pair.Align,
			HasHostLayout:   elemL.HasHostLayout,
			HasSIMDStorage:  elemL.HasSIMDStorage,
			IsRegularMemory: elemL.IsRegularMemory || u.length == 0,
		}
	case *Basic, *Pointer, *Map, *Chan:
		return c.layoutInMode(t, modeHostField)
	case *Signature:
		if u.IsNative() {
			return c.layoutInMode(t, modeHostField)
		}
		return c.layoutHostNested(t)
	default:
		return c.layoutHostNested(t)
	}
}

func (c *Context) layoutHostNested(t Type) *Layout {
	if !c.HostUsesNativeGoLayout() {
		return c.layoutInMode(t, modeGo)
	}
	return c.layoutInMode(t, modeHostNative)
}

func (c *Context) assembleAggregate(fieldLayouts []*Layout, fieldVars []*Var, padTail, hasHost, hasSIMD bool, wasmWideOpt ...bool) *Layout {
	n := len(fieldLayouts)
	pair, offsets := aggregateLayout(n, func(i int) LayoutPair {
		return LayoutPair{Size: fieldLayouts[i].Size, Align: fieldLayouts[i].Align}
	}, padTail)

	word := int64(c.spec.WordSize)
	fields := make([]FieldLayout, n)
	var ptrOffsets []int64
	var ptrBytes int64
	regular := true
	wasmWide := len(wasmWideOpt) > 0 && wasmWideOpt[0]

	for i := 0; i < n; i++ {
		fl := fieldLayouts[i]
		off := offsets[i]
		var padAfter int64
		if pair.Size >= 0 && off >= 0 && fl.Size >= 0 {
			nextOff := pair.Size
			if i+1 < n && offsets[i+1] >= 0 {
				nextOff = offsets[i+1]
			}
			padAfter = nextOff - (off + fl.Size)
			if padAfter < 0 {
				padAfter = 0
			}
		}
		slotWide := false
		if fieldVars != nil && i < len(fieldVars) && c.spec.UsesWideGoStorage() && isWasmWideGoSlot(fieldVars[i].typ) {
			slotWide = true
		}
		fields[i] = FieldLayout{
			Offset:       off,
			Size:         fl.Size,
			Align:        fl.Align,
			PaddingAfter: padAfter,
			WasmWideSlot: slotWide,
		}
		if fl.PtrBytes > 0 && off >= 0 {
			if end := off + fl.PtrBytes; end > ptrBytes {
				ptrBytes = end
			}
			for _, po := range fl.PtrOffsets {
				ptrOffsets = append(ptrOffsets, off+po)
			}
		}
		if regular {
			if fieldVars != nil && i < len(fieldVars) && fieldVars[i].name == "_" {
				regular = false
			} else if !fl.IsRegularMemory {
				regular = false
			} else if padAfter > 0 {
				regular = false
			}
		}
		if fl.HasHostLayout {
			hasHost = true
		}
		if fl.HasSIMDStorage {
			hasSIMD = true
		}
		if fl.HasWasmWideSlots {
			wasmWide = true
		}
	}

	return &Layout{
		Size:             pair.Size,
		Align:            pair.Align,
		PtrBytes:         ptrBytes,
		PtrOffsets:       ptrOffsets,
		GCBitmap:         gcBitmap(ptrOffsets, ptrBytes, word),
		Fields:           fields,
		IsRegularMemory:  regular,
		HasWasmWideSlots: wasmWide,
		HasHostLayout:    hasHost,
		HasSIMDStorage:   hasSIMD,
	}
}

func isWasmWideGoSlot(t Type) bool {
	if t == nil {
		return false
	}
	switch u := t.Underlying().(type) {
	case *Basic:
		return u.kind == Int || u.kind == Uint || u.kind == Uintptr || u.kind == UnsafePointer || u.kind == String
	case *Pointer, *Signature, *Slice, *Interface, *Map, *Chan:
		return true
	}
	return false
}

func isRegularBasic(k BasicKind) bool {
	switch k {
	case Float32, Float64, Complex64, Complex128, String:
		return false
	default:
		return true
	}
}

// GoSizes returns a go/types.Sizes adapter backed directly by Context.Layout
// (single source of truth, zero duplicated layout logic).
func (c *Context) GoSizes() gotypes.Sizes {
	return &goSizesAdapter{ctx: c}
}

type goSizesAdapter struct {
	ctx *Context
}

// LLGoContext allows callers holding a gotypes.Sizes (such as abi.Builder) to
// recover the underlying *Context without allocating a second one.
func (g *goSizesAdapter) LLGoContext() *Context {
	return g.ctx
}

func (g *goSizesAdapter) Alignof(T gotypes.Type) int64 {
	if T == nil {
		return 1
	}
	return g.ctx.Layout(g.ctx.FromGo(T)).Align
}

func (g *goSizesAdapter) Sizeof(T gotypes.Type) int64 {
	if T == nil {
		return 0
	}
	return g.ctx.Layout(g.ctx.FromGo(T)).Size
}

func (g *goSizesAdapter) Offsetsof(fields []*gotypes.Var) []int64 {
	n := len(fields)
	if n == 0 {
		return nil
	}
	c := g.ctx
	c.mu.RLock()
	needCheck := len(c.incompleteNamed) > 0
	c.mu.RUnlock()
	if needCheck {
		c.mu.Lock()
		c.checkIncompleteNamedLocked()
		c.mu.Unlock()
	}

	if isGoClosureFields(fields) {
		return []int64{0, int64(c.spec.WordSize)}
	}
	native := c.hasNativeFields(fields)
	host := false
	containsHost := false
	for _, f := range fields {
		ft := f.Type()
		// Never call Underlying() on pointer/slice/map/chan/sig fields!
		switch ft.(type) {
		case *gotypes.Basic, *gotypes.Pointer, *gotypes.Slice, *gotypes.Map, *gotypes.Chan, *gotypes.Signature, *gotypes.Interface:
			continue
		}
		if goIsHostLayoutMarker(ft) {
			host = true
			containsHost = true
			break
		}
		if !containsHost && goContainsHostLayout(ft) {
			containsHost = true
		}
	}

	mode := modeGo
	switch {
	case native && c.spec.UsesWideGoStorage():
		mode = modeWasmNative
	case host && !native:
		mode = modeHostField
	case !native && containsHost && c.HostUsesNativeGoLayout():
		mode = modeHostNative
	}

	offsets := make([]int64, n)
	var size int64
	for i, f := range fields {
		ft := f.Type()
		var llT Type
		if native {
			if _, isSig := ft.Underlying().(*gotypes.Signature); isSig {
				llT = c.FromGoWithCallConv(ft, CDecl)
			} else {
				llT = c.FromGo(ft)
			}
		} else {
			llT = c.FromGo(ft)
		}
		var fl *Layout
		switch mode {
		case modeGo:
			fl = c.Layout(llT)
		case modeHostField:
			fl = c.layoutHostField(llT)
		default:
			fl = c.layoutInMode(llT, mode)
		}
		size = alignSize(size, fl.Align)
		offsets[i] = size
		if size < 0 || fl.Size < 0 || fl.Size > math.MaxInt64-size {
			size = -1
		} else {
			size += fl.Size
		}
	}
	return offsets
}
