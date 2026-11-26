//go:build !go1.22

package types

import (
	"go/ast"
	gotypes "go/types"
)

func initGoInfoFileVersions(goInfo *gotypes.Info) {}

func copyGoInfoFileVersions(dst map[*ast.File]string, goInfo *gotypes.Info) bool {
	return false
}

func deleteGoInfoFileVersion(goInfo *gotypes.Info, f *ast.File) {}
