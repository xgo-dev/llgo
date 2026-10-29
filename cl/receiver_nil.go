package cl

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
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

func (p *context) checkMethodCallReceiver(b llssa.Builder, call *ssa.CallCommon) {
	if len(call.Args) == 0 {
		return
	}
	// Promoted method expressions and interface adapters also contain
	// implicit field-address arithmetic, but have no source selector or call
	// position. Keep the required base checks in the generated wrapper itself
	// so storing (*Outer).M in a function value cannot bypass them. The final
	// declared pointer receiver may still legally be nil.
	if isPointerMethodWrapperCall(p.currentSSA(), call) {
		p.checkAddressedMethodReceiver(b, call.Args[0], receiverNilCheck{pos: p.goFn.SSA.Pos()})
		return
	}
	checks := p.options.ReceiverNilChecks
	if checks == nil {
		return
	}
	if check, ok := checks.calls[call.Pos()]; ok {
		// SSA has evaluated the argument expressions by this instruction.
		// Checking while compiling FieldAddr would move the panic before
		// those side effects. The saved receiver also avoids re-reading a
		// variable that an argument expression may have reassigned.
		p.checkAddressedMethodReceiver(b, call.Args[0], check)
	}
}

func isPointerMethodWrapperCall(owner *ssa.Function, call *ssa.CallCommon) bool {
	if !isMethodReceiverWrapper(owner) {
		return false
	}
	fn := call.StaticCallee()
	return fn != nil && fn.Signature.Recv() != nil && isPointerGoType(fn.Signature.Recv().Type())
}

func isMethodReceiverWrapper(fn *ssa.Function) bool {
	return fn != nil && (strings.HasPrefix(fn.Synthetic, "wrapper for ") || strings.HasPrefix(fn.Synthetic, "thunk for "))
}

func (p *context) checkBoundMethodReceiver(b llssa.Builder, closure *ssa.MakeClosure) {
	checks := p.options.ReceiverNilChecks
	if checks == nil || len(closure.Bindings) == 0 {
		return
	}
	if check, ok := checks.values[closure.Pos()]; ok {
		// Method-value formation must panic now, even if the value is never
		// invoked; there is no later argument-evaluation phase to preserve.
		p.checkAddressedMethodReceiver(b, closure.Bindings[0], check)
	}
}

func (p *context) checkAddressedMethodReceiver(b llssa.Builder, receiver ssa.Value, check receiverNilCheck) {
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
	switch receiver := receiver.(type) {
	case *ssa.UnOp:
		return receiver.Op == token.MUL && !isKnownNonNilAddr(receiver.X) && !isWrapNilCheckCall(receiver.X)
	case *ssa.FieldAddr:
		return !isKnownNonNilAddr(receiver.X) && !isWrapNilCheckCall(receiver.X)
	}
	return false
}
