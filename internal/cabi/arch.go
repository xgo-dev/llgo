package cabi

import (
	"slices"

	"github.com/xgo-dev/llvm"
)

func elementTypesCount(typ llvm.Type) int {
	switch typ.TypeKind() {
	case llvm.VoidTypeKind:
		return 0
	case llvm.StructTypeKind:
		var count int
		for _, t := range typ.StructElementTypes() {
			count += elementTypesCount(t)
		}
		return count
	case llvm.ArrayTypeKind:
		return typ.ArrayLength() * elementTypesCount(typ.ElementType())
	}
	return 1
}

func elementTypes(td llvm.TargetData, typ llvm.Type) (types []llvm.Type) {
	switch typ.TypeKind() {
	case llvm.VoidTypeKind:
	case llvm.StructTypeKind:
		for _, t := range typ.StructElementTypes() {
			types = append(types, elementTypes(td, t)...)
		}
	case llvm.ArrayTypeKind:
		n := typ.ArrayLength()
		switch n {
		case 0:
		case 1:
			return elementTypes(td, typ.ElementType())
		default:
			sub := elementTypes(td, typ.ElementType())
			return slices.Repeat(sub, n)
		}
	default:
		return []llvm.Type{typ}
	}
	return
}

func checkTypes(typs []llvm.Type, typ llvm.Type) bool {
	for _, t := range typs {
		if t != typ {
			return false
		}
	}
	return true
}

func containsType(typs []llvm.Type, typ llvm.Type) bool {
	for _, t := range typs {
		if t == typ {
			return true
		}
	}
	return false
}

func containsTypeAny(typs []llvm.Type, typ1, typ2 llvm.Type) bool {
	for _, t := range typs {
		if t == typ1 || t == typ2 {
			return true
		}
	}
	return false
}

func hasTypesKind(typs []llvm.Type, kind llvm.TypeKind) bool {
	for _, t := range typs {
		if t.TypeKind() == kind {
			return true
		}
	}
	return false
}

type TypeInfoAmd64 struct {
	*Transformer
}

func (p *TypeInfoAmd64) SupportByVal() bool {
	return true
}

func (p *TypeInfoAmd64) SkipEmptyParams() bool {
	return true
}

func (p *TypeInfoAmd64) IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool {
	return elementTypesCount(typ) >= 2
}

func (p *TypeInfoAmd64) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	info := &TypeInfo{}
	info.Type = typ
	info.Type1 = typ
	if typ.TypeKind() == llvm.VoidTypeKind {
		info.Kind = AttrVoid
		return info
	}
	info.Size = p.Sizeof(typ)
	info.Align = p.Alignof(typ)
	if p.hasUnalignedField(typ, 0) {
		// SysV classifies aggregates with unaligned fields as MEMORY, even
		// below 16 bytes. Flattening them loses the packed member offsets.
		info.Kind = AttrPointer
		info.Type1 = llvm.PointerType(typ, 0)
		info.ByValAlign = max(8, info.Align)
		return info
	}
	if n := elementTypesCount(typ); n >= 2 {
		if info.Size > 16 {
			info.Kind = AttrPointer
			info.Type1 = llvm.PointerType(typ, 0)
		} else if info.Size <= 8 {
			info.Kind = AttrWidthType
			info.Type1 = ctx.IntType(info.Size * 8)
			types := elementTypes(p.td, typ)
			if types[0] == ctx.FloatType() && types[1] == ctx.FloatType() {
				info.Type1 = llvm.VectorType(ctx.FloatType(), 2)
			}
		} else {
			types := elementTypes(p.td, typ)
			if n == 2 {
				// skip (i64|double,*) (*,i64/double)
				if p.Sizeof(types[0]) == 8 || p.Sizeof(types[1]) == 8 {
					info.Kind = AttrWidthType2
					info.Type1 = types[0]
					info.Type2 = types[1]
					return info
				}
			}
			var offset int
			var index int
			for i, et := range types {
				align := p.Alignof(et)
				offset = (offset + p.Sizeof(et) + align - 1) &^ (align - 1)
				if offset < 8 {
					continue
				} else if offset > 8 {
					index = i
				} else {
					index = i + 1
				}
				break
			}
			subType := func(subs []llvm.Type, left bool) llvm.Type {
				if len(subs) == 1 {
					return subs[0]
				} else if len(subs) == 2 && subs[0] == ctx.FloatType() && subs[1] == ctx.FloatType() {
					return llvm.VectorType(ctx.FloatType(), 2)
				}
				if left {
					return ctx.Int64Type()
				}
				var n int
				for _, sub := range subs {
					align := p.Alignof(sub)
					n = (n + p.Sizeof(sub) + align - 1) &^ (align - 1)
				}
				// Each SysV register component occupies at most one
				// eightbyte, even when the aggregate is over-aligned.
				align := min(info.Align, 8)
				n = (n + align - 1) &^ (align - 1)
				return ctx.IntType(n * 8)
			}
			info.Kind = AttrWidthType2
			info.Type1 = subType(types[0:index], true)
			info.Type2 = subType(types[index:], false)
		}
	}
	return info
}

func (p *TypeInfoAmd64) hasUnalignedField(typ llvm.Type, offset uint64) bool {
	if p.td.TypeAllocSize(typ) == 0 {
		return false
	}
	if offset%uint64(p.Alignof(typ)) != 0 {
		return true
	}
	switch typ.TypeKind() {
	case llvm.StructTypeKind:
		for i, field := range typ.StructElementTypes() {
			if p.hasUnalignedField(field, offset+p.td.ElementOffset(typ, i)) {
				return true
			}
		}
	case llvm.ArrayTypeKind:
		elem := typ.ElementType()
		if p.hasUnalignedField(elem, offset) {
			return true
		}
		// A packed element can start with an aligned field but have a stride
		// that misaligns that same field in subsequent elements.
		return typ.ArrayLength() > 1 && p.hasUnalignedField(elem, offset+p.td.TypeAllocSize(elem))
	}
	return false
}

type TypeInfoArm64 struct {
	*Transformer
	hfaStackAlign bool // Emit AAPCS64's HFA alignstack attribute; omitted on Darwin/Windows.
}

func (p *TypeInfoArm64) SupportByVal() bool {
	return false
}

func (p *TypeInfoArm64) SkipEmptyParams() bool {
	return true
}

func (p *TypeInfoArm64) IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool {
	bret := index == 0
	switch typ.TypeKind() {
	case llvm.StructTypeKind, llvm.ArrayTypeKind:
		if bret && elementTypesCount(typ) == 1 {
			return false
		}
		return true
	default:
		return false
	}
}

func (p *TypeInfoArm64) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	bret := index == 0
	info := &TypeInfo{}
	info.Type = typ
	info.Type1 = typ
	kind := typ.TypeKind()
	if kind == llvm.VoidTypeKind {
		info.Kind = AttrVoid
		return info
	}
	info.Size = p.Sizeof(typ)
	info.Align = p.Alignof(typ)
	switch kind {
	case llvm.StructTypeKind, llvm.ArrayTypeKind:
		n := elementTypesCount(typ)
		if bret && n == 1 {
			return info
		}
		if n <= 4 {
			types := elementTypes(p.td, typ)
			// skip (i64/ptr,i64/ptr)
			if n == 2 && (types[0].TypeKind() == llvm.PointerTypeKind || types[0] == ctx.Int64Type()) &&
				(types[1].TypeKind() == llvm.PointerTypeKind || types[1] == ctx.Int64Type()) {
				return info
			}
			if checkTypes(types, ctx.FloatType()) || checkTypes(types, ctx.DoubleType()) {
				if !bret {
					// LLVM must see one HFA array, not independent struct fields:
					// an exhausted FP bank spills the entire argument together.
					info.Kind = AttrWidthType
					info.Type1 = llvm.ArrayType(types[0], n)
					if p.hfaStackAlign {
						// AAPCS64 C.4: HFA stack alignment is 8, or 16 for
						// naturally 16-byte-aligned aggregates, capped at 16.
						info.StackAlign = 8
						if info.Align >= 16 {
							info.StackAlign = 16
						}
					}
				}
				return info
			}
		}
		if info.Size > 16 {
			info.Kind = AttrPointer
			info.Type1 = llvm.PointerType(typ, 0)
		} else if info.Size <= 8 {
			info.Kind = AttrWidthType
			if bret {
				info.Type1 = ctx.IntType(info.Size * 8)
			} else {
				info.Type1 = ctx.Int64Type()
			}
		} else {
			info.Kind = AttrWidthType
			info.Type1 = llvm.ArrayType(ctx.Int64Type(), 2)
		}
	}

	return info
}

type TypeInfoArm struct {
	*Transformer
}

func (p *TypeInfoArm) SupportByVal() bool {
	return false
}

func (p *TypeInfoArm) SkipEmptyParams() bool {
	return true
}

func (p *TypeInfoArm) IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool {
	switch typ.TypeKind() {
	case llvm.StructTypeKind, llvm.ArrayTypeKind:
		return true
	default:
		return false
	}
}

func (p *TypeInfoArm) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	bret := index == 0
	info := &TypeInfo{}
	info.Type = typ
	info.Type1 = typ
	kind := typ.TypeKind()
	if kind == llvm.VoidTypeKind {
		info.Kind = AttrVoid
		return info
	}
	info.Size = p.Sizeof(typ)
	info.Align = p.Alignof(typ)
	switch kind {
	case llvm.StructTypeKind, llvm.ArrayTypeKind:
		n := elementTypesCount(typ)
		var types []llvm.Type
		if n <= 4 {
			types = elementTypes(p.td, typ)
			if checkTypes(types, ctx.FloatType()) || checkTypes(types, ctx.DoubleType()) {
				return info
			}
		}
		if bret {
			if info.Size > 4 {
				info.Kind = AttrPointer
				info.Type1 = llvm.PointerType(typ, 0)
			} else {
				info.Kind = AttrWidthType
				info.Type1 = ctx.IntType(info.Size * 8)
			}
		} else {
			if info.Size > 64 {
				info.Kind = AttrPointer
				info.Type1 = llvm.PointerType(typ, 0)
			} else {
				if types == nil {
					types = elementTypes(p.td, typ)
				}
				info.Kind = AttrWidthType
				if containsTypeAny(types, ctx.Int64Type(), ctx.DoubleType()) {
					size := (info.Size + 7) &^ 7
					info.Type1 = llvm.ArrayType(ctx.Int64Type(), size/8)
				} else {
					size := (info.Size + 3) &^ 3
					info.Type1 = llvm.ArrayType(ctx.Int32Type(), size/4)
				}
			}
		}
	}

	return info
}

type TypeInfoWasm struct {
	*Transformer
}

func (p *TypeInfoWasm) SupportByVal() bool {
	return true
}

func (p *TypeInfoWasm) SkipEmptyParams() bool {
	return true
}

func (p *TypeInfoWasm) IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool {
	return elementTypesCount(typ) >= 2
}

func (p *TypeInfoWasm) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	info := &TypeInfo{}
	info.Type = typ
	info.Type1 = typ
	if typ.TypeKind() == llvm.VoidTypeKind {
		info.Kind = AttrVoid
		return info
	}
	info.Size = p.Sizeof(typ)
	info.Align = p.Alignof(typ)
	if n := elementTypesCount(typ); n >= 2 {
		info.Kind = AttrPointer
		info.Type1 = llvm.PointerType(typ, 0)
	}
	return info
}

type TypeInfoRiscv64 struct {
	*Transformer
	mabi string
}

func (p *TypeInfoRiscv64) SupportByVal() bool {
	return false
}

func (p *TypeInfoRiscv64) SkipEmptyParams() bool {
	return true
}

func (p *TypeInfoRiscv64) IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool {
	switch typ.TypeKind() {
	case llvm.StructTypeKind, llvm.ArrayTypeKind:
		return true
	}
	return false
}

func (p *TypeInfoRiscv64) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	info := &TypeInfo{}
	info.Type = typ
	info.Type1 = typ
	if typ.TypeKind() == llvm.VoidTypeKind {
		info.Kind = AttrVoid
		return info
	}
	info.Size = p.Sizeof(typ)
	info.Align = p.Alignof(typ)
	switch typ.TypeKind() {
	case llvm.StructTypeKind, llvm.ArrayTypeKind:
		n := elementTypesCount(typ)
		switch n {
		case 1:
			types := elementTypes(p.td, typ)
			if types[0].TypeKind() == llvm.PointerTypeKind || types[0] == ctx.Int64Type() {
				return info
			}
			switch p.mabi {
			case MABI_LP64F, MABI_LP64D:
				if types[0] == ctx.FloatType() {
					return info
				}
			}
		case 2:
			types := elementTypes(p.td, typ)
			if (types[0].TypeKind() == llvm.PointerTypeKind || types[0] == ctx.Int64Type()) &&
				(types[1].TypeKind() == llvm.PointerTypeKind || types[1] == ctx.Int64Type()) {
				return info
			}
			switch p.mabi {
			case MABI_LP64F:
				if containsType(types, ctx.FloatType()) && !containsType(types, ctx.DoubleType()) && !hasTypesKind(types, llvm.PointerTypeKind) {
					info.Kind = AttrWidthType2
					info.Type1 = types[0]
					info.Type2 = types[1]
					return info
				}
			case MABI_LP64D:
				if (containsTypeAny(types, ctx.FloatType(), ctx.DoubleType())) && !hasTypesKind(types, llvm.PointerTypeKind) {
					info.Kind = AttrWidthType2
					info.Type1 = types[0]
					info.Type2 = types[1]
					return info
				}
			}
		}
		if info.Size > 16 {
			info.Kind = AttrPointer
			info.Type1 = llvm.PointerType(typ, 0)
		} else if info.Size <= 8 {
			info.Kind = AttrWidthType
			info.Type1 = ctx.Int64Type()
		} else {
			info.Kind = AttrWidthType
			info.Type1 = llvm.ArrayType(ctx.Int64Type(), 2)
		}
	}
	return info
}

const (
	MABI_LP64  = "lp64"
	MABI_LP64F = "lp64f"
	MABI_LP64D = "lp64d"
)

const (
	MABI_ILP32  = "ilp32"
	MABI_ILP32F = "ilp32f"
	MABI_ILP32D = "ilp32d"
)

type TypeInfoRiscv32 struct {
	*Transformer
	mabi string
}

func (p *TypeInfoRiscv32) SupportByVal() bool {
	return false
}

func (p *TypeInfoRiscv32) SkipEmptyParams() bool {
	return true
}

func (p *TypeInfoRiscv32) IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool {
	switch typ.TypeKind() {
	case llvm.StructTypeKind, llvm.ArrayTypeKind:
		return true
	}
	return false
}

func (p *TypeInfoRiscv32) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	info := &TypeInfo{}
	info.Type = typ
	info.Type1 = typ
	if typ.TypeKind() == llvm.VoidTypeKind {
		info.Kind = AttrVoid
		return info
	}
	info.Size = p.Sizeof(typ)
	info.Align = p.Alignof(typ)
	switch typ.TypeKind() {
	case llvm.StructTypeKind, llvm.ArrayTypeKind:
		n := elementTypesCount(typ)
		switch n {
		case 1:
			types := elementTypes(p.td, typ)
			if types[0] == ctx.Int64Type() {
				return info
			}
			if types[0] == ctx.DoubleType() {
				info.Kind = AttrWidthType
				info.Type1 = ctx.Int64Type()
				return info
			}
		case 2:
			types := elementTypes(p.td, typ)
			switch p.mabi {
			case MABI_ILP32F:
				if info.Align != 8 && containsType(types, ctx.FloatType()) && !hasTypesKind(types, llvm.PointerTypeKind) {
					info.Kind = AttrWidthType2
					info.Type1 = types[0]
					info.Type2 = types[1]
					return info
				}
			case MABI_ILP32D:
				if (containsTypeAny(types, ctx.FloatType(), ctx.DoubleType())) &&
					!hasTypesKind(types, llvm.PointerTypeKind) && !containsType(types, ctx.Int64Type()) {
					info.Kind = AttrWidthType2
					info.Type1 = types[0]
					info.Type2 = types[1]
					return info
				}
			}
		}
		if info.Size > 8 {
			info.Kind = AttrPointer
			info.Type1 = llvm.PointerType(typ, 0)
		} else if info.Size <= 4 {
			info.Kind = AttrWidthType
			info.Type1 = ctx.Int32Type()
		} else {
			info.Kind = AttrWidthType
			info.Type1 = llvm.ArrayType(ctx.Int32Type(), 2)
		}
	}
	return info
}

type TypeInfo386 struct {
	*Transformer
}

func (p *TypeInfo386) SupportByVal() bool {
	return true
}

func (p *TypeInfo386) SkipEmptyParams() bool {
	return false
}

func (p *TypeInfo386) IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool {
	switch typ.TypeKind() {
	case llvm.ArrayTypeKind, llvm.StructTypeKind:
		return true
	}
	return false
}

func (p *TypeInfo386) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	bret := index == 0
	info := &TypeInfo{}
	info.Type = typ
	info.Type1 = typ
	if typ.TypeKind() == llvm.VoidTypeKind {
		info.Kind = AttrVoid
		return info
	}
	info.Size = p.Sizeof(typ)
	info.Align = p.Alignof(typ)
	if info.Size == 0 {
		if bret {
			return info
		}
		if index == 1 {
			info.Kind = AttrPointer
			info.Type1 = llvm.PointerType(typ, 0)
			return info
		}
		info.Kind = AttrVoid
		info.Type1 = ctx.VoidType()
		return info
	}
	switch typ.TypeKind() {
	case llvm.StructTypeKind:
		if !bret && info.Size <= 16 {
			var extract bool
			subs := typ.StructElementTypes()
		loop:
			for _, sub := range subs {
				switch sub.TypeKind() {
				case llvm.FloatTypeKind, llvm.DoubleTypeKind, llvm.PointerTypeKind:
					extract = true
				case llvm.IntegerTypeKind:
					if width := sub.IntTypeWidth(); width == 32 || width == 64 {
						extract = true
					} else {
						extract = false
						break loop
					}
				default:
					extract = false
					break loop
				}
			}
			if extract {
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
		info.Type1 = llvm.PointerType(typ, 0)
	case llvm.ArrayTypeKind:
		info.Kind = AttrPointer
		info.Type1 = llvm.PointerType(typ, 0)
	}
	return info
}

type TypeInfoEsp32 struct {
	*Transformer
}

func (p *TypeInfoEsp32) SupportByVal() bool {
	return true
}

func (p *TypeInfoEsp32) SkipEmptyParams() bool {
	return false
}

func (p *TypeInfoEsp32) IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool {
	switch typ.TypeKind() {
	case llvm.ArrayTypeKind, llvm.StructTypeKind:
		return true
	}
	return false
}

func (p *TypeInfoEsp32) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	bret := index == 0
	info := &TypeInfo{}
	info.Type = typ
	info.Type1 = typ
	if typ.TypeKind() == llvm.VoidTypeKind {
		info.Kind = AttrVoid
		return info
	}
	info.Size = p.Sizeof(typ)
	info.Align = p.Alignof(typ)
	if info.Size == 0 {
		if bret {
			return info
		}
		if index == 1 {
			info.Kind = AttrPointer
			info.Type1 = llvm.PointerType(typ, 0)
			return info
		}
		info.Kind = AttrVoid
		info.Type1 = ctx.VoidType()
		return info
	}
	switch typ.TypeKind() {
	case llvm.StructTypeKind, llvm.ArrayTypeKind:
		if (bret && (info.Size > 16 ||
			(info.Size <= 4 && info.Align < 4) || (info.Size < 8 && info.Align >= 4))) ||
			(info.Size > 24) {
			info.Kind = AttrPointer
			info.Type1 = llvm.PointerType(typ, 0)
			return info
		}
		info.Kind = AttrWidthType
		types := elementTypes(p.td, typ)
		if containsTypeAny(types, ctx.Int64Type(), ctx.DoubleType()) {
			size := (info.Size + 7) &^ 7
			info.Type1 = llvm.ArrayType(ctx.Int64Type(), size/8)
		} else {
			size := (info.Size + 3) &^ 3
			info.Type1 = llvm.ArrayType(ctx.Int32Type(), size/4)
		}
	}
	return info
}
