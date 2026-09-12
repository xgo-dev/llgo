package ssa

import (
	"go/types"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/internal/funcattrs"
	"github.com/xgo-dev/llvm"
)

func (p Program) SetFunctionAttributes(name string, attrs []funcattrs.Attribute) error {
	if len(attrs) == 0 {
		return nil
	}
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	merged, err := funcattrs.Merge(p.packageSyntax.functionAttributes[name], attrs)
	if err == nil {
		p.packageSyntax.functionAttributes[name] = merged
	}
	return err
}

// CheckAttributeInstrumentation accounts for compiler-owned operations. Target
// mode effects are already widened on every declaration; these local calls
// document the emitted operation and protect future instrumentation additions.
func (f Function) CheckAttributeInstrumentation(reason string, names ...string) {
	switch reason {
	case "compiler-generated GC root publication":
		if !f.Prog.GCRootsEnabled() && !f.Prog.CooperativeSafepointsEnabled() {
			if err := funcattrs.CheckInstrumentation(f.impl, reason, "memory", "capture", "noalias"); err != nil {
				panic(err)
			}
		}
		funcattrs.WidenForGCRootPublication(f.impl)
		return
	case "cooperative safepoints":
		if !f.Prog.GCRootsEnabled() && !f.Prog.CooperativeSafepointsEnabled() {
			if err := funcattrs.CheckInstrumentation(f.impl, reason, "memory", "capture", "access", "noalias", "nofree", "nosync", "nounwind", "willreturn"); err != nil {
				panic(err)
			}
		}
		funcattrs.WidenForUnknownInstrumentation(f.impl)
		return
	}
	if err := funcattrs.CheckInstrumentation(f.impl, reason, names...); err != nil {
		panic(err)
	}
}

// CheckImplicitRuntimeEffects guards compiler-owned runtime protocols whose
// footprint is not yet available in every caller's declaration. It leaves
// explicit source operations governed by their ordinary contract promises.
func (f Function) CheckImplicitRuntimeEffects(reason string) {
	f.CheckAttributeInstrumentation(reason, "memory", "nofree", "nosync", "nounwind", "willreturn", "capture", "access", "noalias")
}

// SetFunctionAttributeOrigin binds a concrete generic symbol to its declaration.
// Logical indices are preserved, and Apply validates the instantiated types.
func (p Program) SetFunctionAttributeOrigin(instance, origin string) {
	if instance == origin {
		return
	}
	p.packageSyntax.mu.Lock()
	p.packageSyntax.attributeOrigins[instance] = origin
	p.packageSyntax.mu.Unlock()
}

func (p Program) functionAttributes(name string) ([]funcattrs.Attribute, error) {
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	data := p.packageSyntax
	if origin, ok := data.attributeOrigins[name]; ok {
		name = origin
	}
	resolve := func(s string) string {
		// Linknames are symbol mappings, not an instruction to follow arbitrary
		// alias chains. This mirrors the frontend's one-step resolution.
		if link, ok := data.linknames[s]; ok {
			return strings.TrimPrefix(strings.TrimPrefix(link, "C."), "stdcall.")
		}
		return s
	}
	resolved := resolve(name)
	var owners []string
	for source := range data.functionAttributes {
		if source == name || resolve(source) == resolved {
			owners = append(owners, source)
		}
	}
	sort.Strings(owners)
	var sets [][]funcattrs.Attribute
	for _, owner := range owners {
		sets = append(sets, data.functionAttributes[owner])
	}
	return funcattrs.Merge(sets...)
}

func (p Program) applyFunctionAttributes(fn llvm.Value, name string, sig *types.Signature, hasEnvironment bool, bg Background) {
	attrs, err := p.functionAttributes(name)
	if err != nil {
		panic(err)
	}
	offset := 0
	if hasEnvironment {
		offset = 1
	}
	resolve := func(index int) []int {
		return p.functionAttributeResultPath(sig, index, bg)
	}
	if err = funcattrs.Apply(p.ctx, fn, sig, attrs, offset, p.Int().ll.IntTypeWidth(), resolve); err != nil {
		panic(err)
	}
	if p.GCRootsEnabled() || p.CooperativeSafepointsEnabled() {
		funcattrs.WidenForUnknownInstrumentation(fn)
	}
}

// functionAttributeResultPath locates a whole Go result in a multi-result
// LLVM value. On 386 an extra wrapper preserves Go alignment and padding.
func (p Program) functionAttributeResultPath(sig *types.Signature, index int, bg Background) []int {
	indices := []int{index}
	converted := p.FuncDecl(sig, bg).raw.Type.(*types.Signature)
	tuple := p.retType(converted)
	if layout, ok := p.structLayout(tuple); ok && layout.wrapped[index] {
		indices = append(indices, 0)
	}
	return indices
}
