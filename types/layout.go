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

// Sizes specifies the sizing functions for LLGo Types (mirroring go/types.Sizes).
type Sizes interface {
	Alignof(T Type) int64
	Offsetsof(fields []*Var) []int64
	Sizeof(T Type) int64
}

// SizesFor returns a Sizes implementation (backed by *Context) for the given
// compiler ("llgo" or "gc") and GOARCH (mirroring go/types.SizesFor).
func SizesFor(compiler, arch string) Sizes {
	if compiler != "llgo" && compiler != "gc" {
		return nil
	}
	spec := target.ResolveSpec(target.Spec{GOARCH: arch})
	if spec.WordSize == 0 {
		return nil
	}
	return NewContext(spec)
}

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
	for _, f := range fields {
		if TypeIsHostLayoutMarker(f.typ) {
			st.hostLayout = true
			break
		}
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
	needCheck := len(c.incompleteNamed) > 0
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
		if u.hasCallConv && u.callConv.IsNative() {
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
		if u.simdTagIndex >= 0 {
			return &Layout{
				Size:            0,
				Align:           c.SIMDAlignment(u.simdTagIndex),
				IsRegularMemory: true,
				HasSIMDStorage:  true,
			}
		}
		if u.hasCallConv && u.callConv.IsNative() {
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
		if u.nativeLayout && c.spec.UsesWideGoStorage() {
			return c.layoutInMode(u.underlying, modeWasmNative)
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
				l.GCBitmap = GCBitmap(l.PtrOffsets, l.PtrBytes, ptr)
			}
			return l
		}
		if mode == modeHostField {
			if u.kind == Uintptr || u.kind == UnsafePointer {
				l := &Layout{Size: word, Align: word, IsRegularMemory: true}
				if u.kind == UnsafePointer {
					l.PtrOffsets = []int64{0}
					l.PtrBytes = word
					l.GCBitmap = GCBitmap(l.PtrOffsets, l.PtrBytes, word)
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
				l.GCBitmap = GCBitmap(l.PtrOffsets, l.PtrBytes, word)
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
			l.GCBitmap = GCBitmap(l.PtrOffsets, l.PtrBytes, word)
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
			l.GCBitmap = GCBitmap(l.PtrOffsets, l.PtrBytes, sz)
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
		l.GCBitmap = GCBitmap(l.PtrOffsets, l.PtrBytes, sz)
		return l

	case *Slice:
		l := &Layout{
			Size:            3 * word,
			Align:           c.alignForMode(word, mode),
			PtrOffsets:      []int64{0},
			PtrBytes:        word,
			IsRegularMemory: false,
		}
		l.GCBitmap = GCBitmap(l.PtrOffsets, l.PtrBytes, word)
		return l

	case *Interface:
		l := &Layout{
			Size:            2 * word,
			Align:           c.alignForMode(word, mode),
			PtrOffsets:      []int64{0, word},
			PtrBytes:        2 * word,
			IsRegularMemory: false,
		}
		l.GCBitmap = GCBitmap(l.PtrOffsets, l.PtrBytes, word)
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
			l.GCBitmap = GCBitmap(l.PtrOffsets, l.PtrBytes, word)
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
		pair := ArrayLayout(LayoutPair{Size: elemL.Size, Align: elemL.Align}, u.length)
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
				l.GCBitmap = GCBitmap(offsets, l.PtrBytes, word)
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

func (c *Context) computeStructLayout(st *Struct, mode layoutMode) *Layout {
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
			GCBitmap:        GCBitmap(ptrOff, 2*word, word),
			IsRegularMemory: false,
			Fields: []FieldLayout{
				{Offset: 0, Size: word, Align: al, WasmWideSlot: c.spec.UsesWideGoStorage()},
				{Offset: word, Size: word, Align: al, WasmWideSlot: c.spec.UsesWideGoStorage()},
			},
			HasWasmWideSlots: c.spec.UsesWideGoStorage(),
		}
	}

	// Determine effective field layout mode and tail zero-sized field padding.
	effectiveMode := mode
	hasHost := st.hostLayout
	padTail := true

	switch {
	case st.nativeLayout && c.spec.UsesWideGoStorage():
		effectiveMode = modeWasmNative
		padTail = false
	case mode == modeWasmNative:
		if !st.nativeLayout {
			effectiveMode = modeGo
			padTail = true
		} else {
			padTail = false
		}
	case st.hostLayout && !st.nativeLayout:
		effectiveMode = modeHostField
		padTail = true
	case mode == modeHostNative:
		effectiveMode = modeHostNative
		padTail = false
	case !st.nativeLayout && c.HostUsesNativeGoLayout() && TypeContainsHostLayout(st):
		effectiveMode = modeHostNative
		padTail = false
	default:
		if st.nativeLayout && TypeContainsSIMDStorage(st) {
			padTail = false
		}
	}

	n := len(st.fields)
	fieldLayouts := make([]*Layout, n)
	wasmWide := false
	hasSIMD := false
	for i := 0; i < n; i++ {
		ft := st.fields[i].typ
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

	if st.nativeLayout && hasSIMD {
		padTail = false
	}

	return c.assembleAggregate(fieldLayouts, st.fields, padTail, hasHost, hasSIMD, wasmWide)
}

func (c *Context) layoutHostField(t Type) *Layout {
	if t == nil {
		return &Layout{Align: 1, IsRegularMemory: true}
	}
	switch u := Unalias(t).(type) {
	case *Named:
		u.resolve()
		if u.simdTagIndex >= 0 || u.hasCallConv && u.callConv.IsNative() {
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
		pair := ArrayLayout(LayoutPair{Size: elemL.Size, Align: elemL.Align}, u.length)
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
	pair, offsets := AggregateLayout(n, func(i int) LayoutPair {
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
		GCBitmap:         GCBitmap(ptrOffsets, ptrBytes, word),
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
	native := c.hasNativeFieldsLocked(fields)
	if !native && n > 0 && c.hasNativeFieldsLocked(fields[:1]) {
		native = true
	}
	host := false
	containsHost := false
	for _, f := range fields {
		ft := f.Type()
		// Never call Underlying() on pointer/slice/map/chan/sig fields!
		switch ft.(type) {
		case *gotypes.Basic, *gotypes.Pointer, *gotypes.Slice, *gotypes.Map, *gotypes.Chan, *gotypes.Signature, *gotypes.Interface:
			continue
		}
		if IsHostLayoutMarker(ft) {
			host = true
			containsHost = true
			break
		}
		if !containsHost && ContainsHostLayout(ft) {
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
		size = AlignSize(size, fl.Align)
		offsets[i] = size
		if size < 0 || fl.Size < 0 || fl.Size > math.MaxInt64-size {
			size = -1
		} else {
			size += fl.Size
		}
	}
	return offsets
}
