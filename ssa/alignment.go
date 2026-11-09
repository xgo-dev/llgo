package ssa

import (
	"fmt"
	"go/token"
	"go/types"
)

// AlignmentError reports a source-level alignment restriction discovered while
// lowering a native signature or call. Frontends may add their file-set position.
type AlignmentError struct {
	Pos  token.Pos
	Type types.Type
}

func (e *AlignmentError) Error() string {
	return fmt.Sprintf("explicitly aligned values at native C call boundaries are not supported (%s); use a pointer", e.Type)
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
