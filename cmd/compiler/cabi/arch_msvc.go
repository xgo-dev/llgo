package cabi

import "github.com/xgo-dev/llvm"

// TypeInfoWindowsAmd64 implements the Microsoft x64 ABI aggregate rules.
// Unlike the SysV ABI, ordinary aggregates are never split across multiple
// registers. Aggregates of exactly 1, 2, 4, or 8 bytes use one integer
// register; all other non-empty aggregates are passed or returned indirectly.
type TypeInfoWindowsAmd64 struct {
	*Transformer
}

func (p *TypeInfoWindowsAmd64) SupportByVal() bool {
	return false
}

func (p *TypeInfoWindowsAmd64) SkipEmptyParams() bool {
	return false
}

func (p *TypeInfoWindowsAmd64) IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool {
	return isAggregateType(typ)
}

func (p *TypeInfoWindowsAmd64) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	info := &TypeInfo{Type: typ, Type1: typ}
	if typ.TypeKind() == llvm.VoidTypeKind {
		info.Kind = AttrVoid
		return info
	}
	if !isAggregateType(typ) {
		return info
	}

	info.Size = p.Sizeof(typ)
	info.Align = p.Alignof(typ)
	if info.Size == 0 {
		// Clang models Microsoft's empty-structure extension as a four-byte
		// integer argument/result on x64, even though LLVM's {} has size zero.
		info.Kind = AttrWidthType
		info.Type1 = ctx.Int32Type()
		return info
	}
	if isRegisterAggregateSize(info.Size) {
		info.Kind = AttrWidthType
		info.Type1 = ctx.IntType(info.Size * 8)
		return info
	}
	// The Microsoft x64 ABI passes larger aggregates through caller-owned
	// temporaries. This is a plain pointer parameter, not LLVM byval.
	info.Kind = AttrPointer
	info.Type1 = llvm.PointerType(typ, 0)
	return info
}

// Windows on ARM64 follows the AAPCS64 aggregate classification implemented
// by TypeInfoArm64. Keep a distinct selected type so OS-specific ABI changes
// cannot silently fall back to an architecture-only implementation.
type TypeInfoWindowsArm64 struct {
	*TypeInfoArm64
}

func (p *TypeInfoWindowsArm64) SkipEmptyParams() bool {
	// Clang lowers Microsoft's empty-structure extension to void on ARM64
	// for both parameters and results. TypeInfoArm64 keeps an empty result
	// for Go-level ABI purposes, so handle the C ABI case here explicitly.
	return false
}

func (p *TypeInfoWindowsArm64) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	if isAggregateType(typ) && p.Sizeof(typ) == 0 {
		return &TypeInfo{Type: typ, Kind: AttrVoid, Type1: ctx.VoidType()}
	}
	return p.TypeInfoArm64.GetTypeInfo(ctx, ftyp, typ, index)
}

// TypeInfoWindows386 implements the Microsoft x86 structure-return rules and
// the cdecl aggregate parameter rules emitted by Clang for the MSVC target.
type TypeInfoWindows386 struct {
	*Transformer
}

func (p *TypeInfoWindows386) SupportByVal() bool {
	return true
}

func (p *TypeInfoWindows386) SkipEmptyParams() bool {
	return false
}

func (p *TypeInfoWindows386) IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool {
	return isAggregateType(typ)
}

func (p *TypeInfoWindows386) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	bret := index == 0
	nativeType, _ := windows386NativeAggregateType(ctx, typ)
	info := &TypeInfo{Type: typ, NativeType: nativeType, Type1: nativeType}
	if typ.TypeKind() == llvm.VoidTypeKind {
		info.Kind = AttrVoid
		return info
	}
	if !isAggregateType(typ) {
		return info
	}

	info.Size = p.Sizeof(nativeType)
	info.Align = p.Alignof(nativeType)
	if info.Size == 0 {
		if bret {
			info.Kind = AttrVoid
			info.Type1 = ctx.VoidType()
		} else {
			info.Kind = AttrPointer
			info.Type1 = llvm.PointerType(nativeType, 0)
			info.ByValAlign = 4
		}
		return info
	}
	if bret {
		if isRegisterAggregateSize(info.Size) {
			info.Kind = AttrWidthType
			info.Type1 = windows386ReturnType(ctx, nativeType, info.Size)
		} else {
			info.Kind = AttrPointer
			info.Type1 = llvm.PointerType(nativeType, 0)
		}
		return info
	}

	// MSVC x86 passes a small structure's direct register-sized scalar
	// members as separate arguments. Arrays and structures that contain
	// smaller integers or nested aggregates remain indirect byval arguments.
	if nativeType.TypeKind() == llvm.StructTypeKind && info.Size <= 16 {
		subs := nativeType.StructElementTypes()
		if windows386CanExtract(subs) && p.unpaddedSize(subs) == info.Size {
			if len(subs) == 1 {
				info.Kind = AttrWidthType
				info.Type1 = subs[0]
			} else {
				info.Kind = AttrExtract
			}
			return info
		}
	}
	info.Kind = AttrPointer
	info.Type1 = llvm.PointerType(nativeType, 0)
	info.ByValAlign = 4
	return info
}

// windows386NativeAggregateType reverses the explicit wrappers used by the
// SSA backend to give ordinary Go/386 structs their gc-compatible four-byte
// field alignment. A value crossing a native C boundary must instead use the
// layout selected by Clang for the MSVC target (where int64 and double fields
// have eight-byte alignment).
//
// The wrappers are packed anonymous structs containing the real field and,
// when necessary, byte padding. Go has no packed-struct source type, so this
// representation is private to the Go/386 layout lowering and can be decoded
// here without changing native C types.
func windows386NativeAggregateType(ctx llvm.Context, typ llvm.Type) (llvm.Type, bool) {
	switch typ.TypeKind() {
	case llvm.ArrayTypeKind:
		elem, changed := windows386NativeAggregateType(ctx, typ.ElementType())
		if !changed {
			return typ, false
		}
		return llvm.ArrayType(elem, typ.ArrayLength()), true
	case llvm.StructTypeKind:
		sourceFields := typ.StructElementTypes()
		nativeFields := make([]llvm.Type, 0, len(sourceFields))
		changed := false
		for _, field := range sourceFields {
			if inner, ok := windows386GoFieldWrapper(field); ok {
				field = inner
				changed = true
			}
			native, nestedChanged := windows386NativeAggregateType(ctx, field)
			nativeFields = append(nativeFields, native)
			changed = changed || nestedChanged
		}
		if changed && len(nativeFields) != 0 {
			last := nativeFields[len(nativeFields)-1]
			if last.TypeKind() == llvm.ArrayTypeKind && last.ArrayLength() == 0 {
				// toLLVMStructBody may append a zero-length integer array solely
				// to restore Go's outer alignment after packing a field wrapper.
				nativeFields = nativeFields[:len(nativeFields)-1]
			}
		}
		if !changed {
			return typ, false
		}
		return ctx.StructType(nativeFields, false), true
	default:
		return typ, false
	}
}

func windows386GoFieldWrapper(typ llvm.Type) (llvm.Type, bool) {
	if typ.TypeKind() != llvm.StructTypeKind || !typ.IsStructPacked() {
		return llvm.Type{}, false
	}
	fields := typ.StructElementTypes()
	if len(fields) == 1 {
		return fields[0], true
	}
	if len(fields) == 2 && fields[1].TypeKind() == llvm.ArrayTypeKind &&
		fields[1].ElementType().TypeKind() == llvm.IntegerTypeKind &&
		fields[1].ElementType().IntTypeWidth() == 8 {
		return fields[0], true
	}
	return llvm.Type{}, false
}

func (p *TypeInfoWindows386) unpaddedSize(types []llvm.Type) int {
	size := 0
	for _, typ := range types {
		size += p.Sizeof(typ)
	}
	return size
}

func isAggregateType(typ llvm.Type) bool {
	switch typ.TypeKind() {
	case llvm.ArrayTypeKind, llvm.StructTypeKind:
		return true
	}
	return false
}

func isRegisterAggregateSize(size int) bool {
	return size == 1 || size == 2 || size == 4 || size == 8
}

func windows386ReturnType(ctx llvm.Context, typ llvm.Type, size int) llvm.Type {
	if typ.TypeKind() == llvm.StructTypeKind {
		subs := typ.StructElementTypes()
		if len(subs) == 1 && subs[0].TypeKind() == llvm.PointerTypeKind && size == 4 {
			return subs[0]
		}
	}
	return ctx.IntType(size * 8)
}

func windows386CanExtract(types []llvm.Type) bool {
	if len(types) == 0 {
		return false
	}
	for _, typ := range types {
		switch typ.TypeKind() {
		case llvm.FloatTypeKind, llvm.DoubleTypeKind, llvm.PointerTypeKind:
		case llvm.IntegerTypeKind:
			if width := typ.IntTypeWidth(); width != 32 && width != 64 {
				return false
			}
		default:
			return false
		}
	}
	return true
}
