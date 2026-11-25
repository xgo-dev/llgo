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
	"go/ast"
	"go/token"
	gotypes "go/types"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/target"
)

// ContextConfig configures a *Context.
type ContextConfig struct {
	Spec target.Spec

	// HostBasicLayouts optionally overrides default C ABI basic scalar layouts
	// when HostBasicLayouts[Bool].Size != 0.
	HostBasicLayouts [UnsafePointer + 1]LayoutPair

	// SIMDAlignments optionally overrides v128/v256/v512 alignments when non-zero.
	SIMDAlignments [3]int64
}

type ptrInternKey struct {
	elem Type
	kind PtrKind
}

type mapInternKey struct {
	key  Type
	elem Type
}

type chanInternKey struct {
	dir  ChanDir
	elem Type
}

type arrayInternKey struct {
	elem   Type
	length int64
	native bool
}

type simdInternKey struct {
	elem   BasicKind
	length int
	mask   bool
}

type goTypeCacheKey struct {
	typ         gotypes.Type
	cc          CallConv
	hasOverride bool
}

// Context is the single unified engine for LLGo's type system:
//   - Canonical Type interning and construction
//   - One-shot and memoized lowering from go/types + //llgo:* directives
//   - Single-source-of-truth Layout, GC bitmap, and FuncABI computation
//   - Sizes (on LLGo Type) and GoSizes() (go/types.Sizes adapter)
//
// All public methods are safe for concurrent use.
type Context struct {
	mu sync.RWMutex

	spec             target.Spec
	hostBasicLayouts [UnsafePointer + 1]LayoutPair
	simdAlignments   [3]int64

	// Registered type-level calling conventions and native struct field markers
	// (from Check pre-scan, SetTypeCallConv, SetNativeStructFields, MarkNativeField).
	hasNativeMetadata  atomic.Bool
	typeCallConvs      map[string]CallConv
	nativeStructFields map[token.Pos]bool
	nativeGoVars       map[gotypes.Object]bool
	nativeVars         map[*Var]bool

	// Interning tables.
	anyType *Interface
	ptrs    map[ptrInternKey]*Pointer
	slices  map[Type]*Slice
	maps    map[mapInternKey]*Map
	chans   map[chanInternKey]*Chan
	arrays  map[arrayInternKey]*Array
	simds   map[simdInternKey]*SIMDVec

	// Memoized go/types -> llgo/types lowering tables.
	goPkgs          map[*gotypes.Package]*Package
	goScopes        map[*gotypes.Scope]*Scope
	goObjs          map[gotypes.Object]Object
	goTypes         map[goTypeCacheKey]Type
	goNamed         map[*gotypes.Named]*Named
	goAliases       map[gotypes.Type]*Alias
	goTParams       map[*gotypes.TypeParam]*TypeParam
	incompleteNamed []*Named

	// Memoized physical layout and ABI caches.
	layouts  map[Type]*Layout
	funcABIs map[*Signature]*FuncABI
}

// NewContext creates a Context for spec with default target layouts and alignments.
func NewContext(spec target.Spec) *Context {
	return NewContextWithConfig(ContextConfig{Spec: spec})
}

// NewContextWithConfig creates a Context with explicit target and metadata config.
func NewContextWithConfig(cfg ContextConfig) *Context {
	spec := cfg.Spec
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	if spec.MaxAlign == 0 {
		spec.MaxAlign = spec.WordSize
	}
	hostLayouts := cfg.HostBasicLayouts
	if hostLayouts[Bool].Size == 0 {
		hostLayouts = DefaultHostBasicLayouts(spec)
	}
	simdAligns := cfg.SIMDAlignments
	if simdAligns[0] == 0 {
		simdAligns = DefaultSIMDAlignments(spec)
	}
	return &Context{
		spec:               spec,
		hostBasicLayouts:   hostLayouts,
		simdAlignments:     simdAligns,
		typeCallConvs:      make(map[string]CallConv),
		nativeStructFields: make(map[token.Pos]bool),
		nativeGoVars:       make(map[gotypes.Object]bool),
		nativeVars:         make(map[*Var]bool),
		anyType:            &Interface{resolved: true, comparable: true},
		ptrs:               make(map[ptrInternKey]*Pointer),
		slices:             make(map[Type]*Slice),
		maps:               make(map[mapInternKey]*Map),
		chans:              make(map[chanInternKey]*Chan),
		arrays:             make(map[arrayInternKey]*Array),
		simds:              make(map[simdInternKey]*SIMDVec),
		goPkgs:             make(map[*gotypes.Package]*Package),
		goScopes:           make(map[*gotypes.Scope]*Scope),
		goObjs:             make(map[gotypes.Object]Object),
		goTypes:            make(map[goTypeCacheKey]Type),
		goNamed:            make(map[*gotypes.Named]*Named),
		goAliases:          make(map[gotypes.Type]*Alias),
		goTParams:          make(map[*gotypes.TypeParam]*TypeParam),
		layouts:            make(map[Type]*Layout),
		funcABIs:           make(map[*Signature]*FuncABI),
	}
}

// Spec returns the resolved target specification of c.
func (c *Context) Spec() target.Spec {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.spec
}

// SIMDAlignments returns the resolved v128/v256/v512 alignments of c.
func (c *Context) SIMDAlignments() [3]int64 {
	return c.simdAlignments
}

// HasNativeMetadata reports whether any native calling-convention type or
// native struct field has been registered in c.
func (c *Context) HasNativeMetadata() bool {
	return c.hasNativeMetadata.Load()
}

// SetTypeCallConv registers a type-level calling convention directive (such as
// //llgo:cdecl or //llgo:stdcall) by qualified type name ("pkg.Name") and
// invalidates cached go/types lowerings so subsequent queries reflect it.
func (c *Context) SetTypeCallConv(typeKey string, cc CallConv) {
	c.mu.Lock()
	c.typeCallConvs[typeKey] = cc
	if cc.IsNative() {
		c.hasNativeMetadata.Store(true)
	}
	c.resetGoCachesLocked()
	c.mu.Unlock()
}

// TypeCallConv looks up a calling convention previously registered for typeKey.
func (c *Context) TypeCallConv(typeKey string) (CallConv, bool) {
	c.mu.RLock()
	cc, ok := c.typeCallConvs[typeKey]
	c.mu.RUnlock()
	return cc, ok
}

// NativeTypeCallConv resolves the effective native calling convention attached
// directly to t (walking *Alias chains before unaliasing).
func (c *Context) NativeTypeCallConv(t Type) (CallConv, bool) {
	if t == nil {
		return Go, false
	}
	cur := t
	for {
		switch u := cur.(type) {
		case *Named:
			if u.hasCallConv && u.callConv.IsNative() {
				return u.callConv, true
			}
			if key := typeNameKey(u.obj); key != "" && c.hasNativeMetadata.Load() {
				if cc, ok := c.TypeCallConv(key); ok && cc.IsNative() {
					return cc, true
				}
			}
			return Go, false
		case *Alias:
			if u.hasCallConv && u.callConv.IsNative() {
				return u.callConv, true
			}
			if key := typeNameKey(u.obj); key != "" && c.hasNativeMetadata.Load() {
				if cc, ok := c.TypeCallConv(key); ok && cc.IsNative() {
					return cc, true
				}
			}
			if u.rhs == nil || u.rhs == cur {
				return Go, false
			}
			cur = u.rhs
		default:
			return Go, false
		}
	}
}

// SetNativeStructFields registers source positions of fields belonging to
// native structs (//llgo:cdecl / //llgo:stdcall) so Offsetsof can identify
// native struct layouts even when invoked without the enclosing named type.
func (c *Context) SetNativeStructFields(positions []token.Pos) {
	c.mu.Lock()
	changed := false
	for _, pos := range positions {
		if pos.IsValid() {
			c.nativeStructFields[pos] = true
			c.hasNativeMetadata.Store(true)
			changed = true
		}
	}
	if changed {
		c.resetGoCachesLocked()
	}
	c.mu.Unlock()
}

// MarkNativeField marks a struct field (typically field 0 of a programmatically
// constructed native struct without source positions) as belonging to a native
// struct layout.
func (c *Context) MarkNativeField(v *Var) {
	if v == nil {
		return
	}
	c.mu.Lock()
	c.nativeVars[v] = true
	if v.goObj != nil {
		c.nativeGoVars[v.goObj] = true
	}
	c.hasNativeMetadata.Store(true)
	c.resetGoCachesLocked()
	c.mu.Unlock()
}

// SetSizes updates the Go word size and max alignment used by c (for callers
// that swap target go/types.Sizes dynamically) and clears cached layouts.
func (c *Context) SetSizes(wordSize, maxAlign int64) {
	c.mu.Lock()
	if wordSize > 0 {
		c.spec.WordSize = int(wordSize)
	}
	if maxAlign > 0 {
		c.spec.MaxAlign = int(maxAlign)
	}
	c.layouts = make(map[Type]*Layout)
	c.funcABIs = make(map[*Signature]*FuncABI)
	c.mu.Unlock()
}

// InvalidateGoCache clears memoized go/types lowering and layout entries after
// external directive metadata changes.
func (c *Context) InvalidateGoCache() {
	c.mu.Lock()
	c.resetGoCachesLocked()
	c.mu.Unlock()
}

func (c *Context) resetGoCachesLocked() {
	c.goTypes = make(map[goTypeCacheKey]Type)
	c.goNamed = make(map[*gotypes.Named]*Named)
	c.goAliases = make(map[gotypes.Type]*Alias)
	c.incompleteNamed = nil
	c.layouts = make(map[Type]*Layout)
	c.funcABIs = make(map[*Signature]*FuncABI)
}

func (c *Context) checkIncompleteNamedLocked() {
	if len(c.incompleteNamed) == 0 {
		return
	}
	for _, n := range c.incompleteNamed {
		if n.resolved && n.goNamed != nil && n.goNamed.Underlying() != n.goUnderlying {
			c.resetGoCachesLocked()
			return
		}
	}
}

// --- Canonical Type Interning ---

// Basic returns the canonical *Basic type for kind.
func (c *Context) Basic(kind BasicKind) *Basic {
	if kind >= 0 && int(kind) < len(Typ) {
		return Typ[kind]
	}
	return Typ[Invalid]
}

// Any returns the canonical empty interface type (any).
func (c *Context) Any() *Interface { return c.anyType }

// Pointer returns an interned *Pointer for (elem, kind).
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
	p := &Pointer{elem: elem, kind: kind, goType: goType}
	c.ptrs[k] = p
	return p
}

// Slice returns an interned *Slice for elem.
func (c *Context) Slice(elem Type) *Slice {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.internSliceLocked(elem, nil)
}

func (c *Context) internSliceLocked(elem Type, goType gotypes.Type) *Slice {
	if s, ok := c.slices[elem]; ok {
		return s
	}
	s := &Slice{elem: elem, goType: goType}
	c.slices[elem] = s
	return s
}

// Map returns an interned *Map for (key, elem).
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
	m := &Map{key: key, elem: elem, goType: goType}
	c.maps[k] = m
	return m
}

// Chan returns an interned *Chan for (dir, elem).
func (c *Context) Chan(dir ChanDir, elem Type) *Chan {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.internChanLocked(dir, elem, nil)
}

func (c *Context) internChanLocked(dir ChanDir, elem Type, goType gotypes.Type) *Chan {
	k := chanInternKey{dir: dir, elem: elem}
	if ch, ok := c.chans[k]; ok {
		return ch
	}
	ch := &Chan{dir: dir, elem: elem, goType: goType}
	c.chans[k] = ch
	return ch
}

// Array returns an interned *Array for (elem, length, nativeStorage).
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
	a := &Array{elem: elem, length: length, nativeStorage: nativeStorage, goType: goType}
	c.arrays[k] = a
	return a
}

// SIMDVec returns an interned *SIMDVec for (elemKind, length, isMask).
func (c *Context) SIMDVec(elemKind BasicKind, length int, isMask bool) *SIMDVec {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.internSIMDLocked(elemKind, length, isMask, nil)
}

func (c *Context) internSIMDLocked(elemKind BasicKind, length int, isMask bool, goType gotypes.Type) *SIMDVec {
	k := simdInternKey{elem: elemKind, length: length, mask: isMask}
	if v, ok := c.simds[k]; ok {
		return v
	}
	elem := Typ[Invalid]
	if elemKind >= 0 && int(elemKind) < len(Typ) {
		elem = Typ[elemKind]
	}
	v := &SIMDVec{
		elem:     elem,
		length:   length,
		isMask:   isMask,
		tagIndex: simdTagForBits(basicBits(elem) * length),
		goType:   goType,
	}
	c.simds[k] = v
	return v
}

// FuncABI computes and caches the pure-Go ABI classification for sig.
func (c *Context) FuncABI(sig *Signature) *FuncABI {
	if sig == nil {
		return nil
	}
	c.mu.RLock()
	if abi, ok := c.funcABIs[sig]; ok {
		c.mu.RUnlock()
		return abi
	}
	c.mu.RUnlock()

	abi := ClassifyFuncABI(c.spec, sig, c.Layout)

	c.mu.Lock()
	if existing, ok := c.funcABIs[sig]; ok {
		c.mu.Unlock()
		return existing
	}
	c.funcABIs[sig] = abi
	c.mu.Unlock()
	return abi
}

// --- Native Directive Helpers ---

func typeNameKey(obj *TypeName) string {
	if obj == nil {
		return ""
	}
	if obj.pkg != nil && obj.pkg.path != "" {
		return obj.pkg.path + "." + obj.name
	}
	return obj.name
}

func goTypeNameKey(obj *gotypes.TypeName) string {
	if obj == nil {
		return ""
	}
	if obj.Pkg() != nil && obj.Pkg().Path() != "" {
		return obj.Pkg().Path() + "." + obj.Name()
	}
	return obj.Name()
}

func (c *Context) lookupTypeCallConvLocked(t gotypes.Type) (CallConv, bool) {
	if len(c.typeCallConvs) == 0 {
		return Go, false
	}
	cur := t
	for {
		var obj *gotypes.TypeName
		var next gotypes.Type
		var isAlias bool
		if n, ok := cur.(*gotypes.Named); ok {
			obj = n.Obj()
		} else if aObj, rhs, ok := aliasObjAndRhs(cur); ok {
			obj = aObj
			next = rhs
			isAlias = true
		}
		if obj != nil {
			if cc, ok := c.typeCallConvs[goTypeNameKey(obj)]; ok {
				return cc, true
			}
		}
		if !isAlias || next == cur {
			return Go, false
		}
		cur = next
	}
}

func (c *Context) hasLegacyNativeLayoutLocked(raw gotypes.Type) bool {
	if !c.hasNativeMetadata.Load() {
		return false
	}
	switch t := unaliasType(raw).(type) {
	case *gotypes.Named:
		cc, ok := c.lookupTypeCallConvLocked(t)
		return ok && cc.IsNative()
	case *gotypes.Array:
		return c.hasLegacyNativeLayoutLocked(t.Elem())
	}
	return false
}

func (c *Context) hasNativeFieldsLocked(fields []*gotypes.Var) bool {
	if len(fields) == 0 || !c.hasNativeMetadata.Load() {
		return false
	}
	return c.nativeGoVars[fields[0]] || c.nativeStructFields[fields[0].Pos()]
}

func (c *Context) hasNativeFields(fields []*gotypes.Var) bool {
	if len(fields) == 0 || !c.hasNativeMetadata.Load() {
		return false
	}
	c.mu.RLock()
	ok := c.nativeGoVars[fields[0]] || c.nativeStructFields[fields[0].Pos()]
	c.mu.RUnlock()
	return ok
}

func (c *Context) hasNativeLLFields(fields []*Var) bool {
	if len(fields) == 0 || !c.hasNativeMetadata.Load() {
		return false
	}
	f0 := fields[0]
	if f0 == nil {
		return false
	}
	c.mu.RLock()
	ok := c.nativeVars[f0] || (f0.goObj != nil && c.nativeGoVars[f0.goObj]) || c.nativeStructFields[f0.pos]
	c.mu.RUnlock()
	return ok
}

func isGoClosureFields(fields []*gotypes.Var) bool {
	if len(fields) != 2 {
		return false
	}
	f0, f1 := fields[0], fields[1]
	if _, ok := f0.Type().(*gotypes.Signature); !ok || f0.Name() != "$f" {
		return false
	}
	return f1.Type() == gotypes.Typ[gotypes.UnsafePointer] && f1.Name() == "$data"
}

// --- Lazy Resolution for Named & Interface Types Lowered from go/types ---

func (c *Context) resolveNamed(n *Named) {
	c.mu.RLock()
	if n.resolved && (n.goNamed == nil || n.goNamed.Underlying() == n.goUnderlying) {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	c.resolveNamedLocked(n)
}

func (c *Context) resolveNamedLocked(n *Named) {
	gt := n.goNamed
	if gt == nil {
		n.resolved = true
		return
	}
	under := gt.Underlying()
	if n.resolved && under == n.goUnderlying {
		return
	}

	// Mark resolved before recursing to support cyclic named types safely.
	n.resolved = true
	n.goUnderlying = under
	if under == nil {
		c.incompleteNamed = append(c.incompleteNamed, n)
	} else if st, ok := under.(*gotypes.Struct); ok && st.NumFields() == 0 {
		c.incompleteNamed = append(c.incompleteNamed, n)
	}

	tagIdx := -1
	if idx, ok := goSIMDTagIndex(gt); ok {
		tagIdx = idx
	}
	var vec *SIMDVec
	if lanes, ok := goSIMDVectorShape(gt); ok {
		if elem, ok := lanes.Elem().(*gotypes.Basic); ok {
			isMask := strings.HasPrefix(gt.Obj().Name(), "Mask")
			vec = c.internSIMDLocked(elem.Kind(), int(lanes.Len()), isMask, gt)
		}
	}

	host := goHasHostLayout(gt)
	_, isUnderStruct := under.(*gotypes.Struct)
	_, isUnderArray := under.(*gotypes.Array)
	native := (isUnderStruct || isUnderArray) && ((n.hasCallConv && n.callConv.IsNative()) || c.hasLegacyNativeLayoutLocked(gt))
	if !native && isUnderStruct {
		if st := under.(*gotypes.Struct); st.NumFields() > 0 {
			if c.hasNativeFieldsLocked([]*gotypes.Var{st.Field(0)}) {
				native = true
			}
		}
	}

	n.hostLayout = host
	n.nativeLayout = native
	n.simdTagIndex = tagIdx
	n.simdVec = vec
	if n.obj != nil {
		n.obj.nativeLayout = native
		n.obj.hostLayout = host
	}

	if orig := gt.Origin(); orig != nil && orig != gt {
		n.origin = c.fromGoNamedShellLocked(orig, Go, false)
	}
	if tps := gt.TypeParams(); tps != nil && tps.Len() > 0 {
		list := make([]*TypeParam, tps.Len())
		for i := 0; i < tps.Len(); i++ {
			list[i] = c.fromGoTypeParamLocked(tps.At(i))
		}
		n.typeParams = newTypeParamList(list)
	}
	if tas := gt.TypeArgs(); tas != nil && tas.Len() > 0 {
		list := make([]Type, tas.Len())
		for i := 0; i < tas.Len(); i++ {
			list[i] = c.fromGoLocked(tas.At(i), Go, false)
		}
		n.typeArgs = newTypeList(list)
	}

	effCC := n.callConv
	effOverride := n.hasCallConv
	if arr, ok := under.(*gotypes.Array); ok && native {
		var elem Type
		if effOverride && effCC.IsNative() {
			elem = c.fromGoLocked(arr.Elem(), effCC, true)
		} else {
			elem = c.fromGoLocked(arr.Elem(), Go, false)
		}
		n.underlying = c.internArrayLocked(elem, arr.Len(), c.spec.UsesWideGoStorage(), arr)
	} else if st, ok := under.(*gotypes.Struct); ok && (native || tagIdx >= 0) {
		num := st.NumFields()
		fields := make([]*gotypes.Var, num)
		tags := make([]string, num)
		hasTags := false
		for i := 0; i < num; i++ {
			fields[i] = st.Field(i)
			tags[i] = st.Tag(i)
			if tags[i] != "" {
				hasTags = true
			}
		}
		if !hasTags {
			tags = nil
		}
		llSt := c.structFromGoFieldsLocked(fields, tags, native, st)
		llSt.simdTagIndex = tagIdx
		n.underlying = llSt
	} else if effOverride {
		n.underlying = c.fromGoLocked(under, effCC, true)
	} else {
		n.underlying = c.fromGoLocked(under, Go, false)
	}
}

func (c *Context) resolveNamedMethods(n *Named) {
	c.mu.RLock()
	if n.methodsResolved || n.goNamed == nil {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if n.methodsResolved || n.goNamed == nil {
		return
	}
	n.methodsResolved = true
	gt := n.goNamed
	if nm := gt.NumMethods(); nm > 0 {
		n.methods = make([]*Func, nm)
		for i := 0; i < nm; i++ {
			n.methods[i] = c.fromGoFuncLocked(gt.Method(i))
		}
	}
}

func (c *Context) resolveInterface(iface *Interface) {
	c.mu.RLock()
	if iface.resolved || iface.goIface == nil {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if iface.resolved || iface.goIface == nil {
		return
	}
	iface.resolved = true
	gt := iface.goIface
	if n := gt.NumMethods(); n > 0 {
		iface.methods = make([]*Func, n)
		for i := 0; i < n; i++ {
			iface.methods[i] = c.fromGoFuncLocked(gt.Method(i))
		}
	}
	if n := gt.NumEmbeddeds(); n > 0 {
		iface.embeddeds = make([]Type, n)
		for i := 0; i < n; i++ {
			iface.embeddeds[i] = c.fromGoLocked(gt.EmbeddedType(i), Go, false)
		}
	}
}

// --- go/types -> llgo/types Lowering (Single Conversion Boundary) ---

// FromGo lowers a go/types.Type into its canonical LLGo Type (O(1) cached).
// Named and Interface shells are resolved lazily so invoking FromGo during
// go/types.Config.Check never triggers premature go/types assertions on
// unresolved pointer targets.
func (c *Context) FromGo(t gotypes.Type) Type {
	if t == nil {
		return nil
	}
	c.mu.RLock()
	if len(c.incompleteNamed) == 0 {
		if cached, ok := c.goTypes[goTypeCacheKey{typ: t, cc: Go, hasOverride: false}]; ok {
			c.mu.RUnlock()
			return cached
		}
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	c.checkIncompleteNamedLocked()
	return c.fromGoLocked(t, Go, false)
}

// FromGoWithCallConv lowers a go/types.Type with an explicit calling convention
// override (for example CDecl or Stdcall on a signature).
func (c *Context) FromGoWithCallConv(t gotypes.Type, cc CallConv) Type {
	if t == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checkIncompleteNamedLocked()
	return c.fromGoLocked(t, cc, true)
}

func (c *Context) fromGoLocked(t gotypes.Type, cc CallConv, hasOverride bool) Type {
	if t == nil {
		return nil
	}
	key := goTypeCacheKey{typ: t, cc: cc, hasOverride: hasOverride}
	if cached, ok := c.goTypes[key]; ok {
		return cached
	}

	// Preserve go/types.Alias identity and any calling convention attached to it.
	if aObj, rhs, ok := aliasObjAndRhs(t); ok {
		if !hasOverride {
			if existing, ok := c.goAliases[t]; ok {
				return existing
			}
		}
		dirCC, hasDir := c.lookupTypeCallConvLocked(t)
		effCC := cc
		effOverride := hasOverride
		if !effOverride && hasDir {
			effCC = dirCC
			effOverride = true
		}
		tn := c.fromGoTypeNameShellLocked(aObj)
		tn.isAlias = true
		if hasDir {
			tn.callConv = dirCC
			tn.hasCallConv = true
		}
		a := &Alias{
			obj:         tn,
			callConv:    effCC,
			hasCallConv: effOverride,
			goType:      t,
		}
		if tn.typ == nil {
			tn.typ = a
		}
		if !hasOverride {
			c.goAliases[t] = a
		}
		c.goTypes[key] = a
		a.rhs = c.fromGoLocked(rhs, effCC, effOverride)
		return a
	}

	switch gt := t.(type) {
	case *gotypes.Basic:
		b := Typ[Invalid]
		if gt.Kind() >= 0 && int(gt.Kind()) < len(Typ) {
			b = Typ[gt.Kind()]
		}
		c.goTypes[key] = b
		return b

	case *gotypes.Pointer:
		elem := c.fromGoLocked(gt.Elem(), Go, false)
		res := c.internPtrLocked(elem, GoPtr, gt)
		c.goTypes[key] = res
		return res

	case *gotypes.Slice:
		elem := c.fromGoLocked(gt.Elem(), Go, false)
		res := c.internSliceLocked(elem, gt)
		c.goTypes[key] = res
		return res

	case *gotypes.Array:
		var elem Type
		if hasOverride && cc.IsNative() {
			elem = c.fromGoLocked(gt.Elem(), cc, true)
		} else {
			elem = c.fromGoLocked(gt.Elem(), Go, false)
		}
		res := c.internArrayLocked(elem, gt.Len(), false, gt)
		c.goTypes[key] = res
		return res

	case *gotypes.Map:
		k := c.fromGoLocked(gt.Key(), Go, false)
		v := c.fromGoLocked(gt.Elem(), Go, false)
		res := c.internMapLocked(k, v, gt)
		c.goTypes[key] = res
		return res

	case *gotypes.Chan:
		elem := c.fromGoLocked(gt.Elem(), Go, false)
		res := c.internChanLocked(gt.Dir(), elem, gt)
		c.goTypes[key] = res
		return res

	case *gotypes.Tuple:
		res := c.fromGoTupleLocked(gt)
		c.goTypes[key] = res
		return res

	case *gotypes.Signature:
		effCC := Go
		if hasOverride {
			effCC = cc
		}
		res := c.fromGoSignatureLocked(gt, effCC)
		c.goTypes[key] = res
		return res

	case *gotypes.Interface:
		iface := &Interface{
			ctx:        c,
			goIface:    gt,
			comparable: true,
			goType:     gt,
		}
		c.goTypes[key] = iface
		return iface

	case *gotypes.Struct:
		n := gt.NumFields()
		fields := make([]*gotypes.Var, n)
		tags := make([]string, n)
		hasTags := false
		for i := 0; i < n; i++ {
			fields[i] = gt.Field(i)
			tags[i] = gt.Tag(i)
			if tags[i] != "" {
				hasTags = true
			}
		}
		if !hasTags {
			tags = nil
		}
		res := c.structFromGoFieldsLocked(fields, tags, hasOverride && cc.IsNative(), gt)
		c.goTypes[key] = res
		return res

	case *gotypes.TypeParam:
		res := c.fromGoTypeParamLocked(gt)
		c.goTypes[key] = res
		return res

	case *gotypes.Named:
		return c.fromGoNamedShellLocked(gt, cc, hasOverride)
	}

	return Typ[Invalid]
}

// fromGoNamedShellLocked creates or returns the cached *Named shell for gt
// without calling gt.Underlying() or gt.NumMethods().
func (c *Context) fromGoNamedShellLocked(gt *gotypes.Named, cc CallConv, hasOverride bool) *Named {
	if !hasOverride {
		if cached, ok := c.goNamed[gt]; ok {
			return cached
		}
	}

	dirCC, hasDir := c.lookupTypeCallConvLocked(gt)
	effCC := cc
	effOverride := hasOverride
	if !effOverride && hasDir {
		effCC = dirCC
		effOverride = true
	}

	tn := c.fromGoTypeNameShellLocked(gt.Obj())
	if hasDir {
		tn.callConv = dirCC
		tn.hasCallConv = true
	}

	n := &Named{
		ctx:           c,
		goNamed:       gt,
		overrideCC:    cc,
		hasOverrideCC: hasOverride,
		obj:           tn,
		callConv:      effCC,
		hasCallConv:   effOverride,
		simdTagIndex:  -1,
		goType:        gt,
	}
	if tn.typ == nil {
		tn.typ = n
	}

	if !hasOverride {
		c.goNamed[gt] = n
	}
	c.goTypes[goTypeCacheKey{typ: gt, cc: cc, hasOverride: hasOverride}] = n
	return n
}

func (c *Context) structFromGoFieldsLocked(fields []*gotypes.Var, tags []string, forceNative bool, goType gotypes.Type) *Struct {
	isClosure := isGoClosureFields(fields)
	host := false
	for _, f := range fields {
		ft := f.Type()
		switch ft.(type) {
		case *gotypes.Basic, *gotypes.Pointer, *gotypes.Slice, *gotypes.Map, *gotypes.Chan, *gotypes.Signature, *gotypes.Interface:
			continue
		}
		if goIsHostLayoutMarker(ft) {
			host = true
			break
		}
	}
	native := forceNative || c.hasNativeFieldsLocked(fields)
	if !native && len(fields) > 0 && c.hasNativeFieldsLocked(fields[:1]) {
		native = true
	}

	llFields := make([]*Var, len(fields))
	for i, f := range fields {
		var ft Type
		if native {
			if _, isSig := f.Type().Underlying().(*gotypes.Signature); isSig {
				ft = c.fromGoLocked(f.Type(), CDecl, true)
			} else {
				ft = c.fromGoLocked(f.Type(), Go, false)
			}
		} else {
			ft = c.fromGoLocked(f.Type(), Go, false)
		}
		var pkg *Package
		if f.Pkg() != nil {
			pkg = c.fromGoPkgLocked(f.Pkg())
		}
		llFields[i] = &Var{
			object: object{
				pos:   f.Pos(),
				pkg:   pkg,
				name:  f.Name(),
				typ:   ft,
				goObj: f,
			},
			embedded: f.Embedded(),
			isField:  true,
		}
	}

	return &Struct{
		fields:          llFields,
		tags:            tags,
		hostLayout:      host,
		nativeLayout:    native,
		isClosureStruct: isClosure,
		simdTagIndex:    -1,
		goType:          goType,
	}
}

func (c *Context) fromGoTupleLocked(gt *gotypes.Tuple) *Tuple {
	if gt == nil || gt.Len() == 0 {
		return nil
	}
	vars := make([]*Var, gt.Len())
	for i := 0; i < gt.Len(); i++ {
		vars[i] = c.fromGoVarLocked(gt.At(i))
	}
	return &Tuple{vars: vars, goType: gt}
}

func (c *Context) fromGoSignatureLocked(gt *gotypes.Signature, cc CallConv) *Signature {
	sig := &Signature{
		callConv: cc,
		variadic: gt.Variadic(),
		goType:   gt,
	}
	if recv := gt.Recv(); recv != nil {
		sig.recv = c.fromGoVarLocked(recv)
	}
	if rtps := gt.RecvTypeParams(); rtps != nil && rtps.Len() > 0 {
		list := make([]*TypeParam, rtps.Len())
		for i := 0; i < rtps.Len(); i++ {
			list[i] = c.fromGoTypeParamLocked(rtps.At(i))
		}
		sig.recvTypeParams = newTypeParamList(list)
	}
	if tps := gt.TypeParams(); tps != nil && tps.Len() > 0 {
		list := make([]*TypeParam, tps.Len())
		for i := 0; i < tps.Len(); i++ {
			list[i] = c.fromGoTypeParamLocked(tps.At(i))
		}
		sig.typeParams = newTypeParamList(list)
	}
	sig.params = c.fromGoTupleLocked(gt.Params())
	sig.results = c.fromGoTupleLocked(gt.Results())
	return sig
}

func (c *Context) fromGoTypeParamLocked(gt *gotypes.TypeParam) *TypeParam {
	if cached, ok := c.goTParams[gt]; ok {
		return cached
	}
	tp := &TypeParam{
		index:  gt.Index(),
		goType: gt,
	}
	c.goTParams[gt] = tp
	if gt.Obj() != nil {
		tn := c.fromGoTypeNameShellLocked(gt.Obj())
		if tn.typ == nil {
			tn.typ = tp
		}
		tp.obj = tn
	}
	if con := gt.Constraint(); con != nil {
		tp.constraint = c.fromGoLocked(con, Go, false)
	}
	return tp
}

// --- Object, Package, Scope, Selection, MethodSet & Info Lowering ---

// FromGoObject lowers any go/types.Object into its canonical LLGo Object.
func (c *Context) FromGoObject(obj gotypes.Object) Object {
	if obj == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fromGoObjectLocked(obj)
}

func (c *Context) fromGoObjectLocked(obj gotypes.Object) Object {
	if obj == nil {
		return nil
	}
	if cached, ok := c.goObjs[obj]; ok {
		return cached
	}
	switch o := obj.(type) {
	case *gotypes.Func:
		return c.fromGoFuncLocked(o)
	case *gotypes.Var:
		return c.fromGoVarLocked(o)
	case *gotypes.TypeName:
		return c.fromGoTypeNameLocked(o)
	case *gotypes.Const:
		res := &Const{
			object: object{
				pos:   o.Pos(),
				pkg:   c.fromGoPkgLocked(o.Pkg()),
				name:  o.Name(),
				goObj: o,
			},
			val: o.Val(),
		}
		c.goObjs[o] = res
		res.typ = c.fromGoLocked(o.Type(), Go, false)
		return res
	case *gotypes.PkgName:
		res := &PkgName{
			object: object{
				pos:   o.Pos(),
				pkg:   c.fromGoPkgLocked(o.Pkg()),
				name:  o.Name(),
				goObj: o,
			},
			imported: c.fromGoPkgLocked(o.Imported()),
		}
		c.goObjs[o] = res
		return res
	case *gotypes.Label:
		res := &Label{
			object: object{
				pos:   o.Pos(),
				pkg:   c.fromGoPkgLocked(o.Pkg()),
				name:  o.Name(),
				typ:   Typ[Invalid],
				goObj: o,
			},
		}
		c.goObjs[o] = res
		return res
	case *gotypes.Builtin:
		res := &Builtin{
			object: object{
				pos:   o.Pos(),
				name:  o.Name(),
				typ:   Typ[Invalid],
				goObj: o,
			},
		}
		c.goObjs[o] = res
		return res
	case *gotypes.Nil:
		res := &Nil{
			object: object{
				pos:   o.Pos(),
				name:  o.Name(),
				typ:   Typ[UntypedNil],
				goObj: o,
			},
		}
		c.goObjs[o] = res
		return res
	}
	return nil
}

func (c *Context) fromGoFuncLocked(o *gotypes.Func) *Func {
	if cached, ok := c.goObjs[o]; ok {
		return cached.(*Func)
	}
	f := &Func{
		object: object{
			pos:   o.Pos(),
			pkg:   c.fromGoPkgLocked(o.Pkg()),
			name:  o.Name(),
			goObj: o,
		},
		callConv: Go,
	}
	c.goObjs[o] = f
	if orig := o.Origin(); orig != nil && orig != o {
		f.origin = c.fromGoFuncLocked(orig)
	}
	if sig, ok := o.Type().(*gotypes.Signature); ok {
		f.typ = c.fromGoSignatureLocked(sig, Go)
	} else if o.Type() != nil {
		f.typ = c.fromGoLocked(o.Type(), Go, false)
	}
	return f
}

func (c *Context) fromGoVarLocked(o *gotypes.Var) *Var {
	if cached, ok := c.goObjs[o]; ok {
		return cached.(*Var)
	}
	v := &Var{
		object: object{
			pos:   o.Pos(),
			pkg:   c.fromGoPkgLocked(o.Pkg()),
			name:  o.Name(),
			goObj: o,
		},
		embedded: o.Embedded(),
		isField:  o.IsField(),
	}
	c.goObjs[o] = v
	if orig := o.Origin(); orig != nil && orig != o {
		v.origin = c.fromGoVarLocked(orig)
	}
	if o.Type() != nil {
		v.typ = c.fromGoLocked(o.Type(), Go, false)
	}
	return v
}

func (c *Context) fromGoTypeNameShellLocked(o *gotypes.TypeName) *TypeName {
	if o == nil {
		return nil
	}
	if cached, ok := c.goObjs[o]; ok {
		return cached.(*TypeName)
	}
	tn := &TypeName{
		object: object{
			pos:   o.Pos(),
			pkg:   c.fromGoPkgLocked(o.Pkg()),
			name:  o.Name(),
			goObj: o,
		},
		isAlias: o.IsAlias(),
	}
	c.goObjs[o] = tn
	return tn
}

func (c *Context) fromGoTypeNameLocked(o *gotypes.TypeName) *TypeName {
	tn := c.fromGoTypeNameShellLocked(o)
	if tn != nil && tn.typ == nil && o.Type() != nil {
		tn.typ = c.fromGoLocked(o.Type(), Go, false)
	}
	return tn
}

func (c *Context) fromGoPkgLocked(gp *gotypes.Package) *Package {
	if gp == nil {
		return nil
	}
	if cached, ok := c.goPkgs[gp]; ok {
		return cached
	}
	p := NewPackage(gp.Path(), gp.Name())
	p.complete = gp.Complete()
	p.goPkg = gp
	c.goPkgs[gp] = p
	return p
}

func (c *Context) fromGoScopeLocked(gs *gotypes.Scope) *Scope {
	if gs == nil {
		return nil
	}
	if cached, ok := c.goScopes[gs]; ok {
		return cached
	}
	var parent *Scope
	if gs.Parent() != nil && gs.Parent() != gotypes.Universe {
		parent = c.fromGoScopeLocked(gs.Parent())
	}
	s := NewScope(parent, gs.Pos(), gs.End(), "")
	c.goScopes[gs] = s
	for _, name := range gs.Names() {
		if obj := c.fromGoObjectLocked(gs.Lookup(name)); obj != nil {
			s.Insert(obj)
		}
	}
	return s
}

func (c *Context) fromGoSelectionLocked(gs *gotypes.Selection) *Selection {
	if gs == nil {
		return nil
	}
	idx := make([]int, len(gs.Index()))
	copy(idx, gs.Index())
	return &Selection{
		kind:     gs.Kind(),
		recv:     c.fromGoLocked(gs.Recv(), Go, false),
		obj:      c.fromGoObjectLocked(gs.Obj()),
		typ:      c.fromGoLocked(gs.Type(), Go, false),
		index:    idx,
		indirect: gs.Indirect(),
	}
}

// MethodSet returns the LLGo MethodSet of t.
func (c *Context) MethodSet(t Type) *MethodSet {
	return methodSetOf(c, t)
}

func methodSetOf(c *Context, t Type) *MethodSet {
	if t == nil {
		return &MethodSet{}
	}
	if c != nil {
		if gt := t.goTypeRef(); gt != nil {
			gms := gotypes.NewMethodSet(gt)
			c.mu.Lock()
			defer c.mu.Unlock()
			list := make([]*Selection, gms.Len())
			for i := 0; i < gms.Len(); i++ {
				list[i] = c.fromGoSelectionLocked(gms.At(i))
			}
			return &MethodSet{list: list}
		}
	}
	var list []*Selection
	switch u := Unalias(t).(type) {
	case *Named:
		for i := 0; i < u.NumMethods(); i++ {
			m := u.Method(i)
			if sig := m.Signature(); sig != nil && sig.recv != nil {
				if _, ptrRecv := sig.recv.typ.(*Pointer); ptrRecv {
					continue
				}
			}
			list = append(list, &Selection{
				kind:  MethodVal,
				recv:  t,
				obj:   m,
				typ:   m.typ,
				index: []int{i},
			})
		}
	case *Pointer:
		if n, ok := Unalias(u.elem).(*Named); ok {
			for i := 0; i < n.NumMethods(); i++ {
				m := n.Method(i)
				list = append(list, &Selection{
					kind:     MethodVal,
					recv:     t,
					obj:      m,
					typ:      m.typ,
					index:    []int{i},
					indirect: true,
				})
			}
		}
	case *Interface:
		for i := 0; i < u.NumMethods(); i++ {
			m := u.Method(i)
			list = append(list, &Selection{
				kind:  MethodVal,
				recv:  t,
				obj:   m,
				typ:   m.typ,
				index: []int{i},
			})
		}
	}
	return &MethodSet{list: list}
}

// Check type-checks files using go/types with LLGo's exact GoSizes (after
// pre-scanning type-level //llgo:* directives) and lowers the result once into
// a completely self-contained LLGo (*Package, *Info) carrying all //llgo:* and
// //go:wasm* symbol attributes.
func (c *Context) Check(pkgPath string, fset *token.FileSet, files []*ast.File, cfg *gotypes.Config) (*Package, *Info, error) {
	c.prescanTypeDirectives(pkgPath, files)

	var goCfg gotypes.Config
	if cfg != nil {
		goCfg = *cfg
	}
	if goCfg.Sizes == nil {
		goCfg.Sizes = c.GoSizes()
	}

	goInfo := &gotypes.Info{
		Types:      make(map[ast.Expr]gotypes.TypeAndValue),
		Defs:       make(map[*ast.Ident]gotypes.Object),
		Uses:       make(map[*ast.Ident]gotypes.Object),
		Implicits:  make(map[ast.Node]gotypes.Object),
		Selections: make(map[*ast.SelectorExpr]*gotypes.Selection),
		Scopes:     make(map[ast.Node]*gotypes.Scope),
		Instances:  make(map[*ast.Ident]gotypes.Instance),
	}

	goPkg, err := goCfg.Check(pkgPath, fset, files, goInfo)
	if goPkg == nil {
		return nil, nil, err
	}
	pkg, info := c.FromGoPackage(goPkg, goInfo, fset, files)
	return pkg, info, err
}

// FromGoPackage lowers an already-checked go/types Package and Info (plus
// optional AST files for //llgo:* directive binding) into a native LLGo
// (*Package, *Info).
func (c *Context) FromGoPackage(goPkg *gotypes.Package, goInfo *gotypes.Info, fset *token.FileSet, files []*ast.File) (*Package, *Info) {
	if goPkg != nil && len(files) > 0 {
		c.prescanTypeDirectives(goPkg.Path(), files)
	}

	c.mu.Lock()
	pkg := c.fromGoPkgLocked(goPkg)
	if goPkg != nil {
		pkg.complete = goPkg.Complete()
		pkg.scope = c.fromGoScopeLocked(goPkg.Scope())
		if imps := goPkg.Imports(); len(imps) > 0 {
			pkg.imports = make([]*Package, len(imps))
			for i, imp := range imps {
				pkg.imports[i] = c.fromGoPkgLocked(imp)
			}
		}
	}

	info := &Info{
		Context:    c,
		Types:      make(map[ast.Expr]TypeAndValue),
		Defs:       make(map[*ast.Ident]Object),
		Uses:       make(map[*ast.Ident]Object),
		Implicits:  make(map[ast.Node]Object),
		Selections: make(map[*ast.SelectorExpr]*Selection),
		Scopes:     make(map[ast.Node]*Scope),
		Instances:  make(map[*ast.Ident]Instance),
		goInfo:     goInfo,
	}

	if goInfo != nil {
		for id, obj := range goInfo.Defs {
			info.Defs[id] = c.fromGoObjectLocked(obj)
		}
		for id, obj := range goInfo.Uses {
			info.Uses[id] = c.fromGoObjectLocked(obj)
		}
		for node, obj := range goInfo.Implicits {
			info.Implicits[node] = c.fromGoObjectLocked(obj)
		}
		for expr, tv := range goInfo.Types {
			info.Types[expr] = TypeAndValue{
				Type:        c.fromGoLocked(tv.Type, Go, false),
				Value:       tv.Value,
				isType:      tv.IsType(),
				isValue:     tv.IsValue(),
				isNil:       tv.IsNil(),
				addressable: tv.Addressable(),
				assignable:  tv.Assignable(),
				hasOk:       tv.HasOk(),
				isBuiltin:   tv.IsBuiltin(),
			}
		}
		for selExpr, sel := range goInfo.Selections {
			info.Selections[selExpr] = c.fromGoSelectionLocked(sel)
		}
		for node, sc := range goInfo.Scopes {
			info.Scopes[node] = c.fromGoScopeLocked(sc)
		}
		if len(goInfo.InitOrder) > 0 {
			info.InitOrder = make([]*Initializer, len(goInfo.InitOrder))
			for i, init := range goInfo.InitOrder {
				lhs := make([]*Var, len(init.Lhs))
				for j, v := range init.Lhs {
					lhs[j] = c.fromGoVarLocked(v)
				}
				info.InitOrder[i] = &Initializer{Lhs: lhs, Rhs: init.Rhs}
			}
		}
		for id, inst := range goInfo.Instances {
			var args []Type
			if inst.TypeArgs != nil && inst.TypeArgs.Len() > 0 {
				args = make([]Type, inst.TypeArgs.Len())
				for i := 0; i < inst.TypeArgs.Len(); i++ {
					args[i] = c.fromGoLocked(inst.TypeArgs.At(i), Go, false)
				}
			}
			info.Instances[id] = Instance{
				TypeArgs: newTypeList(args),
				Type:     c.fromGoLocked(inst.Type, Go, false),
			}
		}
	}
	c.mu.Unlock()

	if len(files) > 0 {
		c.bindDeclDirectives(pkg, info, files)
	}
	return pkg, info
}

func (c *Context) prescanTypeDirectives(pkgPath string, files []*ast.File) {
	if len(files) == 0 {
		return
	}
	changed := false
	c.mu.Lock()
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name == nil {
					continue
				}
				doc := ts.Doc
				if doc == nil && len(gd.Specs) == 1 {
					doc = gd.Doc
				}
				if cc, ok := parseTypeCallConvDirective(doc); ok {
					key := ts.Name.Name
					if pkgPath != "" {
						key = pkgPath + "." + key
					}
					c.typeCallConvs[key] = cc
					if cc.IsNative() {
						c.hasNativeMetadata.Store(true)
					}
					changed = true
				}
			}
		}
	}
	if changed {
		c.resetGoCachesLocked()
	}
	c.mu.Unlock()
}

func parseTypeCallConvDirective(doc *ast.CommentGroup) (CallConv, bool) {
	res, err := directive.ParseTypeCallConv(doc)
	if err != nil {
		return Go, false
	}
	switch res.CallConv {
	case directive.CallConvC:
		return CDecl, true
	case directive.CallConvStdcall:
		return Stdcall, true
	default:
		return Go, false
	}
}

func (c *Context) bindDeclDirectives(pkg *Package, info *Info, files []*ast.File) {
	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name == nil {
					continue
				}
				fn, ok := info.Defs[d.Name].(*Func)
				if !ok || fn == nil {
					continue
				}
				if d.Body == nil {
					fn.isExternal = true
				}
				for _, dir := range directive.ParseGroup(d.Doc) {
					switch dir.Name {
					case "llgo:link", "go:linkname":
						if fields := strings.Fields(dir.Args); len(fields) >= 2 {
							target := fields[1]
							fn.linkName = target
							fn.isExternal = true
							if cc, ok := callConvFromLinkTarget(target); ok {
								fn.callConv = cc
								if sig := fn.Signature(); sig != nil {
									sig.callConv = cc
								}
							}
						}
					case "llgo:cdecl":
						fn.callConv = CDecl
						if sig := fn.Signature(); sig != nil {
							sig.callConv = CDecl
						}
					case "llgo:stdcall":
						fn.callConv = Stdcall
						if sig := fn.Signature(); sig != nil {
							sig.callConv = Stdcall
						}
					case "llgo:skip":
						fn.isSkip = true
					case "llgo:pylist":
						fn.isPyList = true
					case "go:wasmimport":
						if fields := strings.Fields(dir.Args); len(fields) >= 2 {
							fn.wasmMod = fields[0]
							fn.wasmName = fields[1]
							fn.callConv = Wasm
							fn.isExternal = true
							if sig := fn.Signature(); sig != nil {
								sig.callConv = Wasm
							}
						}
					case "go:wasmexport":
						if fields := strings.Fields(dir.Args); len(fields) >= 1 {
							fn.wasmExport = fields[0]
						}
					}
				}
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					doc := vs.Doc
					if doc == nil && len(d.Specs) == 1 {
						doc = d.Doc
					}
					for _, dir := range directive.ParseGroup(doc) {
						if dir.Name != "llgo:link" && dir.Name != "go:linkname" {
							continue
						}
						fields := strings.Fields(dir.Args)
						if len(fields) < 2 {
							continue
						}
						for _, id := range vs.Names {
							if id.Name == fields[0] {
								if v, ok := info.Defs[id].(*Var); ok && v != nil {
									v.linkName = fields[1]
									v.isExternal = true
								}
							}
						}
					}
				}
			}
		}
	}
	_ = pkg
}

func callConvFromLinkTarget(target string) (CallConv, bool) {
	switch {
	case strings.HasPrefix(target, "C."):
		return CDecl, true
	case strings.HasPrefix(target, "py."):
		return Py, true
	default:
		return Go, false
	}
}
