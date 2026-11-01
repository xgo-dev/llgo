// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package types

import (
	"fmt"
	"go/token"
)

// Validate rejects managed references whose packed storage cannot be scanned
// by LLGo's word-aligned conservative collectors. Native C storage is explicitly
// outside this ownership contract; callers must keep Go owners alive separately.
func (s *LLGoSizes) Validate(t Type) error {
	seen := make(map[Type]bool)
	var visit func(Type) error
	visit = func(t Type) error {
		if t == nil || seen[t] {
			return nil
		}
		seen[t] = true
		if n, ok := t.(*Named); ok && n.TypeParams().Len() > 0 && (n.TypeArgs().Len() == 0 || llgoVariableLayout(t, make(map[Type]bool))) {
			return nil
		}
		switch u := t.Underlying().(type) {
		case *Struct:
			attrs := LLGoTypeAttrs{}
			if s.Layout != nil {
				attrs = s.Layout.TypeAttrs(t)
			}
			if attrs == (LLGoTypeAttrs{}) && u.NumFields() > 0 {
				attrs, _, _ = llgoTagLayout(u.Tag(0))
			}
			if _, named := t.(*Named); named && attrs.Native && u.NumFields() > 0 {
				underlying, _, _ := llgoTagLayout(u.Tag(0))
				if !underlying.Native {
					return fmt.Errorf("native type %s reuses a Go struct's field identity; declare the native struct explicitly or derive from a native type", t)
				}
			}
			if !attrs.Native && attrs.Pack > 0 && attrs.Pack < s.PointerSize && s.managedReferences(t, make(map[Type]bool)) {
				return fmt.Errorf("packed Go type %s contains managed references below pointer alignment; use pack %d or native C storage with separately retained Go owners", t, s.PointerSize)
			}
			for i := 0; i < u.NumFields(); i++ {
				if err := visit(u.Field(i).Type()); err != nil {
					return err
				}
			}
		case *Array:
			return visit(u.Elem())
		case *Slice:
			return visit(u.Elem())
		case *Pointer:
			return visit(u.Elem())
		case *Map:
			if err := visit(u.Key()); err != nil {
				return err
			}
			return visit(u.Elem())
		case *Chan:
			return visit(u.Elem())
		case *Signature:
			for _, list := range []*Tuple{u.Params(), u.Results()} {
				for i := 0; i < list.Len(); i++ {
					if err := visit(list.At(i).Type()); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	return visit(t)
}

func (s *LLGoSizes) managedReferences(t Type, seen map[Type]bool) bool {
	if seen[t] {
		return false
	}
	seen[t] = true
	if s.Layout != nil && s.Layout.TypeAttrs(t).Native {
		return false
	}
	switch u := t.Underlying().(type) {
	case *Pointer, *Map, *Chan, *Slice, *Interface, *Signature:
		return true
	case *Basic:
		return u.Kind() == String || u.Kind() == UnsafePointer
	case *Array:
		return u.Len() != 0 && s.managedReferences(u.Elem(), seen)
	case *Struct:
		for i := 0; i < u.NumFields(); i++ {
			if s.managedReferences(u.Field(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

func (s *LLGoSizes) ValidatePackage(pkg *Package) error {
	for _, name := range pkg.Scope().Names() {
		if err := s.Validate(pkg.Scope().Lookup(name).Type()); err != nil {
			return err
		}
	}
	return nil
}

// ValidateInstances also checks concrete types used only inside function bodies.
func (s *LLGoSizes) ValidateInstances(fset *token.FileSet, info *Info) error {
	if info == nil {
		return nil
	}
	for id, instance := range info.Instances {
		if err := s.Validate(instance.Type); err != nil {
			return fmt.Errorf("%s: %w", fset.Position(id.Pos()), err)
		}
	}
	return nil
}
