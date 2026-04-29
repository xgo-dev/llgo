/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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
	"go/token"
	"go/types"
	"reflect"
	"sync"
	"unsafe"
)

// -----------------------------------------------------------------------------

type goTypes struct {
	typs  map[unsafe.Pointer]unsafe.Pointer
	typbg sync.Map
}

func newGoTypes() goTypes {
	typs := make(map[unsafe.Pointer]unsafe.Pointer)
	return goTypes{typs: typs}
}

type Background int

const (
	inUnknown Background = iota
	InGo
	InC
	InPython
)

// Type convert a Go/C type into raw type.
// C type = raw type
// Go type: convert to raw type (because of closure)
func (p Program) Type(typ types.Type, bg Background) Type {
	if bg == InGo {
		typ, _ = p.gocvt.cvtType(typ)
	}
	return p.rawType(typ)
}

// FuncDecl converts a Go/C function declaration into raw type.
func (p Program) FuncDecl(sig *types.Signature, bg Background) Type {
	recv := sig.Recv()
	if bg == InGo {
		sig = p.gocvt.cvtFunc(sig, recv)
	} else if recv != nil { // even in C, we need to add ctx for method
		sig = FuncAddCtx(recv, sig)
	}
	return &aType{p.toLLVMFunc(sig), rawType{sig}, vkFuncDecl}
}

// Closure creates a closture type for a function.
func (p Program) Closure(sig *types.Signature) Type {
	closure := p.gocvt.cvtClosure(sig)
	return p.rawType(closure)
}

func (p goTypes) cvtType(typ types.Type) (raw types.Type, cvt bool) {
	if raw, ok := cvtGoSSAOpaqueType(typ); ok {
		return raw, true
	}
	switch t := typ.(type) {
	case *types.Basic:
	case *types.Pointer:
		if elem, cvt := p.cvtType(t.Elem()); cvt {
			return types.NewPointer(elem), true
		}
	case *types.Interface:
		return p.cvtInterface(t)
	case *types.Slice:
		if elem, cvt := p.cvtType(t.Elem()); cvt {
			return types.NewSlice(elem), true
		}
	case *types.Map:
		key, cvt1 := p.cvtType(t.Key())
		elem, cvt2 := p.cvtType(t.Elem())
		if cvt1 || cvt2 {
			return types.NewMap(key, elem), true
		}
	case *types.Struct:
		if IsClosure(t) {
			return typ, false
		}
		return p.cvtStruct(t)
	case *types.Named:
		if v, ok := p.typbg.Load(namedLinkname(t)); ok && v.(Background) == InC {
			break
		}
		return p.cvtNamed(t)
	case *types.Signature:
		return p.cvtClosure(t), true
	case *types.Array:
		if elem, cvt := p.cvtType(t.Elem()); cvt {
			return types.NewArray(elem, t.Len()), true
		}
	case *types.Chan:
		if elem, cvt := p.cvtType(t.Elem()); cvt {
			return types.NewChan(t.Dir(), elem), true
		}
	case *types.Tuple:
		return p.cvtTuple(t)
	case *types.TypeParam:
		return typ.Underlying(), false
	case *types.Alias:
		return p.cvtType(types.Unalias(t))
	default:
		panic(fmt.Sprintf("cvtType: unexpected type - %T", typ))
	}
	return typ, false
}

func cvtGoSSAOpaqueType(typ types.Type) (types.Type, bool) {
	if ptr, ok := typ.(*types.Pointer); ok && isGoSSAOpaqueType(ptr.Elem()) {
		return types.Typ[types.UnsafePointer], true
	}
	if isGoSSAOpaqueType(typ) {
		return types.Typ[types.UnsafePointer], true
	}
	return nil, false
}

func isGoSSAOpaqueType(typ types.Type) bool {
	// opaqueType is unexported in x/tools/go/ssa with no public detection API.
	// We fall back to reflection here; TestGoSSAOpaqueTypeConversion guards
	// against upstream renames or representation changes.
	rt := reflect.TypeOf(typ)
	if rt == nil {
		return false
	}
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	return rt.PkgPath() == "golang.org/x/tools/go/ssa" && rt.Name() == "opaqueType"
}

func namedLinkname(t *types.Named) string {
	obj := t.Obj()
	if obj.Pkg() != nil {
		return obj.Pkg().Path() + "." + obj.Name()
	}
	return obj.Name()
}

func (p goTypes) cvtNamed(t *types.Named) (raw *types.Named, cvt bool) {
	if v, ok := p.typs[unsafe.Pointer(t)]; ok {
		raw = (*types.Named)(v)
		cvt = t != raw
		return
	}
	n := t.NumMethods()
	methods := make([]*types.Func, n)
	for i := 0; i < n; i++ {
		m := t.Method(i) // don't need to convert method signature
		methods[i] = m
	}
	named := types.NewNamed(t.Obj(), types.Typ[types.Int], methods)
	if tp := t.TypeParams(); tp != nil {
		list := make([]*types.TypeParam, tp.Len())
		for i := 0; i < tp.Len(); i++ {
			param := tp.At(i)
			list[i] = types.NewTypeParam(param.Obj(), param.Constraint())
		}
		named.SetTypeParams(list)
	}
	p.typs[unsafe.Pointer(t)] = unsafe.Pointer(t)
	if tund, cvt := p.cvtType(t.Underlying()); cvt {
		named.SetUnderlying(tund)
		if typ, ok := Instantiate(named, t); ok {
			named = typ.(*types.Named)
		}
		p.typs[unsafe.Pointer(t)] = unsafe.Pointer(named)
		return named, true
	}
	return t, false
}

func Instantiate(orig types.Type, t *types.Named) (types.Type, bool) {
	if tp := t.TypeArgs(); tp != nil {
		targs := make([]types.Type, tp.Len())
		for i := 0; i < tp.Len(); i++ {
			targs[i] = tp.At(i)
		}
		// Use validate=false because the newly created type parameters may not
		// match the original constraint references. The original type was already
		// validated by the Go type checker, so we can skip validation here.
		if typ, err := types.Instantiate(nil, orig, targs, false); err == nil {
			return typ, true
		}
	}
	return orig, false
}

func (p goTypes) cvtClosure(sig *types.Signature) *types.Struct {
	if sig.Recv() != nil {
		sig = types.NewSignatureType(nil, nil, nil, sig.Params(), sig.Results(), sig.Variadic())
	}
	raw := p.cvtFunc(sig, nil)
	flds := []*types.Var{
		types.NewField(token.NoPos, nil, "$f", raw, false),
		types.NewField(token.NoPos, nil, "$data", types.Typ[types.UnsafePointer], false),
	}
	return types.NewStruct(flds, nil)
}

func (p goTypes) cvtFunc(sig *types.Signature, recv *types.Var) (raw *types.Signature) {
	if recv != nil {
		sig = FuncAddCtx(recv, sig)
	}
	params, cvt1 := p.cvtTuple(sig.Params())
	results, cvt2 := p.cvtTuple(sig.Results())
	if cvt1 || cvt2 {
		return types.NewSignatureType(nil, nil, nil, params, results, sig.Variadic())
	}
	return sig
}

func (p goTypes) cvtTuple(t *types.Tuple) (*types.Tuple, bool) {
	n := t.Len()
	var vars []*types.Var
	for i := 0; i < n; i++ {
		v := t.At(i)
		if raw, cvt := p.cvtType(v.Type()); cvt {
			if vars == nil {
				vars = make([]*types.Var, n)
				for j := 0; j < i; j++ {
					vars[j] = t.At(j)
				}
			}
			v = types.NewParam(v.Pos(), v.Pkg(), v.Name(), raw)
		}
		if vars != nil {
			vars[i] = v
		}
	}
	if vars != nil {
		return types.NewTuple(vars...), true
	}
	return t, false
}

func explicitMethods(typ *types.Interface) []*types.Func {
	n := typ.NumExplicitMethods()
	methods := make([]*types.Func, n)
	for i := 0; i < n; i++ {
		methods[i] = typ.ExplicitMethod(i)
	}
	return methods
}

func embeddedTypes(typ *types.Interface) []types.Type {
	n := typ.NumEmbeddeds()
	embeddeds := make([]types.Type, n)
	for i := 0; i < n; i++ {
		embeddeds[i] = typ.EmbeddedType(i)
	}
	return embeddeds
}

func (p goTypes) cvtExplicitMethods(typ *types.Interface) ([]*types.Func, bool) {
	n := typ.NumExplicitMethods()
	var methods []*types.Func
	for i := 0; i < n; i++ {
		m := typ.ExplicitMethod(i)
		sig := m.Type().(*types.Signature)
		if raw := p.cvtFunc(sig, nil); sig != raw {
			if methods == nil {
				methods = make([]*types.Func, n)
				for j := 0; j < i; j++ {
					methods[j] = typ.ExplicitMethod(j)
				}
			}
			m = types.NewFunc(m.Pos(), m.Pkg(), m.Name(), raw)
		}
		if methods != nil {
			methods[i] = m
		}
	}
	return methods, methods != nil
}

func (p goTypes) cvtEmbeddedTypes(typ *types.Interface) ([]types.Type, bool) {
	n := typ.NumEmbeddeds()
	var embeddeds []types.Type
	for i := 0; i < n; i++ {
		t := typ.EmbeddedType(i)
		if raw, cvt := p.cvtType(t); cvt {
			if embeddeds == nil {
				embeddeds = make([]types.Type, n)
				for j := 0; j < i; j++ {
					embeddeds[j] = typ.EmbeddedType(j)
				}
			}
			t = raw
		}
		if embeddeds != nil {
			embeddeds[i] = t
		}
	}
	return embeddeds, embeddeds != nil
}

func (p goTypes) cvtInterface(typ *types.Interface) (raw *types.Interface, cvt bool) {
	if v, ok := p.typs[unsafe.Pointer(typ)]; ok {
		raw = (*types.Interface)(v)
		cvt = typ != raw
		return
	}
	defer func() {
		p.typs[unsafe.Pointer(typ)] = unsafe.Pointer(raw)
	}()
	methods, cvt1 := p.cvtExplicitMethods(typ)
	embeddeds, cvt2 := p.cvtEmbeddedTypes(typ)
	if cvt1 || cvt2 {
		if !cvt1 {
			methods = explicitMethods(typ)
		}
		if !cvt2 {
			embeddeds = embeddedTypes(typ)
		}
		return types.NewInterfaceType(methods, embeddeds), true
	}
	return typ, false
}

func (p goTypes) cvtStruct(typ *types.Struct) (raw *types.Struct, cvt bool) {
	if v, ok := p.typs[unsafe.Pointer(typ)]; ok {
		raw = (*types.Struct)(v)
		cvt = typ != raw
		return
	}
	defer func() {
		p.typs[unsafe.Pointer(typ)] = unsafe.Pointer(raw)
	}()
	n := typ.NumFields()
	var flds []*types.Var
	for i := 0; i < n; i++ {
		f := typ.Field(i)
		if raw, cvt := p.cvtType(f.Type()); cvt {
			if flds == nil {
				flds = make([]*types.Var, n)
				for j := 0; j < i; j++ {
					flds[j] = typ.Field(j)
				}
			}
			f = types.NewField(f.Pos(), f.Pkg(), f.Name(), raw, f.Anonymous())
		}
		if flds != nil {
			flds[i] = f
		}
	}
	if flds != nil {
		return types.NewStruct(flds, nil), true
	}
	return typ, false
}

// -----------------------------------------------------------------------------

// FuncAddCtx adds a ctx to a function signature.
func FuncAddCtx(ctx *types.Var, sig *types.Signature) *types.Signature {
	tParams := sig.Params()
	nParams := tParams.Len()
	params := make([]*types.Var, nParams+1)
	params[0] = ctx
	for i := 0; i < nParams; i++ {
		params[i+1] = tParams.At(i)
	}
	return types.NewSignatureType(
		nil, nil, nil, types.NewTuple(params...), sig.Results(), sig.Variadic())
}

// -----------------------------------------------------------------------------
