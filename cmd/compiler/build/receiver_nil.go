package build

import (
	"github.com/xgo-dev/llgo/types"
	"go/ast"

	"github.com/xgo-dev/llgo/cmd/compiler/cl"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

// collectReceiverNilChecks runs once on the coordinator, before any package
// lowering. Imported generic bodies can be emitted in a caller's module while
// retaining the defining package's source positions. Every backend therefore
// shares this immutable table, including selections from runtime overlays.
func collectReceiverNilChecks(groups ...[]*packages.Package) *cl.ReceiverNilChecks {
	var files []*ast.File
	var infos []*types.Info
	seen := make(map[*types.Info]bool)
	for _, roots := range groups {
		packages.Visit(roots, nil, func(pkg *packages.Package) {
			if pkg.TypesInfo == nil || seen[pkg.TypesInfo] {
				return
			}
			seen[pkg.TypesInfo] = true
			files = append(files, pkg.Syntax...)
			infos = append(infos, pkg.TypesInfo)
		})
	}
	return cl.CollectReceiverNilChecks(files, infos...)
}
