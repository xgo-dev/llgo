package genmethod

import "go/types"

func IsGenericMethod(typ types.Type) bool {
	return typ.(*types.Signature).TypeParams().Len() > 0
}
