package ssa

import "go/types"

// An explicit native function type has one representation everywhere. In
// particular, *Callback cannot depend on whether it points into a Go record,
// HostLayout record, or legacy native record.
func (p Program) isNativeCallbackType(t types.Type) bool {
	_, signature := t.Underlying().(*types.Signature)
	return signature && p.hasNativeTypeDirective(t)
}

func (p Program) containsNativeCallback(t types.Type) bool {
	if p.isNativeCallbackType(t) {
		return true
	}
	switch u := t.Underlying().(type) {
	case *types.Array:
		return p.containsNativeCallback(u.Elem())
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if p.containsNativeCallback(u.Field(i).Type()) {
				return true
			}
		}
	}
	return false
}

func (p *goProgram) nativeCallbackLayout(t types.Type) (sourceLayout, bool) {
	prog := Program(p)
	// Equal-width targets and packages without native directives already
	// have the correct source layout.
	if !prog.usesWideGoStorage() || !prog.packageSyntax.hasNativeTypeMetadata.Load() {
		return sourceLayout{}, false
	}
	if prog.isNativeCallbackType(t) {
		width := int64(prog.PointerSize())
		return sourceLayout{width, width}, true
	}
	switch u := t.Underlying().(type) {
	case *types.Array:
		if elem, ok := p.nativeCallbackLayout(u.Elem()); ok {
			return arraySourceLayout(elem, u.Len()), true
		}
	case *types.Struct:
		if !prog.containsNativeCallback(t) {
			return sourceLayout{}, false
		}
		fields := make([]*types.Var, u.NumFields())
		for i := range fields {
			fields[i] = u.Field(i)
		}
		layout, _, ok := p.nativeCallbackFieldsLayout(fields)
		return layout, ok
	}
	return sourceLayout{}, false
}

func (p *goProgram) nativeCallbackFieldsLayout(fields []*types.Var) (sourceLayout, []int64, bool) {
	prog := Program(p)
	if !prog.usesWideGoStorage() || !prog.packageSyntax.hasNativeTypeMetadata.Load() {
		return sourceLayout{}, nil, false
	}
	found := false
	for _, field := range fields {
		if prog.containsNativeCallback(field.Type()) {
			found = true
			break
		}
	}
	if !found {
		return sourceLayout{}, nil, false
	}
	layout, offsets := aggregateSourceLayout(len(fields), func(i int) sourceLayout {
		t := fields[i].Type()
		return sourceLayout{p.Sizeof(t), p.Alignof(t)}
	}, !prog.nativeFields(fields))
	return layout, offsets, true
}
