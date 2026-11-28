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

package cl

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"
)

// RewriteUnsafeOperands replaces nonconstant generic Sizeof, Alignof, and
// Offsetof operands after type checking and before SSA construction. Internal
// calls carry a typed nil pointer and constant metadata instead of the expression.
func RewriteUnsafeOperands(files []*ast.File, info *types.Info) {
	if info == nil {
		return
	}
	for _, file := range files {
		usesUnsafe := false
		for _, spec := range file.Imports {
			if spec.Name != nil && spec.Name.Name == "_" {
				continue
			}
			path, err := strconv.Unquote(spec.Path.Value)
			if err == nil && path == "unsafe" {
				usesUnsafe = true
				break
			}
		}
		if !usesUnsafe {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 || info.Types[call].Value != nil {
				return true
			}
			var id *ast.Ident
			switch fun := ast.Unparen(call.Fun).(type) {
			case *ast.SelectorExpr:
				id = fun.Sel
			case *ast.Ident:
				id = fun
			}
			builtin, ok := info.Uses[id].(*types.Builtin)
			if !ok || builtin.Pkg() != types.Unsafe || (builtin.Name() != "Offsetof" && builtin.Name() != "Sizeof" && builtin.Name() != "Alignof") {
				return true
			}
			operand := call.Args[0]
			if !unsafeOperandHasTypeParam(operand, info) {
				return true
			}
			receiver := info.TypeOf(operand)
			// Sizeof/Alignof carry a type root followed by representation operations.
			indices := []int{0}
			if builtin.Name() == "Offsetof" {
				selector, ok := ast.Unparen(operand).(*ast.SelectorExpr)
				if !ok {
					return true
				}
				selection := info.Selections[selector]
				if selection == nil || selection.Kind() != types.FieldVal {
					return true
				}
				receiver = selection.Recv()
				if ptr, ok := receiver.Underlying().(*types.Pointer); ok {
					receiver = ptr.Elem()
				}
				indices = selection.Index()
			}
			if builtin.Name() != "Offsetof" {
				receiver, indices = unsafeOperandPath(operand, info)
			}
			ptr := types.NewPointer(receiver)
			placeholder := &ast.Ident{NamePos: operand.Pos(), Name: "nil"}
			info.Uses[placeholder] = types.Universe.Lookup("nil")
			info.Types[placeholder] = types.TypeAndValue{Type: ptr}
			args := []ast.Expr{placeholder}
			params := []*types.Var{types.NewParam(token.NoPos, nil, "", ptr)}
			for _, index := range indices {
				arg := &ast.BasicLit{ValuePos: operand.Pos(), Kind: token.INT, Value: strconv.Itoa(index)}
				info.Types[arg] = types.TypeAndValue{Type: types.Typ[types.Uintptr], Value: constant.MakeInt64(int64(index))}
				args = append(args, arg)
				params = append(params, types.NewParam(token.NoPos, nil, "", types.Typ[types.Uintptr]))
			}
			fun := &ast.Ident{NamePos: call.Fun.Pos(), Name: builtin.Name()}
			info.Uses[fun] = builtin
			info.Types[fun] = types.TypeAndValue{Type: types.NewSignatureType(nil, nil, nil,
				types.NewTuple(params...), types.NewTuple(types.NewParam(token.NoPos, nil, "", types.Typ[types.Uintptr])), false)}
			call.Fun = fun
			call.Args = args
			return false
		})
	}
}

const (
	unsafeValue = iota
	unsafeField
	unsafeIndex
	unsafePointer
	unsafeElem
	unsafeGenericIndex
)

// unsafeOperandHasTypeParam reports whether a generic operand contains a type
// parameter in its checked expression tree.
func unsafeOperandHasTypeParam(expr ast.Expr, info *types.Info) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if e, ok := node.(ast.Expr); ok && containsTypeParam(info.TypeOf(e)) {
			found = true
			return false
		}
		return true
	})
	return found
}

func containsTypeParam(typ types.Type) bool {
	return containsTypeParamSeen(typ, make(map[types.Type]bool))
}

func containsTypeParamSeen(typ types.Type, seen map[types.Type]bool) bool {
	if typ == nil || seen[typ] {
		return false
	}
	seen[typ] = true
	switch typ := typ.(type) {
	case *types.TypeParam:
		return true
	case *types.Pointer:
		return containsTypeParamSeen(typ.Elem(), seen)
	case *types.Array:
		return containsTypeParamSeen(typ.Elem(), seen)
	case *types.Slice:
		return containsTypeParamSeen(typ.Elem(), seen)
	case *types.Map:
		return containsTypeParamSeen(typ.Key(), seen) || containsTypeParamSeen(typ.Elem(), seen)
	case *types.Chan:
		return containsTypeParamSeen(typ.Elem(), seen)
	case *types.Alias:
		if args := typ.TypeArgs(); args != nil {
			for i := 0; i < args.Len(); i++ {
				if containsTypeParamSeen(args.At(i), seen) {
					return true
				}
			}
		}
		return containsTypeParamSeen(typ.Rhs(), seen)
	case *types.Named:
		if args := typ.TypeArgs(); args != nil {
			for i := 0; i < args.Len(); i++ {
				if containsTypeParamSeen(args.At(i), seen) {
					return true
				}
			}
		}
		return containsTypeParamSeen(typ.Underlying(), seen)
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if containsTypeParamSeen(typ.Field(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

// unsafeOperandPath describes only representation-preserving operations. Calls,
// conversions, and other expressions start a new root at their result type.
func unsafeOperandPath(expr ast.Expr, info *types.Info) (types.Type, []int) {
	expr = ast.Unparen(expr)
	switch expr := expr.(type) {
	case *ast.SelectorExpr:
		if selection := info.Selections[expr]; selection != nil && selection.Kind() == types.FieldVal {
			root, path := unsafeOperandPath(expr.X, info)
			typ := selection.Recv()
			for _, index := range selection.Index() {
				if ptr, ok := typ.Underlying().(*types.Pointer); ok {
					path = append(path, unsafeElem)
					typ = ptr.Elem()
				}
				path = append(path, unsafeField, index)
				typ = typ.Underlying().(*types.Struct).Field(index).Type()
			}
			return root, append(path, unsafeValue)
		}
	case *ast.IndexExpr:
		if param, ok := types.Unalias(info.TypeOf(expr.X)).(*types.TypeParam); ok && unsafeIndexConstraint(param.Constraint(), make(map[types.Type]bool)) {
			root, path := unsafeOperandPath(expr.X, info)
			return root, append(path, unsafeGenericIndex, unsafeValue)
		}
		switch typ := info.TypeOf(expr.X).Underlying().(type) {
		case *types.Array, *types.Slice:
			root, path := unsafeOperandPath(expr.X, info)
			return root, append(path, unsafeIndex, unsafeValue)
		case *types.Pointer:
			if _, ok := typ.Elem().Underlying().(*types.Array); ok {
				root, path := unsafeOperandPath(expr.X, info)
				return root, append(path, unsafeElem, unsafeIndex, unsafeValue)
			}
		}
	case *ast.StarExpr:
		root, path := unsafeOperandPath(expr.X, info)
		return root, append(path, unsafeElem, unsafeValue)
	case *ast.UnaryExpr:
		if expr.Op == token.AND {
			root, path := unsafeOperandPath(expr.X, info)
			// Addressing a value uses its storage representation before loading.
			if len(path) > 0 && path[len(path)-1] == unsafeValue {
				path = path[:len(path)-1]
			}
			return root, append(path, unsafePointer)
		}
	}
	return info.TypeOf(expr), []int{unsafeValue}
}

// unsafeIndexConstraint reports whether all types are arrays, array pointers, or slices.
func unsafeIndexConstraint(typ types.Type, visiting map[types.Type]bool) bool {
	typ = types.Unalias(typ)
	if visiting[typ] {
		return false
	}
	visiting[typ] = true
	defer delete(visiting, typ)
	switch typ := typ.(type) {
	case *types.Named:
		return unsafeIndexConstraint(typ.Underlying(), visiting)
	case *types.Union:
		if typ.Len() == 0 {
			return false
		}
		for i := 0; i < typ.Len(); i++ {
			if !unsafeIndexConstraint(typ.Term(i).Type(), visiting) {
				return false
			}
		}
		return true
	case *types.Interface:
		for i := 0; i < typ.NumEmbeddeds(); i++ {
			if unsafeIndexConstraint(typ.EmbeddedType(i), visiting) {
				return true
			}
		}
		return false
	default:
		switch t := typ.Underlying().(type) {
		case *types.Array, *types.Slice:
			return true
		case *types.Pointer:
			_, ok := types.Unalias(t.Elem()).Underlying().(*types.Array)
			return ok
		}
		return false
	}
}
