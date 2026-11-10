package ssa

import "go/types"

// wasmNativeLayout keeps unsafe/reflect sizes consistent with the explicit C
// storage view on Memory32. Compute from source metadata: the checker can query
// an incomplete named type before it is safe to materialize its LLVM type.
func (p *goProgram) wasmNativeLayout(t types.Type) (sourceLayout, bool) {
	prog := Program(p)
	if !prog.usesWideGoStorage() || !prog.packageSyntax.hasNativeTypeMetadata.Load() {
		return sourceLayout{}, false
	}
	switch u := t.Underlying().(type) {
	case *types.Array:
		// An array's own directive selects C slots. An ordinary Go array of
		// native callbacks must retain the callbacks' Go storage width.
		if _, named := types.Unalias(t).(*types.Named); named && prog.hasLegacyNativeTypeLayout(t) {
			return arraySourceLayout(p.wasmNativeFieldLayout(u.Elem()), u.Len()), true
		}
		if elem, ok := p.wasmNativeLayout(u.Elem()); ok {
			return arraySourceLayout(elem, u.Len()), true
		}
	case *types.Struct:
		native := prog.hasLegacyNativeTypeLayout(t)
		if !native && u.NumFields() > 0 {
			native = prog.nativeFields([]*types.Var{u.Field(0)})
		}
		if native {
			layout, _ := p.wasmNativeFieldsLayout(u.NumFields(), u.Field)
			return layout, true
		}
		for i := 0; i < u.NumFields(); i++ {
			if known, ok := p.wasmNativeLayout(u.Field(i).Type()); ok {
				layout, _ := aggregateSourceLayout(u.NumFields(), func(j int) sourceLayout {
					if j == i {
						return known
					}
					return p.wasmGoFieldLayout(u.Field(j).Type())
				}, true)
				return layout, true
			}
		}
	}
	return sourceLayout{}, false
}

func (p *goProgram) wasmNativeFieldsLayout(n int, field func(int) *types.Var) (sourceLayout, []int64) {
	return aggregateSourceLayout(n, func(i int) sourceLayout {
		return p.wasmNativeFieldLayout(field(i).Type())
	}, false)
}

// Query a subtree once and retain the size/alignment pair. Do not memoize
// across checker calls, since a named type can still be incomplete.
func (p *goProgram) wasmGoFieldLayout(t types.Type) sourceLayout {
	if layout, ok := p.wasmNativeLayout(t); ok {
		return layout
	}
	return sourceLayout{p.goSizeof(t), p.goAlignof(t)}
}

func (p *goProgram) wasmNativeFieldLayout(t types.Type) sourceLayout {
	word := int64(p.ptrSize)
	switch u := t.Underlying().(type) {
	case *types.Basic:
		if u.Kind() == types.Uintptr || u.Kind() == types.UnsafePointer {
			return sourceLayout{word, word}
		}
	case *types.Pointer, *types.Signature, *types.Map, *types.Chan:
		return sourceLayout{word, word}
	case *types.Array:
		return arraySourceLayout(p.wasmNativeFieldLayout(u.Elem()), u.Len())
	}
	// A nested record owns its storage policy; an outer C directive does not
	// recursively turn an ordinary Go record into a C record.
	return p.wasmGoFieldLayout(t)
}
