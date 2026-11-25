// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by an Apache 2.0 license.

package midway

import (
	"go/ast"
	"reflect"
)

// cloneSyntax copies syntax, retaining source positions and a correspondence
// with the initially checked tree. Parser Object/Scope links are deliberately
// discarded: the new checker provides all binding information. Copying only
// the AST's public syntax fields also avoids cycles through those old links.
func cloneSyntax(node ast.Node) (ast.Node, map[ast.Node]ast.Node) {
	original := make(map[ast.Node]ast.Node)
	var copyValue func(reflect.Value) reflect.Value
	copyValue = func(v reflect.Value) reflect.Value {
		switch v.Kind() {
		case reflect.Interface:
			if v.IsNil() {
				return reflect.Zero(v.Type())
			}
			out := reflect.New(v.Type()).Elem()
			out.Set(copyValue(v.Elem()))
			return out
		case reflect.Pointer:
			if v.IsNil() || v.Type() == reflect.TypeFor[*ast.Object]() || v.Type() == reflect.TypeFor[*ast.Scope]() {
				return reflect.Zero(v.Type())
			}
			out := reflect.New(v.Type().Elem())
			out.Elem().Set(copyValue(v.Elem()))
			if old, ok := v.Interface().(ast.Node); ok {
				original[out.Interface().(ast.Node)] = old
			}
			return out
		case reflect.Struct:
			out := reflect.New(v.Type()).Elem()
			for i := 0; i < v.NumField(); i++ {
				out.Field(i).Set(copyValue(v.Field(i)))
			}
			return out
		case reflect.Slice:
			if v.IsNil() {
				return reflect.Zero(v.Type())
			}
			out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
			for i := 0; i < v.Len(); i++ {
				out.Index(i).Set(copyValue(v.Index(i)))
			}
			return out
		}
		return v
	}
	return copyValue(reflect.ValueOf(node)).Interface().(ast.Node), original
}
