//go:build go1.23

package types

import gotypes "go/types"

func unaliasType(t gotypes.Type) gotypes.Type {
	return gotypes.Unalias(t)
}

func aliasObjAndRhs(t gotypes.Type) (*gotypes.TypeName, gotypes.Type, bool) {
	if a, ok := t.(*gotypes.Alias); ok {
		return a.Obj(), a.Rhs(), true
	}
	return nil, nil, false
}
