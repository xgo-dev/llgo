package goescape

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

func (f *functionAnalysis) borrowableConversion(v *ssa.Convert) bool {
	if byteStringView(v.Type(), v.X.Type()) {
		return true // Mutation and retention are checked by the flow graph.
	}
	if !byteStringView(v.X.Type(), v.Type()) || v.Referrers() == nil {
		return false
	}
	// Find the entire live region of the immutable snapshot. Examine every
	// path to every consumer, including paths that do not themselves use it.
	last := make(map[*ssa.BasicBlock]int)
	for _, ref := range *v.Referrers() {
		if _, debug := ref.(*ssa.DebugRef); debug {
			continue
		}
		if !f.stringConsumer(ref, v) {
			return false
		}
		for i, instr := range ref.Block().Instrs {
			if instr == ref && i+1 > last[ref.Block()] {
				last[ref.Block()] = i + 1
			}
		}
	}
	if len(last) == 0 {
		return false
	}
	region := make(map[*ssa.BasicBlock]bool)
	var mark func(*ssa.BasicBlock)
	mark = func(block *ssa.BasicBlock) {
		if region[block] || len(region) > 128 {
			return
		}
		region[block] = true
		if block != v.Block() {
			for _, pred := range block.Preds {
				mark(pred)
			}
		}
	}
	for block := range last {
		mark(block)
	}
	if len(region) > 128 {
		return false
	}
	work := 0
	for block := range region {
		if !v.Block().Dominates(block) {
			return false
		}
		end := last[block]
		for _, succ := range block.Succs {
			if region[succ] {
				// A cycle could reuse the snapshot after an unexamined write.
				if succ.Dominates(block) {
					return false
				}
				end = len(block.Instrs)
			}
		}
		start := block != v.Block()
		for i, instr := range block.Instrs[:end] {
			if instr == v {
				start = true
				continue
			}
			if !start {
				continue
			}
			work++
			if work > 512 || !f.readOnlyInstruction(block.Instrs[i], make(map[*ssa.Function]bool), &work) {
				return false
			}
		}
	}
	return true
}

func (f *functionAnalysis) stringConsumer(instr ssa.Instruction, v *ssa.Convert) bool {
	switch use := instr.(type) {
	case *ssa.BinOp:
		return use.Op == token.EQL || use.Op == token.NEQ || use.Op == token.LSS || use.Op == token.LEQ || use.Op == token.GTR || use.Op == token.GEQ
	case *ssa.Lookup:
		return use.Index == v
	case *ssa.Call:
		if builtin, ok := use.Call.Value.(*ssa.Builtin); ok {
			return builtin.Name() == "len"
		}
		for _, target := range callTargets(&use.Call) {
			summary := f.a.summary(target.fn)
			if len(summary.Parameters) != len(target.args) {
				return false
			}
			for i, arg := range target.args {
				if arg != v {
					continue
				}
				p := summary.Parameters[i]
				if p.Heap != 0 || p.Mutator != 0 || p.Callee != 0 {
					return false
				}
				for _, result := range p.Results {
					if result != 0 {
						return false
					}
				}
			}
		}
		return len(callTargets(&use.Call)) != 0
	}
	return false
}

// Checking only parameter mutation is insufficient: a helper could write a
// global alias of the byte buffer. Require a bounded, inspectable pure body.
func (f *functionAnalysis) readOnlyInstruction(instr ssa.Instruction, active map[*ssa.Function]bool, work *int) bool {
	switch v := instr.(type) {
	case *ssa.DebugRef, *ssa.BinOp, *ssa.Phi, *ssa.Extract, *ssa.Field, *ssa.FieldAddr, *ssa.Index, *ssa.IndexAddr, *ssa.Lookup, *ssa.Slice, *ssa.ChangeType, *ssa.ChangeInterface, *ssa.TypeAssert, *ssa.If, *ssa.Jump, *ssa.Return:
		return true
	case *ssa.UnOp:
		return v.Op != token.ARROW
	case *ssa.Call:
		if builtin, ok := v.Call.Value.(*ssa.Builtin); ok {
			return builtin.Name() == "len" || builtin.Name() == "cap" || builtin.Name() == "real" || builtin.Name() == "imag"
		}
		targets := callTargets(&v.Call)
		if len(targets) == 0 {
			return false
		}
		for _, target := range targets {
			fn := target.fn
			if active[fn] || !f.a.inspect(fn) {
				return false
			}
			active[fn] = true
			for _, block := range fn.Blocks {
				for _, nested := range block.Instrs {
					*work++
					if *work > 512 || !f.readOnlyInstruction(nested, active, work) {
						return false
					}
				}
			}
			delete(active, fn)
		}
		return true
	}
	return false
}
