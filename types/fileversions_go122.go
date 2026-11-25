//go:build go1.22

package types

import (
	"go/ast"
	gotypes "go/types"
)

func initGoInfoFileVersions(goInfo *gotypes.Info) {
	goInfo.FileVersions = make(map[*ast.File]string)
}

func copyGoInfoFileVersions(dst map[*ast.File]string, goInfo *gotypes.Info) bool {
	if goInfo == nil || len(goInfo.FileVersions) == 0 {
		return false
	}
	for f, v := range goInfo.FileVersions {
		dst[f] = v
	}
	return true
}
