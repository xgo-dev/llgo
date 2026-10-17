package importer

import "go/types"

// Default reads the selected Go toolchain's gc export data. LLGo's compiler
// identity does not make its native archives compatible with this importer.
func Default() types.Importer {
	return For("gc", nil)
}
