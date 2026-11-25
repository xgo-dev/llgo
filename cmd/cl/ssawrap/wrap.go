package ssawrap

import (
	"fmt"
	"go/token"
	"go/types"
	"unsafe"

	"golang.org/x/tools/go/ssa"
)

// CallWrapper identifies a compiler-owned forwarding body. Its source symbol
// is retained, but the compiler must not apply the binding again to this entry.
const CallWrapper = "llgo instruction wrapper"

type domInfo struct {
	idom      *ssa.BasicBlock   // immediate dominator (parent in domtree)
	children  []*ssa.BasicBlock // nodes immediately dominated by this one
	pre, post int32             // pre- and post-order numbering within domtree
}

type _BasicBlock struct {
	Index        int                // index of this block within Parent().Blocks
	Comment      string             // optional label; no semantic significance
	parent       *ssa.Function      // parent function
	Instrs       []ssa.Instruction  // instructions in order
	Preds, Succs []*ssa.BasicBlock  // predecessors and successors
	succs2       [2]*ssa.BasicBlock // initial space for Succs
	dom          domInfo            // dominator tree info
	gaps         int                // number of nil Instrs (transient)
	rundefers    int                // number of rundefers (transient)
}

type anInstruction struct {
	block *ssa.BasicBlock // the basic block of this instruction
}

type _Return struct {
	anInstruction
	Results []ssa.Value
	pos     token.Pos
}

type register struct {
	anInstruction
	num       int        // "name" of virtual register, e.g. "t0".  Not guaranteed unique.
	typ       types.Type // type of virtual register
	pos       token.Pos  // position of source expression, or NoPos
	referrers []ssa.Instruction
}

type _Call struct {
	register
	Call ssa.CallCommon
}

type _Parameter struct {
	name      string
	object    *types.Var
	typ       types.Type
	parent    *ssa.Function
	referrers []ssa.Instruction
}

type _Extract struct {
	register
	Tuple ssa.Value
	Index int
}

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
	entry := &ssa.BasicBlock{
		Index:   0,
		Comment: "entry",
	}
	(*_BasicBlock)(unsafe.Pointer(entry)).parent = fn
	fn.Blocks = append(fn.Blocks, entry)
	addParam := func(v *types.Var) {
		name := v.Name()
		if name == "" {
			name = fmt.Sprintf("arg%d", len(fn.Params))
		}
		param := &_Parameter{name: name, object: v, typ: v.Type(), parent: fn}
		fn.Params = append(fn.Params, (*ssa.Parameter)(unsafe.Pointer(param)))
	}
	if recv := f.Signature.Recv(); recv != nil {
		addParam(recv)
	}
	for i := 0; i < f.Signature.Params().Len(); i++ {
		addParam(f.Signature.Params().At(i))
	}
	var args []ssa.Value
	for _, param := range fn.Params {
		args = append(args, param)
	}
	call := &ssa.Call{
		Call: ssa.CallCommon{
			Value: f,
			Args:  args,
		},
	}
	(*_Call)(unsafe.Pointer(call)).block = entry
	(*_Call)(unsafe.Pointer(call)).typ = f.Signature.Results()
	(*_Call)(unsafe.Pointer(call)).pos = f.Pos()
	if f.Signature.Results().Len() == 1 {
		(*_Call)(unsafe.Pointer(call)).typ = f.Signature.Results().At(0).Type()
	}
	for _, param := range fn.Params {
		refs := param.Referrers()
		*refs = append(*refs, call)
	}
	entry.Instrs = append(entry.Instrs, call)
	ret := &ssa.Return{}
	switch f.Signature.Results().Len() {
	case 0:
	case 1:
		ret.Results = []ssa.Value{call}
		*call.Referrers() = append(*call.Referrers(), ret)
	default:
		for i := 0; i < f.Signature.Results().Len(); i++ {
			extract := &_Extract{
				register: register{anInstruction: anInstruction{block: entry}, num: i + 1,
					typ: f.Signature.Results().At(i).Type(), pos: f.Pos()},
				Tuple: call, Index: i,
			}
			value := (*ssa.Extract)(unsafe.Pointer(extract))
			extract.referrers = []ssa.Instruction{ret}
			*call.Referrers() = append(*call.Referrers(), value)
			entry.Instrs = append(entry.Instrs, value)
			ret.Results = append(ret.Results, value)
		}
	}
	(*_Return)(unsafe.Pointer(ret)).block = entry
	(*_Return)(unsafe.Pointer(ret)).pos = f.Pos()
	entry.Instrs = append(entry.Instrs, ret)
	return fn
}
