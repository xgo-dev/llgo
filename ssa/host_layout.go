package ssa

import "go/types"

// isHostLayoutMarker recognizes the private identity retained by aliases and
// defined types of structs.HostLayout. Do not recursively search arbitrary
// structs: HostLayout deliberately does not propagate across struct boundaries.
func isHostLayoutMarker(t types.Type) bool {
	st, ok := t.Underlying().(*types.Struct)
	if !ok || st.NumFields() != 1 || st.Field(0).Name() != "_" {
		return false
	}
	n, ok := types.Unalias(st.Field(0).Type()).(*types.Named)
	if !ok || n.Obj().Pkg() == nil || (n.Obj().Pkg().Path() != "structs" && n.Obj().Pkg().Path() != "github.com/xgo-dev/llgo/runtime/internal/structs") || n.Obj().Name() != "hostLayout" {
		return false
	}
	body, ok := n.Underlying().(*types.Struct)
	return ok && body.NumFields() == 0
}

func hasHostLayout(t types.Type) bool {
	st, ok := t.Underlying().(*types.Struct)
	if ok {
		for i := 0; i < st.NumFields(); i++ {
			if isHostLayoutMarker(st.Field(i).Type()) {
				return true
			}
		}
	}
	return false
}

func hasHostMarkerField(fields []*types.Var) bool {
	for _, f := range fields {
		if isHostLayoutMarker(f.Type()) {
			return true
		}
	}
	return false
}

func containsHostLayout(t types.Type) bool {
	switch t := t.Underlying().(type) {
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			f := t.Field(i).Type()
			if isHostLayoutMarker(f) || containsHostLayout(f) {
				return true
			}
		}
	case *types.Array:
		return containsHostLayout(t.Elem())
	}
	return false
}

// hostLayout computes source sizes without materializing LLVM named types.
// The checker can call this while other named types are still incomplete.
// Enclosing Go aggregates inherit field sizes/alignment, not the host policy.
func (p *goProgram) hostLayout(t types.Type) (sourceLayout, bool) {
	switch t := t.Underlying().(type) {
	case *types.Struct:
		if !containsHostLayout(t) {
			return sourceLayout{}, false
		}
		fields := make([]*types.Var, t.NumFields())
		for i := range fields {
			fields[i] = t.Field(i)
		}
		layout, _ := p.hostFieldsLayout(fields)
		return layout, true
	case *types.Array:
		if elem, ok := p.hostLayout(t.Elem()); ok {
			return arraySourceLayout(elem, t.Len()), true
		}
	}
	return sourceLayout{}, false
}

func (p *goProgram) hostFieldsLayout(fields []*types.Var) (sourceLayout, []int64) {
	host := hasHostMarkerField(fields)
	fieldLayout := func(i int) sourceLayout {
		t := fields[i].Type()
		if host {
			return p.hostFieldLayout(t, false)
		}
		return p.hostNestedLayout(t)
	}
	return aggregateSourceLayout(len(fields), fieldLayout, host || !Program(p).hostUsesNativeGoLayout())
}

func (p *goProgram) hostFieldLayout(t types.Type, native bool) sourceLayout {
	prog := Program(p)
	native = native || prog.hasNativeTypeDirective(t)
	switch u := t.Underlying().(type) {
	case *types.Basic:
		// Strings retain their Go aggregate representation. Numeric types
		// retain their widths but use the target ABI's alignment. A marker
		// must not narrow an addressable Go uintptr or pointer slot.
		if u.Kind() == types.Uintptr || u.Kind() == types.UnsafePointer {
			word := int64(prog.GoWordSize())
			return sourceLayout{word, word}
		}
		if u.Kind() != types.String {
			return prog.hostBasicLayouts[u.Kind()]
		}
	case *types.Pointer, *types.Map, *types.Chan:
		word := int64(prog.GoWordSize())
		return sourceLayout{word, word}
	case *types.Signature:
		// A layout marker never changes a Go closure into a C callback.
		if native {
			// Explicit native function types use one physical code pointer in
			// every container, including ordinary Go variables and arrays.
			width := int64(prog.PointerSize())
			return sourceLayout{width, width}
		}
	case *types.Array:
		return arraySourceLayout(p.hostFieldLayout(u.Elem(), native), u.Len())
	}
	// A nested struct, including a closure's representation, owns its layout.
	return p.hostNestedLayout(t)
}

// hasNativeTypeDirective preserves declarations on aliases before unaliasing.
func (p Program) hasNativeTypeDirective(t types.Type) bool {
	for {
		var obj *types.TypeName
		switch n := t.(type) {
		case *types.Named:
			obj = n.Obj()
		case *types.Alias:
			obj = n.Obj()
		}
		if obj != nil {
			if cc, ok := p.packageTypeCallConv(typeNameLinkname(obj)); ok && isNativeCallConv(cc) {
				return true
			}
		}
		alias, ok := t.(*types.Alias)
		if !ok {
			return false
		}
		t = alias.Rhs()
	}
}

// These targets already use LLVM's native layout for ordinary aggregates.
// Consult that storage policy only inside HostLayout aggregates and their
// containers. Ordinary Go source sizes and storage stay unchanged by this feature.
func (p Program) hostUsesNativeGoLayout() bool {
	arch := p.target.effectiveGOARCH()
	return arch != "386" && arch != "wasm" && p.GoWordSize() == 4 && p.hostBasicLayouts[types.Int64].align > 4
}

func (p *goProgram) hostNestedLayout(t types.Type) sourceLayout {
	if !Program(p).hostUsesNativeGoLayout() {
		return sourceLayout{p.Sizeof(t), p.Alignof(t)}
	}
	return p.hostNativeLayout(t, false)
}

// hostNativeLayout describes existing nested storage without materializing LLVM
// types during source checking. It is reachable only through HostLayout sizing.
func (p *goProgram) hostNativeLayout(t types.Type, native bool) sourceLayout {
	prog := Program(p)
	native = native || prog.hasNativeTypeDirective(t)
	if l, ok := p.hostLayout(t); ok {
		return l
	}
	if l, ok := p.vectorLayout(t, native); ok {
		return l
	}
	word := int64(prog.GoWordSize())
	switch u := t.Underlying().(type) {
	case *types.Basic:
		if u.Kind() == types.String {
			return sourceLayout{2 * word, word}
		}
		if u.Kind() > types.Invalid && u.Kind() <= types.UnsafePointer {
			return prog.hostBasicLayouts[u.Kind()]
		}
	case *types.Pointer, *types.Chan, *types.Map:
		return sourceLayout{word, word}
	case *types.Interface:
		return sourceLayout{2 * word, word}
	case *types.Slice:
		return sourceLayout{3 * word, word}
	case *types.Signature:
		if native {
			return sourceLayout{word, word}
		}
		return sourceLayout{2 * word, word}
	case *types.Array:
		return arraySourceLayout(p.hostNativeLayout(u.Elem(), native), u.Len())
	case *types.Struct:
		native = native || IsClosure(u)
		fields := make([]*types.Var, u.NumFields())
		for i := range fields {
			fields[i] = u.Field(i)
		}
		native = native || prog.nativeFields(fields)
		l, _ := aggregateSourceLayout(len(fields), func(i int) sourceLayout { return p.hostNativeLayout(fields[i].Type(), native) }, false)
		return l
	}
	return sourceLayout{p.sizes.Sizeof(t), p.sizes.Alignof(t)}
}
