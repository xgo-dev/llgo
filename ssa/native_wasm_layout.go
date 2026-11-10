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
		if elem, ok := p.wasmNativeLayout(u.Elem()); ok {
			return arraySourceLayout(elem, u.Len()), true
		}
	case *types.Struct:
		native := prog.hasLegacyNativeTypeLayout(t)
		if !native && u.NumFields() > 0 {
			native = prog.nativeFields([]*types.Var{u.Field(0)})
		}
		changed := native
		for i := 0; i < u.NumFields() && !changed; i++ {
			_, changed = p.wasmNativeLayout(u.Field(i).Type())
		}
		if changed {
			layout, _ := p.wasmNativeFieldsLayout(u.NumFields(), u.Field, native)
			return layout, true
		}
	}
	return sourceLayout{}, false
}

func (p *goProgram) wasmNativeFieldsLayout(n int, field func(int) *types.Var, native bool) (sourceLayout, []int64) {
	return aggregateSourceLayout(n, func(i int) sourceLayout {
		t := field(i).Type()
		if native {
			return p.wasmNativeFieldLayout(t)
		}
		return sourceLayout{p.Sizeof(t), p.Alignof(t)}
	}, !native)
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
	return sourceLayout{p.Sizeof(t), p.Alignof(t)}
}
