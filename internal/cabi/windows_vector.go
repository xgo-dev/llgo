package cabi

import "github.com/xgo-dev/llvm"

// LowerWindowsVectorParams exposes the native Win64 indirect vector-parameter
// ABI before FMV creates musttail forwarding functions. Otherwise LLVM's late
// ABI lowering copies a vector into the forwarding function's own stack frame,
// then passes a dangling pointer when the tail jump releases that frame.
// Vector results still use XMM0. This does not lower aggregate parameters or
// zero-sized CPU-query receivers, which FMV must inspect in their source form.
func (p *Transformer) LowerWindowsVectorParams(m llvm.Module) {
	if _, ok := p.sys.(*TypeInfoWindowsAmd64); !ok {
		return
	}
	narrow := *p
	narrow.sys = &windowsVectorParams{p}
	narrow.transformModule(m, func(cc llvm.CallConv) bool {
		// 79 is LLVM's explicit Win64 calling convention.
		return cc == llvm.CCallConv || cc == llvm.CallConv(79)
	})
}

type windowsVectorParams struct{ *Transformer }

func (*windowsVectorParams) SupportByVal() bool    { return false }
func (*windowsVectorParams) SkipEmptyParams() bool { return false }

func (p *windowsVectorParams) IsWrapType(_ llvm.Context, _ llvm.Type, typ llvm.Type, index int) bool {
	return index > 0 && typ.TypeKind() == llvm.VectorTypeKind && p.td.TypeSizeInBits(typ) == 128
}

func (p *windowsVectorParams) GetTypeInfo(ctx llvm.Context, ft, typ llvm.Type, index int) *TypeInfo {
	info := &TypeInfo{Type: typ, Type1: typ}
	if p.IsWrapType(ctx, ft, typ, index) {
		info.Kind = AttrPointer
		info.Type1 = llvm.PointerType(typ, 0)
		info.Size, info.Align, info.ByValAlign = 16, 16, 16
	}
	return info
}
