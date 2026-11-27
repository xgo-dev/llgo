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

type aliasCallConv struct {
	name string
	cc   CallConv
}

// Context is the single unified engine for LLGo's type system:
//   - Canonical Type interning and go/types lowering
//   - Single-source-of-truth Layout, GC bitmap, and target sizing
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
	typeAliasCallConvs map[string][]aliasCallConv
	checkedGoPkgs      map[string]*gotypes.Package
	nativeStructFields map[token.Pos]bool
	nativeGoVars       map[gotypes.Object]bool
	nativeVars         map[*Var]bool

	// Interning tables.
	ptrs   map[ptrInternKey]*Pointer
	slices map[Type]*Slice
	maps   map[mapInternKey]*Map
	chans  map[chanInternKey]*Chan
	arrays map[arrayInternKey]*Array
	simds  map[simdInternKey]*SIMDVec

	// Memoized go/types -> llgo/types lowering tables.
	goCtx             *gotypes.Context
	goPkgs            map[*gotypes.Package]*Package
	llToGoPkgs        map[*Package]*gotypes.Package
	llToGoTypes       map[Type]gotypes.Type
	llToGoObjs        map[Object]gotypes.Object
	llToGoTParams     map[*TypeParam]*gotypes.TypeParam
	goScopes          map[*gotypes.Scope]*Scope
	goObjs            map[gotypes.Object]Object
	goTypes           map[goTypeCacheKey]Type
	goNamed           map[*gotypes.Named]*Named
	goAliases         map[gotypes.Type]*Alias
	goTParams         map[*gotypes.TypeParam]*TypeParam
	incompleteNamed   []*Named
	incompleteLLNamed map[*Named]Type

	// Memoized physical layout cache.
	layouts map[Type]*Layout
}

// NewContext creates a new Context using the host's default target spec
// (matching go/types.NewContext).
func NewContext() *Context {
	return NewContextWithConfig(ContextConfig{})
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
		hostLayouts = defaultHostBasicLayouts(spec)
	}
	simdAligns := cfg.SIMDAlignments
	if simdAligns[0] == 0 {
		simdAligns = defaultSIMDAlignments(spec)
	}
	return &Context{
		spec:               spec,
		hostBasicLayouts:   hostLayouts,
		simdAlignments:     simdAligns,
		typeCallConvs:      make(map[string]CallConv),
		nativeStructFields: make(map[token.Pos]bool),
		nativeGoVars:       make(map[gotypes.Object]bool),
		nativeVars:         make(map[*Var]bool),
		ptrs:               make(map[ptrInternKey]*Pointer),
		slices:             make(map[Type]*Slice),
		maps:               make(map[mapInternKey]*Map),
		chans:              make(map[chanInternKey]*Chan),
		arrays:             make(map[arrayInternKey]*Array),
		simds:              make(map[simdInternKey]*SIMDVec),
		goPkgs:             make(map[*gotypes.Package]*Package),
		llToGoPkgs:         make(map[*Package]*gotypes.Package),
		llToGoTypes:        make(map[Type]gotypes.Type),
		llToGoObjs:         make(map[Object]gotypes.Object),
		llToGoTParams:      make(map[*TypeParam]*gotypes.TypeParam),
		goScopes:           make(map[*gotypes.Scope]*Scope),
		goObjs:             make(map[gotypes.Object]Object),
		goTypes:            make(map[goTypeCacheKey]Type),
		goNamed:            make(map[*gotypes.Named]*Named),
		goAliases:          make(map[gotypes.Type]*Alias),
		goTParams:          make(map[*gotypes.TypeParam]*TypeParam),
		layouts:            make(map[Type]*Layout),
	}
}

func (c *Context) ensureGoContext() *gotypes.Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.goCtx == nil {
		c.goCtx = gotypes.NewContext()
	}
	return c.goCtx
}

// HasNativeMetadata reports whether any native calling-convention type or
// native struct field has been registered in c.
func (c *Context) HasNativeMetadata() bool {
	return c.hasNativeMetadata.Load()
}

// Spec returns the target ABI specification configured for c.
func (c *Context) Spec() target.Spec {
	if c == nil {
		return target.ResolveSpec(target.Spec{})
	}
	return c.spec
}

// SetTypeCallConv registers a type-level calling convention directive (such as
// //llgo:cdecl or //llgo:stdcall) by qualified type name ("pkg.Name") and
// invalidates cached layouts so subsequent queries reflect it.
func (c *Context) SetTypeCallConv(typeKey string, cc CallConv) {
	c.mu.Lock()
	c.typeCallConvs[typeKey] = cc
	if cc.IsNative() {
		c.hasNativeMetadata.Store(true)
	}
	for _, n := range c.goNamed {
		if typeNameKey(n.obj) == typeKey {
			n.callConv = cc
			n.hasCallConv = true
			n.resolved = false
		}
	}
	for _, a := range c.goAliases {
		if typeNameKey(a.obj) == typeKey {
			a.callConv = cc
			a.hasCallConv = true
			if sig, ok := a.rhs.(*Signature); ok {
				a.rhs = sig.WithCallConv(cc)
			}
		}
	}
	c.goTypes = make(map[goTypeCacheKey]Type)
	c.layouts = make(map[Type]*Layout)
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
	c.mu.Unlock()
}

func (c *Context) resetGoCachesLocked() {
	c.goTypes = make(map[goTypeCacheKey]Type)
	for _, n := range c.goNamed {
		if n.goNamed != nil {
			if st, ok := n.goNamed.Underlying().(*gotypes.Struct); ok && st.NumFields() > 0 {
				if c.hasNativeFieldsLocked([]*gotypes.Var{st.Field(0)}) {
					n.resolved = false
				}
			}
		}
	}
	c.layouts = make(map[Type]*Layout)
}

func (c *Context) checkIncompleteNamedLocked() {
	changed := false
	if len(c.incompleteNamed) > 0 {
		remaining := c.incompleteNamed[:0]
		for _, n := range c.incompleteNamed {
			if n.resolved && n.goNamed != nil && n.goNamed.Underlying() != n.goUnderlying {
				n.resolved = false
				changed = true
			} else {
				remaining = append(remaining, n)
			}
		}
		c.incompleteNamed = remaining
	}
	if len(c.incompleteLLNamed) > 0 {
		for n, prev := range c.incompleteLLNamed {
			if n.underlying != prev {
				c.incompleteLLNamed = nil
				changed = true
				break
			}
		}
	}
	if changed {
		c.layouts = make(map[Type]*Layout)
	}
}

func (c *Context) trackIncompleteLLNamed(n *Named) {
	if n == nil || n.goNamed != nil || c.layouts == nil {
		return
	}
	c.mu.Lock()
	if c.incompleteLLNamed == nil {
		c.incompleteLLNamed = make(map[*Named]Type)
	}
	c.incompleteLLNamed[n] = n.underlying
	c.mu.Unlock()
}

// --- Canonical Type Interning (Internal) ---

func (c *Context) internPtrLocked(elem Type, kind PtrKind, goType gotypes.Type) *Pointer {
	k := ptrInternKey{elem: elem, kind: kind}
	if p, ok := c.ptrs[k]; ok {
		return p
	}
	p := &Pointer{elem: elem, kind: kind, goType: goType}
	c.ptrs[k] = p
	return p
}

func (c *Context) internSliceLocked(elem Type, goType gotypes.Type) *Slice {
	if s, ok := c.slices[elem]; ok {
		return s
	}
	s := &Slice{elem: elem, goType: goType}
	c.slices[elem] = s
	return s
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

func (c *Context) internChanLocked(dir ChanDir, elem Type, goType gotypes.Type) *Chan {
	k := chanInternKey{dir: dir, elem: elem}
	if ch, ok := c.chans[k]; ok {
		return ch
	}
	ch := &Chan{dir: dir, elem: elem, goType: goType}
	c.chans[k] = ch
	return ch
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
	c.resolveNamedMethodsLocked(n)
}

func (c *Context) resolveNamedMethodsLocked(n *Named) {
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
	c.resolveInterfaceLocked(iface)
}

func (c *Context) resolveInterfaceLocked(iface *Interface) {
	if iface.resolved || iface.goIface == nil {
		return
	}
	iface.resolved = true
	iface.completed = true
	gt := iface.goIface
	iface.comparable = gt.IsComparable()
	iface.implicit = gt.IsImplicit()
	iface.methodSet = gt.IsMethodSet()
	if n := gt.NumExplicitMethods(); n > 0 {
		iface.explicitMethods = make([]*Func, n)
		for i := 0; i < n; i++ {
			iface.explicitMethods[i] = c.fromGoFuncLocked(gt.ExplicitMethod(i))
		}
	}
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
		if !hasOverride && aObj != nil && aObj.Parent() == gotypes.Universe {
			if uo := Universe.Lookup(aObj.Name()); uo != nil && uo.Type() != nil {
				c.goTypes[key] = uo.Type()
				return uo.Type()
			}
		}
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
		a := &Alias{
			obj:         tn,
			callConv:    effCC,
			hasCallConv: effOverride,
			goType:      t,
		}
		if tn.typ == nil && !hasOverride && (aObj == nil || aObj.Type() == t) {
			tn.typ = a
		}
		if !hasOverride {
			c.goAliases[t] = a
		}
		c.goTypes[key] = a
		if orig, tps, targs := aliasGenericInfo(t); orig != nil || (tps != nil && tps.Len() > 0) || (targs != nil && targs.Len() > 0) {
			if orig != nil {
				if origA, ok := c.fromGoLocked(orig, Go, false).(*Alias); ok {
					a.origin = origA
				}
			}
			if tps != nil && tps.Len() > 0 {
				list := make([]*TypeParam, tps.Len())
				for i := 0; i < tps.Len(); i++ {
					list[i] = c.fromGoTypeParamLocked(tps.At(i))
				}
				a.typeParams = newTypeParamList(list)
			}
			if targs != nil && targs.Len() > 0 {
				list := make([]Type, targs.Len())
				for i := 0; i < targs.Len(); i++ {
					list[i] = c.fromGoLocked(targs.At(i), Go, false)
				}
				a.typeArgs = newTypeList(list)
			}
		}
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
		} else if len(c.typeAliasCallConvs) > 0 {
			if aliasCC, ok := c.lookupErasedAliasCallConvLocked(gt); ok {
				effCC = aliasCC
			}
		}
		res := c.fromGoSignatureLocked(gt, effCC)
		c.goTypes[key] = res
		return res

	case *gotypes.Interface:
		iface := &Interface{
			ctx:       c,
			goIface:   gt,
			methodSet: true,
			goType:    gt,
		}
		c.goTypes[key] = iface
		return iface

	case *gotypes.Union:
		n := gt.Len()
		terms := make([]*Term, n)
		for i := 0; i < n; i++ {
			term := gt.Term(i)
			terms[i] = NewTerm(term.Tilde(), c.fromGoLocked(term.Type(), Go, false))
		}
		u := &Union{terms: terms, goType: gt}
		c.goTypes[key] = u
		return u

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
	if !hasOverride && gt.Obj() != nil && gt.Obj().Parent() == gotypes.Universe {
		if uo := Universe.Lookup(gt.Obj().Name()); uo != nil {
			if un, ok := uo.Type().(*Named); ok {
				c.goNamed[gt] = un
				c.goTypes[goTypeCacheKey{typ: gt, cc: cc, hasOverride: hasOverride}] = un
				return un
			}
		}
	}
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
	orig := gt.Origin()
	if tn.typ == nil && !hasOverride && (orig == nil || orig == gt) && (gt.Obj() == nil || gt.Obj().Type() == gt) {
		tn.typ = n
	}

	if !hasOverride {
		c.goNamed[gt] = n
	}
	c.goTypes[goTypeCacheKey{typ: gt, cc: cc, hasOverride: hasOverride}] = n
	if !hasOverride && orig != nil && orig != gt {
		n.origin = c.fromGoNamedShellLocked(orig, Go, false)
	}
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
		if !native {
			llFields[i] = c.fromGoVarWithKindLocked(f, FieldVar)
			continue
		}
		var ft Type
		if _, isSig := f.Type().Underlying().(*gotypes.Signature); isSig {
			ft = c.fromGoLocked(f.Type(), CDecl, true)
		} else {
			ft = c.fromGoLocked(f.Type(), Go, false)
		}
		var pkg *Package
		if f.Pkg() != nil {
			pkg = c.fromGoPkgLocked(f.Pkg())
		}
		lf := &Var{
			object: object{
				pos:   f.Pos(),
				pkg:   pkg,
				name:  f.Name(),
				typ:   ft,
				goObj: f,
			},
			embedded: f.Embedded(),
			isField:  true,
			kind:     FieldVar,
		}
		if orig := f.Origin(); orig != nil && orig != f {
			lf.origin = c.fromGoVarWithKindLocked(orig, FieldVar)
		}
		if _, exists := c.goObjs[f]; !exists {
			c.goObjs[f] = lf
		}
		llFields[i] = lf
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
	return c.fromGoTupleWithKindLocked(gt, ParamVar)
}

func (c *Context) fromGoTupleWithKindLocked(gt *gotypes.Tuple, kind VarKind) *Tuple {
	if gt == nil || gt.Len() == 0 {
		return nil
	}
	vars := make([]*Var, gt.Len())
	for i := 0; i < gt.Len(); i++ {
		vars[i] = c.fromGoVarWithKindLocked(gt.At(i), kind)
	}
	return &Tuple{vars: vars, goType: gt}
}

func (c *Context) fromGoSignatureLocked(gt *gotypes.Signature, cc CallConv) *Signature {
	sig := &Signature{
		ctx:      c,
		callConv: cc,
		variadic: gt.Variadic(),
		goType:   gt,
	}
	if recv := gt.Recv(); recv != nil {
		sig.recv = c.fromGoVarWithKindLocked(recv, RecvVar)
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
	sig.params = c.fromGoTupleWithKindLocked(gt.Params(), ParamVar)
	sig.results = c.fromGoTupleWithKindLocked(gt.Results(), ResultVar)
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
		if tn, isTN := cached.(*TypeName); isTN && tn.typ == nil && obj.Type() != nil {
			tn.typ = c.fromGoLocked(obj.Type(), Go, false)
		}
		return cached
	}
	if pub := c.findPublishedGoObjectLocked(obj); pub != nil {
		if tn, isTN := pub.(*TypeName); isTN && tn.typ == nil && obj.Type() != nil {
			tn.typ = c.fromGoLocked(obj.Type(), Go, false)
		}
		return pub
	}
	if obj.Parent() == gotypes.Universe {
		if uo := Universe.Lookup(obj.Name()); uo != nil {
			c.goObjs[obj] = uo
			return uo
		}
	}
	if obj.Pkg() == gotypes.Unsafe {
		if uo := Unsafe.Scope().Lookup(obj.Name()); uo != nil {
			c.goObjs[obj] = uo
			return uo
		}
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
		if o.Parent() != nil {
			res.parent = c.fromGoScopeLocked(o.Parent())
		}
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
		if o.Parent() != nil {
			res.parent = c.fromGoScopeLocked(o.Parent())
		}
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
		if o.Parent() != nil {
			res.parent = c.fromGoScopeLocked(o.Parent())
		}
		return res
	case *gotypes.Builtin:
		res := &Builtin{
			object: object{
				pos:    o.Pos(),
				parent: Universe,
				name:   o.Name(),
				typ:    Typ[Invalid],
				goObj:  o,
			},
		}
		c.goObjs[o] = res
		return res
	case *gotypes.Nil:
		res := &Nil{
			object: object{
				pos:    o.Pos(),
				parent: Universe,
				name:   o.Name(),
				typ:    Typ[UntypedNil],
				goObj:  o,
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
	if pub, ok := c.findPublishedGoObjectLocked(o).(*Func); ok {
		return pub
	}
	f := &Func{
		object: object{
			pos:   o.Pos(),
			pkg:   c.fromGoPkgLocked(o.Pkg()),
			name:  o.Name(),
			goObj: o,
		},
	}
	c.goObjs[o] = f
	if o.Parent() != nil {
		f.parent = c.fromGoScopeLocked(o.Parent())
	}
	if _, ok := o.Type().(*gotypes.Signature); ok && o.Scope() != nil {
		f.scope = c.fromGoScopeLocked(o.Scope())
	}
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

func inferGoVarKind(o *gotypes.Var) VarKind {
	if o.IsField() {
		return FieldVar
	}
	if p := o.Parent(); p != nil {
		if o.Pkg() != nil && p == o.Pkg().Scope() {
			return PackageVar
		}
		return LocalVar
	}
	return PackageVar
}

func (c *Context) fromGoVarLocked(o *gotypes.Var) *Var {
	return c.fromGoVarWithKindLocked(o, inferGoVarKind(o))
}

func (c *Context) fromGoVarWithKindLocked(o *gotypes.Var, kind VarKind) *Var {
	if cached, ok := c.goObjs[o]; ok {
		v := cached.(*Var)
		if kind == RecvVar || kind == ParamVar || kind == ResultVar || kind == FieldVar {
			v.kind = kind
		}
		return v
	}
	if pub, ok := c.findPublishedGoObjectLocked(o).(*Var); ok {
		if kind == RecvVar || kind == ParamVar || kind == ResultVar || kind == FieldVar {
			pub.kind = kind
		}
		return pub
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
		kind:     kind,
	}
	c.goObjs[o] = v
	if o.Parent() != nil {
		v.parent = c.fromGoScopeLocked(o.Parent())
	}
	if orig := o.Origin(); orig != nil && orig != o {
		v.origin = c.fromGoVarWithKindLocked(orig, kind)
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
	if pub, ok := c.findPublishedGoObjectLocked(o).(*TypeName); ok {
		return pub
	}
	if o.Parent() == gotypes.Universe {
		if uo, ok := Universe.Lookup(o.Name()).(*TypeName); ok {
			c.goObjs[o] = uo
			return uo
		}
	}
	if o.Pkg() == gotypes.Unsafe {
		if uo, ok := Unsafe.Scope().Lookup(o.Name()).(*TypeName); ok {
			c.goObjs[o] = uo
			return uo
		}
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
	if o.Parent() != nil {
		tn.parent = c.fromGoScopeLocked(o.Parent())
	}
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
	if gp == gotypes.Unsafe {
		return Unsafe
	}
	if cached, ok := c.goPkgs[gp]; ok {
		return cached
	}
	p := NewPackage(gp.Path(), gp.Name())
	p.complete = gp.Complete()
	p.ctx = c
	p.goPkg = gp
	c.goPkgs[gp] = p
	if gs := gp.Scope(); gs != nil {
		p.scope.pos = gs.Pos()
		p.scope.end = gs.End()
		p.scope.goScope = gs
		c.goScopes[gs] = p.scope
	}
	return p
}

func findChildByGoScope(parent *Scope, gs *gotypes.Scope) *Scope {
	if parent == nil || gs == nil {
		return nil
	}
	for _, ch := range parent.children {
		if ch.goScope == gs {
			return ch
		}
	}
	return nil
}

// findPublishedGoScopeLocked locates an already-published LLGo *Scope matching
// gs by walking up to the cached package scope and descending existing children,
// without allocating new scopes or repopulating pruned cache entries.
func (c *Context) findPublishedGoScopeLocked(gs *gotypes.Scope) *Scope {
	if gs == nil {
		return nil
	}
	if gs == gotypes.Universe {
		return Universe
	}
	if cached, ok := c.goScopes[gs]; ok {
		return cached
	}
	gp := gs.Parent()
	if gp == nil {
		return nil
	}
	parent := c.findPublishedGoScopeLocked(gp)
	if parent == nil {
		return nil
	}
	return findChildByGoScope(parent, gs)
}

func (c *Context) findPublishedGoObjectLocked(obj gotypes.Object) Object {
	if obj == nil {
		return nil
	}
	gp := obj.Parent()
	if gp == nil {
		return nil
	}
	parent := c.findPublishedGoScopeLocked(gp)
	if parent == nil {
		return nil
	}
	existing := parent.Lookup(obj.Name())
	if carrier, ok := existing.(goObjectCarrier); ok && carrier.goObjectRef() == obj {
		return existing
	}
	return nil
}

func (c *Context) fromGoScopeLocked(gs *gotypes.Scope) *Scope {
	if gs == nil {
		return nil
	}
	if gs == gotypes.Universe {
		return Universe
	}
	if cached, ok := c.goScopes[gs]; ok {
		return cached
	}
	if pub := c.findPublishedGoScopeLocked(gs); pub != nil {
		return pub
	}
	var parent *Scope
	if gp := gs.Parent(); gp != nil {
		parent = c.fromGoScopeLocked(gp)
		if gp != gotypes.Universe {
			c.lowerGoScopeChildrenLocked(gp, parent)
			if cached, ok := c.goScopes[gs]; ok {
				return cached
			}
			if pub := findChildByGoScope(parent, gs); pub != nil {
				return pub
			}
		}
	}
	s := NewScope(parent, gs.Pos(), gs.End(), "")
	s.goScope = gs
	c.goScopes[gs] = s
	c.populateGoScopeObjectsLocked(gs, s)
	return s
}

func (c *Context) lowerGoScopeChildrenLocked(gp *gotypes.Scope, parent *Scope) {
	n := gp.NumChildren()
	if n == 0 {
		return
	}
	var created []*gotypes.Scope
	for i := 0; i < n; i++ {
		gc := gp.Child(i)
		if _, ok := c.goScopes[gc]; ok {
			continue
		}
		if pub := findChildByGoScope(parent, gc); pub != nil {
			c.goScopes[gc] = pub
			continue
		}
		cs := NewScope(parent, gc.Pos(), gc.End(), "")
		cs.goScope = gc
		c.goScopes[gc] = cs
		created = append(created, gc)
	}
	for _, gc := range created {
		c.populateGoScopeObjectsLocked(gc, c.goScopes[gc])
	}
}

func (c *Context) lowerGoScopeTreeLocked(gs *gotypes.Scope, s *Scope) {
	c.lowerGoScopeChildrenLocked(gs, s)
	for i := 0; i < gs.NumChildren(); i++ {
		gc := gs.Child(i)
		c.lowerGoScopeTreeLocked(gc, c.goScopes[gc])
	}
}

func (c *Context) populateGoScopeObjectsLocked(gs *gotypes.Scope, s *Scope) {
	for _, name := range gs.Names() {
		if obj := c.fromGoObjectLocked(gs.Lookup(name)); obj != nil {
			s.Insert(obj)
		}
	}
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

func (c *Context) fromGoTypeAndValueLocked(tv gotypes.TypeAndValue) TypeAndValue {
	return TypeAndValue{
		Type:        c.fromGoLocked(tv.Type, Go, false),
		Value:       tv.Value,
		isVoid:      tv.IsVoid(),
		isType:      tv.IsType(),
		isValue:     tv.IsValue(),
		isNil:       tv.IsNil(),
		addressable: tv.Addressable(),
		assignable:  tv.Assignable(),
		hasOk:       tv.HasOk(),
		isBuiltin:   tv.IsBuiltin(),
	}
}

func (c *Context) populateFromGoPackage(dstPkg *Package, goPkg *gotypes.Package, dstInfo *Info, goInfo *gotypes.Info, goVersion string, files []*ast.File, startChild int) *Package {
	c.mu.Lock()
	defer c.mu.Unlock()

	pkg := dstPkg
	if goPkg != nil {
		if pkg == nil {
			pkg = c.fromGoPkgLocked(goPkg)
		} else {
			c.goPkgs[goPkg] = pkg
			pkg.goPkg = goPkg
			pkg.ctx = c
			if goPkg.Name() != "" {
				pkg.name = goPkg.Name()
			}
		}
		pkg.complete = goPkg.Complete()
		if goVersion != "" {
			pkg.goVersion = goVersion
		}
		if gs := goPkg.Scope(); gs != nil {
			if pkg.scope == nil {
				pkg.scope = c.fromGoScopeLocked(gs)
			} else {
				pkg.scope.goScope = gs
				c.goScopes[gs] = pkg.scope
				c.populateGoScopeObjectsLocked(gs, pkg.scope)
			}
			if startChild < 0 {
				startChild = 0
			}
			for i := startChild; i < gs.NumChildren(); i++ {
				gc := gs.Child(i)
				cs, ok := c.goScopes[gc]
				if !ok {
					cs = findChildByGoScope(pkg.scope, gc)
				}
				if cs == nil {
					cs = NewScope(pkg.scope, gc.Pos(), gc.End(), "")
					cs.goScope = gc
					c.goScopes[gc] = cs
					c.populateGoScopeObjectsLocked(gc, cs)
				} else {
					c.goScopes[gc] = cs
				}
				c.lowerGoScopeTreeLocked(gc, cs)
			}
		}
		if imps := goPkg.Imports(); len(imps) > 0 {
			pkg.imports = make([]*Package, len(imps))
			for i, imp := range imps {
				pkg.imports[i] = c.fromGoPkgLocked(imp)
			}
		}
	}

	c.populateInfoFromGoLocked(dstInfo, goInfo, goVersion, files)
	if goPkg != nil {
		if gs := goPkg.Scope(); gs != nil {
			for i := startChild; i < gs.NumChildren(); i++ {
				c.pruneLocalGoScopeLocked(gs.Child(i))
			}
		}
	}
	return pkg
}

// pruneLocalGoScopeLocked drops file-, function-, and block-local go/types
// scopes and the declarations owned by those scopes (obj.Parent() == gs) from
// the Context cache after a package has finished lowering into its LLGo
// Scope/Info. Objects owned by other scopes (such as dot-imported package
// declarations) remain cached for cross-package identity.
func (c *Context) pruneLocalGoScopeLocked(gs *gotypes.Scope) {
	if gs == nil {
		return
	}
	delete(c.goScopes, gs)
	for _, name := range gs.Names() {
		if obj := gs.Lookup(name); obj != nil && obj.Parent() == gs {
			delete(c.goObjs, obj)
		}
	}
	for i := 0; i < gs.NumChildren(); i++ {
		c.pruneLocalGoScopeLocked(gs.Child(i))
	}
}

func (c *Context) populateInfoFromGoLocked(info *Info, goInfo *gotypes.Info, goVersion string, files []*ast.File) {
	if info == nil {
		return
	}
	if goInfo != nil {
		if info.Defs != nil {
			for id, obj := range goInfo.Defs {
				info.Defs[id] = c.fromGoObjectLocked(obj)
			}
		}
		if info.Uses != nil {
			for id, obj := range goInfo.Uses {
				info.Uses[id] = c.fromGoObjectLocked(obj)
			}
		}
		if info.Implicits != nil {
			for node, obj := range goInfo.Implicits {
				info.Implicits[node] = c.fromGoObjectLocked(obj)
			}
		}
		if info.Types != nil {
			for expr, tv := range goInfo.Types {
				info.Types[expr] = c.fromGoTypeAndValueLocked(tv)
			}
		}
		if info.Selections != nil {
			for selExpr, sel := range goInfo.Selections {
				info.Selections[selExpr] = c.fromGoSelectionLocked(sel)
			}
		}
		if info.Scopes != nil {
			for node, sc := range goInfo.Scopes {
				info.Scopes[node] = c.fromGoScopeLocked(sc)
			}
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
		if info.Instances != nil {
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
	}
	if info.FileVersions != nil {
		copyGoInfoFileVersions(info.FileVersions, goInfo)
		for _, f := range files {
			if f != nil {
				if _, ok := info.FileVersions[f]; !ok {
					info.FileVersions[f] = goVersion
				}
			}
		}
	}
}

func (c *Context) registerCheckedGoPackage(pkg *Package, goPkg *gotypes.Package) {
	if c == nil || goPkg == nil {
		return
	}
	c.mu.Lock()
	// Bind goPkg -> pkg before go/types runs so that any Sizes/FromGo query
	// triggered during type checking (e.g. unsafe.Sizeof/Offsetof constants)
	// reuses the caller's *Package instead of creating an orphan duplicate.
	if pkg != nil {
		c.goPkgs[goPkg] = pkg
		pkg.goPkg = goPkg
		pkg.ctx = c
		if gs := goPkg.Scope(); gs != nil && pkg.scope != nil {
			pkg.scope.goScope = gs
			c.goScopes[gs] = pkg.scope
		}
	}
	if c.checkedGoPkgs == nil {
		c.checkedGoPkgs = make(map[string]*gotypes.Package)
	}
	pkgPath := goPkg.Path()
	if pkg != nil && pkg.path != "" {
		pkgPath = pkg.path
	}
	c.checkedGoPkgs[pkgPath] = goPkg
	c.mu.Unlock()
}

// lookupErasedAliasCallConvLocked recovers a native calling convention attached
// to a function type alias on Go toolchains where go/types erases aliases into
// bare *gotypes.Signature values (Go 1.20/1.21, or Go 1.22 with gotypesalias=0).
func (c *Context) lookupErasedAliasCallConvLocked(sig *gotypes.Signature) (CallConv, bool) {
	for pkgPath, aliases := range c.typeAliasCallConvs {
		gp := c.checkedGoPkgs[pkgPath]
		if gp == nil || gp.Scope() == nil {
			continue
		}
		scope := gp.Scope()
		for _, a := range aliases {
			if obj := scope.Lookup(a.name); obj != nil && obj.Type() == sig {
				return a.cc, true
			}
		}
	}
	return Go, false
}

func (c *Context) prescanTypeDirectives(pkgPath string, files []*ast.File, isolatedAliases map[string]bool) {
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
					if prev, exists := c.typeCallConvs[key]; !exists || prev != cc {
						c.typeCallConvs[key] = cc
						if cc.IsNative() {
							c.hasNativeMetadata.Store(true)
						}
						changed = true
					}
					if cc.IsNative() {
						if st, ok := ts.Type.(*ast.StructType); ok && st.Fields != nil {
							if c.nativeStructFields == nil {
								c.nativeStructFields = make(map[token.Pos]bool)
							}
							for _, field := range st.Fields.List {
								if len(field.Names) == 0 {
									if pos := embeddedStructFieldPos(field.Type); pos.IsValid() {
										c.nativeStructFields[pos] = true
										c.hasNativeMetadata.Store(true)
										changed = true
									}
									continue
								}
								for _, name := range field.Names {
									if pos := name.Pos(); pos.IsValid() {
										c.nativeStructFields[pos] = true
										c.hasNativeMetadata.Store(true)
										changed = true
									}
								}
							}
						}
					}
					if ts.Assign.IsValid() && isolatedAliases[ts.Name.Name] {
						if c.typeAliasCallConvs == nil {
							c.typeAliasCallConvs = make(map[string][]aliasCallConv)
						}
						found := false
						for i, existing := range c.typeAliasCallConvs[pkgPath] {
							if existing.name == ts.Name.Name {
								c.typeAliasCallConvs[pkgPath][i].cc = cc
								found = true
								break
							}
						}
						if !found {
							c.typeAliasCallConvs[pkgPath] = append(c.typeAliasCallConvs[pkgPath], aliasCallConv{name: ts.Name.Name, cc: cc})
						}
					}
				}
			}
		}
	}
	if changed && len(c.goTypes) > 0 {
		for k := range c.goTypes {
			if n, ok := k.typ.(*gotypes.Named); ok && n.Obj() != nil {
				if _, hit := c.typeCallConvs[goTypeNameKey(n.Obj())]; hit {
					delete(c.goTypes, k)
					delete(c.goNamed, n)
				}
			}
		}
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
