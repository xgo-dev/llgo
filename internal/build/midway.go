package build

import (
	"fmt"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/midway"
	"github.com/xgo-dev/llgo/internal/packages"
)

func rewritePortableSIMD(pkg *packages.Package, index *directive.Index) (bool, error) {
	// Keep the metadata graph in step with the transformed syntax. Internal
	// bridge access is added only after the user's original source has passed
	// normal package loading and type checking.
	byPath := make(map[string]*packages.Package)
	packages.Visit([]*packages.Package{pkg}, func(p *packages.Package) bool {
		byPath[p.PkgPath] = p
		return true
	}, nil)
	changed, imports, err := midway.Rewrite(pkg.Fset, pkg.Types, pkg.TypesInfo, pkg.Syntax, index)
	if err != nil {
		return false, err
	}
	for _, path := range imports {
		dep := byPath[path]
		if dep == nil {
			return false, fmt.Errorf("Midway: missing dependency metadata for %s in %s", path, pkg.PkgPath)
		}
		pkg.Imports[path] = dep
	}
	return changed, nil
}
