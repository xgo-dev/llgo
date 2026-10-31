// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package types

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// Native function fields need their own type identity: tagging their shared
// structural Signature would also change unrelated Go closures and generics.
// A canonical generic named type represents each native signature shape. Type
// arguments retain real parameter/result types, including enclosing parameters.
func (m *LLGoLayout) nativeFunctionExpr(pkg *Package, expr ast.Expr, cc string) ast.Expr {
	switch x := expr.(type) {
	case *ast.IndexListExpr:
		x.X = m.nativeFunctionExpr(pkg, x.X, cc)
		for i, arg := range x.Indices {
			x.Indices[i] = m.nativeFunctionExpr(pkg, arg, cc)
		}
	case *ast.IndexExpr:
		x.X = m.nativeFunctionExpr(pkg, x.X, cc)
		x.Index = m.nativeFunctionExpr(pkg, x.Index, cc)
	case *ast.Ident:
		if strings.HasPrefix(x.Name, "__llgo_native_") {
			parts := strings.Split(strings.TrimPrefix(x.Name, "__llgo_native_"), "_")
			if len(parts) == 4 {
				nin, e1 := strconv.Atoi(parts[1])
				nout, e2 := strconv.Atoi(parts[2])
				variadic, e3 := strconv.ParseBool(parts[3])
				convention := "C"
				if parts[0] == "Stdcall" {
					convention = "stdcall"
				}
				if e1 == nil && e2 == nil && e3 == nil && nin >= 0 && nout >= 0 && (!variadic || nin > 0) {
					m.nativeFunctionName(pkg, convention, nin, nout, variadic)
				}
			}
		}
	case *ast.ParenExpr:
		x.X = m.nativeFunctionExpr(pkg, x.X, cc)
	case *ast.ArrayType:
		x.Elt = m.nativeFunctionExpr(pkg, x.Elt, cc)
	case *ast.StarExpr:
		x.X = m.nativeFunctionExpr(pkg, x.X, cc)
	case *ast.MapType:
		x.Key = m.nativeFunctionExpr(pkg, x.Key, cc)
		x.Value = m.nativeFunctionExpr(pkg, x.Value, cc)
	case *ast.ChanType:
		x.Value = m.nativeFunctionExpr(pkg, x.Value, cc)
	case *ast.FuncType:
		var args []ast.Expr
		variadic := false
		for _, list := range []*ast.FieldList{x.Params, x.Results} {
			if list == nil {
				continue
			}
			for _, f := range list.List {
				t := f.Type
				if ellipsis, ok := t.(*ast.Ellipsis); ok {
					t, variadic = ellipsis.Elt, true
				}
				t = m.nativeFunctionExpr(pkg, t, cc)
				for n := max(1, len(f.Names)); n > 0; n-- {
					args = append(args, t)
				}
			}
		}
		nin, nout := x.Params.NumFields(), x.Results.NumFields()
		local := m.nativeFunctionName(pkg, cc, nin, nout, variadic)
		var result ast.Expr = &ast.Ident{NamePos: expr.Pos(), Name: local}
		if len(args) != 0 {
			result = &ast.IndexListExpr{X: result, Lbrack: expr.Pos(), Indices: args, Rbrack: expr.End() - 1}
		}
		return result
	}
	return expr
}

func (m *LLGoLayout) nativeFunctionName(pkg *Package, cc string, nin, nout int, variadic bool) string {
	prefix := "Cdecl"
	if cc == "stdcall" {
		prefix = "Stdcall"
	}
	name := fmt.Sprintf("%s_%d_%d_%t", prefix, nin, nout, variadic)
	if m.nativeFuncs == nil {
		m.nativeFuncs = make(map[string]*Named)
		m.nativePkg = NewPackage("llgo.internal/native", "native")
		m.nativePkg.MarkComplete()
	}
	named := m.nativeFuncs[name]
	if named == nil {
		tparams := make([]*TypeParam, nin+nout)
		params, results := make([]*Var, nin), make([]*Var, nout)
		constraint := NewInterfaceType(nil, nil).Complete()
		for i := range tparams {
			tp := NewTypeParam(NewTypeName(token.NoPos, m.nativePkg, fmt.Sprintf("T%d", i), nil), constraint)
			tparams[i] = tp
			var t Type = tp
			if variadic && i == nin-1 {
				t = NewSlice(t)
			}
			v := NewVar(token.NoPos, m.nativePkg, "", t)
			if i < nin {
				params[i] = v
			} else {
				results[i-nin] = v
			}
		}
		sig := NewSignatureType(nil, nil, nil, NewTuple(params...), NewTuple(results...), variadic)
		obj := NewTypeName(token.NoPos, m.nativePkg, name, nil)
		named = NewNamed(obj, sig, nil)
		named.SetTypeParams(tparams)
		m.nativePkg.Scope().Insert(obj)
		m.nativeFuncs[name] = named
	}
	local := "__llgo_native_" + name
	if pkg.Scope().Lookup(local) == nil {
		pkg.Scope().Insert(NewTypeName(token.NoPos, pkg, local, named))
	}

	return local
}

// A shared ValueSpec type cannot describe both a Go closure and a native slot.
func llgoHasAnonymousCallback(expr ast.Expr) bool {
	found := false
	llgoInspect(expr, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncType:
			found = true
		case *ast.Ident:
			found = found || strings.HasPrefix(node.Name, "__llgo_native_")
		}
		return !found
	})
	return found
}
