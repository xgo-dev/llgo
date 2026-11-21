package build

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// wasmSynchronousFunctions proves that the Go entry points and everything
// they call directly can run on one stack. Allocation and implicit panics use the
// ordinary runtime. Channel operations, deferred calls, and unknown targets
// fall back to the scheduler.
func wasmSynchronousFunctions(roots []*ssa.Function, closedCall func(*ssa.Function) bool, globalSafe func(*ssa.Global) bool) bool {
	if len(roots) == 0 {
		return false
	}
	seen := make(map[*ssa.Function]bool)
	var visit func(*ssa.Function) bool
	visit = func(fn *ssa.Function) bool {
		if fn == nil || len(fn.Blocks) == 0 || len(fn.FreeVars) != 0 || !closedCall(fn) {
			return false
		}
		if seen[fn] {
			return true
		}
		seen[fn] = true
		var operands [8]*ssa.Value
		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				for _, operand := range inst.Operands(operands[:0]) {
					if operand == nil {
						continue
					}
					if global, ok := (*operand).(*ssa.Global); ok && !globalSafe(global) {
						return false
					}
				}
				switch instr := inst.(type) {
				case *ssa.DebugRef, *ssa.Return, *ssa.Jump, *ssa.If, *ssa.Phi, *ssa.BinOp,
					*ssa.Alloc, *ssa.Store, *ssa.Field, *ssa.FieldAddr, *ssa.Index, *ssa.IndexAddr,
					*ssa.MakeMap, *ssa.MapUpdate, *ssa.Lookup, *ssa.MakeSlice, *ssa.Slice,
					*ssa.Range, *ssa.Next, *ssa.Extract, *ssa.Convert, *ssa.ChangeType,
					*ssa.SliceToArrayPointer, *ssa.MakeInterface, *ssa.ChangeInterface, *ssa.TypeAssert:
				case *ssa.UnOp:
					if instr.Op == token.ARROW {
						return false
					}
				case *ssa.Call:
					if builtin, ok := instr.Call.Value.(*ssa.Builtin); ok {
						switch builtin.Name() {
						case "len", "cap", "append", "copy", "delete", "clear", "min", "max", "real", "imag", "complex":
						case "print", "println":
							for _, arg := range instr.Call.Args {
								if !runtimeFreeScalar(arg.Type()) {
									return false
								}
							}
						default:
							return false
						}
					} else if !visit(instr.Call.StaticCallee()) {
						return false
					}
				default:
					return false
				}
			}
		}
		return true
	}
	for _, root := range roots {
		if !visit(root) {
			return false
		}
	}
	return true
}
