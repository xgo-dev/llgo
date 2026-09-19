package build

import (
	"fmt"
	"slices"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/exportdata"
)

func collectPackageExports(pkg *aPackage) (*exportdata.Package, error) {
	files := slices.Clone(pkg.Syntax)
	if pkg.AltPkg != nil {
		files = append(files, pkg.AltPkg.Syntax...)
	}
	return cl.CollectPackageExports(pkg.Fset, files)
}

// preparePackageExports runs on the coordinator after cache lookup and before
// any backend in the group starts. Declaration-only and deferred runtime
// packages also need records, even when they produce no archive in this group.
func (c *context) preparePackageExports() error {
	if c.frontendOptions.Exports == nil {
		c.frontendOptions.Exports = new(cl.PackageExports)
	}
	for _, pkg := range c.pkgs {
		if pkg.Exports == nil {
			data, err := collectPackageExports(pkg)
			if err != nil {
				return fmt.Errorf("export %s: %w", pkg.PkgPath, err)
			}
			pkg.Exports = data
		}
		if err := c.frontendOptions.Exports.Set(pkg.Types, pkg.Exports); err != nil {
			return err
		}
		if patch, ok := c.patches[pkg.PkgPath]; ok {
			if err := c.frontendOptions.Exports.Set(patch.Types, pkg.Exports); err != nil {
				return err
			}
		}
	}
	return nil
}
