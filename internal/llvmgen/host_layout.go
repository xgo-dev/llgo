package llvmgen

import (
	"go/types"

	lltypes "github.com/xgo-dev/llgo/types"
)

// isHostLayoutMarker recognizes the private identity retained by aliases and
// defined types of structs.HostLayout. Do not recursively search arbitrary
// structs: HostLayout deliberately does not propagate across struct boundaries.
func isHostLayoutMarker(t types.Type) bool {
	return lltypes.IsHostLayoutMarker(t)
}

func hasHostLayout(t types.Type) bool {
	return lltypes.HasHostLayout(t)
}

func containsHostLayout(t types.Type) bool {
	return lltypes.ContainsHostLayout(t)
}

func toLLCallConv(cc CallConv) lltypes.CallConv {
	switch cc {
	case GoDecl:
		return lltypes.GoDecl
	case CDecl:
		return lltypes.CDecl
	case PyDecl:
		return lltypes.PyDecl
	case Stdcall:
		return lltypes.Stdcall
	default:
		return lltypes.CallConvUnknown
	}
}

// nativeFuncAliasABI distinguishes one-word function aliases from the public
// Go signature, which can also be used as the code field of a Go closure.
func (p Program) nativeFuncAliasABI(alias *types.Alias) string {
	if _, ok := types.Unalias(alias).(*types.Signature); !ok {
		return ""
	}
	switch p.nativeTypeDirective(alias) {
	case CDecl:
		return "cdecl"
	case Stdcall:
		return "stdcall"
	}
	return ""
}

// hasNativeTypeDirective preserves declarations on aliases before unaliasing.
func (p Program) hasNativeTypeDirective(t types.Type) bool {
	return p.nativeTypeDirective(t) != ccUnknown
}

func (p Program) nativeTypeDirective(t types.Type) CallConv {
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
				return cc
			}
		}
		alias, ok := t.(*types.Alias)
		if !ok {
			return ccUnknown
		}
		t = alias.Rhs()
	}
}

func (p Program) hostUsesNativeGoLayout() bool {
	return p.sourceSizes.HostUsesNativeGoLayout()
}
