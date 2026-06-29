package ssawrap

import (
	"go/token"
	"go/types"
	"unsafe"

	"golang.org/x/tools/go/ssa"
)

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

func MakeCallWrapper(prog *ssa.Program, f *ssa.Function) *ssa.Function {
	fn := prog.NewFunction(f.Name()+"$wrapper", f.Signature, "wrapper")
	entry := &ssa.BasicBlock{
		Index:   0,
		Comment: "entry",
	}
	(*_BasicBlock)(unsafe.Pointer(entry)).parent = fn
	fn.Blocks = append(fn.Blocks, entry)
	var args []ssa.Value
	fn.Params = f.Params
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
	entry.Instrs = append(entry.Instrs, call)
	var ret *ssa.Return
	if f.Signature.Results() != nil {
		ret = &ssa.Return{
			Results: []ssa.Value{call},
		}
		call := (*_Call)(unsafe.Pointer(call))
		call.referrers = append(call.referrers, ret)
	} else {
		ret = &ssa.Return{}
	}
	(*_Return)(unsafe.Pointer(ret)).block = entry
	entry.Instrs = append(entry.Instrs, ret)
	return fn
}
