package cl

import (
	"github.com/xgo-dev/llgo/types"
	"go/ast"
	"go/token"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/ssaopt"
	"github.com/xgo-dev/llgo/ssa"
)

// ReceiverNilChecks preserves implicit address-taking in pointer method
// selections. SSA folds (*p).M into p.M and represents promoted value fields
// with FieldAddr, neither of which retains the required nil dereference.
// The maps are immutable after collection and shared by package lowering.
type ReceiverNilChecks struct {
	// AST CallExpr.Lparen matches ssa.CallCommon.Pos; SelectorExpr.Sel.Pos
	// matches the position of SSA MakeClosure for a bound method value.
	// NoPos belongs to synthesized SSA, never to either source lookup table.
	calls  map[token.Pos]receiverNilCheck
	values map[token.Pos]receiverNilCheck
}

type receiverNilCheck struct {
	pos     token.Pos
	address bool
}

// CollectReceiverNilChecks uses checked selections, not selector spelling:
// (*T).M is a method expression, whereas (*p).M selects a method value.
// infos may include both an original package and its runtime source overlay.
func CollectReceiverNilChecks(files []*ast.File, infos ...*types.Info) *ReceiverNilChecks {
	selections := make(map[*ast.SelectorExpr]receiverNilCheck)
	for _, info := range infos {
		if info != nil {
			for expr, sel := range info.Selections {
				if isPointerMethodSelection(sel) && expr.Pos().IsValid() && expr.Sel.Pos().IsValid() {
					selections[expr] = receiverNilCheck{expr.Pos(), receiverNeedsAddressCheck(sel)}
				}
			}
		}
	}
	if len(selections) == 0 {
		return nil
	}
	checks := &ReceiverNilChecks{calls: make(map[token.Pos]receiverNilCheck), values: make(map[token.Pos]receiverNilCheck)}
	for expr, check := range selections {
		checks.values[expr.Sel.Pos()] = check
	}
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
			if check, found := selections[selector]; ok && found && call.Lparen.IsValid() {
				checks.calls[call.Lparen] = check
			}
			return true
		})
	}
	return checks
}

func isPointerMethodSelection(sel *types.Selection) bool {
	if sel == nil || sel.Kind() != types.MethodVal {
		return false
	}
	sig, ok := sel.Obj().Type().(*types.Signature)
	return ok && sig.Recv() != nil && isPointerGoType(sig.Recv().Type())
}

func receiverNeedsAddressCheck(sel *types.Selection) bool {
	if !isPointerMethodSelection(sel) {
		return false
	}
	// An embedded pointer is already a pointer receiver and may legally be
	// nil. A value field, in contrast, requires the address of the field;
	// selecting it through a nil enclosing pointer must panic.
	typ := sel.Recv()
	indices := sel.Index()
	for _, index := range indices[:len(indices)-1] {
		if ptr, ok := types.Unalias(typ).Underlying().(*types.Pointer); ok {
			typ = ptr.Elem()
		}
		st, ok := types.Unalias(typ).Underlying().(*types.Struct)
		if !ok || index >= st.NumFields() {
			return false
		}
		typ = st.Field(index).Type()
	}
	return !isPointerGoType(typ)
}

func (p *context) checkMethodCallReceiver(b llvmgen.Builder, ins ssa.Instruction, call *ssa.CallCommon) {
	if len(call.Args) == 0 {
		return
	}
	if p.optPlan != nil {
		if nc, ok := p.optPlan.NilChecks[ins]; ok {
			p.emitPlannedReceiverNilCheck(b, call.Args[0], nc)
			return
		}
		// Single owner: ssaopt owns every call with RecvNilCheck or wrapper call.
		if isPointerMethodWrapperCall(p.goFn, call) || call.RecvNilCheck != nil {
			return
		}
	}
	// Legacy fallback only when Options.ReceiverNilChecks is explicitly supplied
	// by direct cl unit tests without ssa.CallCommon.RecvNilCheck.
	checks := p.options.ReceiverNilChecks
	if checks == nil {
		return
	}
	if check, ok := checks.calls[call.Pos()]; ok {
		p.checkAddressedMethodReceiver(b, call.Args[0], check)
	}
}

func isPointerMethodWrapperCall(owner *ssa.Function, call *ssa.CallCommon) bool {
	return ssaopt.IsPointerMethodWrapperCall(owner, call)
}

func isMethodReceiverWrapper(fn *ssa.Function) bool {
	return ssaopt.IsMethodReceiverWrapper(fn)
}

func (p *context) checkBoundMethodReceiver(b llvmgen.Builder, closure *ssa.MakeClosure) {
	if len(closure.Bindings) == 0 {
		return
	}
	if p.optPlan != nil {
		if nc, ok := p.optPlan.NilChecks[closure]; ok {
			p.emitPlannedReceiverNilCheck(b, closure.Bindings[0], nc)
			return
		}
		if closure.RecvNilCheck != nil {
			return
		}
	}
	checks := p.options.ReceiverNilChecks
	if checks == nil {
		return
	}
	if check, ok := checks.values[closure.Pos()]; ok {
		p.checkAddressedMethodReceiver(b, closure.Bindings[0], check)
	}
}

func (p *context) emitPlannedReceiverNilCheck(b llvmgen.Builder, receiver ssa.Value, nc ssaopt.NilCheckPlan) {
	if nc.Elided || (!nc.NeedBaseCheck && !nc.NeedAddrCheck) {
		return
	}
	p.recordPanicSite(b, nc.Pos)
	if nc.NeedBaseCheck {
		p.emitNilDerefBaseCheck(b, receiver)
	}
	if nc.NeedAddrCheck {
		b.AssertNilDeref(p.compileValue(b, receiver))
	}
}

func (p *context) checkAddressedMethodReceiver(b llvmgen.Builder, receiver ssa.Value, check receiverNilCheck) {
	if isKnownNonNilAddr(receiver) || isWrapNilCheckCall(receiver) {
		return
	}
	if load, ok := receiver.(*ssa.UnOp); ok && !check.address {
		if _, protected := p.recvNilDerefChecks[load]; protected {
			return
		}
	}
	if !check.address && !methodReceiverHasUncheckedBase(receiver) {
		return
	}
	p.recordPanicSite(b, check.pos)
	p.emitNilDerefBaseCheck(b, receiver)
	if check.address {
		b.AssertNilDeref(p.compileValue(b, receiver))
	}
}

func methodReceiverHasUncheckedBase(receiver ssa.Value) bool {
	return ssaopt.MethodReceiverHasUncheckedBase(receiver)
}
