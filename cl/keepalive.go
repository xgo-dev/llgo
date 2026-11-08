package cl

import "golang.org/x/tools/go/ssa"

// KeepAlive observes only liveness. Avoid retaining static type descriptors
// and their method tables when all executable uses are direct KeepAlive calls.
// Dynamic interfaces and boxes shared with other operations keep their types.
func (p *context) keepAliveDataOnly(value *ssa.MakeInterface) (only, stack bool) {
	refs, _ := nonDebugReferrers(value)
	if len(refs) == 0 {
		return false, false
	}
	stack = true
	for _, ref := range refs {
		call, ok := ref.(ssa.CallInstruction)
		if !ok {
			return false, false
		}
		callee := call.Common().StaticCallee()
		if callee == nil {
			return false, false
		}
		if _, name, kind := p.funcName(callee); kind != llgoInstr || name != "keepAlive" {
			return false, false
		}
		if _, immediate := ref.(*ssa.Call); !immediate {
			// Defer/go retain a snapshot beyond this instruction. A reused
			// entry-block stack slot would lose earlier loop iterations,
			// and a goroutine may also outlive the current frame.
			stack = false
		}
	}
	return true, stack
}
