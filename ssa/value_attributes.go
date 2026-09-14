package ssa

import (
	"go/types"
	"sort"

	"github.com/xgo-dev/llgo/internal/funcattrs"
	"github.com/xgo-dev/llvm"
)

func (p Program) SetValueAttributes(name string, attrs []funcattrs.Attribute) error {
	if len(attrs) == 0 {
		return nil
	}
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	merged, err := funcattrs.Merge(p.packageSyntax.valueAttributes[name], attrs)
	if err == nil {
		p.packageSyntax.valueAttributes[name] = merged
	}
	return err
}

func (p Program) valueAttributes(name string) ([]funcattrs.Attribute, error) {
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	data := p.packageSyntax
	if origin, ok := data.attributeOrigins[name]; ok {
		name = origin
	}
	resolved := data.resolveAttributeName(name)
	var sets [][]funcattrs.Attribute
	var owners []string
	for source := range data.valueAttributes {
		if source == name || data.resolveAttributeName(source) == resolved {
			owners = append(owners, source)
		}
	}
	sort.Strings(owners)
	for _, owner := range owners {
		sets = append(sets, data.valueAttributes[owner])
	}
	return funcattrs.Merge(sets...)
}

func (p Program) applyValueAttributes(fn llvm.Value, name string, sig *types.Signature, environment bool, bg Background) {
	attrs, err := p.valueAttributes(name)
	if err != nil {
		panic(err)
	}
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
	// Root publication and safepoints can introduce accesses outside the
	// source body. Use the same conservative policy for imported declarations.
	if !p.GCRootsEnabled() && !p.CooperativeSafepointsEnabled() {
		if err := funcattrs.ApplyPointerEffects(p.ctx, fn, sig, attrs, offset); err != nil {
			panic(err)
		}
		for _, attr := range attrs {
			if attr.Name == "noalias" || attr.Name == "access" {
				if p.pointerEffects == nil {
					p.pointerEffects = make(map[string][]funcattrs.Attribute)
				}
				p.pointerEffects[fn.Name()] = append(p.pointerEffects[fn.Name()], attr)
			}
		}
	}
	plan, err := funcattrs.PrepareResultAttributes(p.ctx, fn, sig, attrs, offset, p.Int().ll.IntTypeWidth(), func(index int) []int {
		path := []int{index}
		converted := p.FuncDecl(sig, bg).raw.Type.(*types.Signature)
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
