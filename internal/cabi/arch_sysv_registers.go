package cabi

import "github.com/xgo-dev/llvm"

func setSysVMemory(info *TypeInfo, isParam bool) {
	info.Kind = AttrPointer
	info.Type1 = llvm.PointerType(info.Type, 0)
	info.Type2 = llvm.Type{}
	// ByValAlign describes arguments only; results use sret.
	info.ByValAlign = 0
	if isParam {
		info.ByValAlign = max(8, info.Align)
	}
}

// SysV allocates an aggregate's integer/SSE classes atomically. When either
// register bank lacks space, the whole value is passed in memory and neither
// bank is consumed. Letting LLVM spill independent coercions does not preserve
// the C aggregate's stack layout or this register rollback rule.
func (p *Transformer) allocateSysVAggregateRegisters(info *FuncInfo, getAttr func(int, uint) llvm.Attribute) {
	integer, floating := 6, 8
	if info.Return.Kind == AttrPointer {
		integer-- // Hidden return storage is the first integer argument.
	}
	for index, param := range info.Params {
		if getAttr != nil {
			hidden := false
			for _, kind := range closureEnvAttributeKinds {
				hidden = hidden || !getAttr(index+1, kind).IsNil()
			}
			if hidden {
				continue // Closure environments use a dedicated register, such as R10.
			}
		}
		if param.Kind == AttrPointer || param.Kind == AttrVoid {
			continue // Memory aggregates and elided empty values consume no registers.
		}
		gp, fp := sysVRegisterClasses(param.Type1)
		if param.Kind == AttrWidthType2 {
			gp2, fp2 := sysVRegisterClasses(param.Type2)
			gp, fp = gp+gp2, fp+fp2
		}
		if gp > integer || fp > floating {
			if kind := param.Type.TypeKind(); kind == llvm.StructTypeKind || kind == llvm.ArrayTypeKind {
				setSysVMemory(param, true)
			}
			continue
		}
		integer, floating = integer-gp, floating-fp
	}
}

func sysVRegisterClasses(typ llvm.Type) (integer, floating int) {
	// Count the existing classifier's lowered Go/C types: integer eightbytes,
	// pointers, float/double and <2 x float>, including single-field wrappers.
	// Extended floating-point ABIs are outside the current Go/C type mapping.
	switch typ.TypeKind() {
	case llvm.IntegerTypeKind:
		return (typ.IntTypeWidth() + 63) / 64, 0
	case llvm.PointerTypeKind:
		return 1, 0
	case llvm.FloatTypeKind, llvm.DoubleTypeKind, llvm.VectorTypeKind:
		return 0, 1
	case llvm.StructTypeKind:
		for _, element := range typ.StructElementTypes() {
			gp, fp := sysVRegisterClasses(element)
			integer, floating = integer+gp, floating+fp
		}
	case llvm.ArrayTypeKind:
		gp, fp := sysVRegisterClasses(typ.ElementType())
		integer, floating = gp*typ.ArrayLength(), fp*typ.ArrayLength()
	default:
		panic("cabi: unsupported SysV register class: " + typ.String())
	}
	return
}
