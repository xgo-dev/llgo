package cl

import (
	"go/types"

	"github.com/xgo-dev/llgo/internal/llvmfmv"
	"github.com/xgo-dev/llgo/internal/midway"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func simdCPUQuery(fn *ssa.Function) string {
	obj, ok := fn.Object().(*types.Func)
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "simd/archsimd" ||
		!types.Identical(obj.Type(), fn.Signature) {
		return ""
	}
	sig := fn.Signature
	if sig.Recv() == nil || sig.Params().Len() != 0 || sig.Results().Len() != 1 ||
		!types.Identical(sig.Results().At(0).Type(), types.Typ[types.Bool]) {
		return ""
	}
	recv, ok := sig.Recv().Type().(*types.Named)
	if !ok || recv.Obj().Pkg() != obj.Pkg() || recv.Obj().Name() != "X86Features" {
		return ""
	}
	return llvmfmv.X86Query(obj.Name())
}

func (p *context) markSIMDFMV(fn llssa.Function, source *ssa.Function) {
	if fn == nil || p.prog.Target().GOARCH != "amd64" {
		return
	}
	query := simdCPUQuery(source)
	if origin := source.Origin(); origin != nil {
		source = origin
	}
	var width int
	if obj, ok := source.Object().(*types.Func); ok && source.Synthetic == "" {
		width = midway.ImplementationWidth(obj)
	}
	fn.MarkSIMDFMV(width, query)
}
