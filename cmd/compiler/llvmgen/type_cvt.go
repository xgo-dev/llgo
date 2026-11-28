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

package llvmgen

import (
	"github.com/xgo-dev/llgo/types"
	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

type CallConv int

const (
	ccUnknown CallConv = iota
	GoDecl
	CDecl
	PyDecl
	Stdcall
)

type nativeViewKey struct {
	typ types.Type
	cc  CallConv
}

// Type lowers a Go or native llgo/types.Type into its LLVM representation
// without cloning or rewriting the underlying semantic type graph.
// Go function signatures use the 2-word closure representation (vkClosure),
// while native C/stdcall signatures use a 1-word function pointer (vkFuncPtr).
func (p Program) Type(typ types.Type, cc CallConv) Type {
	if cc == Stdcall {
		p.validateStdcallType(typ)
	}
	if isNativeCallConv(cc) {
		return p.withNativeStorage(p.nativeType(typ, cc))
	}
	return p.rawType(typ)
}

func (p Program) nativeCallConvOf(t Type) CallConv {
	if t == nil || p.nativeCC == nil {
		return ccUnknown
	}
	return p.nativeCC[t]
}

func (p Program) nativeView(base Type, cc CallConv) Type {
	if base == nil || !isNativeCallConv(cc) {
		return base
	}
	if p.nativeCC[base] == cc {
		return base
	}
	k := nativeViewKey{typ: base.raw.Type, cc: cc}
	if v := p.nativeViews[k]; v != nil {
		return v
	}
	view := *base
	return p.registerNativeView(k, &view, cc)
}

func (p Program) hasNativeFuncDiff(typ types.Type) bool {
	switch u := types.Unalias(typ).(type) {
	case *types.Signature:
		return p.IsClosureType(typ)
	case *types.Pointer:
		return p.hasNativeFuncDiff(u.Elem())
	case *types.Array:
		return p.hasNativeFuncDiff(u.Elem())
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if p.hasNativeFuncDiff(p.patch(u.Field(i).Type())) {
				return true
			}
		}
		return false
	case *types.Named:
		switch uu := u.Underlying().(type) {
		case *types.Signature:
			return p.IsClosureType(typ)
		case *types.Array:
			return p.hasNativeFuncDiff(uu.Elem())
		}
	}
	return false
}

func (p Program) nativeType(typ types.Type, cc CallConv) Type {
	if !p.hasNativeFuncDiff(typ) {
		return p.rawType(typ)
	}
	switch u := types.Unalias(typ).(type) {
	case *types.Signature:
		return p.funcPtrType(typ)
	case *types.Pointer:
		return p.nativeView(p.rawType(typ), cc)
	case *types.Array:
		k := nativeViewKey{typ: typ, cc: cc}
		if v := p.nativeViews[k]; v != nil {
			return v
		}
		elem := p.nativeType(u.Elem(), cc)
		view := &aType{llvm.ArrayType(elem.ll, int(u.Len())), rawType{typ}, vkArray}
		return p.registerNativeView(k, view, cc)
	case *types.Struct:
		k := nativeViewKey{typ: typ, cc: cc}
		if v := p.nativeViews[k]; v != nil {
			return v
		}
		n := u.NumFields()
		fields := make([]llvm.Type, n)
		for i := 0; i < n; i++ {
			fieldRaw := p.patch(u.Field(i).Type())
			ft := p.nativeType(fieldRaw, cc)
			if ft != p.rawType(fieldRaw) {
				fields[i] = ft.ll
			} else {
				fields[i] = p.storageType(ft)
			}
		}
		ll := p.ctx.StructType(fields, false)
		view := &aType{ll, rawType{typ}, vkStruct}
		return p.registerNativeView(k, view, cc)
	case *types.Named:
		if _, ok := u.Underlying().(*types.Signature); ok {
			return p.funcPtrType(typ)
		}
		if arr, ok := u.Underlying().(*types.Array); ok {
			k := nativeViewKey{typ: typ, cc: cc}
			if v := p.nativeViews[k]; v != nil {
				return v
			}
			elem := p.nativeType(arr.Elem(), cc)
			view := &aType{llvm.ArrayType(elem.ll, int(arr.Len())), rawType{typ}, vkArray}
			return p.registerNativeView(k, view, cc)
		}
	}
	return p.rawType(typ)
}

func (p Program) registerNativeView(k nativeViewKey, view Type, cc CallConv) Type {
	if p.nativeViews == nil {
		p.nativeViews = make(map[nativeViewKey]Type)
	}
	p.nativeViews[k] = view
	return p.markNativeCallConv(view, cc)
}

func (p Program) markNativeCallConv(t Type, cc CallConv) Type {
	if t == nil || !isNativeCallConv(cc) {
		return t
	}
	if p.nativeCC == nil {
		p.nativeCC = make(map[Type]CallConv)
	}
	p.nativeCC[t] = cc
	return t
}

// FuncDecl converts a Go/C function declaration into its LLVM function type.
func (p Program) FuncDecl(sig *types.Signature, cc CallConv) Type {
	decl := p.funcDeclType(sig, cc)
	if isNativeCallConv(cc) {
		return p.withNativeStorage(p.markNativeCallConv(&decl, cc))
	}
	return &decl
}

// Prototype-only queries do not need a registered native storage view.
func (p Program) funcDeclType(sig *types.Signature, cc CallConv) aType {
	if cc == Stdcall {
		p.validateStdcallSignature(sig)
	}
	if recv := sig.Recv(); recv != nil {
		sig = FuncAddCtx(recv, sig)
	}
	return aType{p.toLLVMFuncBackground(sig, cc), rawType{sig}, vkFuncDecl}
}

// Closure returns the two-word Go closure LLVM Type for sig while keeping sig
// itself as RawType().
func (p Program) Closure(sig *types.Signature) Type {
	if sig.Recv() != nil {
		sig = types.NewSignatureType(nil, nil, nil, sig.Params(), sig.Results(), sig.Variadic())
	}
	if sig.CallConv() != types.Go {
		sig = sig.WithCallConv(types.Go)
	}
	return p.rawType(sig)
}

// IsClosureType reports whether typ is represented as a two-word Go closure
// value (vkClosure) rather than a one-word native function pointer.
func (p Program) IsClosureType(typ types.Type) bool {
	if typ == nil || p.isNativeFuncType(typ) {
		return false
	}
	sig, ok := types.Unalias(typ).Underlying().(*types.Signature)
	return ok && sig.CallConv().IsClosure()
}

func (p Program) isNativeFuncType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	switch t := typ.(type) {
	case *types.Alias:
		if _, ok := types.Unalias(t).Underlying().(*types.Signature); !ok {
			return false
		}
		if t.CallConv().IsNative() {
			return true
		}
		if p.nativeTypeDirective(t) != ccUnknown {
			return true
		}
		return p.isNativeFuncType(t.Rhs())
	case *types.Named:
		if _, ok := t.Underlying().(*types.Signature); !ok {
			return false
		}
		if t.CallConv().IsNative() {
			return true
		}
		return p.hasLegacyNativeTypeLayout(t) || p.nativeTypeDirective(t) != ccUnknown
	case *types.Signature:
		return t.CallConv().IsNative()
	}
	return false
}

func namedLinkname(t *types.Named) string {
	return typeNameLinkname(t.Obj())
}

func typeNameLinkname(obj *types.TypeName) string {
	if obj.Pkg() != nil {
		return obj.Pkg().Path() + "." + obj.Name()
	}
	return obj.Name()
}

// FuncAddCtx prepends ctx (a receiver or closure environment parameter) to sig.
func FuncAddCtx(ctx *types.Var, sig *types.Signature) *types.Signature {
	params := sig.Params()
	n := params.Len()
	vars := make([]*types.Var, n+1)
	vars[0] = ctx
	for i := 0; i < n; i++ {
		vars[i+1] = params.At(i)
	}
	res := types.NewSignatureType(nil, nil, nil, types.NewTuple(vars...), sig.Results(), sig.Variadic())
	if cc := sig.CallConv(); cc != types.Go {
		res = res.WithCallConv(cc)
	}
	return res
}
