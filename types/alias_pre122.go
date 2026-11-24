//go:build !go1.22

package types

import gotypes "go/types"

func unaliasType(t gotypes.Type) gotypes.Type {
	return t
}

func aliasObjAndRhs(t gotypes.Type) (*gotypes.TypeName, gotypes.Type, bool) {
	return nil, nil, false
}
