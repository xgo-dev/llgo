package ssa

import (
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/funcattrs"
	"github.com/xgo-dev/llvm"
	"go/types"
)

func (f Function) ApplyValueAttributes(sig *types.Signature, attrs []funcattrs.Attribute) {
	p, fn, environment := f.Prog, f.impl, f.NeedsEnv()
	if len(attrs) == 0 {
		return
	}
	offset := 0
	if environment {
		offset = 1
	}
	if err := funcattrs.Apply(p.ctx, fn, sig, attrs, offset, p.Int().ll.IntTypeWidth()); err != nil {
		panic(err)
	}
	// Scanning collectors and cooperative scheduling introduce accesses that
	// are not yet modeled by pointer contracts. Definitions and imports use the
	// same weaker LLVM policy in these modes, while retaining value guarantees.
	if !p.GCRootsEnabled() && !p.CooperativeSafepointsEnabled() {
		if err := funcattrs.ApplyPointerEffects(p.ctx, fn, sig, attrs, offset); err != nil {
			panic(err)
		}
		var effects []funcattrs.Attribute
		for _, attr := range attrs {
			if attr.Name == "access" || attr.Name == "noalias" {
				effects = append(effects, attr)
			}
		}
		if len(effects) != 0 {
			if p.pointerEffects == nil {
				p.pointerEffects = make(map[string][]funcattrs.Attribute)
			}
			merged, err := directive.Merge(p.pointerEffects[fn.Name()], effects)
			if err != nil {
				panic(err)
			}
			p.pointerEffects[fn.Name()] = merged
		}
	}
	plan, err := funcattrs.PrepareResultAttributes(p.ctx, fn, sig, attrs, offset, p.Int().ll.IntTypeWidth(), func(index int) []int {
		path := []int{index}
		converted := f.Type.raw.Type.(*types.Signature)
		if layout, ok := p.structLayout(p.retType(converted)); ok && layout.wrapped[index] {
			path = append(path, 0)
		}
		return path
	})
	if err != nil {
		panic(err)
	}
	if len(plan) != 0 {
		if p.valuePlans == nil {
			p.valuePlans = make(map[llvm.Value]funcattrs.ValuePlan)
		}
		p.valuePlans[fn] = plan
	}
}

// MaterializeValueAttributes consumes only plans belonging to this module.
func (p Program) MaterializeValueAttributes(m llvm.Module) error {
	plans := make(map[llvm.Value]funcattrs.ValuePlan)
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if plan, ok := p.valuePlans[fn]; ok {
			plans[fn] = plan
		}
	}
	if err := funcattrs.MaterializeValueContracts(m, plans); err != nil {
		return err
	}
	for fn := range plans {
		delete(p.valuePlans, fn)
	}
	return nil
}
