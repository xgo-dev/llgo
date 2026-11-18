package cl

import "golang.org/x/tools/go/ssa"

// KeepAlive observes only liveness. Avoid retaining static type descriptors
// and their method tables when all executable uses are direct KeepAlive calls.
// Dynamic interfaces and boxes shared with other operations keep their types.
func (p *context) keepAliveDataOnly(value *ssa.MakeInterface) bool {
	refs, _ := nonDebugReferrers(value)
	if len(refs) == 0 {
		return false
	}
	for _, ref := range refs {
		call, ok := ref.(ssa.CallInstruction)
		if !ok {
			return false
		}
		callee := call.Common().StaticCallee()
		if callee == nil {
			return false
		}
		if _, name, kind := p.funcName(callee); kind != llgoInstr || name != "keepAlive" {
			return false
		}
	}
	return true
}
