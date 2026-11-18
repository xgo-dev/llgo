package cabi

import "github.com/xgo-dev/llvm"

// LowerWideVectorCalls gives LLGo's amd64 wide-vector values a feature-neutral
// memory ABI before FMV builds forwarding entries. A baseline caller and an
// AVX2 callee must not independently choose XMM versus YMM call carriers.
// This also covers multiple-result tuples; source Go aggregates already use
// their Go storage representation. Native C wide-vector bindings are rejected
// by the frontend and are not part of this internal ABI.
func (p *Transformer) LowerWideVectorCalls(m llvm.Module) {
	if p.arch != "amd64" {
		return
	}
	wide := *p
	wide.sys = &wideVectorABI{p}
	wide.transformModule(m, nil)
}

type wideVectorABI struct{ *Transformer }

func (*wideVectorABI) SupportByVal() bool    { return false }
func (*wideVectorABI) SkipEmptyParams() bool { return false }

func (p *wideVectorABI) containsWideVector(typ llvm.Type) bool {
	switch typ.TypeKind() {
	case llvm.VectorTypeKind:
		return p.td.TypeSizeInBits(typ) > 128
	case llvm.ArrayTypeKind:
		return typ.ArrayLength() != 0 && p.containsWideVector(typ.ElementType())
	case llvm.StructTypeKind:
		if !typ.IsStructOpaque() {
			for _, field := range typ.StructElementTypes() {
				if p.containsWideVector(field) {
					return true
				}
			}
		}
	}
	return false
}

func (p *wideVectorABI) IsWrapType(_ llvm.Context, _ llvm.Type, typ llvm.Type, _ int) bool {
	return p.containsWideVector(typ)
}

func (p *wideVectorABI) GetTypeInfo(_ llvm.Context, _ llvm.Type, typ llvm.Type, _ int) *TypeInfo {
	info := &TypeInfo{Type: typ, Type1: typ}
	if p.containsWideVector(typ) {
		info.Kind = AttrPointer
		info.Type1 = llvm.PointerType(typ, 0)
		info.Size, info.Align = p.Sizeof(typ), p.Alignof(typ)
		info.PreservePointerABI = true
	}
	return info
}
