package llvmattr

import (
	"maps"
	"slices"
)

var RuntimeModelNames = slices.Sorted(maps.Keys(runtimeFunctionModels))
