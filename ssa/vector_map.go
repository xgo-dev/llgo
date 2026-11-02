package ssa

import (
	"fmt"
	"go/types"
)

// ValidateVectorMaps rejects map storage requiring native vector alignment.
// Follow containers to find maps, but check each key/value's by-value layout:
// a pointer to a vector is allowed, while arrays and structs containing one
// are not. Keep the runtime's bucket layout and indirection policy unchanged.
func (p Program) ValidateVectorMaps(t types.Type) error {
	return p.VectorMapValidator()(t)
}

// VectorMapValidator returns a checker with a per-compilation type cache.
// Use after source metadata and type checking are complete.
func (p Program) VectorMapValidator() func(types.Type) error {
	seen := make(map[types.Type]bool)
	var visit func(types.Type) error
	visit = func(t types.Type) error {
		if t == nil || seen[t] {
			return nil
		}
		seen[t] = true
		var children []types.Type
		switch t := types.Unalias(t).(type) {
		case *types.Named:
			children = append(children, t.Underlying())
		case *types.Map:
			if err := p.validateVectorMap(t); err != nil {
				return err
			}
			children = append(children, t.Key(), t.Elem())
		case *types.Array:
			children = append(children, t.Elem())
		case *types.Pointer:
			children = append(children, t.Elem())
		case *types.Slice:
			children = append(children, t.Elem())
		case *types.Chan:
			children = append(children, t.Elem())
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				children = append(children, t.Field(i).Type())
			}
		case *types.Tuple:
			if t != nil {
				for i := 0; i < t.Len(); i++ {
					children = append(children, t.At(i).Type())
				}
			}
		case *types.Interface:
			for i := 0; i < t.NumMethods(); i++ {
				children = append(children, t.Method(i).Type())
			}
		case *types.Signature:
			children = append(children, t.Params(), t.Results())
		}
		for _, child := range children {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	return visit
}

func (p Program) validateVectorMap(t *types.Map) error {
	for _, part := range []struct {
		name string
		typ  types.Type
	}{{"key", t.Key()}, {"value", t.Elem()}} {
		if p.containsVectorStorage(part.typ) {
			return fmt.Errorf("vector storage is not supported in map %s type %s; use an array of scalars or a pointer", part.name, part.typ)
		}
	}
	return nil
}

func (p Program) containsVectorStorage(t types.Type) bool {
	if _, ok := simdTagIndex(t); ok {
		return true
	}
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		return p.containsVectorStorage(t.Underlying())
	case *types.Array:
		return p.containsVectorStorage(t.Elem())
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if p.containsVectorStorage(t.Field(i).Type()) {
				return true
			}
		}
	}
	return false
}
