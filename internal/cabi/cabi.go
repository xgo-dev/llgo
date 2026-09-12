package cabi

import (
	"fmt"
	"strings"

	"github.com/xgo-dev/llgo/internal/funcattrs"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func targetArch(llvmTarget string) string {
	if pos := strings.Index(llvmTarget, "-"); pos != -1 {
		llvmTarget = llvmTarget[:pos]
	}
	switch llvmTarget {
	case "x86_64":
		return "amd64"
	case "aarch64":
		return "arm64"
	case "i386", "i486", "i586", "i686":
		return "386"
	case "wasm32", "wasm64":
		return "wasm"
	}
	if strings.HasPrefix(llvmTarget, "armv") || strings.HasPrefix(llvmTarget, "thumb") {
		return "arm"
	}
	return llvmTarget
}

// usesWindowsCABI reports whether the target uses the native Windows C ABI.
// MinGW uses that ABI too; its GNU environment selects a toolchain and CRT,
// not the SysV calling convention. MSYS and Cygwin are POSIX-emulation ABIs
// and are deliberately excluded.
func usesWindowsCABI(target *ssa.Target, llvmTarget string) bool {
	if llvmTarget == "" || !strings.Contains(llvmTarget, "-") {
		return target != nil && target.GOOS == "windows"
	}
	parts := strings.Split(strings.ToLower(llvmTarget), "-")
	windows := false
	for _, part := range parts[1:] {
		switch {
		case strings.Contains(part, "cygwin"), part == "cygnus", strings.Contains(part, "msys"):
			return false
		case part == "windows", part == "win32", strings.Contains(part, "mingw"):
			windows = true
		}
	}
	return windows
}

func NewTransformer(prog ssa.Program, llvmTarget string, targetAbi string, optimize bool) *Transformer {
	target := prog.Target()
	arch := target.GOARCH
	if llvmTarget != "" {
		arch = targetArch(llvmTarget)
	}
	tr := &Transformer{
		prog:     prog,
		td:       prog.TargetData(),
		arch:     arch,
		optimize: optimize,
	}
	if usesWindowsCABI(target, llvmTarget) {
		switch arch {
		case "amd64":
			tr.sys = &TypeInfoWindowsAmd64{tr}
		case "arm64":
			tr.sys = &TypeInfoWindowsArm64{&TypeInfoArm64{tr}}
		case "386":
			tr.sys = &TypeInfoWindows386{tr}
		}
		if tr.sys != nil {
			return tr
		}
	}
	switch arch {
	case "xtensa":
		tr.sys = &TypeInfoEsp32{tr}
	case "riscv32":
		tr.sys = &TypeInfoRiscv32{tr, targetAbi}
	case "amd64":
		tr.sys = &TypeInfoAmd64{tr}
	case "arm64":
		tr.sys = &TypeInfoArm64{tr}
	case "arm":
		tr.sys = &TypeInfoArm{tr}
	case "wasm":
		tr.sys = &TypeInfoWasm{tr}
	case "riscv64":
		tr.sys = &TypeInfoRiscv64{tr, targetAbi}
	case "386":
		tr.sys = &TypeInfo386{tr}
	}
	return tr
}

type Transformer struct {
	prog     ssa.Program
	td       llvm.TargetData
	arch     string
	sys      TypeInfoSys
	optimize bool
	skipFns  map[string]struct{}
}

// SetSkipFuncs configures function full names that should not be transformed.
// Names are LLVM symbol names (for example, "internal/bytealg.Compare").
// The list is applied to both function signature rewriting and call-site rewriting.
func (p *Transformer) SetSkipFuncs(names []string) {
	p.skipFns = make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		p.skipFns[name] = struct{}{}
	}
}

func (p *Transformer) shouldSkipFunc(name string) bool {
	if name == "" || len(p.skipFns) == 0 {
		return false
	}
	_, ok := p.skipFns[name]
	return ok
}

func (p *Transformer) shouldSkipCall(call llvm.Value) bool {
	callee := call.CalledValue()
	if !callee.IsAInlineAsm().IsNil() {
		return true
	}
	if callee.IsAFunction().IsNil() {
		return false
	}
	return p.shouldSkipFunc(callee.Name())
}

type CallInstr struct {
	call llvm.Value
	fn   llvm.Value
}

func (p *Transformer) TransformModule(path string, m llvm.Module) {
	ctx := m.Context()
	var fns []llvm.Value
	var callInstrs []CallInstr
	fn := m.FirstFunction()
	for !fn.IsNil() {
		if !p.shouldSkipFunc(fn.Name()) && p.isWrapFunctionType(ctx, fn.GlobalValueType()) {
			fns = append(fns, fn)
		}
		bb := fn.FirstBasicBlock()
		for !bb.IsNil() {
			instr := bb.FirstInstruction()
			for !instr.IsNil() {
				if !instr.IsAInvokeInst().IsNil() && p.isWrapFunctionType(ctx, instr.CalledFunctionType()) {
					panic("cabi: invoke requires unsupported ABI signature conversion")
				}
				if call := instr.IsACallInst(); !call.IsNil() {
					if p.shouldSkipCall(call) {
						instr = llvm.NextInstruction(instr)
						continue
					}
					if p.isWrapFunctionType(ctx, call.CalledFunctionType()) {
						callInstrs = append(callInstrs, CallInstr{call, fn})
					}
				}
				instr = llvm.NextInstruction(instr)
			}
			bb = llvm.NextBasicBlock(bb)
		}
		fn = llvm.NextFunction(fn)
	}
	for _, call := range callInstrs {
		p.transformCallInstr(m, ctx, call.call, call.fn)
	}
	for _, fn := range fns {
		p.transformFunc(m, fn)
	}
}

func (p *Transformer) isWrapFunctionType(ctx llvm.Context, ft llvm.Type) bool {
	if p.IsWrapType(ctx, ft, ft.ReturnType(), 0) {
		return true
	}
	for i, typ := range ft.ParamTypes() {
		if p.IsWrapType(ctx, ft, typ, i+1) {
			return true
		}
	}
	return false
}

type TypeInfoSys interface {
	SupportByVal() bool
	SkipEmptyParams() bool
	IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool
	GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo
}

type AttrKind int

const (
	AttrNone       AttrKind = iota // keep org type
	AttrVoid                       // return type void / param type void (size == 0) skip
	AttrPointer                    // type => type*
	AttrWidthType                  // type => width int i16/i24/i32/i40/i48/i56/i64 float/double
	AttrWidthType2                 // type => width two int {i64,i16} float/double
	AttrExtract                    // extract struct type
)

type FuncInfo struct {
	Type   llvm.Type   // func type
	Return *TypeInfo   // return info
	Params []*TypeInfo // params info
}

func (p *FuncInfo) HasWrap() bool {
	if p.Return.Kind > AttrVoid {
		return true
	}
	for _, t := range p.Params {
		if t.Kind > AttrNone {
			return true
		}
	}
	return false
}

type TypeInfo struct {
	Type       llvm.Type
	NativeType llvm.Type // native aggregate layout when it differs from Type
	Kind       AttrKind
	Type1      llvm.Type // AttrWidthType
	Type2      llvm.Type // AttrWidthType2
	Size       int
	Align      int
	ByValAlign int // explicit stack alignment for AttrPointer parameters
}

func (p *TypeInfo) nativeType() llvm.Type {
	if p.NativeType.C != nil {
		return p.NativeType
	}
	return p.Type
}

func (p *TypeInfo) hasNativeLayoutConversion() bool {
	return p.NativeType.C != nil && p.NativeType != p.Type
}

func byvalAttribute(ctx llvm.Context, typ llvm.Type) llvm.Attribute {
	id := llvm.AttributeKindID("byval")
	return ctx.CreateTypeAttribute(id, typ)
}

func sretAttribute(ctx llvm.Context, typ llvm.Type) llvm.Attribute {
	id := llvm.AttributeKindID("sret")
	return ctx.CreateTypeAttribute(id, typ)
}

func alignAttribute(ctx llvm.Context, align int) llvm.Attribute {
	return ctx.CreateEnumAttribute(llvm.AttributeKindID("align"), uint64(align))
}

func funcNoUnwind(ctx llvm.Context) llvm.Attribute {
	return ctx.CreateEnumAttribute(llvm.AttributeKindID("nounwind"), 0)
}

func (p *Transformer) IsWrapType(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) bool {
	if p.sys != nil {
		bret := index == 0
		if p.sys.SkipEmptyParams() && p.isWrapEmptyType(ctx, typ, bret) {
			return true
		}
		return p.sys.IsWrapType(ctx, ftyp, typ, index)
	}
	return false
}

func (p *Transformer) isWrapEmptyType(ctx llvm.Context, typ llvm.Type, bret bool) bool {
	if !bret && (typ.TypeKind() == llvm.VoidTypeKind || p.Sizeof(typ) == 0) {
		return true
	}
	return false
}

func (p *Transformer) getEmptyType(ctx llvm.Context, typ llvm.Type, bret bool) (*TypeInfo, bool) {
	if typ.TypeKind() == llvm.VoidTypeKind {
		return &TypeInfo{Type: typ, Kind: AttrVoid, Type1: ctx.VoidType()}, true
	} else if p.Sizeof(typ) == 0 {
		if bret {
			return &TypeInfo{Type: typ, Kind: AttrNone, Type1: typ}, true
		}
		return &TypeInfo{Type: typ, Kind: AttrVoid, Type1: ctx.VoidType()}, true
	}
	return nil, false
}

func (p *Transformer) GetTypeInfo(ctx llvm.Context, ftyp llvm.Type, typ llvm.Type, index int) *TypeInfo {
	if p.sys != nil {
		bret := index == 0
		if p.sys.SkipEmptyParams() {
			if info, ok := p.getEmptyType(ctx, typ, bret); ok {
				return info
			}
		}
		return p.sys.GetTypeInfo(ctx, ftyp, typ, index)
	}
	panic("not implment: " + p.arch)
}

func (p *Transformer) Sizeof(typ llvm.Type) int {
	return int(p.td.TypeAllocSize(typ))
}

func (p *Transformer) Alignof(typ llvm.Type) int {
	return int(p.td.ABITypeAlignment(typ))
}

func (p *Transformer) GetFuncInfo(ctx llvm.Context, typ llvm.Type) (info FuncInfo) {
	info.Type = typ
	info.Return = p.GetTypeInfo(ctx, typ, typ.ReturnType(), 0)
	params := typ.ParamTypes()
	info.Params = make([]*TypeInfo, len(params))
	for i, t := range params {
		info.Params[i] = p.GetTypeInfo(ctx, typ, t, i+1)
	}
	return
}

func (p *Transformer) transformFuncType(
	ctx llvm.Context, info *FuncInfo,
) (llvm.Type, map[int][]llvm.Attribute, []int) {
	var paramTypes []llvm.Type
	var returnType llvm.Type
	attrs := make(map[int][]llvm.Attribute)
	addAttr := func(index int, attr llvm.Attribute) {
		attrs[index] = append(attrs[index], attr)
	}
	// paramMap maps each zero-based source parameter to its one-based
	// transformed LLVM attribute index. Zero means the parameter was elided.
	paramMap := make([]int, len(info.Params))
	switch info.Return.Kind {
	case AttrPointer:
		returnType = ctx.VoidType()
		paramTypes = append(paramTypes, info.Return.Type1)
		addAttr(1, sretAttribute(ctx, info.Return.nativeType()))
	case AttrWidthType:
		returnType = info.Return.Type1
	case AttrWidthType2:
		returnType = ctx.StructType([]llvm.Type{info.Return.Type1, info.Return.Type2}, false)
	default:
		returnType = info.Return.Type1
	}

	for i, ti := range info.Params {
		if ti.Kind != AttrVoid {
			paramMap[i] = len(paramTypes) + 1
		}
		switch ti.Kind {
		case AttrVoid:
			// skip
		case AttrNone, AttrWidthType:
			paramTypes = append(paramTypes, ti.Type1)
		case AttrPointer:
			paramTypes = append(paramTypes, ti.Type1)
			if p.sys.SupportByVal() {
				index := len(paramTypes)
				addAttr(index, byvalAttribute(ctx, ti.nativeType()))
				if ti.ByValAlign != 0 {
					addAttr(index, alignAttribute(ctx, ti.ByValAlign))
				}
			}
		case AttrWidthType2:
			paramTypes = append(paramTypes, ti.Type1, ti.Type2)
		case AttrExtract:
			subs := ti.nativeType().StructElementTypes()
			paramTypes = append(paramTypes, subs...)
		}
	}
	return llvm.FunctionType(returnType, paramTypes, info.Type.IsFunctionVarArg()), attrs, paramMap
}

func (p *Transformer) transformFunc(m llvm.Module, fn llvm.Value) bool {
	ctx := m.Context()
	if fn.IntrinsicID() != 0 {
		return false
	}
	info := p.GetFuncInfo(ctx, fn.GlobalValueType())
	if !info.HasWrap() {
		return false
	}
	nft, attrs, paramMap := p.transformFuncType(ctx, &info)
	preloweredSRet := fn.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("sret"))
	fname := fn.Name()
	fn.SetName("")
	nfn := llvm.AddFunction(m, fname, nft)
	for i, list := range attrs {
		for _, attr := range list {
			nfn.AddAttributeAtIndex(i, attr)
		}
	}
	copyClosureEnvFunctionAttrs(fn, nfn, paramMap)
	if !preloweredSRet.IsNil() {
		nfn.AddAttributeAtIndex(1, preloweredSRet)
	}
	nfn.SetLinkage(fn.Linkage())
	nfn.SetComdat(fn.Comdat())
	nfn.SetFunctionCallConv(fn.FunctionCallConv())
	for _, attr := range fn.GetFunctionAttributes() {
		nfn.AddAttributeAtIndex(-1, attr)
	}
	if err := funcattrs.RemapFunction(fn, nfn, attributeMapping(&info, paramMap)); err != nil {
		panic(err)
	}
	if sp := fn.Subprogram(); !sp.IsNil() {
		nfn.SetSubprogram(sp)
	}

	if !fn.IsDeclaration() {
		p.transformFuncBody(m, ctx, &info, fn, nfn, nft)
	}

	fn.ReplaceAllUsesWith(nfn)
	fn.EraseFromParentAsFunction()
	return true
}

func loadIndirectParam(b llvm.Builder, info *TypeInfo, ptr llvm.Value) llvm.Value {
	value := b.CreateLoad(info.nativeType(), ptr, "")
	if info.ByValAlign != 0 {
		value.SetAlignment(info.ByValAlign)
	}
	return value
}

// windows386AggregateToNative and windows386AggregateFromNative translate
// values between the explicit Go/386 field wrappers emitted by SSA and the
// natural aggregate layout consumed by native C code. They are no-ops for all
// other targets and for aggregates whose Go and C layouts already agree.
func windows386AggregateToNative(b llvm.Builder, value llvm.Value, source, native llvm.Type) llvm.Value {
	if source == native {
		return value
	}
	switch source.TypeKind() {
	case llvm.ArrayTypeKind:
		ret := llvm.Undef(native)
		for i := 0; i < source.ArrayLength(); i++ {
			item := b.CreateExtractValue(value, i, "")
			item = windows386AggregateToNative(b, item, source.ElementType(), native.ElementType())
			ret = b.CreateInsertValue(ret, item, i, "")
		}
		return ret
	case llvm.StructTypeKind:
		ret := llvm.Undef(native)
		sourceFields := source.StructElementTypes()
		nativeFields := native.StructElementTypes()
		nativeIndex := 0
		for sourceIndex, sourceField := range sourceFields {
			if nativeIndex == len(nativeFields) {
				if sourceIndex != len(sourceFields)-1 || !windows386GoAlignmentMarker(sourceField) {
					panic("cabi: invalid trailing Go/386 aggregate field")
				}
				break
			}
			item := b.CreateExtractValue(value, sourceIndex, "")
			if inner, wrapped := windows386GoFieldWrapper(sourceField); wrapped {
				item = b.CreateExtractValue(item, 0, "")
				sourceField = inner
			}
			item = windows386AggregateToNative(b, item, sourceField, nativeFields[nativeIndex])
			ret = b.CreateInsertValue(ret, item, nativeIndex, "")
			nativeIndex++
		}
		if nativeIndex != len(nativeFields) {
			panic("cabi: missing Go/386 aggregate field")
		}
		return ret
	default:
		panic(fmt.Sprintf("cabi: unsupported Go/386 aggregate layout conversion from %s (%v) to %s (%v)", source, source.TypeKind(), native, native.TypeKind()))
	}
}

func windows386AggregateFromNative(b llvm.Builder, value llvm.Value, source, native llvm.Type) llvm.Value {
	if source == native {
		return value
	}
	switch source.TypeKind() {
	case llvm.ArrayTypeKind:
		ret := llvm.Undef(source)
		for i := 0; i < source.ArrayLength(); i++ {
			item := b.CreateExtractValue(value, i, "")
			item = windows386AggregateFromNative(b, item, source.ElementType(), native.ElementType())
			ret = b.CreateInsertValue(ret, item, i, "")
		}
		return ret
	case llvm.StructTypeKind:
		ret := llvm.Undef(source)
		sourceFields := source.StructElementTypes()
		nativeFields := native.StructElementTypes()
		nativeIndex := 0
		for sourceIndex, sourceField := range sourceFields {
			if nativeIndex == len(nativeFields) {
				if sourceIndex != len(sourceFields)-1 || !windows386GoAlignmentMarker(sourceField) {
					panic("cabi: invalid trailing Go/386 aggregate field")
				}
				break
			}
			item := b.CreateExtractValue(value, nativeIndex, "")
			fieldNative := nativeFields[nativeIndex]
			if inner, wrapped := windows386GoFieldWrapper(sourceField); wrapped {
				item = windows386AggregateFromNative(b, item, inner, fieldNative)
				wrapper := llvm.Undef(sourceField)
				item = b.CreateInsertValue(wrapper, item, 0, "")
			} else {
				item = windows386AggregateFromNative(b, item, sourceField, fieldNative)
			}
			ret = b.CreateInsertValue(ret, item, sourceIndex, "")
			nativeIndex++
		}
		if nativeIndex != len(nativeFields) {
			panic("cabi: missing native Go/386 aggregate field")
		}
		return ret
	default:
		panic(fmt.Sprintf("cabi: unsupported native-to-Go/386 aggregate layout conversion from %s (%v) to %s (%v)", native, native.TypeKind(), source, source.TypeKind()))
	}
}

func windows386GoAlignmentMarker(typ llvm.Type) bool {
	return typ.TypeKind() == llvm.ArrayTypeKind && typ.ArrayLength() == 0 &&
		typ.ElementType().TypeKind() == llvm.IntegerTypeKind
}

func aggregateToNative(b llvm.Builder, info *TypeInfo, value llvm.Value) llvm.Value {
	return windows386AggregateToNative(b, value, info.Type, info.nativeType())
}

func aggregateFromNative(b llvm.Builder, info *TypeInfo, value llvm.Value) llvm.Value {
	return windows386AggregateFromNative(b, value, info.Type, info.nativeType())
}

func (p *Transformer) transformFuncBody(m llvm.Module, ctx llvm.Context, info *FuncInfo, fn llvm.Value, nfn llvm.Value, nft llvm.Type) {
	var blocks []llvm.BasicBlock
	bb := fn.FirstBasicBlock()
	for !bb.IsNil() {
		blocks = append(blocks, bb)
		bb = llvm.NextBasicBlock(bb)
	}
	for _, bb := range blocks {
		bb.RemoveFromParent()
		llvm.AppendExistingBasicBlock(nfn, bb)
	}

	b := ctx.NewBuilder()
	b.SetInsertPointBefore(nfn.EntryBasicBlock().FirstInstruction())

	params := nfn.Params()
	index := 0
	if info.Return.Kind == AttrPointer {
		index++
	}
	for i, ti := range info.Params {
		var nv llvm.Value
		switch ti.Kind {
		default:
			nv = params[index]
		case AttrVoid:
			nv = llvm.ConstNull(ti.Type)
			fn.Param(i).ReplaceAllUsesWith(nv)
			// skip
			continue
		case AttrPointer:
			// void @fn(%typ %0)
			// %1 = alloca %typ, align 8
			// call void @llvm.memset(ptr %1, i8 0, i64 36, i1 false)
			// store %typ %0, ptr %1, align 4
			//
			// void @fn(ptr byval(%typ) %0)
			// %1 = load %typ, ptr %0, align 4
			// %2 = alloca %typ, align 8
			// call void @llvm.memset(ptr %2, i8 0, i64 36, i1 false)
			// store %typ %1, ptr %2, align 4
			nv = aggregateFromNative(b, ti, loadIndirectParam(b, ti, params[index]))
			// Reuse the incoming ABI storage for one proven parameter home.
			if p.optimize && !ti.hasNativeLayoutConversion() {
				storageAlign := ti.Align
				if ti.ByValAlign != 0 {
					storageAlign = ti.ByValAlign
				}
				reuseParamHome(fn.Param(i), params[index], nfn.EntryBasicBlock(), ti.Align, storageAlign)
			}
		case AttrWidthType:
			iptr := llvm.CreateAlloca(b, ti.Type1)
			storageAlign := max(ti.Align, p.Alignof(ti.Type1))
			iptr.SetAlignment(storageAlign)
			b.CreateStore(params[index], iptr)
			ptr := b.CreateBitCast(iptr, llvm.PointerType(ti.nativeType(), 0), "")
			nv = aggregateFromNative(b, ti, b.CreateLoad(ti.nativeType(), ptr, ""))
			if p.optimize && !ti.hasNativeLayoutConversion() {
				reuseParamHome(fn.Param(i), ptr, nfn.EntryBasicBlock(), ti.Align, storageAlign)
			}
		case AttrWidthType2:
			typ := ctx.StructType([]llvm.Type{ti.Type1, ti.Type2}, false)
			iptr := llvm.CreateAlloca(b, typ)
			storageAlign := max(ti.Align, p.Alignof(typ))
			iptr.SetAlignment(storageAlign)
			b.CreateStore(params[index], b.CreateStructGEP(typ, iptr, 0, ""))
			index++
			b.CreateStore(params[index], b.CreateStructGEP(typ, iptr, 1, ""))
			ptr := b.CreateBitCast(iptr, llvm.PointerType(ti.nativeType(), 0), "")
			nv = aggregateFromNative(b, ti, b.CreateLoad(ti.nativeType(), ptr, ""))
			if p.optimize && !ti.hasNativeLayoutConversion() {
				reuseParamHome(fn.Param(i), ptr, nfn.EntryBasicBlock(), ti.Align, storageAlign)
			}
		case AttrExtract:
			nsubs := ti.nativeType().StructElementTypesCount()
			nv = llvm.Undef(ti.nativeType())
			for i := 0; i < nsubs; i++ {
				nv = b.CreateInsertValue(nv, params[index], i, "")
				index++
			}
			nv = aggregateFromNative(b, ti, nv)
			fn.Param(i).ReplaceAllUsesWith(nv)
			continue
		}
		fn.Param(i).ReplaceAllUsesWith(nv)
		index++
	}

	voidAggregateReturn := info.Return.Kind == AttrVoid && info.Return.Type.TypeKind() != llvm.VoidTypeKind
	if info.Return.Kind >= AttrPointer || voidAggregateReturn {
		var retInstrs []llvm.Value
		bb := nfn.FirstBasicBlock()
		for !bb.IsNil() {
			instr := bb.FirstInstruction()
			for !instr.IsNil() {
				if !instr.IsAReturnInst().IsNil() {
					retInstrs = append(retInstrs, instr)
				}
				instr = llvm.NextInstruction(instr)
			}
			bb = llvm.NextBasicBlock(bb)
		}
		for _, instr := range retInstrs {
			ret := instr.Operand(0)
			b.SetInsertPointBefore(instr)
			nativeRet := aggregateToNative(b, info.Return, ret)
			var rv llvm.Value
			switch info.Return.Kind {
			case AttrVoid:
				rv = b.CreateRetVoid()
			case AttrPointer:
				// %typ @fn()
				// %2 = load %typ, ptr %1
				// ret %typ %2
				//
				// void @fn(ptr sret(%typ) %0)
				// %2 = load %typ, ptr %1
				// store %typ %2, ptr %0
				// ret void
				//
				// Note: We don't use memcpy optimization here because the source
				// address content may be modified between load and ret.
				// See: https://github.com/xgo-dev/llgo/issues/1608
				b.CreateStore(nativeRet, params[0])
				rv = b.CreateRetVoid()
			case AttrWidthType:
				if p.optimize && !info.Return.hasNativeLayoutConversion() {
					if load := ret.IsALoadInst(); !load.IsNil() && !load.IsVolatile() && llvm.NextInstruction(load) == instr {
						iptr := b.CreateBitCast(ret.Operand(0), llvm.PointerType(nft.ReturnType(), 0), "")
						value := b.CreateLoad(nft.ReturnType(), iptr, "")
						value.SetAlignment(load.Alignment())
						rv = b.CreateRet(value)
						break
					}
				}
				// Materialize the saved SSA value. The return may be a load whose
				// source was modified after that load but before the return.
				ptr := llvm.CreateAlloca(b, info.Return.nativeType())
				b.CreateStore(nativeRet, ptr)
				iptr := b.CreateBitCast(ptr, llvm.PointerType(nft.ReturnType(), 0), "")
				rv = b.CreateRet(b.CreateLoad(nft.ReturnType(), iptr, ""))
			case AttrWidthType2:
				ptr := llvm.CreateAlloca(b, info.Return.nativeType())
				b.CreateStore(nativeRet, ptr)
				iptr := b.CreateBitCast(ptr, llvm.PointerType(nft.ReturnType(), 0), "")
				rv = b.CreateRet(b.CreateLoad(nft.ReturnType(), iptr, ""))
			}
			instr.ReplaceAllUsesWith(rv)
			instr.EraseFromParentAsInstruction()
		}
	}
}

func (p *Transformer) transformCallInstr(m llvm.Module, ctx llvm.Context, call llvm.Value, fn llvm.Value) bool {
	nfn := call.CalledValue()
	if nfn.IntrinsicID() != 0 {
		return false
	}
	info := p.GetFuncInfo(ctx, call.CalledFunctionType())
	if !info.HasWrap() {
		return false
	}
	nft, attrs, paramMap := p.transformFuncType(ctx, &info)
	preloweredSRet := call.GetCallSiteEnumAttribute(1, llvm.AttributeKindID("sret"))
	reflectMethodByNameAttr := call.GetCallSiteStringAttribute(-1, "llgo.reflect.methodbyname")
	b := ctx.NewBuilder()
	b.SetInsertPointBefore(call)

	first := fn.EntryBasicBlock().FirstInstruction()
	createAlloca := func(t llvm.Type) (ret llvm.Value) {
		b.SetInsertPointBefore(first)
		ret = llvm.CreateAlloca(b, t)
		b.SetInsertPointBefore(call)
		return
	}

	operandCount := len(info.Params)
	returnArgOffset := 0
	if info.Return.Kind == AttrPointer {
		returnArgOffset = 1
	}
	remappedReflectMethodByNameArgAttrIndex := -1
	var nparams []llvm.Value
	for i := 0; i < operandCount; i++ {
		param := call.Operand(i)
		ti := info.Params[i]
		reflectMethodByNameArgAttr := call.GetCallSiteStringAttribute(i+1, "llgo.reflect.methodbyname.name")
		if !reflectMethodByNameArgAttr.IsNil() {
			remappedReflectMethodByNameArgAttrIndex = returnArgOffset + len(nparams) + 1
		}
		switch ti.Kind {
		default:
			nparams = append(nparams, param)
		case AttrVoid:
			// none
		case AttrPointer:
			// Do not pass a load's source pointer directly. The source memory may
			// be modified between the load and this call; pass the loaded value.
			param = aggregateToNative(b, ti, param)
			ptr := createAlloca(ti.nativeType())
			b.CreateStore(param, ptr)
			nparams = append(nparams, ptr)
		case AttrWidthType:
			param = aggregateToNative(b, ti, param)
			ptr := createAlloca(ti.nativeType())
			b.CreateStore(param, ptr)
			iptr := b.CreateBitCast(ptr, llvm.PointerType(ti.Type1, 0), "")
			nparams = append(nparams, b.CreateLoad(ti.Type1, iptr, ""))
		case AttrWidthType2:
			param = aggregateToNative(b, ti, param)
			ptr := createAlloca(ti.nativeType())
			b.CreateStore(param, ptr)
			typ := ctx.StructType([]llvm.Type{ti.Type1, ti.Type2}, false) // {i8,i64}
			iptr := b.CreateBitCast(ptr, llvm.PointerType(typ, 0), "")
			nparams = append(nparams, b.CreateLoad(ti.Type1, b.CreateStructGEP(typ, iptr, 0, ""), ""))
			nparams = append(nparams, b.CreateLoad(ti.Type2, b.CreateStructGEP(typ, iptr, 1, ""), ""))
		case AttrExtract:
			param = aggregateToNative(b, ti, param)
			nsubs := ti.nativeType().StructElementTypesCount()
			for i := 0; i < nsubs; i++ {
				nparams = append(nparams, b.CreateExtractValue(param, i, ""))
			}
		}
	}
	if info.Type.IsFunctionVarArg() {
		// CallBase stores the callee as its final operand. LLGo does not emit
		// operand bundles here, so the operands between the fixed parameters
		// and the callee are exactly the already-promoted C varargs.
		for i, n := operandCount, call.OperandsCount()-1; i < n; i++ {
			nparams = append(nparams, call.Operand(i))
		}
	}

	updateCallAttr := func(replacement llvm.Value) {
		replacement.SetInstructionCallConv(call.InstructionCallConv())
		for i, list := range attrs {
			for _, attr := range list {
				replacement.AddCallSiteAttribute(i, attr)
			}
		}
		if !preloweredSRet.IsNil() {
			replacement.AddCallSiteAttribute(1, preloweredSRet)
		}
		if !reflectMethodByNameAttr.IsNil() {
			replacement.AddCallSiteAttribute(-1, reflectMethodByNameAttr)
		}
		if remappedReflectMethodByNameArgAttrIndex >= 0 {
			replacement.AddCallSiteAttribute(remappedReflectMethodByNameArgAttrIndex, ctx.CreateStringAttribute(
				"llgo.reflect.methodbyname.name", "1",
			))
		}
		copyClosureEnvCallAttrs(call, replacement, paramMap)
		if err := funcattrs.RemapCall(call, replacement, attributeMapping(&info, paramMap)); err != nil {
			panic(err)
		}
	}

	var instr llvm.Value
	switch info.Return.Kind {
	case AttrVoid:
		loweredCall := llvm.CreateCall(b, nft, nfn, nparams)
		updateCallAttr(loweredCall)
		if info.Return.Type.TypeKind() == llvm.VoidTypeKind {
			instr = loweredCall
		} else {
			// The target ABI omits zero-sized aggregate results. Preserve the
			// original SSA value for users even though no value crosses the ABI.
			instr = llvm.ConstNull(info.Return.Type)
		}
	case AttrPointer:
		ret := createAlloca(info.Return.nativeType())
		call := llvm.CreateCall(b, nft, nfn, append([]llvm.Value{ret}, nparams...))
		updateCallAttr(call)
		instr = aggregateFromNative(b, info.Return, b.CreateLoad(info.Return.nativeType(), ret, ""))
	case AttrWidthType, AttrWidthType2:
		ret := llvm.CreateCall(b, nft, nfn, nparams)
		updateCallAttr(ret)
		ptr := createAlloca(nft.ReturnType())
		b.CreateStore(ret, ptr)
		pret := b.CreateBitCast(ptr, llvm.PointerType(info.Return.nativeType(), 0), "")
		instr = aggregateFromNative(b, info.Return, b.CreateLoad(info.Return.nativeType(), pret, ""))
	default:
		instr = llvm.CreateCall(b, nft, nfn, nparams)
		updateCallAttr(instr)
	}
	call.ReplaceAllUsesWith(instr)
	call.EraseFromParentAsInstruction()
	return true
}

var closureEnvAttributeKinds = []uint{
	llvm.AttributeKindID("nest"),
	llvm.AttributeKindID("swiftself"),
}

func copyClosureEnvFunctionAttrs(from, to llvm.Value, paramMap []int) {
	for oldIndex, newIndex := range paramMap {
		if newIndex == 0 {
			continue
		}
		for _, kind := range closureEnvAttributeKinds {
			if attr := from.GetEnumAttributeAtIndex(oldIndex+1, kind); !attr.IsNil() {
				to.AddAttributeAtIndex(newIndex, attr)
			}
		}
	}
}

func copyClosureEnvCallAttrs(from, to llvm.Value, paramMap []int) {
	for oldIndex, newIndex := range paramMap {
		if newIndex == 0 {
			continue
		}
		for _, kind := range closureEnvAttributeKinds {
			if attr := from.GetCallSiteEnumAttribute(oldIndex+1, kind); !attr.IsNil() {
				to.AddCallSiteAttribute(newIndex, attr)
			}
		}
	}
}

func (p *Transformer) callMemcpy(_ llvm.Module, ctx llvm.Context, b llvm.Builder, dst llvm.Value, src llvm.Value, size int) llvm.Value {
	sz := llvm.ConstInt(ctx.IntType(p.prog.PointerSize()*8), uint64(size), false)
	return b.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.memcpy"), []llvm.Value{
		dst, src, sz, llvm.ConstInt(ctx.Int1Type(), 0, false),
	}, "")
}

// reuseParamHome replaces at most one local copy of param with storage. The
// incoming ABI storage can stand in for one parameter home, but independent
// copies must remain distinct objects.
func reuseParamHome(param, storage llvm.Value, entry llvm.BasicBlock, naturalAlign, storageAlign int) {
	seen := make(map[llvm.Value]bool)
	for instr := entry.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
		store := instr.IsAStoreInst()
		if store.IsNil() || store.Operand(0) != param {
			continue
		}
		alloc := store.Operand(1).IsAAllocaInst()
		if alloc.IsNil() || seen[alloc] {
			continue
		}
		seen[alloc] = true
		if canReuseParamHome(alloc, store, entry, param.Type(), naturalAlign, storageAlign) {
			replaceParamHomeUses(alloc, storage, store)
			return
		}
	}
}

func canReuseParamHome(alloc, initStore llvm.Value, entry llvm.BasicBlock, paramType llvm.Type, naturalAlign, storageAlign int) bool {
	if alloc.InstructionParent() != entry || initStore.InstructionParent() != entry {
		return false
	}
	if alloc.AllocatedType() != paramType {
		return false
	}
	count := alloc.Operand(0).IsAConstantInt()
	if count.IsNil() || count.ZExtValue() != 1 {
		return false
	}
	allocAlign := alloc.Alignment()
	if allocAlign == 0 {
		allocAlign = naturalAlign
	}
	if storageAlign < allocAlign {
		return false
	}

	// The initialization in the entry block dominates every later block. In
	// the entry prefix, only a direct memset is harmless: it neither exposes a
	// pointer that would keep referring to the old object nor observes the
	// parameter value that begins at initStore.
	prefix := make(map[llvm.Value]bool)
	foundInit := false
	for instr := llvm.NextInstruction(alloc); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
		if instr == initStore {
			foundInit = true
			break
		}
		prefix[instr] = true
	}
	if !foundInit {
		return false
	}
	for use := alloc.FirstUse(); !use.IsNil(); use = use.NextUse() {
		user := use.User()
		if prefix[user] && !isDirectMemset(user, alloc) {
			return false
		}
	}
	return true
}

func isDirectMemset(instr, ptr llvm.Value) bool {
	call := instr.IsACallInst()
	return !call.IsNil() && call.CalledValue().IntrinsicID() == llvm.LookupIntrinsicID("llvm.memset") && call.Operand(0) == ptr
}

func replaceParamHomeUses(alloc, storage, initStore llvm.Value) {
	prefix := make(map[llvm.Value]bool)
	for instr := llvm.NextInstruction(alloc); !instr.IsNil() && instr != initStore; instr = llvm.NextInstruction(instr) {
		prefix[instr] = true
	}
	var users []llvm.Value
	for use := alloc.FirstUse(); !use.IsNil(); use = use.NextUse() {
		user := use.User()
		if prefix[user] {
			continue
		}
		users = append(users, user)
	}
	for _, user := range users {
		for i, n := 0, user.OperandsCount(); i < n; i++ {
			if user.Operand(i) == alloc {
				user.SetOperand(i, storage)
			}
		}
	}
}
