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
	"sync"

	"github.com/xgo-dev/llgo/target"
)

type ptrInternKey struct {
	elem Type
	kind PtrKind
}

type mapInternKey struct {
	key  Type
	elem Type
}

type chanInternKey struct {
	dir  gotypes.ChanDir
	elem Type
}

type arrayInternKey struct {
	elem   Type
	length int64
	native bool
}

type simdInternKey struct {
	kind gotypes.BasicKind
	len  int
	mask bool
}

type basicInternKey struct {
	kind    gotypes.BasicKind
	ptrKind PtrKind
}

// Context is a canonical, target-bound physical type universe and layout cache.
//
// Structurally identical primitive and compound types (Basic, String, empty
// Interface, Pointer, Slice, Map, Chan, Array, SIMDVec) are interned so
// physical identity can be compared directly with ==. Named, Struct, Func,
// Closure, and Tuple types are memoized by (go/types.Type, CallConv) with
// cycle-safe back-edge resolution, and every Type resolves to a cached
// immutable Layout (Size, Align, FieldLayouts, PtrBytes, GCBitmap,
// IsRegularMemory, WasmWideSlots) plus target-aware FuncABI classification.
type Context struct {
	spec           target.Spec
	sizes          *Sizes
	sourceSizes    *SourceSizes
	simdAlignments [3]int64

	mu         sync.RWMutex
	lookup     CallConvLookup
	mapped     map[mapKey]Type
	layouts    map[Type]*Layout
	basics     map[basicInternKey]*Basic
	stringType *String
	anyType    *Interface
	ptrs       map[ptrInternKey]*Pointer
	slices     map[Type]*Slice
	maps       map[mapInternKey]*Map
	chans      map[chanInternKey]*Chan
	arrays     map[arrayInternKey]*Array
	simds      map[simdInternKey]*SIMDVec
}

// NewContext creates a canonical physical Type & Layout Context for spec.
func NewContext(spec target.Spec, lookup CallConvLookup) *Context {
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	simdAligns := DefaultSIMDAlignments(spec)
	c := &Context{
		spec:           spec,
		sizes:          NewSizes(spec),
		simdAlignments: simdAligns,
		lookup:         lookup,
		mapped:         make(map[mapKey]Type),
		layouts:        make(map[Type]*Layout),
		basics:         make(map[basicInternKey]*Basic),
		stringType:     &String{goType: gotypes.Typ[gotypes.String]},
		anyType:        &Interface{Empty: true},
		ptrs:           make(map[ptrInternKey]*Pointer),
		slices:         make(map[Type]*Slice),
		maps:           make(map[mapInternKey]*Map),
		chans:          make(map[chanInternKey]*Chan),
		arrays:         make(map[arrayInternKey]*Array),
		simds:          make(map[simdInternKey]*SIMDVec),
	}
	c.sourceSizes = NewSourceSizes(SourceSizesConfig{
		Spec:           spec,
		SIMDAlignments: simdAligns,
		CallConvLookup: lookup,
	})
	return c
}

// Spec returns the resolved target specification of c.
func (c *Context) Spec() target.Spec { return c.spec }

// Sizes returns the LLGo physical Sizes calculator for c.
func (c *Context) Sizes() *Sizes { return c.sizes }

// SourceSizes returns the unified go/types.Sizes engine bound to c's target
// spec and calling-convention lookup.
func (c *Context) SourceSizes() *SourceSizes { return c.sourceSizes }

// Basic returns the canonically interned Basic type for (kind, ptrKind).
func (c *Context) Basic(kind gotypes.BasicKind, ptrKind PtrKind) *Basic {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.internBasicLocked(kind, ptrKind, nil)
}

func (c *Context) internBasicLocked(kind gotypes.BasicKind, ptrKind PtrKind, goType gotypes.Type) *Basic {
	if kind != gotypes.UnsafePointer {
		ptrKind = GoPtr
	}
	k := basicInternKey{kind: kind, ptrKind: ptrKind}
	if b, ok := c.basics[k]; ok {
		return b
	}
	if goType == nil && kind > gotypes.Invalid && int(kind) < len(gotypes.Typ) {
		goType = gotypes.Typ[kind]
	}
	b := &Basic{Kind: kind, PtrKind: ptrKind, goType: goType}
	c.basics[k] = b
	return b
}

// String returns the canonically interned Go string type.
func (c *Context) String() *String { return c.stringType }

// Any returns the canonically interned empty interface type.
func (c *Context) Any() *Interface { return c.anyType }

// Pointer returns the canonically interned Pointer type for (elem, kind).
func (c *Context) Pointer(elem Type, kind PtrKind) *Pointer {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.internPtrLocked(elem, kind, nil)
}

func (c *Context) internPtrLocked(elem Type, kind PtrKind, goType gotypes.Type) *Pointer {
	k := ptrInternKey{elem: elem, kind: kind}
	if p, ok := c.ptrs[k]; ok {
		return p
	}
	p := &Pointer{Elem: elem, Kind: kind, goType: goType}
	c.ptrs[k] = p
	return p
}

// Slice returns the canonically interned Slice type for elem.
func (c *Context) Slice(elem Type) *Slice {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.internSliceLocked(elem, nil)
}

func (c *Context) internSliceLocked(elem Type, goType gotypes.Type) *Slice {
	if s, ok := c.slices[elem]; ok {
		return s
	}
	s := &Slice{Elem: elem, goType: goType}
	c.slices[elem] = s
	return s
}

// Map returns the canonically interned Map type for (key, elem).
func (c *Context) Map(key, elem Type) *Map {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.internMapLocked(key, elem, nil)
}

func (c *Context) internMapLocked(key, elem Type, goType gotypes.Type) *Map {
	k := mapInternKey{key: key, elem: elem}
	if m, ok := c.maps[k]; ok {
		return m
	}
	m := &Map{Key: key, Elem: elem, goType: goType}
	c.maps[k] = m
	return m
}

// Chan returns the canonically interned Chan type for (dir, elem).
func (c *Context) Chan(dir gotypes.ChanDir, elem Type) *Chan {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.internChanLocked(dir, elem, nil)
}

func (c *Context) internChanLocked(dir gotypes.ChanDir, elem Type, goType gotypes.Type) *Chan {
	k := chanInternKey{dir: dir, elem: elem}
	if ch, ok := c.chans[k]; ok {
		return ch
	}
	ch := &Chan{Dir: dir, Elem: elem, goType: goType}
	c.chans[k] = ch
	return ch
}

// Array returns the canonically interned Array type for (elem, length, nativeStorage).
func (c *Context) Array(elem Type, length int64, nativeStorage bool) *Array {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.internArrayLocked(elem, length, nativeStorage, nil)
}

func (c *Context) internArrayLocked(elem Type, length int64, nativeStorage bool, goType gotypes.Type) *Array {
	k := arrayInternKey{elem: elem, length: length, native: nativeStorage}
	if a, ok := c.arrays[k]; ok {
		return a
	}
	a := &Array{Elem: elem, Len: length, NativeStorage: nativeStorage, goType: goType}
	c.arrays[k] = a
	return a
}

// SIMDVec returns the canonically interned SIMDVec type for (elemKind, length, mask).
func (c *Context) SIMDVec(elemKind gotypes.BasicKind, length int, mask bool) *SIMDVec {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.internSIMDLocked(elemKind, length, mask, nil)
}

func (c *Context) internSIMDLocked(elemKind gotypes.BasicKind, length int, mask bool, goType gotypes.Type) *SIMDVec {
	k := simdInternKey{kind: elemKind, len: length, mask: mask}
	if v, ok := c.simds[k]; ok {
		return v
	}
	v := &SIMDVec{
		Elem:   c.internBasicLocked(elemKind, GoPtr, nil),
		Len:    length,
		Mask:   mask,
		goType: goType,
	}
	c.simds[k] = v
	return v
}

// Type maps a read-only go/types.Type into its canonical physical Type in c
// under calling convention cc (defaulting to GoDecl unless overridden by a
// directive). Recognized simd/archsimd vectors map directly to *SIMDVec.
func (c *Context) Type(t gotypes.Type, cc CallConv) Type {
	if t == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.mapLocked(t, cc)
}

func (c *Context) mapLocked(t gotypes.Type, cc CallConv) Type {
	if cc == CallConvUnknown {
		cc = GoDecl
	}
	if override := lookupNativeCallConv(t, c.lookup); override.IsNative() {
		cc = override
	}
	key := mapKey{t: t, cc: cc}
	if cached, ok := c.mapped[key]; ok {
		return cached
	}
	if _, rhs, ok := aliasObjAndRhs(t); ok {
		res := c.mapLocked(rhs, cc)
		c.mapped[key] = res
		return res
	}
	if lanes, ok := SIMDVectorShape(t); ok {
		elem := lanes.Elem().(*gotypes.Basic)
		_, isMask := SIMDMaskShape(t)
		res := c.internSIMDLocked(elem.Kind(), int(lanes.Len()), isMask, t)
		c.mapped[key] = res
		return res
	}
	switch u := t.(type) {
	case *gotypes.Named:
		obj := u.Obj()
		pkgPath := ""
		if obj.Pkg() != nil {
			pkgPath = obj.Pkg().Path()
		}
		named := &Named{
			PkgPath:  pkgPath,
			Name:     obj.Name(),
			CallConv: cc,
			goType:   u,
		}
		c.mapped[key] = named
		named.underlying = c.mapLocked(u.Underlying(), cc)
		return named
	case *gotypes.Basic:
		if u.Kind() == gotypes.String {
			c.mapped[key] = c.stringType
			return c.stringType
		}
		pk := GoPtr
		if cc.IsNative() && u.Kind() == gotypes.UnsafePointer {
			pk = NativePtr
		}
		res := c.internBasicLocked(u.Kind(), pk, u)
		c.mapped[key] = res
		return res
	case *gotypes.Pointer:
		pk := GoPtr
		if cc.IsNative() {
			pk = NativePtr
		}
		// Allocate placeholder before descending to handle self-referential pointer types.
		ptr := &Pointer{Kind: pk, goType: u}
		c.mapped[key] = ptr
		elem := c.mapLocked(u.Elem(), cc)
		ptr.Elem = elem
		// If elem does not reference ptr cyclically and an interned entry already exists, reuse it.
		if existing, ok := c.ptrs[ptrInternKey{elem: elem, kind: pk}]; ok {
			c.mapped[key] = existing
			return existing
		}
		c.ptrs[ptrInternKey{elem: elem, kind: pk}] = ptr
		return ptr
	case *gotypes.Signature:
		fn := c.mapSignatureLocked(u, cc)
		if cc.IsNative() || cc == PyDecl {
			c.mapped[key] = fn
			return fn
		}
		res := &Closure{Sig: fn, goType: u}
		c.mapped[key] = res
		return res
	case *gotypes.Slice:
		res := c.internSliceLocked(c.mapLocked(u.Elem(), GoDecl), u)
		c.mapped[key] = res
		return res
	case *gotypes.Interface:
		if u.NumMethods() == 0 {
			c.mapped[key] = c.anyType
			return c.anyType
		}
		res := &Interface{Empty: false, goType: u}
		c.mapped[key] = res
		return res
	case *gotypes.Map:
		res := c.internMapLocked(c.mapLocked(u.Key(), GoDecl), c.mapLocked(u.Elem(), GoDecl), u)
		c.mapped[key] = res
		return res
	case *gotypes.Chan:
		res := c.internChanLocked(u.Dir(), c.mapLocked(u.Elem(), GoDecl), u)
		c.mapped[key] = res
		return res
	case *gotypes.Array:
		res := c.internArrayLocked(c.mapLocked(u.Elem(), cc), u.Len(), cc.IsNative(), u)
		c.mapped[key] = res
		return res
	case *gotypes.Struct:
		host := HasHostLayout(u)
		st := &Struct{
			Fields:        make([]*Field, u.NumFields()),
			HostLayout:    host,
			NativeStorage: cc.IsNative(),
			goType:        u,
		}
		c.mapped[key] = st
		for i := 0; i < u.NumFields(); i++ {
			f := u.Field(i)
			fieldCC := cc
			if fieldCC.IsNative() {
				if _, isStruct := f.Type().Underlying().(*gotypes.Struct); isStruct && !lookupNativeCallConv(f.Type(), c.lookup).IsNative() {
					fieldCC = GoDecl
				}
			}
			st.Fields[i] = &Field{
				Name:     f.Name(),
				Type:     c.mapLocked(f.Type(), fieldCC),
				Tag:      u.Tag(i),
				Embedded: f.Embedded(),
			}
		}
		return st
	case *gotypes.Tuple:
		res := &Tuple{Elems: make([]*Var, u.Len()), goType: u}
		for i := 0; i < u.Len(); i++ {
			v := u.At(i)
			res.Elems[i] = &Var{Name: v.Name(), Type: c.mapLocked(v.Type(), cc)}
		}
		c.mapped[key] = res
		return res
	case *gotypes.TypeParam:
		return c.mapLocked(u.Underlying(), cc)
	default:
		panic("llgo/types: unsupported go/types.Type: " + t.String())
	}
}

func (c *Context) mapSignatureLocked(sig *gotypes.Signature, cc CallConv) *Func {
	if cc == CallConvUnknown {
		cc = GoDecl
	}
	var recv *Var
	if r := sig.Recv(); r != nil {
		recv = &Var{Name: r.Name(), Type: c.mapLocked(r.Type(), cc)}
	}
	params := make([]*Var, sig.Params().Len())
	for i := 0; i < sig.Params().Len(); i++ {
		p := sig.Params().At(i)
		params[i] = &Var{Name: p.Name(), Type: c.mapLocked(p.Type(), cc)}
	}
	results := make([]*Var, sig.Results().Len())
	for i := 0; i < sig.Results().Len(); i++ {
		r := sig.Results().At(i)
		results[i] = &Var{Name: r.Name(), Type: c.mapLocked(r.Type(), cc)}
	}
	return &Func{
		CallConv: cc,
		Recv:     recv,
		Params:   params,
		Results:  results,
		Variadic: sig.Variadic(),
		goType:   sig,
	}
}

// Layout returns the cached, immutable physical Layout for t.
func (c *Context) Layout(t Type) *Layout {
	if t == nil {
		return &Layout{Align: 1, IsRegularMemory: true}
	}
	c.mu.RLock()
	if l, ok := c.layouts[t]; ok {
		c.mu.RUnlock()
		return l
	}
	c.mu.RUnlock()

	l := c.computeLayout(t)
	c.mu.Lock()
	if existing, ok := c.layouts[t]; ok {
		c.mu.Unlock()
		return existing
	}
	c.layouts[t] = l
	c.mu.Unlock()
	return l
}

// GoLayout maps t under cc and returns its cached immutable physical Layout.
func (c *Context) GoLayout(t gotypes.Type, cc CallConv) *Layout {
	return c.Layout(c.Type(t, cc))
}

// FuncABI classifies the target calling convention and parameter/return passing
// modes for sig using c's cached Layouts.
func (c *Context) FuncABI(sig *Func) *FuncABI {
	return ClassifyFuncABI(c.spec, sig, c.Layout)
}

func (c *Context) computeLayout(t Type) *Layout {
	sz, al := c.sizes.layout(t, false)
	// Enforce Clang AArch64 16-byte cap on SIMDVec alignment when computing Layout.
	if v, ok := t.Underlying().(*SIMDVec); ok {
		es, _ := c.sizes.basicLayout(v.Elem.Kind, true, true)
		total := es * int64(v.Len)
		sz = total
		al = total
		if c.spec.GOARCH == "arm64" && al > 16 {
			al = 16
		}
	}
	ptrBytes := c.sizes.PtrDataSize(t)
	ptrOffs := c.sizes.PtrOffsets(t)
	word := int64(c.spec.WordSize)
	l := &Layout{
		Size:             sz,
		Align:            al,
		PtrBytes:         ptrBytes,
		PtrOffsets:       ptrOffs,
		GCBitmap:         GCBitmap(ptrOffs, ptrBytes, word),
		HasWasmWideSlots: c.hasWasmWideSlot(t),
		HasHostLayout:    typeHasHostLayout(t),
		HasSIMDStorage:   typeHasSIMDStorage(t),
	}
	switch u := t.Underlying().(type) {
	case *Struct:
		stSize, _, offs := c.sizes.structLayout(u)
		l.Fields = make([]FieldLayout, len(u.Fields))
		for i, f := range u.Fields {
			fs, fa := c.sizes.layout(f.Type, u.HostLayout)
			end := stSize
			if i+1 < len(offs) {
				end = offs[i+1]
			}
			pad := end - offs[i] - fs
			if pad < 0 {
				pad = 0
			}
			l.Fields[i] = FieldLayout{
				Offset:       offs[i],
				Size:         fs,
				Align:        fa,
				PaddingAfter: pad,
				WasmWideSlot: c.hasWasmWideSlot(f.Type),
			}
		}
	case *Tuple:
		tuSize, _, offs := c.sizes.tupleLayout(u)
		l.Fields = make([]FieldLayout, len(u.Elems))
		for i, e := range u.Elems {
			es, ea := c.sizes.layout(e.Type, false)
			end := tuSize
			if i+1 < len(offs) {
				end = offs[i+1]
			}
			pad := end - offs[i] - es
			if pad < 0 {
				pad = 0
			}
			l.Fields[i] = FieldLayout{
				Offset:       offs[i],
				Size:         es,
				Align:        ea,
				PaddingAfter: pad,
				WasmWideSlot: c.hasWasmWideSlot(e.Type),
			}
		}
	}
	l.IsRegularMemory = c.isRegularMemory(t, l)
	return l
}

func (c *Context) hasWasmWideSlot(t Type) bool {
	if !c.spec.UsesWideGoStorage() || t == nil {
		return false
	}
	switch u := t.Underlying().(type) {
	case *Basic:
		return (u.Kind == gotypes.Uintptr || u.Kind == gotypes.UnsafePointer) && u.PtrKind == GoPtr
	case *Pointer:
		return u.Kind == GoPtr
	case *Func:
		return !u.CallConv.IsNative()
	case *Closure, *String, *Slice, *Interface, *Map, *Chan:
		return true
	case *Array:
		return u.Len > 0 && c.hasWasmWideSlot(u.Elem)
	case *Struct:
		for _, f := range u.Fields {
			if c.hasWasmWideSlot(f.Type) {
				return true
			}
		}
	case *Tuple:
		for _, e := range u.Elems {
			if c.hasWasmWideSlot(e.Type) {
				return true
			}
		}
	}
	return false
}

func typeHasHostLayout(t Type) bool {
	if t == nil {
		return false
	}
	switch u := t.Underlying().(type) {
	case *Struct:
		if u.HostLayout {
			return true
		}
		for _, f := range u.Fields {
			if typeHasHostLayout(f.Type) {
				return true
			}
		}
	case *Array:
		return typeHasHostLayout(u.Elem)
	}
	return false
}

func typeHasSIMDStorage(t Type) bool {
	if t == nil {
		return false
	}
	switch u := t.Underlying().(type) {
	case *SIMDVec:
		return true
	case *Struct:
		for _, f := range u.Fields {
			if typeHasSIMDStorage(f.Type) {
				return true
			}
		}
	case *Array:
		return typeHasSIMDStorage(u.Elem)
	}
	return false
}

func (c *Context) isRegularMemory(t Type, l *Layout) bool {
	switch u := t.Underlying().(type) {
	case *Basic:
		switch u.Kind {
		case gotypes.Float32, gotypes.Float64, gotypes.Complex64, gotypes.Complex128:
			return false
		}
		return true
	case *Pointer, *Chan:
		return true
	case *SIMDVec:
		return u.Elem.Kind != gotypes.Float32 && u.Elem.Kind != gotypes.Float64
	case *String, *Slice, *Func, *Closure, *Interface, *Map:
		return false
	case *Array:
		if u.Len == 0 {
			return true
		}
		return c.Layout(u.Elem).IsRegularMemory
	case *Struct:
		if len(u.Fields) == 0 {
			return true
		}
		if len(u.Fields) == 1 {
			if u.Fields[0].Name == "_" {
				return false
			}
			return c.Layout(u.Fields[0].Type).IsRegularMemory
		}
		for i, f := range u.Fields {
			if f.Name == "_" || l.Fields[i].PaddingAfter != 0 || !c.Layout(f.Type).IsRegularMemory {
				return false
			}
		}
		return true
	case *Tuple:
		for i, e := range u.Elems {
			if l.Fields[i].PaddingAfter != 0 || !c.Layout(e.Type).IsRegularMemory {
				return false
			}
		}
		return true
	}
	return false
}
