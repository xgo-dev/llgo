// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file (LICENSES/Go-BSD-3-Clause.txt).

package ssa

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/xgo-dev/llgo/types"
)

// Adapted from golang.org/x/tools/internal/typesinternal for pure Go 1.20+
// compatibility operating natively on github.com/xgo-dev/llgo/types.

func tiForEachElement(methodSetOf func(types.Type) *types.MethodSet, T types.Type, f func(T types.Type, access bool) bool) {
	var visit func(T types.Type, access bool)
	visit = func(T types.Type, access bool) {
		if f(T, access) {
			return
		}
		tmset := methodSetOf(T)
		for i := 0; i < tmset.Len(); i++ {
			method := tmset.At(i)
			sig := method.Type().(*types.Signature)
			if sig.TypeParams() != nil {
				continue
			}
			visit(sig.Params(), false)
			visit(sig.Results(), false)
		}
		switch T := T.(type) {
		case *types.Alias:
			visit(types.Unalias(T), access)
		case *types.Basic, *types.SIMDVec:
			// nop
		case *types.Interface:
			// nop---handled by recursion over method set.
		case *types.Pointer:
			visit(T.Elem(), true)
		case *types.Slice:
			visit(T.Elem(), true)
		case *types.Chan:
			visit(T.Elem(), true)
		case *types.Map:
			visit(T.Key(), true)
			visit(T.Elem(), true)
		case *types.Signature:
			if T.Recv() != nil {
				panic(fmt.Sprintf("Signature %s has Recv %s", T, T.Recv()))
			}
			visit(T.Params(), false)
			visit(T.Results(), false)
		case *types.Named:
			visit(types.NewPointer(T), true)
			visit(T.Underlying(), false)
		case *types.Array:
			visit(T.Elem(), true)
		case *types.Struct:
			for i, n := 0, T.NumFields(); i < n; i++ {
				visit(T.Field(i).Type(), true)
			}
		case *types.Tuple:
			for i, n := 0, T.Len(); i < n; i++ {
				visit(T.At(i).Type(), true)
			}
		case *types.TypeParam, *types.Union:
			panic(fmt.Sprintf("ForEachElement called on type containing %T", T))
		default:
			panic(fmt.Sprintf("ForEachElement called on unexpected type %T", T))
		}
	}
	visit(T, true)
}

func tiRecvBase(fn *types.Func) (isPtr bool, named *types.Named) {
	if recv := fn.Signature().Recv(); recv != nil {
		t := recv.Type()
		if ptr, ok := types.Unalias(t).(*types.Pointer); ok {
			isPtr = true
			t = ptr.Elem()
		}
		named, _ = types.Unalias(t).(*types.Named)
	}
	return
}

func tiTupleOf(elems ...types.Type) *types.Tuple {
	vars := make([]*types.Var, len(elems))
	for i, elem := range elems {
		vars[i] = types.NewVar(token.NoPos, nil, "", elem)
	}
	return types.NewTuple(vars...)
}

func tiZeroString(t types.Type, qual types.Qualifier) (_ string, isValid bool) {
	switch t := t.(type) {
	case *types.Basic:
		switch {
		case t.Info()&types.IsBoolean != 0:
			return "false", true
		case t.Info()&types.IsNumeric != 0:
			return "0", true
		case t.Info()&types.IsString != 0:
			return `""`, true
		case t.Kind() == types.UnsafePointer, t.Kind() == types.UntypedNil:
			return "nil", true
		case t.Kind() == types.Invalid:
			return "invalid", false
		default:
			panic(fmt.Sprintf("ZeroString for unexpected type %v", t))
		}
	case *types.Pointer, *types.Slice, *types.Chan, *types.Map, *types.Signature:
		return "nil", true
	case *types.Interface:
		if !t.IsMethodSet() {
			return "invalid", false
		}
		return "nil", true
	case *types.Named:
		switch under := t.Underlying().(type) {
		case *types.Struct, *types.Array:
			return types.TypeString(t, qual) + "{}", true
		default:
			return tiZeroString(under, qual)
		}
	case *types.Alias:
		switch t.Underlying().(type) {
		case *types.Struct, *types.Array:
			return types.TypeString(t, qual) + "{}", true
		default:
			return tiZeroString(types.Unalias(t), qual)
		}
	case *types.Array, *types.Struct, *types.SIMDVec:
		return types.TypeString(t, qual) + "{}", true
	case *types.TypeParam:
		return "*new(" + types.TypeString(t, qual) + ")", true
	case *types.Tuple:
		isValid := true
		components := make([]string, t.Len())
		for i := 0; i < t.Len(); i++ {
			comp, ok := tiZeroString(t.At(i).Type(), qual)
			components[i] = comp
			isValid = isValid && ok
		}
		return "(" + strings.Join(components, ", ") + ")", isValid
	case *types.Union:
		panic(fmt.Sprintf("invalid type for a variable: %v", t))
	default:
		panic(t)
	}
}
