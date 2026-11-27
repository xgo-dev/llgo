package ssawrap

import (
	"fmt"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// CallWrapper identifies a compiler-owned forwarding body. Its source symbol
// is retained, but the compiler must not apply the binding again to this entry.
const CallWrapper = "llgo instruction wrapper"

// MakeCallWrapper gives an instruction binding an addressable body under its
// source symbol. CallWrapper makes the entry a Go function, while the forwarded
// call still uses the original binding's instruction lowering.
func MakeCallWrapper(prog *ssa.Program, f *ssa.Function) *ssa.Function {
	// Keep declaration metadata, including generic-instance names and source
	// positions, while replacing the source body and its parameter graph.
	fn := new(ssa.Function)
	*fn = *f
	fn.Prog = prog
	fn.Synthetic = CallWrapper
	fn.Params = nil
	fn.FreeVars = nil
	fn.Locals = nil
	fn.Blocks = nil
	fn.Recover = nil
	fn.AnonFuncs = nil

	entry := fn.NewBasicBlock("entry")
	addParam := func(v *types.Var) {
		name := v.Name()
		if name == "" {
			name = fmt.Sprintf("arg%d", len(fn.Params))
		}
		fn.AddParam(name, v, v.Type())
	}
	if recv := f.Signature.Recv(); recv != nil {
		addParam(recv)
	}
	for i := 0; i < f.Signature.Params().Len(); i++ {
		addParam(f.Signature.Params().At(i))
	}

	args := make([]ssa.Value, len(fn.Params))
	for i, param := range fn.Params {
		args[i] = param
	}
	call := &ssa.Call{
		Call: ssa.CallCommon{
			Value: f,
			Args:  args,
		},
	}
	callTyp := types.Type(f.Signature.Results())
	if f.Signature.Results().Len() == 1 {
		callTyp = f.Signature.Results().At(0).Type()
	}
	entry.Emit(call, callTyp, f.Pos())

	ret := &ssa.Return{}
	switch f.Signature.Results().Len() {
	case 0:
	case 1:
		ret.Results = []ssa.Value{call}
	default:
		for i := 0; i < f.Signature.Results().Len(); i++ {
			ext := &ssa.Extract{Tuple: call, Index: i}
			entry.Emit(ext, f.Signature.Results().At(i).Type(), f.Pos())
			ret.Results = append(ret.Results, ext)
		}
	}
	entry.Emit(ret, nil, f.Pos())
	fn.Finish()
	return fn
}
