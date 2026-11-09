package ssa

import (
	"fmt"
	"go/token"
	"go/types"
)

// AlignmentError reports a source-level alignment restriction. Frontends check
// these restrictions before lowering and add their file-set position.
type AlignmentError struct {
	Pos  token.Pos
	Type types.Type
}

func (e *AlignmentError) Error() string {
	return fmt.Sprintf("explicitly aligned values at native C call boundaries are not supported (%s); use a pointer", e.Type)
}

// ValidateNativeValueAlignment checks one value crossing a native call boundary.
// Pointers and descriptors do not inherit their referent's storage alignment.
func ValidateNativeValueAlignment(t types.Type) error {
	if requiredExplicitAlignment(t) != 0 {
		return &AlignmentError{Type: t}
	}
	return nil
}

// NativeAlignmentValidator checks signatures and callback types without creating
// LLVM types. The cache belongs to one validation pass after syntax metadata has
// been collected; a type may have different Go and native representations.
func (p Program) NativeAlignmentValidator() func(types.Type, CallConv) error {
	type key struct {
		typ    types.Type
		native bool
	}
	seen := make(map[key]bool)
	var visit func(types.Type, bool) error
	visit = func(t types.Type, native bool) error {
		if t == nil || seen[key{t, native}] {
			return nil
		}
		seen[key{t, native}] = true
		switch t := t.(type) {
		case *types.Named:
			if cc, ok := p.packageTypeCallConv(namedLinkname(t)); ok && isNativeCallConv(cc) {
				native = true
			}
			return visit(t.Underlying(), native)
		case *types.Alias:
			if cc, ok := p.packageTypeCallConv(typeNameLinkname(t.Obj())); ok && isNativeCallConv(cc) {
				native = true
			}
			return visit(t.Rhs(), native)
		case *types.Signature:
			check := func(v *types.Var) error {
				if v == nil {
					return nil
				}
				if native && requiredExplicitAlignment(v.Type()) != 0 {
					return &AlignmentError{Pos: v.Pos(), Type: v.Type()}
				}
				return visit(v.Type(), native)
			}
			if err := check(t.Recv()); err != nil {
				return err
			}
			for _, tuple := range []*types.Tuple{t.Params(), t.Results()} {
				for i := 0; i < tuple.Len(); i++ {
					if err := check(tuple.At(i)); err != nil {
						return err
					}
				}
			}
		case *types.Pointer:
			// Passing a descriptor does not expose the referent's Go callbacks
			// as native signatures. Explicit native type directives still apply.
			return visit(t.Elem(), false)
		case *types.Array:
			return visit(t.Elem(), native)
		case *types.Slice:
			return visit(t.Elem(), false)
		case *types.Chan:
			return visit(t.Elem(), false)
		case *types.Map:
			if err := visit(t.Key(), false); err != nil {
				return err
			}
			return visit(t.Elem(), false)
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				if err := visit(t.Field(i).Type(), native); err != nil {
					return err
				}
			}
		case *types.Tuple:
			for i := 0; i < t.Len(); i++ {
				if err := visit(t.At(i).Type(), native); err != nil {
					return err
				}
			}
		case *types.Interface:
			for i := 0; i < t.NumMethods(); i++ {
				if err := visit(t.Method(i).Type(), false); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return func(t types.Type, cc CallConv) error { return visit(t, isNativeCallConv(cc)) }
}

// explicitAlignment recognizes only the private, empty alignment carriers.
// Public markers and types defined from them retain a carrier as a field.
func explicitAlignment(t types.Type) int64 {
	n, ok := types.Unalias(t).(*types.Named)
	if !ok || n.Obj().Pkg() == nil || n.Obj().Pkg().Path() != "structs" {
		return 0
	}
	st, ok := n.Underlying().(*types.Struct)
	if !ok || st.NumFields() != 0 {
		return 0
	}
	switch n.Obj().Name() {
	case "align8":
		return 8
	case "align16":
		return 16
	case "align32":
		return 32
	case "align64":
		return 64
	}
	return 0
}

// requiredExplicitAlignment follows only by-value storage. A pointer, slice,
// function, or other descriptor does not inherit its referent's alignment.
func requiredExplicitAlignment(t types.Type) int64 {
	if t == nil {
		return 0
	}
	if a := explicitAlignment(t); a != 0 {
		return a
	}
	switch t := t.Underlying().(type) {
	case *types.Array:
		return requiredExplicitAlignment(t.Elem())
	case *types.Struct:
		var alignment int64
		for i := 0; i < t.NumFields(); i++ {
			alignment = max(alignment, requiredExplicitAlignment(t.Field(i).Type()))
		}
		return alignment
	case *types.Tuple:
		var alignment int64
		for i := 0; i < t.Len(); i++ {
			alignment = max(alignment, requiredExplicitAlignment(t.At(i).Type()))
		}
		return alignment
	}
	return 0
}

func (p Program) containsAlignedStorage(t types.Type) bool {
	return requiredExplicitAlignment(t) != 0 || p.containsVectorStorage(t)
}
