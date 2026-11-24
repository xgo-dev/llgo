package build

import (
	"go/types"

	"github.com/xgo-dev/llgo/internal/build/fmtspecialize"
	"github.com/xgo-dev/llgo/internal/packages"
)

func (ctx *context) prepareValueFormatSource(p *packages.Package) (*packages.Package, error) {
	runtime := ctx.formatRuntime
	if runtime == nil || p.Name != "main" || p.Imports["fmt"] == nil || p.TypesInfo == nil ||
		len(p.Syntax) == 0 || ctx.dedup.IsExported(p.ID) {
		return nil, nil
	}
	names, usesRuntime, err := fmtspecialize.AddValueSource(p, runtime.Types)
	if err != nil || names == nil {
		return nil, err
	}
	if ctx.valueFormatFunctions == nil {
		ctx.valueFormatFunctions = make(map[*types.Package]*fmtspecialize.ValueFunctions)
	}
	ctx.valueFormatFunctions[p.Types] = names
	if usesRuntime {
		return runtime.Package, nil
	}
	return nil, nil
}
