package cl

import (
	"go/token"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// Heap profiles observe callers of implicit allocation calls. Extend the
// existing frame-preservation policy only when a program can read a profile.
func heapProfileFunctions(analyses []callerTrackingPackageAnalysis, observed bool, imported map[*ssa.Function]bool) map[*ssa.Function]bool {
	for _, analysis := range analyses {
		for fn := range analysis.funcs {
			if fn.Pkg == nil || llssa.IsRuntimeSupportPackage(fn.Pkg.Pkg.Path()) {
				continue
			}
			forEachCall(fn, func(call *ssa.CallCommon) {
				callee := call.StaticCallee()
				if callee != nil && callee.Pkg != nil &&
					(callee.Pkg.Pkg.Path() == "runtime" || callee.Pkg.Pkg.Path() == "github.com/xgo-dev/llgo/runtime/internal/lib/runtime") && callee.Name() == "MemProfile" {
					observed = true
				}
			})
		}
	}
	if !observed {
		return nil
	}
	frames := make(map[*ssa.Function]bool)
	for fn := range imported {
		frames[fn] = true
	}
	callers := make(map[*ssa.Function][]*ssa.Function)
	queue := make([]*ssa.Function, 0, len(imported))
	for fn := range imported {
		queue = append(queue, fn)
	}
	add := func(fn *ssa.Function) {
		if !frames[fn] {
			frames[fn] = true
			queue = append(queue, fn)
		}
	}
	for _, analysis := range analyses {
		for fn := range analysis.trackable {
			if fn.Pkg == nil || llssa.IsRuntimeSupportPackage(fn.Pkg.Pkg.Path()) {
				continue
			}
			for _, block := range fn.Blocks {
				for _, instr := range block.Instrs {
					if heapProfileAllocation(instr) {
						add(fn)
					}
					if call, ok := instr.(ssa.CallInstruction); ok {
						common := call.Common()
						if target := common.StaticCallee(); target != nil {
							callers[target] = append(callers[target], fn)
						} else if _, builtin := common.Value.(*ssa.Builtin); !builtin {
							// Indirect callees may allocate or call back into Go.
							add(fn)
						}
					}
				}
			}
		}
	}
	for i := 0; i < len(queue); i++ {
		for _, caller := range callers[queue[i]] {
			add(caller)
		}
	}
	return frames
}

func heapProfileAllocation(instr ssa.Instruction) bool {
	switch v := instr.(type) {
	case *ssa.Alloc:
		return v.Heap
	case *ssa.MakeSlice, *ssa.MakeMap, *ssa.MakeChan, *ssa.MakeClosure, *ssa.MakeInterface:
		return true
	case *ssa.Convert:
		_, slice := types.Unalias(v.Type()).Underlying().(*types.Slice)
		basic, scalar := types.Unalias(v.Type()).Underlying().(*types.Basic)
		return slice || scalar && basic.Kind() == types.String
	case *ssa.BinOp:
		basic, ok := types.Unalias(v.Type()).Underlying().(*types.Basic)
		return v.Op == token.ADD && ok && basic.Kind() == types.String
	case *ssa.Call:
		builtin, ok := v.Call.Value.(*ssa.Builtin)
		return ok && builtin.Name() == "append"
	}
	return false
}

// EnableHeapProfileFrames selects the source-derived whole-program policy
// before frontend archive contracts are imported.
func (c *CallerTracking) EnableHeapProfileFrames(enable bool) {
	if c.precomputed {
		panic("heap profile policy changed after precomputation")
	}
	c.heapProfiles = enable
}
