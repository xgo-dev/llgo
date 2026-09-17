package cl

import (
	"go/types"

	"github.com/xgo-dev/llgo/internal/exportdata"
)

// PackageExports associates manifest records with loaded Go package identities.
// Prepare it before lowering; backend workers share it read-only. It owns no
// source comments or LLVM values. The zero value is ready for use.
type PackageExports struct {
	packages map[*types.Package]*exportdata.Package
}

// Set binds validated records to one loaded package. The caller must not mutate
// data or update this index while backend workers are running.
func (p *PackageExports) Set(pkg *types.Package, data *exportdata.Package) error {
	if err := data.Validate(); err != nil {
		return err
	}
	if p.packages == nil {
		p.packages = make(map[*types.Package]*exportdata.Package)
	}
	p.packages[pkg] = data
	return nil
}

func (p *PackageExports) lookup(pkg *types.Package, name string) exportdata.Function {
	if p == nil {
		return exportdata.Function{}
	}
	return p.packages[pkg].Function(name)
}
