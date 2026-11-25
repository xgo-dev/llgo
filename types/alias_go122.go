//go:build go1.22 && !go1.23

package types

import gotypes "go/types"

func unaliasType(t gotypes.Type) gotypes.Type {
	return gotypes.Unalias(t)
}

func aliasObjAndRhs(t gotypes.Type) (*gotypes.TypeName, gotypes.Type, bool) {
	if a, ok := t.(*gotypes.Alias); ok {
		return a.Obj(), gotypes.Unalias(a), true
	}
	return nil, nil, false
}

func aliasGenericInfo(t gotypes.Type) (origin gotypes.Type, tparams *gotypes.TypeParamList, targs *gotypes.TypeList) {
	return nil, nil, nil
}

func newGoAlias(obj *gotypes.TypeName, rhs gotypes.Type, _ []*gotypes.TypeParam) gotypes.Type {
	return gotypes.NewAlias(obj, rhs)
}
