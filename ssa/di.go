package ssa

import (
	"debug/dwarf"
	"fmt"
	"go/token"
	"go/types"
	"path/filepath"
	"unsafe"

	"github.com/xgo-dev/llvm"
)

type Positioner interface {
	Position(pos token.Pos) token.Position
}

type aDIBuilder struct {
	di         *llvm.DIBuilder
	prog       Program
	types      map[Type]DIType
	positioner Positioner
	m          llvm.Module // Add this field
}

type diBuilder = *aDIBuilder

func newDIBuilder(prog Program, pkg Package, positioner Positioner) diBuilder {
	m := pkg.mod
	ctx := m.Context()

	b := &aDIBuilder{
		di:         llvm.NewDIBuilder(m),
		prog:       prog,
		types:      make(map[*aType]DIType),
		positioner: positioner,
		m:          m, // Initialize the m field
	}

	b.addNamedMetadataOperand("llvm.module.flags", 2, "Debug Info Version", 3)
	b.addNamedMetadataOperand("llvm.module.flags", 7, "Dwarf Version", 4)
	b.addNamedMetadataOperand("llvm.module.flags", 1, "wchar_size", 4)
	b.addNamedMetadataOperand("llvm.module.flags", 8, "PIC Level", 2)
	b.addNamedMetadataOperand("llvm.module.flags", 7, "uwtable", 1)
	b.addNamedMetadataOperand("llvm.module.flags", 7, "frame-pointer", 1)

	// Add llvm.ident metadata
	identNode := ctx.MDNode([]llvm.Metadata{
		ctx.MDString("LLGo Compiler"),
	})
	m.AddNamedMetadataOperand("llvm.ident", identNode)

	return b
}

func hasTypeParam(typ types.Type) bool {
	visited := make(map[types.Type]bool)
	var visit func(types.Type) bool
	visit = func(tt types.Type) bool {
		if tt == nil {
			return false
		}
		if visited[tt] {
			return false
		}
		visited[tt] = true
		switch t := tt.(type) {
		case *types.TypeParam:
			return true
		case *types.Named:
			if tp := t.TypeParams(); tp != nil && tp.Len() > 0 {
				if ta := t.TypeArgs(); ta == nil || ta.Len() == 0 {
					return true
				}
			}
			if ta := t.TypeArgs(); ta != nil {
				for i := 0; i < ta.Len(); i++ {
					if visit(ta.At(i)) {
						return true
					}
				}
			}
			return visit(t.Underlying())
		case *types.Pointer:
			return visit(t.Elem())
		case *types.Slice:
			return visit(t.Elem())
		case *types.Array:
			return visit(t.Elem())
		case *types.Map:
			return visit(t.Key()) || visit(t.Elem())
		case *types.Chan:
			return visit(t.Elem())
		case *types.Signature:
			if tp := t.TypeParams(); tp != nil && tp.Len() > 0 {
				return true
			}
			if params := t.Params(); params != nil {
				for i := 0; i < params.Len(); i++ {
					if visit(params.At(i).Type()) {
						return true
					}
				}
			}
			if results := t.Results(); results != nil {
				for i := 0; i < results.Len(); i++ {
					if visit(results.At(i).Type()) {
						return true
					}
				}
			}
			return false
		case *types.Tuple:
			for i := 0; i < t.Len(); i++ {
				if visit(t.At(i).Type()) {
					return true
				}
			}
			return false
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				if visit(t.Field(i).Type()) {
					return true
				}
			}
			return false
		case *types.Interface:
			for i := 0; i < t.NumMethods(); i++ {
				if visit(t.Method(i).Type()) {
					return true
				}
			}
			for i := 0; i < t.NumEmbeddeds(); i++ {
				if visit(t.EmbeddedType(i)) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return visit(typ)
}

// New method to add named metadata operand
func (b diBuilder) addNamedMetadataOperand(name string, intValue int, stringValue string, intValue2 int) {
	ctx := b.m.Context()
	b.m.AddNamedMetadataOperand(name,
		ctx.MDNode([]llvm.Metadata{
			llvm.ConstInt(ctx.Int32Type(), uint64(intValue), false).ConstantAsMetadata(),
			ctx.MDString(stringValue),
			llvm.ConstInt(ctx.Int32Type(), uint64(intValue2), false).ConstantAsMetadata(),
		}),
	)
}

// ----------------------------------------------------------------------------

type aCompilationUnit struct {
	ll llvm.Metadata
}

type CompilationUnit = *aCompilationUnit

func (c CompilationUnit) scopeMeta(b diBuilder, pos token.Position) DIScopeMeta {
	return &aDIScopeMeta{c.ll}
}

var DWARF_LANG_C llvm.DwarfLang = 0x2
var DWARF_LANG_GO llvm.DwarfLang = 0x16

func (b diBuilder) createCompileUnit(filename, dir string) CompilationUnit {
	return &aCompilationUnit{ll: b.di.CreateCompileUnit(llvm.DICompileUnit{
		// TODO(lijie): use C language for now, change after Go plugin of LLDB is ready
		Language:       DWARF_LANG_C - 1,
		File:           filename,
		Dir:            dir,
		Producer:       "LLGo",
		Optimized:      true,
		RuntimeVersion: 1,
	})}
}

// ----------------------------------------------------------------------------

type aDIScopeMeta struct {
	ll llvm.Metadata
}

type DIScopeMeta = *aDIScopeMeta

type DIScope interface {
	scopeMeta(b diBuilder, pos token.Position) DIScopeMeta
}

// ----------------------------------------------------------------------------

type aDIFile struct {
	ll llvm.Metadata
}

type DIFile = *aDIFile

func (b diBuilder) createFile(filename string) DIFile {
	dir, file := filepath.Split(filename)
	return &aDIFile{ll: b.di.CreateFile(file, dir)}
}

func (f DIFile) scopeMeta(b diBuilder, pos token.Position) DIScopeMeta {
	return &aDIScopeMeta{b.file(pos.Filename).ll}
}

// ----------------------------------------------------------------------------

type aDIType struct {
	ll llvm.Metadata
}

type DIType = *aDIType

func (b diBuilder) createType(name string, ty Type, pos token.Position) DIType {
	var typ llvm.Metadata
	switch t := ty.RawType().(type) {
	case *types.Basic:
		if t.Kind() == types.UnsafePointer {
			typ = b.di.CreatePointerType(llvm.DIPointerType{
				Name:        name,
				SizeInBits:  b.prog.SizeOf(b.prog.rawType(t)) * 8,
				AlignInBits: uint32(b.prog.sizes.Alignof(t) * 8),
			})
			return &aDIType{typ}
		}

		var encoding llvm.DwarfTypeEncoding
		if t.Info()&types.IsBoolean != 0 {
			encoding = llvm.DW_ATE_boolean
		} else if t.Info()&types.IsUnsigned != 0 {
			encoding = llvm.DW_ATE_unsigned
		} else if t.Info()&types.IsInteger != 0 {
			encoding = llvm.DW_ATE_signed
		} else if t.Info()&types.IsFloat != 0 {
			encoding = llvm.DW_ATE_float
		} else if t.Info()&types.IsComplex != 0 {
			return b.createComplexType(ty)
		} else if t.Info()&types.IsString != 0 {
			return b.createStringType()
		} else {
			panic(fmt.Errorf("can't create debug info of basic type: %v, %T", ty.RawType(), ty.RawType()))
		}

		typ = b.di.CreateBasicType(llvm.DIBasicType{
			Name:       name,
			SizeInBits: b.prog.SizeOf(b.prog.rawType(t)) * 8,
			Encoding:   encoding,
		})
	case *types.Pointer:
		return b.createPointerType(name, b.prog.rawType(t.Elem()), pos)
	case *types.Named:
		// Create typedef type for named types
		return b.createTypedefType(name, ty, pos)
	case *types.Interface:
		ty := b.prog.rtType("Iface")
		return b.createInterfaceType(name, ty)
	case *types.Slice:
		ty := b.prog.rtType("Slice")
		tyElem := b.prog.rawType(t.Elem())
		return b.createSliceType(name, ty, tyElem)
	case *types.Struct:
		return b.createStructType(name, ty, pos)
	case *types.Signature:
		tyFn := b.prog.Closure(t)
		return b.createFuncPtrType(name, tyFn, pos)
	case *types.Array:
		return b.createArrayType(ty, t.Len())
	case *types.Chan:
		return b.createChanType(name, ty, pos)
	case *types.Map:
		ty := b.prog.rtType("Map")
		return b.createMapType(name, ty, pos)
	case *types.Tuple:
		return b.createTupleType(name, ty, pos)
	default:
		panic(fmt.Errorf("can't create debug info of type: %v, %T", ty.RawType(), ty.RawType()))
	}
	return &aDIType{typ}
}

// ----------------------------------------------------------------------------

type aDIFunction struct {
	ll llvm.Metadata
}

type DIFunction = *aDIFunction

func (p Function) scopeMeta(b diBuilder, pos token.Position) DIScopeMeta {
	return &aDIScopeMeta{p.diFunc.ll}
}

// ----------------------------------------------------------------------------

type aDIGlobalVariableExpression struct {
	ll llvm.Metadata
}

type DIGlobalVariableExpression = *aDIGlobalVariableExpression

func (b diBuilder) createGlobalVariableExpression(scope DIScope, pos token.Position, name, linkageName string, ty Type, isLocalToUnit bool) DIGlobalVariableExpression {
	return &aDIGlobalVariableExpression{
		ll: b.di.CreateGlobalVariableExpression(
			scope.scopeMeta(b, pos).ll,
			llvm.DIGlobalVariableExpression{
				Name:        name,
				LinkageName: linkageName,
				File:        b.file(pos.Filename).ll,
				Line:        pos.Line,
				Type:        b.diType(ty, pos).ll,
				LocalToUnit: isLocalToUnit,
				AlignInBits: uint32(b.prog.sizes.Alignof(ty.RawType()) * 8),
			},
		),
	}
}

// ----------------------------------------------------------------------------

type aDILexicalBlock struct {
	ll llvm.Metadata
}

type DILexicalBlock = *aDILexicalBlock

func (l *aDILexicalBlock) scopeMeta(b diBuilder, pos token.Position) DIScopeMeta {
	return &aDIScopeMeta{l.ll}
}

// ----------------------------------------------------------------------------

type aDIVar struct {
	ll llvm.Metadata
}

type DIVar = *aDIVar

func (b diBuilder) createParameterVariable(scope DIScope, pos token.Position, name string, argNo int, ty DIType) DIVar {
	return &aDIVar{
		ll: b.di.CreateParameterVariable(
			scope.scopeMeta(b, pos).ll,
			llvm.DIParameterVariable{
				Name:           name,
				File:           b.file(pos.Filename).ll,
				Line:           pos.Line,
				ArgNo:          argNo,
				Type:           ty.ll,
				AlwaysPreserve: true,
			},
		),
	}
}

func (b diBuilder) createAutoVariable(scope DIScope, pos token.Position, name string, ty DIType) DIVar {
	return &aDIVar{
		ll: b.di.CreateAutoVariable(
			scope.scopeMeta(b, pos).ll,
			llvm.DIAutoVariable{
				Name:           name,
				File:           b.file(pos.Filename).ll,
				Line:           pos.Line,
				Type:           ty.ll,
				AlwaysPreserve: true,
			},
		),
	}
}

func (b diBuilder) createTypedefType(name string, ty Type, pos token.Position) DIType {
	underlyingType := b.diType(b.prog.rawType(ty.RawType().(*types.Named).Underlying()), pos)
	typ := b.di.CreateTypedef(llvm.DITypedef{
		Name:        name,
		Type:        underlyingType.ll,
		File:        b.file(pos.Filename).ll,
		Line:        pos.Line,
		AlignInBits: uint32(b.prog.sizes.Alignof(ty.RawType()) * 8),
	})
	return &aDIType{typ}
}

func (b diBuilder) createStringType() DIType {
	ty := b.prog.rtType("String")
	return b.doCreateStructType("string", ty, token.Position{}, func(ditStruct DIType) []llvm.Metadata {
		return []llvm.Metadata{
			b.createMemberType("data", ty, b.prog.CStr(), 0),
			b.createMemberType("len", ty, b.prog.Uint(), 1),
		}
	})
}

func (b diBuilder) createArrayType(ty Type, l int64) DIType {
	tyElem := b.prog.rawType(ty.RawType().(*types.Array).Elem())
	return &aDIType{ll: b.di.CreateArrayType(llvm.DIArrayType{
		SizeInBits:  b.prog.SizeOf(ty) * 8,
		AlignInBits: uint32(b.prog.sizes.Alignof(ty.RawType()) * 8),
		ElementType: b.diType(tyElem, token.Position{}).ll,
		Subscripts: []llvm.DISubrange{{
			Count: l,
		}},
	})}
}

func (b diBuilder) createSliceType(name string, ty, tyElem Type) DIType {
	pos := token.Position{}
	diElemTyPtr := b.prog.Pointer(tyElem)

	return b.doCreateStructType(name, ty, pos, func(ditStruct DIType) []llvm.Metadata {
		return []llvm.Metadata{
			b.createMemberTypeEx("data", ty, diElemTyPtr, 0, pos, 0),
			b.createMemberTypeEx("len", ty, b.prog.Uint(), 1, pos, 0),
			b.createMemberTypeEx("cap", ty, b.prog.Uint(), 2, pos, 0),
		}
	})
}

func (b diBuilder) createInterfaceType(name string, ty Type) DIType {
	tyRaw := ty.RawType().Underlying()
	tyIntr := b.prog.rawType(tyRaw)
	tyType := b.prog.VoidPtr()
	tyData := b.prog.VoidPtr()

	return b.doCreateStructType(name, tyIntr, token.Position{}, func(ditStruct DIType) []llvm.Metadata {
		return []llvm.Metadata{
			b.createMemberType("type", ty, tyType, 0),
			b.createMemberType("data", ty, tyData, 1),
		}
	})
}

func (b diBuilder) createMemberType(name string, tyStruct, tyField Type, idxField int) llvm.Metadata {
	return b.createMemberTypeEx(name, tyStruct, tyField, idxField, token.Position{}, 0)
}

func (b diBuilder) createMemberTypeEx(name string, tyStruct, tyField Type, idxField int, pos token.Position, flags int) llvm.Metadata {
	return b.di.CreateMemberType(
		b.diType(tyStruct, pos).ll,
		llvm.DIMemberType{
			Name:         name,
			SizeInBits:   b.prog.SizeOf(tyField) * 8,
			AlignInBits:  uint32(b.prog.sizes.Alignof(tyField.RawType()) * 8),
			OffsetInBits: b.prog.OffsetOf(tyStruct, idxField) * 8,
			Type:         b.diType(tyField, pos).ll,
			Flags:        flags,
		},
	)
}

func (b diBuilder) createMapType(name string, tyMap Type, pos token.Position) DIType {
	// ty := tyMap.RawType().(*types.Map)
	// tk := b.prog.rawType(ty.Key())
	// tv := b.prog.rawType(ty.Elem())
	tyCount := b.prog.Int()
	return b.doCreateStructType(name, tyMap, pos, func(ditStruct DIType) []llvm.Metadata {
		return []llvm.Metadata{
			b.createMemberType("count", tyMap, tyCount, 0),
		}
	})
}

func (b diBuilder) createChanType(name string, t Type, pos token.Position) DIType {
	return b.doCreateStructType(name, t, pos, func(ditStruct DIType) []llvm.Metadata {
		return []llvm.Metadata{}
	})
}

func (b diBuilder) createComplexType(t Type) DIType {
	var tfield Type
	var tyName string
	if t.RawType().(*types.Basic).Kind() == types.Complex128 {
		tfield = b.prog.Float64()
		tyName = "complex128"
	} else {
		tfield = b.prog.Float32()
		tyName = "complex64"
	}
	return b.doCreateStructType(tyName, t, token.Position{}, func(ditStruct DIType) []llvm.Metadata {
		return []llvm.Metadata{
			b.createMemberType("real", t, tfield, 0),
			b.createMemberType("imag", t, tfield, 1),
		}
	})
}

func (b diBuilder) createPointerType(name string, ty Type, pos token.Position) DIType {
	ptrType := b.prog.VoidPtr()
	return &aDIType{ll: b.di.CreatePointerType(llvm.DIPointerType{
		Name:        name,
		Pointee:     b.diType(ty, pos).ll,
		SizeInBits:  b.prog.SizeOf(ptrType) * 8,
		AlignInBits: uint32(b.prog.sizes.Alignof(ptrType.RawType())) * 8,
	})}
}

func (b diBuilder) doCreateStructType(name string, ty Type, pos token.Position, fn func(ty DIType) []llvm.Metadata) (ret DIType) {
	structType := ty.RawType().Underlying()

	scope := b.file(pos.Filename)
	ret = &aDIType{b.di.CreateReplaceableCompositeType(
		scope.ll,
		llvm.DIReplaceableCompositeType{
			Tag:         dwarf.TagStructType,
			Name:        name,
			File:        b.file(pos.Filename).ll,
			Line:        pos.Line,
			SizeInBits:  b.prog.SizeOf(ty) * 8,
			AlignInBits: uint32(b.prog.sizes.Alignof(structType) * 8),
		},
	)}
	b.types[ty] = ret

	fields := fn(ret)

	st := b.di.CreateStructType(
		scope.ll,
		llvm.DIStructType{
			Name:        name,
			File:        b.file(pos.Filename).ll,
			Line:        pos.Line,
			SizeInBits:  b.prog.SizeOf(ty) * 8,
			AlignInBits: uint32(b.prog.sizes.Alignof(structType) * 8),
			Elements:    fields,
		},
	)
	ret.ll.ReplaceAllUsesWith(st)
	ret.ll = st
	return
}

func (b diBuilder) createStructType(name string, ty Type, pos token.Position) (ret DIType) {
	structType := ty.RawType().(*types.Struct)
	return b.doCreateStructType(name, ty, pos, func(ditStruct DIType) []llvm.Metadata {
		fields := make([]llvm.Metadata, structType.NumFields())
		for i := 0; i < structType.NumFields(); i++ {
			field := structType.Field(i)
			tyField := b.prog.rawType(field.Type())
			flags := 0
			pos := b.positioner.Position(field.Pos())
			fields[i] = b.createMemberTypeEx(field.Name(), ty, tyField, i, pos, flags)
		}
		return fields
	})
}

func (b diBuilder) createTupleType(name string, ty Type, pos token.Position) DIType {
	tupleType := ty.RawType().(*types.Tuple)
	if tupleType.Len() == 0 {
		return &aDIType{}
	}
	if tupleType.Len() == 1 {
		t := b.prog.rawType(tupleType.At(0).Type())
		return b.diType(t, pos)
	}
	return b.doCreateStructType(name, ty, pos, func(ditStruct DIType) []llvm.Metadata {
		fields := make([]llvm.Metadata, ty.RawType().(*types.Tuple).Len())
		for i := 0; i < ty.RawType().(*types.Tuple).Len(); i++ {
			field := ty.RawType().(*types.Tuple).At(i)
			tyField := b.prog.rawType(field.Type())
			fields[i] = b.createMemberTypeEx(field.Name(), ty, tyField, i, pos, 0)
		}
		return fields
	})
}

func (b diBuilder) createFuncPtrType(name string, ty Type, pos token.Position) DIType {
	ptr := b.prog.VoidPtr()
	return &aDIType{ll: b.di.CreatePointerType(llvm.DIPointerType{
		Name:        name,
		Pointee:     b.diType(ptr, pos).ll,
		SizeInBits:  b.prog.SizeOf(ptr) * 8,
		AlignInBits: uint32(b.prog.sizes.Alignof(ptr.RawType()) * 8),
	})}
}

// ----------------------------------------------------------------------------

func (b diBuilder) dbgDeclare(v Expr, dv DIVar, scope DIScope, pos token.Position, expr DIExpression, blk BasicBlock) {
	loc := llvm.DebugLoc{
		Line:  uint(pos.Line),
		Col:   uint(pos.Column),
		Scope: scope.scopeMeta(b, pos).ll,
	}
	b.di.InsertDeclareAtEnd(
		v.impl,
		dv.ll,
		expr.ll,
		loc,
		blk.last,
	)
}

func (b diBuilder) dbgValue(v Expr, dv DIVar, scope DIScope, pos token.Position, expr DIExpression, blk BasicBlock) {
	loc := llvm.DebugLoc{
		Line:  uint(pos.Line),
		Col:   uint(pos.Column),
		Scope: scope.scopeMeta(b, pos).ll,
	}
	b.di.InsertValueAtEnd(
		v.impl,
		dv.ll,
		expr.ll,
		loc,
		blk.last,
	)
}

func (b diBuilder) diType(t Type, pos token.Position) DIType {
	if hasTypeParam(t.RawType()) {
		return &aDIType{}
	}
	name := t.RawType().String()
	return b.diTypeEx(name, t, pos)
}

func (b diBuilder) diTypeEx(name string, t Type, pos token.Position) DIType {
	if ty, ok := b.types[t]; ok {
		return ty
	}
	ty := b.createType(name, t, pos)
	b.types[t] = ty
	return ty
}

func (b diBuilder) varParam(scope DIScope, pos token.Position, varName string, vt DIType, argNo int) DIVar {
	return b.createParameterVariable(
		scope,
		pos,
		varName,
		argNo,
		vt,
	)
}

func (b diBuilder) varAuto(scope DIScope, pos token.Position, varName string, vt DIType) DIVar {
	return b.createAutoVariable(scope, pos, varName, vt)
}

func (b diBuilder) file(filename string) DIFile {
	return b.createFile(filename)
}

// ----------------------------------------------------------------------------

type aDIExpression struct {
	ll llvm.Metadata
}

type DIExpression = *aDIExpression

func (b diBuilder) createExpression(ops []uint64) DIExpression {
	return &aDIExpression{b.di.CreateExpression(ops)}
}

// -----------------------------------------------------------------------------

// Copy to alloca'd memory to get declareable address.
func (b Builder) constructDebugAddr(v Expr) (dbgPtr Expr, dbgVal Expr, exists bool) {
	t := v.Type.RawType().Underlying()
	dbgPtr, dbgVal = b.doConstructDebugAddr(v, t)
	return dbgPtr, dbgVal, false
}

func (b Builder) doConstructDebugAddr(v Expr, t types.Type) (dbgPtr Expr, dbgVal Expr) {
	var ty Type
	switch t := t.(type) {
	case *types.Basic:
		if t.Info()&types.IsComplex != 0 {
			if t.Kind() == types.Complex128 {
				ty = b.Prog.Complex128()
			} else {
				ty = b.Prog.Complex64()
			}
		} else if t.Info()&types.IsString != 0 {
			ty = b.Prog.rtType("String")
		} else {
			ty = v.Type
		}
	case *types.Struct:
		ty = v.Type
	case *types.Slice:
		ty = b.Prog.Type(b.Prog.rtType("Slice").RawType().Underlying(), InGo)
	case *types.Signature:
		ty = b.Prog.Closure(t)
	case *types.Named:
		ty = b.Prog.Type(t.Underlying(), InGo)
	case *types.Map:
		ty = b.Prog.Type(b.Prog.rtType("Map").RawType().Underlying(), InGo)
	default:
		ty = v.Type
	}
	dbgPtr = b.AllocaT(ty)
	dbgPtr.Type = b.Prog.Pointer(v.Type)
	b.Store(dbgPtr, v)
	dbgVal = b.Load(dbgPtr)
	return dbgPtr, dbgVal
}

func (b Builder) di() diBuilder {
	return b.Pkg.di
}

func (b Builder) DIParam(variable *types.Var, v Expr, dv DIVar, scope DIScope, pos token.Position, blk BasicBlock) {
	b.DIValue(variable, v, dv, scope, pos, blk)
}

func (b Builder) DIDeclare(variable *types.Var, v Expr, dv DIVar, scope DIScope, pos token.Position, blk BasicBlock) {
	expr := b.di().createExpression(nil)
	b.di().dbgDeclare(v, dv, scope, pos, expr, blk)
}

func (b Builder) DIValue(variable *types.Var, v Expr, dv DIVar, scope DIScope, pos token.Position, blk BasicBlock) {
	ty := v.Type.RawType().Underlying()
	if !needConstructAddr(ty) {
		expr := b.di().createExpression(nil)
		b.di().dbgValue(v, dv, scope, pos, expr, blk)
	} else {
		dbgPtr, _, _ := b.constructDebugAddr(v)
		expr := b.di().createExpression([]uint64{opDeref})
		b.di().dbgValue(dbgPtr, dv, scope, pos, expr, blk)
	}
}

const (
	opDeref = 0x06
)

func needConstructAddr(t types.Type) bool {
	switch t := t.(type) {
	case *types.Basic:
		if t.Info()&types.IsComplex != 0 {
			return true
		} else if t.Info()&types.IsString != 0 {
			return true
		}
		return false
	case *types.Pointer:
		return false
	default:
		return true
	}
}

func (b Builder) DIVarParam(scope DIScope, pos token.Position, varName string, vt Type, argNo int) DIVar {
	t := b.di().diType(vt, pos)
	return b.di().varParam(scope, pos, varName, t, argNo)
}

func (b Builder) DIVarAuto(scope DIScope, pos token.Position, varName string, vt Type) DIVar {
	t := b.di().diType(vt, pos)
	return b.di().varAuto(scope, pos, varName, t)
}

// hack for types.Scope
type hackScope struct {
	parent   *types.Scope
	children []*types.Scope
	number   int                     // parent.children[number-1] is this scope; 0 if there is no parent
	elems    map[string]types.Object // lazily allocated
	pos, end token.Pos               // scope extent; may be invalid
	comment  string                  // for debugging only
	isFunc   bool                    // set if this is a function scope (internal use only)
}

func isFunc(scope *types.Scope) bool {
	hs := (*hackScope)(unsafe.Pointer(scope))
	return hs.isFunc
}

func (b Builder) DIScope(f Function, scope *types.Scope) DIScope {
	if cachedScope, ok := b.diScopeCache[scope]; ok {
		return cachedScope
	}
	pos := b.di().positioner.Position(scope.Pos())
	// skip package and universe scope
	// if scope.Parent().Parent() == nil {
	// 	return b.di().file(pos.Filename)
	// }

	var result DIScope
	if isFunc(scope) {
		// TODO(lijie): should check scope == function scope
		result = f
	} else {
		parentScope := b.DIScope(f, scope.Parent())
		result = &aDILexicalBlock{b.di().di.CreateLexicalBlock(parentScope.scopeMeta(b.di(), pos).ll, llvm.DILexicalBlock{
			File:   b.di().file(pos.Filename).ll,
			Line:   pos.Line,
			Column: pos.Column,
		})}
	}

	b.diScopeCache[scope] = result
	return result
}

const (
	MD_dbg = 0
)

func (b Builder) DIGlobal(v Expr, name string, pos token.Position) {
	if _, ok := b.Pkg.glbDbgVars[v]; ok {
		return
	}
	gv := b.di().createGlobalVariableExpression(
		b.Pkg.cu,
		pos,
		name,
		name,
		v.Type,
		false,
	)
	v.impl.AddMetadata(MD_dbg, gv.ll)
	b.Pkg.glbDbgVars[v] = true
}

func (b Builder) DISetCurrentDebugLocation(diScope DIScope, pos token.Position) {
	b.impl.SetCurrentDebugLocation(
		uint(pos.Line),
		uint(pos.Column),
		diScope.scopeMeta(b.di(), pos).ll,
		llvm.Metadata{},
	)
}

func (b Builder) DebugFunction(f Function, pos token.Position, bodyPos token.Position) {
	p := f
	if p.diFunc == nil {
		sig := p.Type.raw.Type.(*types.Signature)
		rt := p.Prog.Type(sig.Results(), InGo)
		paramTypes := make([]llvm.Metadata, len(p.params)+1)
		paramTypes[0] = b.di().diType(rt, pos).ll
		for i, t := range p.params {
			paramTypes[i+1] = b.di().diType(t, pos).ll
		}
		diFuncType := b.di().di.CreateSubroutineType(llvm.DISubroutineType{
			File:       b.di().file(pos.Filename).ll,
			Parameters: paramTypes,
		})
		dif := llvm.DIFunction{
			Type:         diFuncType,
			Name:         p.Name(),
			LinkageName:  p.Name(),
			File:         b.di().file(pos.Filename).ll,
			Line:         pos.Line,
			ScopeLine:    bodyPos.Line,
			IsDefinition: true,
			LocalToUnit:  true,
			Optimized:    true,
		}
		p.diFunc = &aDIFunction{
			b.di().di.CreateFunction(b.di().file(pos.Filename).ll, dif),
		}
		p.impl.SetSubprogram(p.diFunc.ll)
	}
	b.impl.SetCurrentDebugLocation(
		uint(bodyPos.Line),
		uint(bodyPos.Column),
		p.diFunc.ll,
		f.impl.InstructionDebugLoc(),
	)
}

func (b Builder) Param(idx int) Expr {
	return b.Func.Param(idx)
}

// -----------------------------------------------------------------------------
