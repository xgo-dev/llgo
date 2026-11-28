package genmethod

import "github.com/xgo-dev/llgo/types"

func IsGenericMethod(typ types.Type) bool {
	return typ.(*types.Signature).TypeParams().Len() > 0
}
