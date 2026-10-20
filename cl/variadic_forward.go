package cl

import (
	"fmt"
	"go/token"
	"go/types"
	"sort"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func (p *context) cVariadicForwardingError(fn *ssa.Function, reason string) error {
	target := p.prog.Target()
	return fmt.Errorf("cannot forward C variadic arguments in %s on %s/%s: %s",
		fn, target.GOOS, target.GOARCH, reason)
}

func (p *context) cVariadicWrapperError(fn *ssa.Function, checked map[*ssa.Function]error) error {
	if fn == nil || fn.Synthetic == "" || !llssa.HasNameValist(fn.Signature) {
		return nil
	}
	if err, ok := checked[fn]; ok {
		return err
	}
	seen := make(map[*ssa.Function]bool)
	work := []*ssa.Function{fn}
	for len(work) > 0 {
		current := work[len(work)-1]
		work = work[:len(work)-1]
		if seen[current] || current.Synthetic == "" || !llssa.HasNameValist(current.Signature) {
			continue
		}
		if err, ok := checked[current]; ok {
			if err != nil {
				checked[fn] = err
				return err
			}
			continue
		}
		seen[current] = true
		if err := p.variadicErrors[current]; err != nil {
			checked[fn] = err
			return err
		}
		for _, block := range current.Blocks {
			for _, instr := range block.Instrs {
				call, ok := instr.(ssa.CallInstruction)
				if !ok || !isCVariadicWrapperForwarding(current, call.Common()) {
					continue
				}
				reason := p.cVariadicWrapperForwardingReason(current, call.Common())
				callee := call.Common().StaticCallee()
				if reason == "" && callee == nil {
					reason = "indirect variadic forwarding is not supported"
				}
				if reason != "" {
					err := p.cVariadicForwardingError(current, reason)
					checked[fn] = err
					return err
				}
				work = append(work, callee)
			}
		}
	}
	// Cache a negative result only after exploring the complete graph. A
	// back-edge alone does not prove that a cycle has no unsupported exit.
	for current := range seen {
		checked[current] = nil
	}
	return nil
}

// compileType eagerly emits both value and pointer method sets, including
// wrappers that are never selected by the source. Diagnose uses instead of
// rejecting an unused value wrapper beside a supported pointer wrapper.
// Explicit C calls already supply their ellipsis and need no forwarding.
func (p *context) validateCVariadicWrapperUses() error {
	hasWrappers := false
	for fn := range p.funcs {
		if fn.Synthetic != "" && llssa.HasNameValist(fn.Signature) {
			hasWrappers = true
			break
		}
	}
	if !hasWrappers {
		return nil
	}
	functions := make([]*ssa.Function, 0, len(p.funcs))
	for fn := range p.funcs {
		if fn.Synthetic == "" || !isRecoverTransparentWrapper(fn) {
			functions = append(functions, fn)
		}
	}
	sort.Slice(functions, func(i, j int) bool { return functions[i].String() < functions[j].String() })
	positioned := func(err error, pos token.Pos, fn *ssa.Function) error {
		if !pos.IsValid() {
			pos = fn.Pos()
		}
		if pos.IsValid() {
			return fmt.Errorf("%s: %w", p.fset.Position(pos), err)
		}
		return err
	}
	checked := make(map[*ssa.Function]error)
	var operands []*ssa.Value
	for _, fn := range functions {
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if conversion, ok := instr.(*ssa.MakeInterface); ok {
					typ := conversion.X.Type()
					methods := p.goProg.MethodSets.MethodSet(typ)
					// Only the destination interface's methods are exposed by
					// this conversion. In particular, boxing into any needs no
					// variadic forwarding, even if the concrete type has it.
					destination := conversion.Type().Underlying().(*types.Interface)
					for i := 0; i < destination.NumMethods(); i++ {
						method := destination.Method(i)
						selection := methods.Lookup(method.Pkg(), method.Name())
						if !llssa.HasNameValist(selection.Type().(*types.Signature)) {
							continue
						}
						if !isPointerGoType(typ) {
							// Only methods in the value's method set are exposed,
							// but interface dispatch selects their pointer entries.
							pointerMethods := p.goProg.MethodSets.MethodSet(types.NewPointer(typ))
							selection = pointerMethods.Lookup(selection.Obj().Pkg(), selection.Obj().Name())
						}
						if wrapper := p.methodValue(selection); wrapper != nil {
							if err := p.cVariadicWrapperError(wrapper, checked); err != nil {
								return positioned(err, instr.Pos(), fn)
							}
						}
					}
				}
				operands = instr.Operands(operands[:0])
				for _, operand := range operands {
					wrapper, ok := (*operand).(*ssa.Function)
					if !ok || wrapper.Synthetic == "" {
						continue
					}
					if err := p.cVariadicWrapperError(wrapper, checked); err != nil {
						return positioned(err, instr.Pos(), fn)
					}
				}
			}
		}
	}
	return nil
}
