package cl

import (
	"go/types"

	"github.com/xgo-dev/llgo/internal/llvmfmv"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func simdCPUQuery(fn *ssa.Function) string {
	obj, ok := fn.Object().(*types.Func)
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "simd/archsimd" || obj.Name() != "AVX2" ||
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
	return llvmfmv.AVX2Query
}

func (p *context) markSIMDFMV(fn llssa.Function, source *ssa.Function) {
	if fn == nil || p.prog.Target().GOARCH != "amd64" {
		return
	}
	hasVector := false
	sig := source.Signature
	var seen map[types.Type]bool
	var check func(types.Type)
	check = func(t types.Type) {
		if hasVector {
			return
		}
		t = types.Unalias(t)
		if simdFMVVector(t) {
			hasVector = true
			return
		}
		// Only by-value containment changes the vector call boundary; do not
		// walk through references or track scalar types in the visited set.
		underlying := t.Underlying()
		switch underlying.(type) {
		case *types.Struct, *types.Array:
		default:
			return
		}
		if seen[t] {
			return
		}
		if seen == nil {
			seen = make(map[types.Type]bool)
		}
		seen[t] = true
		switch t := underlying.(type) {
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				check(t.Field(i).Type())
			}
		case *types.Array:
			check(t.Elem())
		}
	}
	if sig.Recv() != nil {
		check(sig.Recv().Type())
	}
	for i := 0; i < sig.Params().Len(); i++ {
		check(sig.Params().At(i).Type())
	}
	for i := 0; i < sig.Results().Len(); i++ {
		check(sig.Results().At(i).Type())
	}
	// Bodyless assembly/linkname declarations do not promise a compiler-emitted
	// entry. Only check for generated intrinsic bodies when a vector needs one.
	available := len(source.Blocks) != 0
	if hasVector && !available {
		_, available = lookupSIMD(source, "amd64")
	}
	// CPU queries are marked independently of vector signatures.
	fn.MarkSIMDFMV(available && hasVector, simdCPUQuery(source))
}

// Midway's hardware representations are defined bridge types, not aliases of
// archsimd vectors. Recognize their checked storage without changing their ABI.
func simdFMVVector(t types.Type) bool {
	if _, ok := llssa.SIMDVectorShape(t); ok {
		return true
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "simd/internal/bridge" {
		return false
	}
	var arch *types.Package
	for _, pkg := range named.Obj().Pkg().Imports() {
		if pkg.Path() == "simd/archsimd" {
			arch = pkg
			break
		}
	}
	if arch == nil {
		return false
	}
	obj, ok := arch.Scope().Lookup(named.Obj().Name()).(*types.TypeName)
	if !ok {
		return false
	}
	_, vector := llssa.SIMDVectorShape(obj.Type())
	return vector && types.Identical(named.Underlying(), obj.Type().Underlying())
}
