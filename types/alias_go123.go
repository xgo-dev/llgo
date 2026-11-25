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

func aliasGenericInfo(t gotypes.Type) (origin gotypes.Type, tparams *gotypes.TypeParamList, targs *gotypes.TypeList) {
	if a, ok := t.(*gotypes.Alias); ok {
		if orig := a.Origin(); orig != nil && orig != a {
			origin = orig
		}
		return origin, a.TypeParams(), a.TypeArgs()
	}
	return nil, nil, nil
}
