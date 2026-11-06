package ssa

import (
	"debug/dwarf"
	"fmt"
	"go/token"
	"go/types"

	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llgo/internal/debuginfo"
	ssaabi "github.com/xgo-dev/llgo/ssa/abi"
	"github.com/xgo-dev/llvm"
)

type Positioner interface {
	Position(pos token.Pos) token.Position
}

type aDIBuilder struct {
	di         *debuginfo.Builder
	prog       Program
	types      map[Type]DIType
	positioner Positioner
}

type diBuilder = *aDIBuilder

func newDIBuilder(prog Program, pkg Package, positioner Positioner) diBuilder {
	byteOrder := debugabi.ByteOrderLittle
	if prog.TargetData().ByteOrder() == llvm.BigEndian {
		byteOrder = debugabi.ByteOrderBig
	}
	return &aDIBuilder{
		di: debuginfo.New(pkg.mod, debuginfo.Config{
			Producer:       "LLGo",
			Optimized:      prog.debugInfoOptimized,
			EmitCodeView:   prog.emitCodeViewDebugInfo,
			DebuggerRecord: debugabi.NewRecord(uint8(prog.PointerSize()), byteOrder),
		}),
		prog:       prog,
		types:      make(map[*aType]DIType),
		positioner: positioner,
	}
}

func (b diBuilder) finalize() {
	if b == nil || b.di == nil {
		return
	}
	b.di.Finalize()
	b.di = nil
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

// ----------------------------------------------------------------------------

type aCompilationUnit struct {
	ll llvm.Metadata
}

type CompilationUnit = *aCompilationUnit

func (c CompilationUnit) scopeMeta(b diBuilder, pos token.Position) DIScopeMeta {
	return &aDIScopeMeta{c.ll}
}

func (b diBuilder) createCompileUnit(filename, dir string) CompilationUnit {
	return &aCompilationUnit{ll: b.di.CompileUnit(filename, dir)}
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
	return &aDIFile{ll: b.di.File(filename)}
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
			encoding = llvm.DW_ATE_complex_float
		} else if t.Info()&types.IsString != 0 {
			return b.createStringType()
		} else {
			panic(fmt.Errorf("can't create debug info of basic type: %v, %T", ty.RawType(), ty.RawType()))
		}

		basicName := name
		if b.prog.Target().effectiveGOOS() == "windows" {
			switch t.Kind() {
			case types.Int:
				basicName = fmt.Sprintf("int%d", b.prog.SizeOf(ty)*8)
			case types.Uint, types.Uintptr:
				basicName = fmt.Sprintf("uint%d", b.prog.SizeOf(ty)*8)
			}
		}
		typ = b.di.CreateBasicType(llvm.DIBasicType{
			Name:       basicName,
			SizeInBits: b.prog.SizeOf(b.prog.rawType(t)) * 8,
			Encoding:   encoding,
		})
		if basicName != name {
			typ = b.di.CreateTypedef(llvm.DITypedef{
				Name:        name,
				Type:        typ,
				File:        b.file(pos.Filename).ll,
				Line:        pos.Line,
				AlignInBits: uint32(b.prog.sizes.Alignof(t) * 8),
			})
		}
	case *types.Pointer:
		return b.createPointerType(name, b.prog.rawType(t.Elem()), pos)
	case *types.Named:
		return b.createTypedefType(name, ty, b.typeDeclarationPosition(t, pos))
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
		return b.createFuncPtrType(name, ty, pos)
	case *types.Array:
		return b.createArrayType(ty, t.Len())
	case *types.Chan:
		return b.createChanType(name, ty, t)
	case *types.Map:
		return b.createMapType(name, ty, t)
	case *types.Tuple:
		return b.createTupleType(name, ty, pos)
	default:
		panic(fmt.Errorf("can't create debug info of type: %v, %T", ty.RawType(), ty.RawType()))
	}
	return &aDIType{typ}
}

func (b diBuilder) typeDeclarationPosition(typ *types.Named, fallback token.Position) token.Position {
	if obj := typ.Obj(); obj != nil && obj.Pos().IsValid() {
		if pos := b.positioner.Position(obj.Pos()); pos.IsValid() {
			return pos
		}
	}
	return fallback
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
	scope := b.file(pos.Filename)
	ret := &aDIType{ll: b.di.CreateReplaceableCompositeType(
		scope.ll,
		llvm.DIReplaceableCompositeType{
			Tag:         dwarf.TagStructType,
			Name:        name,
			File:        scope.ll,
			Line:        pos.Line,
			SizeInBits:  b.prog.SizeOf(ty) * 8,
			AlignInBits: uint32(b.prog.sizes.Alignof(ty.RawType()) * 8),
		},
	)}
	b.types[ty] = ret

	underlying := ty.RawType().(*types.Named).Underlying()
	rawUnderlying := b.prog.rawType(underlying)
	if signature, ok := underlying.(*types.Signature); ok {
		// Keep the shared LLVM cache unchanged, but preserve the source
		// signature for DWARF. Go type identity ignores __llgo_va_list's name.
		rawUnderlying = b.prog.toType(signature)
	}
	underlyingType := b.diType(rawUnderlying, pos)
	typ := b.di.CreateTypedef(llvm.DITypedef{
		Name:        name,
		Type:        underlyingType.ll,
		File:        scope.ll,
		Line:        pos.Line,
		AlignInBits: uint32(b.prog.sizes.Alignof(ty.RawType()) * 8),
	})
	ret.ll.ReplaceAllUsesWith(typ)
	ret.ll = typ
	return ret
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

func (b diBuilder) createPointerType(name string, ty Type, pos token.Position) DIType {
	ptrType := b.prog.VoidPtr()
	return &aDIType{ll: b.di.CreatePointerType(llvm.DIPointerType{
		Name:        name,
		Pointee:     b.diType(ty, pos).ll,
		SizeInBits:  b.prog.SizeOf(ptrType) * 8,
		AlignInBits: uint32(b.prog.sizes.Alignof(ptrType.RawType())) * 8,
	})}
}

func (b diBuilder) createMapType(name string, ty Type, mapType *types.Map) DIType {
	pos := token.Position{}
	ptr := b.prog.VoidPtr()
	runtimeMap := b.prog.rtType("Map")
	key := b.prog.rawType(mapType.Key())
	elem := b.prog.rawType(mapType.Elem())
	hashName := fmt.Sprintf("hash<%s,%s>", diTypeName(mapType.Key()), diTypeName(mapType.Elem()))
	hash := b.createSyntheticStructPlaceholder(hashName, runtimeMap, pos)
	hashPtr := b.di.CreatePointerType(llvm.DIPointerType{
		Name:        "*" + hashName,
		Pointee:     hash.ll,
		SizeInBits:  b.prog.SizeOf(ptr) * 8,
		AlignInBits: uint32(b.prog.sizes.Alignof(ptr.RawType()) * 8),
	})
	ret := &aDIType{ll: b.createRuntimeContainerTypedef(name, hashPtr, pos)}
	// Map values may be recursive through their key or element type. Cache the
	// pointer before constructing the typed bucket, as the Go linker does when
	// synthesizing map DWARF.
	b.types[ty] = ret

	bucket := b.createMapBucketType(mapType, key, elem, pos)
	bucketPtr := b.createDIPointerType(
		fmt.Sprintf("*bucket<%s,%s>", diTypeName(mapType.Key()), diTypeName(mapType.Elem())),
		bucket.ll,
	)
	replacements := map[string]llvm.Metadata{
		"buckets":    bucketPtr,
		"oldbuckets": bucketPtr,
	}
	b.finishSyntheticStruct(hash, hashName, runtimeMap,
		b.syntheticStructFields(hash, runtimeMap, replacements, pos), pos)
	return ret
}

func (b diBuilder) createMapBucketType(mapType *types.Map, key, elem Type,
	pos token.Position) DIType {
	bucketStorage := b.prog.rawType(ssaabi.MapBucketType(mapType, b.prog.sizes))
	bucketStruct := bucketStorage.RawType().Underlying().(*types.Struct)
	name := fmt.Sprintf("bucket<%s,%s>", diTypeName(key.RawType()), diTypeName(elem.RawType()))
	bucket := b.createSyntheticStructPlaceholder(name, bucketStorage, pos)
	overflow := b.createDIPointerType("*"+name, bucket.ll)
	fields := make([]llvm.Metadata, bucketStruct.NumFields())
	for index := range fields {
		field := bucketStruct.Field(index)
		fieldName := field.Name()
		fieldType := b.prog.rawType(field.Type())
		diType := b.diType(fieldType, pos).ll
		switch fieldName {
		case "topbits":
			fieldName = "tophash"
		case "keys":
			if b.prog.SizeOf(key) > ssaabi.MAXKEYSIZE {
				fieldName = "indirectkeys"
			}
		case "elems":
			fieldName = "values"
			if b.prog.SizeOf(elem) > ssaabi.MAXELEMSIZE {
				fieldName = "indirectvalues"
			}
		case "overflow":
			diType = overflow
		}
		fields[index] = b.createDIMemberType(bucket, fieldName,
			b.prog.SizeOf(fieldType), b.prog.sizes.Alignof(field.Type()),
			b.prog.OffsetOf(bucketStorage, index), diType)
	}
	b.finishSyntheticStruct(bucket, name, bucketStorage, fields, pos)
	return bucket
}

func (b diBuilder) createChanType(name string, ty Type, chanType *types.Chan) DIType {
	pos := token.Position{}
	ptr := b.prog.VoidPtr()
	runtimeChan := b.prog.rtType("Chan")
	chanName := fmt.Sprintf("hchan<%s>", diTypeName(chanType.Elem()))
	channel := b.createSyntheticStructPlaceholder(chanName, runtimeChan, pos)
	channelPtr := b.di.CreatePointerType(llvm.DIPointerType{
		Name:        "*" + chanName,
		Pointee:     channel.ll,
		SizeInBits:  b.prog.SizeOf(ptr) * 8,
		AlignInBits: uint32(b.prog.sizes.Alignof(ptr.RawType()) * 8),
	})
	ret := &aDIType{ll: b.createRuntimeContainerTypedef(name, channelPtr, pos)}
	b.types[ty] = ret

	chanStruct := runtimeChan.RawType().Underlying().(*types.Struct)
	queueIndex := structFieldIndex(chanStruct, "recvq")
	queue := b.prog.rawType(chanStruct.Field(queueIndex).Type())
	queueStruct := queue.RawType().Underlying().(*types.Struct)
	waiterPtrType := queueStruct.Field(structFieldIndex(queueStruct, "first")).Type().(*types.Pointer)
	waiter := b.prog.rawType(waiterPtrType.Elem())

	waiterName := fmt.Sprintf("sudog<%s>", diTypeName(chanType.Elem()))
	typedWaiter := b.createSyntheticStructPlaceholder(waiterName, waiter, pos)
	typedWaiterPtr := b.createDIPointerType("*"+waiterName, typedWaiter.ll)
	elem := b.prog.rawType(chanType.Elem())
	elemPtr := b.createDIPointerType("*"+diTypeName(chanType.Elem()), b.diType(elem, pos).ll)
	waiterReplacements := map[string]llvm.Metadata{
		"prev": typedWaiterPtr,
		"next": typedWaiterPtr,
		"all":  typedWaiterPtr,
		"ch":   ret.ll,
		"elem": elemPtr,
	}
	b.finishSyntheticStruct(typedWaiter, waiterName, waiter,
		b.syntheticStructFields(typedWaiter, waiter, waiterReplacements, pos), pos)

	queueName := fmt.Sprintf("waitq<%s>", diTypeName(chanType.Elem()))
	typedQueue := b.createSyntheticStructPlaceholder(queueName, queue, pos)
	queueReplacements := map[string]llvm.Metadata{
		"first": typedWaiterPtr,
		"last":  typedWaiterPtr,
	}
	b.finishSyntheticStruct(typedQueue, queueName, queue,
		b.syntheticStructFields(typedQueue, queue, queueReplacements, pos), pos)

	channelReplacements := map[string]llvm.Metadata{
		"sendq": typedQueue.ll,
		"recvq": typedQueue.ll,
	}
	b.finishSyntheticStruct(channel, chanName, runtimeChan,
		b.syntheticStructFields(channel, runtimeChan, channelReplacements, pos), pos)
	return ret
}

func (b diBuilder) createSyntheticStructPlaceholder(name string, ty Type,
	pos token.Position) DIType {
	scope := b.file(pos.Filename)
	return &aDIType{ll: b.di.CreateReplaceableCompositeType(
		scope.ll,
		llvm.DIReplaceableCompositeType{
			Tag:         dwarf.TagStructType,
			Name:        name,
			File:        scope.ll,
			Line:        pos.Line,
			SizeInBits:  b.prog.SizeOf(ty) * 8,
			AlignInBits: uint32(b.prog.sizes.Alignof(ty.RawType()) * 8),
		},
	)}
}

func (b diBuilder) createRuntimeContainerTypedef(name string,
	typeMeta llvm.Metadata, pos token.Position) llvm.Metadata {
	ptr := b.prog.VoidPtr()
	return b.di.CreateTypedef(llvm.DITypedef{
		Name:        name,
		Type:        typeMeta,
		File:        b.file(pos.Filename).ll,
		Line:        pos.Line,
		AlignInBits: uint32(b.prog.sizes.Alignof(ptr.RawType()) * 8),
	})
}

func (b diBuilder) finishSyntheticStruct(placeholder DIType, name string, ty Type,
	fields []llvm.Metadata, pos token.Position) {
	scope := b.file(pos.Filename)
	value := b.di.CreateStructType(scope.ll, llvm.DIStructType{
		Name:        name,
		File:        scope.ll,
		Line:        pos.Line,
		SizeInBits:  b.prog.SizeOf(ty) * 8,
		AlignInBits: uint32(b.prog.sizes.Alignof(ty.RawType()) * 8),
		Elements:    fields,
	})
	placeholder.ll.ReplaceAllUsesWith(value)
	placeholder.ll = value
}

func (b diBuilder) syntheticStructFields(owner DIType, ty Type,
	replacements map[string]llvm.Metadata, pos token.Position) []llvm.Metadata {
	structure := ty.RawType().Underlying().(*types.Struct)
	fields := make([]llvm.Metadata, structure.NumFields())
	for index := 0; index < structure.NumFields(); index++ {
		field := structure.Field(index)
		fieldType := b.prog.rawType(field.Type())
		diType := b.diType(fieldType, pos).ll
		if replacement, ok := replacements[field.Name()]; ok {
			diType = replacement
		}
		fields[index] = b.createDIMemberType(owner, field.Name(),
			b.prog.SizeOf(fieldType), b.prog.sizes.Alignof(field.Type()),
			b.prog.OffsetOf(ty, index), diType)
	}
	return fields
}

func (b diBuilder) createDIPointerType(name string, pointee llvm.Metadata) llvm.Metadata {
	ptr := b.prog.VoidPtr()
	return b.di.CreatePointerType(llvm.DIPointerType{
		Name:        name,
		Pointee:     pointee,
		SizeInBits:  b.prog.SizeOf(ptr) * 8,
		AlignInBits: uint32(b.prog.sizes.Alignof(ptr.RawType()) * 8),
	})
}

func (b diBuilder) createDIMemberType(owner DIType, name string, size uint64,
	align int64, offset uint64, ty llvm.Metadata) llvm.Metadata {
	return b.di.CreateMemberType(owner.ll, llvm.DIMemberType{
		Name:         name,
		SizeInBits:   size * 8,
		AlignInBits:  uint32(align * 8),
		OffsetInBits: offset * 8,
		Type:         ty,
	})
}

func structFieldIndex(structure *types.Struct, name string) int {
	for index := 0; index < structure.NumFields(); index++ {
		if structure.Field(index).Name() == name {
			return index
		}
	}
	panic(fmt.Sprintf("runtime field %q not found in %s", name, structure))
}

func (b diBuilder) doCreateStructType(name string, ty Type, pos token.Position, fn func(ty DIType) []llvm.Metadata) (ret DIType) {
	scope := b.file(pos.Filename)
	ret = &aDIType{b.di.CreateReplaceableCompositeType(
		scope.ll,
		llvm.DIReplaceableCompositeType{
			Tag:         dwarf.TagStructType,
			Name:        name,
			File:        b.file(pos.Filename).ll,
			Line:        pos.Line,
			SizeInBits:  b.prog.SizeOf(ty) * 8,
			AlignInBits: uint32(b.prog.AlignOf(ty) * 8),
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
			AlignInBits: uint32(b.prog.AlignOf(ty) * 8),
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
			if signature, ok := field.Type().(*types.Signature); ok {
				// A closure's $f must retain its own variadic convention even
				// when a C callback with a Go-identical signature was cached first.
				tyField = b.prog.toType(signature)
			}
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
	sig := ty.RawType().(*types.Signature)
	params := make([]llvm.Metadata, sig.Params().Len()+1)
	if results := sig.Results(); results.Len() != 0 {
		params[0] = b.diType(b.prog.rawType(results), pos).ll
	}
	n := sig.Params().Len()
	if HasNameValist(sig) {
		// A trailing null type emits DW_TAG_unspecified_parameters. The
		// __llgo_va_list marker is not a fixed []any parameter.
		n--
	}
	for i := 0; i < n; i++ {
		params[i+1] = b.diType(b.prog.rawType(sig.Params().At(i).Type()), pos).ll
	}
	subroutine := b.di.CreateSubroutineType(llvm.DISubroutineType{
		File:       b.file(pos.Filename).ll,
		Parameters: params,
	})
	ptr := b.prog.VoidPtr()
	return &aDIType{ll: b.di.CreatePointerType(llvm.DIPointerType{
		Name:        name,
		Pointee:     subroutine,
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

// Match runtime metadata, including main packages and source overlays.
func diTypeName(t types.Type) string {
	return types.TypeString(t, ssaabi.PathOf)
}

func (b diBuilder) diType(t Type, pos token.Position) DIType {
	if hasTypeParam(t.RawType()) {
		return &aDIType{}
	}
	name := diTypeName(t.RawType())
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
func (b Builder) constructDebugAddr(v Expr) Expr {
	t := v.Type.RawType().Underlying()
	return b.doConstructDebugAddr(v, t)
}

func (b Builder) constructDebugAddrWithStore(v Expr) (Expr, Expr) {
	t := v.Type.RawType().Underlying()
	return b.doConstructDebugAddrWithStore(v, t)
}

func (b Builder) doConstructDebugAddr(v Expr, t types.Type) (dbgPtr Expr) {
	dbgPtr, _ = b.doConstructDebugAddrWithStore(v, t)
	return dbgPtr
}

func (b Builder) doConstructDebugAddrWithStore(v Expr, t types.Type) (dbgPtr, store Expr) {
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
		ty = b.Prog.Type(b.Prog.rtType("Slice").RawType().Underlying(), GoDecl)
	case *types.Signature:
		if v.kind == vkFuncPtr {
			// Native entries already have pointer storage. A debug snapshot must
			// not adapt them into Go closures, especially for C ellipsis entries.
			ty = v.Type
		} else {
			ty = b.Prog.Closure(t)
		}
	case *types.Named:
		ty = b.Prog.Type(t.Underlying(), GoDecl)
	default:
		ty = v.Type
	}
	// Empty aggregates have no SelectionDAG value. LLVM 22 can crash when
	// promoting their debug snapshots to dbg.value(empty zeroinitializer).
	// Back the debug-only address with a byte while keeping the source type
	// in the variable metadata; there are no source value bits to copy.
	storedValue := v
	if b.Prog.SizeOf(ty) == 0 {
		ty = b.Prog.Byte()
		storedValue = b.Prog.IntVal(0, ty)
	}
	// A debug snapshot reserves one slot per function invocation. Allocating
	// it at a DebugRef inside a loop grows the stack on every iteration at O0.
	// Keep the value update at its source position, but reserve the slot with
	// the same entry-block builder used for ordinary stack locals.
	entryBuilder := *b
	entryBuilder.impl = b.Func.entryAllocaBuilder()
	dbgPtr = entryBuilder.AllocaT(ty)
	// Store through the allocated type so closure snapshots also initialize
	// their data field before exposing the source-typed debug address.
	store = b.Store(dbgPtr, storedValue)
	dbgPtr.Type = b.Prog.Pointer(v.Type)
	return dbgPtr, store
}

func (b Builder) di() diBuilder {
	return b.Pkg.di
}

// DIParam emits parameter debug information without exposing its storage. Its
// original signature is retained for callers that use Builder method values.
func (b Builder) DIParam(variable *types.Var, v Expr, dv DIVar, scope DIScope, pos token.Position, blk BasicBlock) {
	b.diParam(variable, v, dv, scope, pos, blk)
}

// DIParamWithHome returns the stable O0 storage backing the parameter.
func (b Builder) DIParamWithHome(variable *types.Var, v Expr, dv DIVar, scope DIScope, pos token.Position, blk BasicBlock) Expr {
	return b.diParam(variable, v, dv, scope, pos, blk)
}

func (b Builder) diParam(variable *types.Var, v Expr, dv DIVar, scope DIScope, pos token.Position, blk BasicBlock) Expr {
	if b.Prog.debugInfoOptimized {
		// Preserve the Windows pointer location policy for optimized CodeView.
		if b.Prog.Target().effectiveGOOS() == "windows" {
			if _, ok := v.Type.RawType().Underlying().(*types.Pointer); ok {
				addr := b.AllocaT(v.Type)
				b.Store(addr, v)
				b.DIDeclare(variable, addr, dv, scope, pos, blk)
				return Nil
			}
		}
		b.DIValue(variable, v, dv, scope, pos, blk)
		return Nil
	}
	dbgPtr, store := b.constructDebugAddrWithStore(v)
	store.impl.InstructionSetDebugLoc(llvm.Metadata{})
	b.DIDeclare(variable, dbgPtr, dv, scope, pos, blk)
	return dbgPtr
}

// DIStore updates the stable debug-only storage for an O0 parameter.
func (b Builder) DIStore(ptr, value Expr) {
	store := b.Store(ptr, value)
	store.impl.InstructionSetDebugLoc(llvm.Metadata{})
}

func (b Builder) DIDeclare(variable *types.Var, v Expr, dv DIVar, scope DIScope, pos token.Position, blk BasicBlock) {
	expr := b.di().createExpression(nil)
	b.di().dbgDeclare(v, dv, scope, pos, expr, blk)
}

func (b Builder) DIValue(variable *types.Var, v Expr, dv DIVar, scope DIScope, pos token.Position, blk BasicBlock) {
	ty := v.Type.RawType().Underlying()
	if !b.needDebugAddr(ty, v.Type) {
		expr := b.di().createExpression(nil)
		b.di().dbgValue(v, dv, scope, pos, expr, blk)
	} else {
		dbgPtr := b.constructDebugAddr(v)
		expr := b.di().createExpression([]uint64{opDeref})
		b.di().dbgValue(dbgPtr, dv, scope, pos, expr, blk)
	}
}

func (b Builder) needDebugAddr(underlying types.Type, ssaType Type) bool {
	if needConstructAddr(underlying) {
		return true
	}
	// On 32-bit Windows, LLVM can lower a wide integer constant to a single
	// address-sized DWARF stack value. That loses its upper half (notably for
	// unsigned values), so debuggers report the variable as unavailable. Keep
	// the complete value in addressable storage and describe it by dereference.
	basic, ok := underlying.(*types.Basic)
	return ok && basic.Info()&types.IsInteger != 0 &&
		b.Prog.Target().effectiveGOOS() == "windows" &&
		b.Prog.PointerSize() == 4 && b.Prog.SizeOf(ssaType) > uint64(b.Prog.PointerSize())
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
	case *types.Pointer, *types.Map, *types.Chan:
		// Map and channel values are pointers to runtime headers. Like other
		// pointers, describe the value itself; a snapshot plus DW_OP_deref can
		// lose its location when LLVM promotes the entry-block slot on Wasm.
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

func (b Builder) DIScope(f Function, scope *types.Scope) DIScope {
	if scope == nil || b.diFuncScope == nil {
		return f
	}
	if cachedScope, ok := b.diScopeCache[scope]; ok {
		return cachedScope
	}
	if scope == b.diFuncScope {
		b.diScopeCache[scope] = f
		return f
	}
	if !isScopeWithin(scope, b.diFuncScope) {
		return f
	}

	pos := b.di().positioner.Position(scope.Pos())
	parentScope := b.DIScope(f, scope.Parent())
	result := &aDILexicalBlock{b.di().di.CreateLexicalBlock(parentScope.scopeMeta(b.di(), pos).ll, llvm.DILexicalBlock{
		File:   b.di().file(pos.Filename).ll,
		Line:   pos.Line,
		Column: pos.Column,
	})}
	b.diScopeCache[scope] = result
	return result
}

func isScopeWithin(scope, root *types.Scope) bool {
	for current := scope; current != nil; current = current.Parent() {
		if current == root {
			return true
		}
	}
	return false
}

const (
	MD_dbg = 0
)

func (b Builder) DIGlobal(v Expr, name string, pos token.Position) {
	// Frontend pseudo-variables, such as Python module attributes, have no
	// native storage to describe or attach metadata to.
	if v.impl.IsNil() {
		return
	}
	if _, ok := b.Pkg.glbDbgVars[v]; ok {
		return
	}
	gv := b.di().createGlobalVariableExpression(
		b.Pkg.cu,
		pos,
		name,
		name,
		b.Prog.Elem(v.Type),
		false,
	)
	v.impl.AddMetadata(MD_dbg, gv.ll)
	b.Pkg.glbDbgVars[v] = true
}

func (b Builder) setDebugLocation(loc llvm.DebugLoc) {
	// The LLVM binding's getter cannot read an unset location. Keep the
	// logical source location here, including for builders created before
	// DebugFunction attaches the function's DISubprogram.
	b.diLocation = loc
	b.restoreDebugLocation()
}

func (b Builder) restoreDebugLocation() {
	loc := b.diLocation
	if loc.Scope.IsNil() {
		return
	}
	b.impl.SetCurrentDebugLocation(loc.Line, loc.Col, loc.Scope, loc.InlinedAt)
}

func (b Builder) DISetCurrentDebugLocation(diScope DIScope, pos token.Position) {
	b.setDebugLocation(llvm.DebugLoc{
		Line:  uint(pos.Line),
		Col:   uint(pos.Column),
		Scope: diScope.scopeMeta(b.di(), pos).ll,
	})
}

func (b Builder) DebugFunction(f Function, funcScope *types.Scope, pos token.Position, bodyPos token.Position) {
	b.diFuncScope = funcScope
	p := f
	if p.diFunc == nil {
		sig := p.Type.raw.Type.(*types.Signature)
		rt := p.Prog.Type(sig.Results(), GoDecl)
		paramTypes := make([]llvm.Metadata, len(p.params)+1)
		paramTypes[0] = b.di().diType(rt, pos).ll
		for i, t := range p.params {
			paramTypes[i+1] = b.di().diType(t, pos).ll
		}
		if p.hasVArg {
			paramTypes = append(paramTypes, llvm.Metadata{})
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
		}
		p.diFunc = &aDIFunction{
			b.di().di.CreateFunction(b.di().file(pos.Filename).ll, dif),
		}
		p.impl.SetSubprogram(p.diFunc.ll)
	}
	b.setDebugLocation(llvm.DebugLoc{
		Line:  uint(bodyPos.Line),
		Col:   uint(bodyPos.Column),
		Scope: p.diFunc.ll,
	})
}

// DIClosureEnv describes the compiler-owned environment separately from source
// parameters. Its type lets debuggers inspect a closure value even when that
// closure is not executing. Opaque //llgo:env entries intentionally keep their
// unsafe.Pointer type and cannot be expanded as captures.
func (b Builder) DIClosureEnv(pos token.Position, blk BasicBlock) {
	if !b.Func.NeedsEnv() {
		return
	}
	ty := b.di().diType(b.Func.EnvType(), pos)
	dv := &aDIVar{ll: b.di().di.CreateAutoVariable(
		b.Func.scopeMeta(b.di(), pos).ll,
		llvm.DIAutoVariable{
			Name: "$env", File: b.di().file(pos.Filename).ll, Line: pos.Line,
			Type: ty.ll, Flags: llvm.FlagArtificial, AlwaysPreserve: true,
		},
	)}
	b.DIValue(nil, b.Func.Env(), dv, b.Func, pos, blk)
}

func (b Builder) Param(idx int) Expr {
	return b.logicalValue(b.Func.Param(idx))
}

// -----------------------------------------------------------------------------
