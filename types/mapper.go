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
	"fmt"
	gotypes "go/types"
)

// CallConvLookup resolves an optional directive-declared calling convention
// (//llgo:cdecl, //llgo:stdcall, legacy //llgo:type C) for a named or aliased
// Go type identified by pkgPath+"."+name (or name for universe/pkg-less types).
type CallConvLookup func(typeName string) (CallConv, bool)

type mapKey struct {
	t  gotypes.Type
	cc CallConv
}

// Mapper maps read-only go/types.Type trees to LLGo's physical/ABI Type graph.
// It never mutates or fabricates go/types nodes and handles recursive cycles
// via memoization on (go/types.Type, CallConv).
type Mapper struct {
	lookup CallConvLookup
	cache  map[mapKey]Type
}

// NewMapper creates a Mapper using an optional calling-convention lookup.
func NewMapper(lookup CallConvLookup) *Mapper {
	return &Mapper{
		lookup: lookup,
		cache:  make(map[mapKey]Type),
	}
}

func typeNameKey(obj *gotypes.TypeName) string {
	if obj == nil {
		return ""
	}
	if obj.Pkg() != nil {
		return obj.Pkg().Path() + "." + obj.Name()
	}
	return obj.Name()
}

func (m *Mapper) lookupCallConv(t gotypes.Type) CallConv {
	if m.lookup == nil {
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
			if cc, ok := m.lookup(typeNameKey(obj)); ok && cc.IsNative() {
				return cc
			}
		}
		if !isAlias || next == t {
			return CallConvUnknown
		}
		t = next
	}
}

// Map converts a read-only go/types.Type into its LLGo physical Type under cc.
// If cc is CallConvUnknown, GoDecl is assumed unless overridden by a type
// directive on a named or aliased declaration.
func (m *Mapper) Map(t gotypes.Type, cc CallConv) Type {
	if cc == CallConvUnknown {
		cc = GoDecl
	}
	if override := m.lookupCallConv(t); override.IsNative() {
		cc = override
	}
	key := mapKey{t: t, cc: cc}
	if cached, ok := m.cache[key]; ok {
		return cached
	}

	if _, rhs, ok := aliasObjAndRhs(t); ok {
		res := m.Map(rhs, cc)
		m.cache[key] = res
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
		// Register placeholder before descending so recursive cycles terminate cleanly.
		m.cache[key] = named
		named.underlying = m.Map(u.Underlying(), cc)
		return named
	case *gotypes.Basic:
		if u.Kind() == gotypes.String {
			res := &String{goType: u}
			m.cache[key] = res
			return res
		}
		pk := GoPtr
		if cc.IsNative() && u.Kind() == gotypes.UnsafePointer {
			pk = NativePtr
		}
		res := &Basic{Kind: u.Kind(), PtrKind: pk, goType: u}
		m.cache[key] = res
		return res
	case *gotypes.Pointer:
		pk := GoPtr
		if cc.IsNative() {
			pk = NativePtr
		}
		ptr := &Pointer{Kind: pk, goType: u}
		m.cache[key] = ptr
		ptr.Elem = m.Map(u.Elem(), cc)
		return ptr
	case *gotypes.Signature:
		fn := m.MapSignature(u, cc)
		if cc.IsNative() || cc == PyDecl {
			m.cache[key] = fn
			return fn
		}
		res := &Closure{Sig: fn, goType: u}
		m.cache[key] = res
		return res
	case *gotypes.Slice:
		res := &Slice{Elem: m.Map(u.Elem(), GoDecl), goType: u}
		m.cache[key] = res
		return res
	case *gotypes.Interface:
		res := &Interface{Empty: u.NumMethods() == 0, goType: u}
		m.cache[key] = res
		return res
	case *gotypes.Map:
		res := &Map{Key: m.Map(u.Key(), GoDecl), Elem: m.Map(u.Elem(), GoDecl), goType: u}
		m.cache[key] = res
		return res
	case *gotypes.Chan:
		res := &Chan{Dir: u.Dir(), Elem: m.Map(u.Elem(), GoDecl), goType: u}
		m.cache[key] = res
		return res
	case *gotypes.Array:
		res := &Array{Elem: m.Map(u.Elem(), cc), Len: u.Len(), NativeStorage: cc.IsNative(), goType: u}
		m.cache[key] = res
		return res
	case *gotypes.Struct:
		host := HasHostLayout(u)
		st := &Struct{
			Fields:        make([]*Field, u.NumFields()),
			HostLayout:    host,
			NativeStorage: cc.IsNative(),
			goType:        u,
		}
		m.cache[key] = st
		for i := 0; i < u.NumFields(); i++ {
			f := u.Field(i)
			fieldCC := cc
			// An outer C directive does not recursively convert nested Go structs
			// unless the field itself carries a native directive or is a pointer/func/array.
			if fieldCC.IsNative() {
				if _, isStruct := f.Type().Underlying().(*gotypes.Struct); isStruct && !m.lookupCallConv(f.Type()).IsNative() {
					fieldCC = GoDecl
				}
			}
			st.Fields[i] = &Field{
				Name:     f.Name(),
				Type:     m.Map(f.Type(), fieldCC),
				Tag:      u.Tag(i),
				Embedded: f.Embedded(),
			}
		}
		return st
	case *gotypes.Tuple:
		res := &Tuple{Elems: make([]*Var, u.Len()), goType: u}
		for i := 0; i < u.Len(); i++ {
			v := u.At(i)
			res.Elems[i] = &Var{Name: v.Name(), Type: m.Map(v.Type(), cc)}
		}
		m.cache[key] = res
		return res
	case *gotypes.TypeParam:
		return m.Map(u.Underlying(), cc)
	default:
		panic(fmt.Sprintf("llgo/types: unsupported go/types.Type %T", t))
	}
}

// MapSignature maps a function declaration signature under calling convention cc.
func (m *Mapper) MapSignature(sig *gotypes.Signature, cc CallConv) *Func {
	if cc == CallConvUnknown {
		cc = GoDecl
	}
	var recv *Var
	if r := sig.Recv(); r != nil {
		recv = &Var{Name: r.Name(), Type: m.Map(r.Type(), cc)}
	}
	params := make([]*Var, sig.Params().Len())
	for i := 0; i < sig.Params().Len(); i++ {
		p := sig.Params().At(i)
		params[i] = &Var{Name: p.Name(), Type: m.Map(p.Type(), cc)}
	}
	results := make([]*Var, sig.Results().Len())
	for i := 0; i < sig.Results().Len(); i++ {
		r := sig.Results().At(i)
		results[i] = &Var{Name: r.Name(), Type: m.Map(r.Type(), cc)}
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
