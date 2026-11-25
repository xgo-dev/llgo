// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by an Apache 2.0 license.

package midway

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"
)

// typeExpr supplies explicit types for inferred global declarations and their
// initializer result tuples. A negative width denotes the retained original.
// Package-level value types cannot contain free type parameters or constraint
// unions; generic declarations are cloned from their checked syntax instead.
func (f *fileRewriter) typeExpr(t types.Type, width int, pos token.Pos) ast.Expr {
	// Keep the first diagnostic and stop descending after an unnameable type.
	// Rewrite discards this partial result as soon as the declaration finishes.
	if f.err != nil {
		return &ast.BadExpr{From: pos, To: pos}
	}
	child := func(t types.Type) ast.Expr { return f.typeExpr(t, width, pos) }
	name := func(obj *types.TypeName, args *types.TypeList) ast.Expr {
		n := obj.Name()
		path := ""
		if obj.Pkg() != nil {
			path = obj.Pkg().Path()
			if obj.Pkg() != f.a.pkg && !obj.Exported() {
				f.err = fmt.Errorf("%s: cannot name foreign unexported type %s in SIMD initializer", f.fset.Position(pos), obj.Name())
				return &ast.BadExpr{From: pos, To: pos}
			}
		}
		if width >= 0 {
			if baseType(obj) {
				n, _ = bridgeName(n, width)
				path = bridgePath
			} else if f.a.dependent(obj) {
				n += suffix(width)
			}
		}
		var expr ast.Expr = ident(n, pos)
		if obj.Pkg() == nil && obj.Parent() == types.Universe {
			expr = f.predeclaredType(n, pos)
		}
		if path != "" {
			expr = f.qualify(path, n, pos)
		}
		if args.Len() != 0 {
			indices := make([]ast.Expr, args.Len())
			for i := range indices {
				indices[i] = child(args.At(i))
			}
			expr = &ast.IndexListExpr{X: expr, Lbrack: pos, Rbrack: pos, Indices: indices}
		}
		return expr
	}
	switch t := t.(type) {
	case *types.Basic:
		if t.Kind() == types.UnsafePointer {
			return f.qualify("unsafe", "Pointer", pos)
		}
		return f.predeclaredType(t.Name(), pos)
	case *types.Named:
		return name(t.Obj(), t.TypeArgs())
	case *types.Alias:
		return name(t.Obj(), t.TypeArgs())
	case *types.Pointer:
		return &ast.StarExpr{Star: pos, X: child(t.Elem())}
	case *types.Slice:
		return &ast.ArrayType{Lbrack: pos, Elt: child(t.Elem())}
	case *types.Array:
		return &ast.ArrayType{Lbrack: pos, Len: &ast.BasicLit{ValuePos: pos, Kind: token.INT, Value: strconv.FormatInt(t.Len(), 10)}, Elt: child(t.Elem())}
	case *types.Map:
		return &ast.MapType{Map: pos, Key: child(t.Key()), Value: child(t.Elem())}
	case *types.Chan:
		dir := ast.SEND | ast.RECV
		if t.Dir() == types.SendOnly {
			dir = ast.SEND
		} else if t.Dir() == types.RecvOnly {
			dir = ast.RECV
		}
		return &ast.ChanType{Begin: pos, Arrow: pos, Dir: dir, Value: child(t.Elem())}
	case *types.Struct:
		fields := &ast.FieldList{Opening: pos, Closing: pos}
		for i := 0; i < t.NumFields(); i++ {
			if f.err != nil {
				return &ast.BadExpr{From: pos, To: pos}
			}
			v := t.Field(i)
			if !v.Exported() && v.Pkg() != nil && v.Pkg() != f.a.pkg {
				f.err = fmt.Errorf("%s: cannot name foreign unexported field %s in SIMD initializer type", f.fset.Position(pos), v.Name())
				return &ast.BadExpr{From: pos, To: pos}
			}
			field := &ast.Field{Type: child(v.Type())}
			if !v.Embedded() {
				field.Names = []*ast.Ident{ident(v.Name(), pos)}
			}
			if tag := t.Tag(i); tag != "" {
				field.Tag = &ast.BasicLit{ValuePos: pos, Kind: token.STRING, Value: strconv.Quote(tag)}
			}
			fields.List = append(fields.List, field)
		}
		return &ast.StructType{Struct: pos, Fields: fields}
	case *types.Signature:
		params := &ast.FieldList{Opening: pos, Closing: pos}
		results := &ast.FieldList{Opening: pos, Closing: pos}
		for i := 0; i < t.Params().Len(); i++ {
			expr := child(t.Params().At(i).Type())
			if t.Variadic() && i == t.Params().Len()-1 {
				expr = &ast.Ellipsis{Ellipsis: pos, Elt: child(t.Params().At(i).Type().(*types.Slice).Elem())}
			}
			params.List = append(params.List, &ast.Field{Type: expr})
		}
		for i := 0; i < t.Results().Len(); i++ {
			results.List = append(results.List, &ast.Field{Type: child(t.Results().At(i).Type())})
		}
		return &ast.FuncType{Func: pos, Params: params, Results: results}
	case *types.Interface:
		methods := &ast.FieldList{Opening: pos, Closing: pos}
		for i := 0; i < t.NumExplicitMethods(); i++ {
			if f.err != nil {
				return &ast.BadExpr{From: pos, To: pos}
			}
			m := t.ExplicitMethod(i)
			if !m.Exported() && m.Pkg() != nil && m.Pkg() != f.a.pkg {
				f.err = fmt.Errorf("%s: cannot name foreign unexported method %s in SIMD initializer type", f.fset.Position(pos), m.Name())
				return &ast.BadExpr{From: pos, To: pos}
			}
			methods.List = append(methods.List, &ast.Field{Names: []*ast.Ident{ident(m.Name(), pos)}, Type: child(m.Type())})
		}
		for i := 0; i < t.NumEmbeddeds(); i++ {
			methods.List = append(methods.List, &ast.Field{Type: child(t.EmbeddedType(i))})
		}
		return &ast.InterfaceType{Interface: pos, Methods: methods}
	}
	f.err = fmt.Errorf("%s: unsupported SIMD initializer type %v", f.fset.Position(pos), t)
	return &ast.BadExpr{From: pos, To: pos}
}

// A bare predeclared name may bind to a user declaration after the second
// check. The official bridge supplies aliases with stable qualified bindings.
func (f *fileRewriter) predeclaredType(name string, pos token.Pos) ast.Expr {
	scope := f.a.info.Scopes[f.file]
	if scope == nil {
		scope = f.a.pkg.Scope()
	}
	_, obj := scope.LookupParent(name, pos)
	if obj == types.Universe.Lookup(name) {
		return ident(name, pos)
	}
	return f.qualify(bridgePath, "LLGoMidway"+strings.ToUpper(name[:1])+name[1:], pos)
}
