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
			if err := funcattrs.CheckInstrumentation(f.impl, reason, "memory", "capture"); err != nil {
				panic(err)
			}
		}
		funcattrs.WidenForGCRootPublication(f.impl)
		return
	case "cooperative safepoints":
		if !f.Prog.GCRootsEnabled() && !f.Prog.CooperativeSafepointsEnabled() {
			if err := funcattrs.CheckInstrumentation(f.impl, reason, "memory", "capture", "access", "nofree", "nosync", "nounwind", "willreturn"); err != nil {
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
	f.CheckAttributeInstrumentation(reason, "memory", "nofree", "nosync", "nounwind", "willreturn", "capture", "access")
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
	resolve := func(target funcattrs.Target, path []int) ([]int, error) {
		return p.functionAttributePath(sig, target, path, bg), nil
	}
	if err = funcattrs.Apply(p.ctx, fn, sig, attrs, offset, p.Int().ll.IntTypeWidth(), resolve); err != nil {
		panic(err)
	}
	if p.GCRootsEnabled() || p.CooperativeSafepointsEnabled() {
		funcattrs.WidenForUnknownInstrumentation(fn)
	}
}

// functionAttributePath composes source selectors with target layout wrappers
// while the original Go types are available. Source field numbering alone is
// insufficient on 386, where both structs and multiple results may wrap fields
// to preserve Go's alignment and padding.
func (p Program) functionAttributePath(sig *types.Signature, target funcattrs.Target, path []int, bg Background) []int {
	var root types.Type
	var indices []int
	switch target.Scope {
	case funcattrs.Receiver:
		root = sig.Recv().Type()
	case funcattrs.Parameter:
		root = sig.Params().At(target.Index).Type()
	case funcattrs.Result:
		root = sig.Results().At(target.Index).Type()
		if sig.Results().Len() > 1 {
			converted := p.FuncDecl(sig, bg).raw.Type.(*types.Signature)
			tuple := p.retType(converted)
			indices = append(indices, target.Index)
			if layout, ok := p.structLayout(tuple); ok && layout.wrapped[target.Index] {
				indices = append(indices, 0)
			}
		}
	}
	typ := p.Type(root, bg)
	for _, index := range path {
		indices = append(indices, index)
		switch typ.raw.Type.Underlying().(type) {
		case *types.Struct:
			if layout, ok := p.structLayout(typ); ok && layout.wrapped[index] {
				indices = append(indices, 0)
			}
			typ = p.Field(typ, index)
		case *types.Array:
			typ = p.Elem(typ)
		}
	}
	return indices
}
