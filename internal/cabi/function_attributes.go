package cabi

import "github.com/xgo-dev/llgo/internal/funcattrs"

// This describes representation changes, not source-language result numbering.
// Field and fragment contracts can be added without changing the public syntax.
func attributeMapping(info *FuncInfo, paramMap []int) funcattrs.ABIMapping {
	value := func(t *TypeInfo, index int) funcattrs.ABIValue {
		v := funcattrs.DirectValue(index)
		switch t.Kind {
		case AttrVoid:
			v.Kind = funcattrs.Omitted
			v.Indices = nil
		case AttrPointer:
			v.Kind = funcattrs.Indirect
		case AttrWidthType, AttrWidthType2, AttrExtract:
			v.Kind = funcattrs.Fragments
		}
		return v
	}
	m := funcattrs.ABIMapping{Result: value(info.Return, 0), Params: make([]funcattrs.ABIValue, len(info.Params))}
	if info.Return.Kind == AttrPointer {
		m.Result.Indices = []int{1}
	}
	for i, t := range info.Params {
		m.Params[i] = value(t, paramMap[i])
		if t.Kind == AttrWidthType2 {
			m.Params[i].Indices = append(m.Params[i].Indices, paramMap[i]+1)
		}
		if t.Kind == AttrExtract {
			m.Params[i].Indices = nil
			for j := range t.nativeType().StructElementTypes() {
				m.Params[i].Indices = append(m.Params[i].Indices, paramMap[i]+j)
			}
		}
	}
	return m
}
