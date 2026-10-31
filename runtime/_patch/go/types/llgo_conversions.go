// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package types

import (
	"fmt"
	"go/ast"
	"go/token"
)

// ValidateConversions complements Go's tag-insensitive pointer conversions.
// A value conversion can copy fields; a pointer conversion cannot do so.
func (s *LLGoSizes) ValidateConversions(fset *token.FileSet, files []*ast.File, info *Info) (err error) {
	if info == nil {
		return nil
	}
	for _, file := range files {
		llgoInspect(file, func(node ast.Node) bool {
			if err != nil {
				return false
			}
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			to := info.Types[call.Fun]
			from := info.Types[call.Args[0]]
			if !to.IsType() || from.Type == nil {
				return true
			}
			x, ok := to.Type.Underlying().(*Pointer)
			if !ok {
				return true
			}
			y, ok := from.Type.Underlying().(*Pointer)
			if !ok {
				return true
			}
			if llgoVariableLayout(x.Elem(), make(map[Type]bool)) || llgoVariableLayout(y.Elem(), make(map[Type]bool)) {
				return true
			}
			if !s.samePointerLayout(x.Elem(), y.Elem(), make(map[[2]Type]bool)) {
				err = fmt.Errorf("%s: cannot convert pointers between different LLGo layouts: %s to %s", fset.Position(call.Pos()), from.Type, to.Type)
			}
			return true
		})
	}
	return
}

func llgoVariableLayout(t Type, seen map[Type]bool) bool {
	if seen[t] {
		return false
	}
	seen[t] = true
	if _, ok := t.(*TypeParam); ok {
		return true
	}
	switch t := t.Underlying().(type) {
	case *Struct:
		for i := 0; i < t.NumFields(); i++ {
			if llgoVariableLayout(t.Field(i).Type(), seen) {
				return true
			}
		}
	case *Array:
		return llgoVariableLayout(t.Elem(), seen)
	case *Pointer:
		return llgoVariableLayout(t.Elem(), seen)
	}
	return false
}

func (s *LLGoSizes) samePointerLayout(x, y Type, seen map[[2]Type]bool) bool {
	key := [2]Type{x, y}
	if seen[key] {
		return true
	}
	seen[key] = true
	if s.Sizeof(x) != s.Sizeof(y) || s.Alignof(x) != s.Alignof(y) {
		return false
	}
	switch x := x.Underlying().(type) {
	case *Struct:
		y, ok := y.Underlying().(*Struct)
		if !ok || x.NumFields() != y.NumFields() {
			return false
		}
		_, xo := s.fieldsLayout(llgoFields{st: x}, s.structNative(x), s.structPack(x))
		_, yo := s.fieldsLayout(llgoFields{st: y}, s.structNative(y), s.structPack(y))
		for i := 0; i < x.NumFields(); i++ {
			if xo[i] != yo[i] || !s.samePointerLayout(x.Field(i).Type(), y.Field(i).Type(), seen) {
				return false
			}
		}
	case *Array:
		y, ok := y.Underlying().(*Array)
		return ok && x.Len() == y.Len() && s.samePointerLayout(x.Elem(), y.Elem(), seen)
	case *Pointer:
		if y, ok := y.Underlying().(*Pointer); ok {
			return s.samePointerLayout(x.Elem(), y.Elem(), seen)
		}
	}
	return true
}

func (s *LLGoSizes) structNative(t *Struct) bool {
	if t.NumFields() == 0 {
		return false
	}
	a, _, _ := llgoTagLayout(t.Tag(0))
	return a.Native
}
func (s *LLGoSizes) structPack(t *Struct) int64 {
	if t.NumFields() == 0 {
		return 0
	}
	a, _, _ := llgoTagLayout(t.Tag(0))
	return a.Pack
}
