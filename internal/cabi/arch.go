package cabi

import (
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
		sub := elementTypes(td, typ.ElementType())
		n := typ.ArrayLength()
		for i := 0; i < n; i++ {
			types = append(types, sub...)
		}
	default:
		types = append(types, typ)
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

func hasTypes(typs []llvm.Type, typ llvm.Type) bool {
	for _, t := range typs {
		if t == typ {
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
				n = (n + info.Align - 1) &^ (info.Align - 1)
				return ctx.IntType(n * 8)
			}
			info.Kind = AttrWidthType2
			info.Type1 = subType(types[0:index], true)
			info.Type2 = subType(types[index:], false)
		}
	}
	return info
}

type TypeInfoArm64 struct {
	*Transformer
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
		types := elementTypes(p.td, typ)
		n := len(types)
		if bret && n == 1 {
			return info
		}
		if n == 2 {
			// skip (i64/ptr,i64/ptr)
			if (types[0].TypeKind() == llvm.PointerTypeKind || types[0] == ctx.Int64Type()) &&
				(types[1].TypeKind() == llvm.PointerTypeKind || types[1] == ctx.Int64Type()) {
				return info
			}
		}
		if n <= 4 {
			if checkTypes(types, ctx.FloatType()) || checkTypes(types, ctx.DoubleType()) {
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
		types := elementTypes(p.td, typ)
		n := len(types)
		if n <= 4 {
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
				info.Kind = AttrWidthType
				if hasTypes(types, ctx.Int64Type()) || hasTypes(types, ctx.DoubleType()) {
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
		types := elementTypes(p.td, typ)
		switch len(types) {
		case 1:
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
			if (types[0].TypeKind() == llvm.PointerTypeKind || types[0] == ctx.Int64Type()) &&
				(types[1].TypeKind() == llvm.PointerTypeKind || types[1] == ctx.Int64Type()) {
				return info
			}
			switch p.mabi {
			case MABI_LP64F:
				if hasTypes(types, ctx.FloatType()) && !hasTypes(types, ctx.DoubleType()) && !hasTypesKind(types, llvm.PointerTypeKind) {
					info.Kind = AttrWidthType2
					info.Type1 = types[0]
					info.Type2 = types[1]
					return info
				}
			case MABI_LP64D:
				if (hasTypes(types, ctx.FloatType()) || hasTypes(types, ctx.DoubleType())) && !hasTypesKind(types, llvm.PointerTypeKind) {
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
		types := elementTypes(p.td, typ)
		switch len(types) {
		case 1:
			if types[0] == ctx.Int64Type() {
				return info
			}
			if types[0] == ctx.DoubleType() {
				info.Kind = AttrWidthType
				info.Type1 = ctx.Int64Type()
				return info
			}
		case 2:
			switch p.mabi {
			case MABI_ILP32F:
				if info.Align != 8 && hasTypes(types, ctx.FloatType()) && !hasTypesKind(types, llvm.PointerTypeKind) {
					info.Kind = AttrWidthType2
					info.Type1 = types[0]
					info.Type2 = types[1]
					return info
				}
			case MABI_ILP32D:
				if (hasTypes(types, ctx.FloatType()) || hasTypes(types, ctx.DoubleType())) &&
					!hasTypesKind(types, llvm.PointerTypeKind) && !hasTypes(types, ctx.Int64Type()) {
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
		types := elementTypes(p.td, typ)
		if (bret && (info.Size > 16 ||
			(info.Size <= 4 && info.Align < 4) || (info.Size < 8 && info.Align >= 4))) ||
			(info.Size > 24) {
			info.Kind = AttrPointer
			info.Type1 = llvm.PointerType(typ, 0)
			return info
		}
		info.Kind = AttrWidthType
		if hasTypes(types, ctx.Int64Type()) || hasTypes(types, ctx.DoubleType()) {
			size := (info.Size + 7) &^ 7
			info.Type1 = llvm.ArrayType(ctx.Int64Type(), size/8)
		} else {
			size := (info.Size + 3) &^ 3
			info.Type1 = llvm.ArrayType(ctx.Int32Type(), size/4)
		}
	}
	return info
}
