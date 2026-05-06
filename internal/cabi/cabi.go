package cabi

import (
	"strings"

	"github.com/goplus/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

type Mode int

const (
	ModeNone Mode = iota
	ModeCFunc
	ModeAllFunc
)

func targetArch(llvmTarget string) string {
	if pos := strings.Index(llvmTarget, "-"); pos != -1 {
		return llvmTarget[:pos]
	}
	return llvmTarget
}

func NewTransformer(prog ssa.Program, llvmTarget string, targetAbi string, mode Mode, optimize bool) *Transformer {
	target := prog.Target()
	arch := target.GOARCH
	if llvmTarget != "" {
		arch = targetArch(llvmTarget)
	}
	tr := &Transformer{
		prog:     prog,
		td:       prog.TargetData(),
		arch:     arch,
		mode:     mode,
		optimize: optimize,
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
	mode     Mode
	optimize bool
	skipFns  map[string]struct{}
}

func (p *Transformer) isCFunc(name string) bool {
	return !strings.Contains(name, ".")
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
	switch p.mode {
	case ModeNone:
		return
	case ModeCFunc:
		fn := m.FirstFunction()
		for !fn.IsNil() {
			if !p.shouldSkipFunc(fn.Name()) && p.isCFunc(fn.Name()) {
				p.transformFuncCall(m, fn)
				if p.isWrapFunctionType(ctx, fn.GlobalValueType()) {
					fns = append(fns, fn)
				}
			}
			bb := fn.FirstBasicBlock()
			for !bb.IsNil() {
				instr := bb.FirstInstruction()
				for !instr.IsNil() {
					if call := instr.IsACallInst(); !call.IsNil() {
						if p.shouldSkipCall(call) {
							instr = llvm.NextInstruction(instr)
							continue
						}
						callee := call.CalledValue()
						// ModeCFunc only targets direct C symbol calls. Indirect calls
						// (callee name is empty under opaque pointers) may be Go closure
						// invocations and must keep Go-level signatures.
						if callee.IsAFunction().IsNil() || !p.isCFunc(callee.Name()) {
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
	case ModeAllFunc:
		fn := m.FirstFunction()
		for !fn.IsNil() {
			if !p.shouldSkipFunc(fn.Name()) && p.isWrapFunctionType(ctx, fn.GlobalValueType()) {
				fns = append(fns, fn)
			}
			bb := fn.FirstBasicBlock()
			for !bb.IsNil() {
				instr := bb.FirstInstruction()
				for !instr.IsNil() {
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
	Type  llvm.Type
	Kind  AttrKind
	Type1 llvm.Type // AttrWidthType
	Type2 llvm.Type // AttrWidthType2
	Size  int
	Align int
}

func byvalAttribute(ctx llvm.Context, typ llvm.Type) llvm.Attribute {
	id := llvm.AttributeKindID("byval")
	return ctx.CreateTypeAttribute(id, typ)
}

func sretAttribute(ctx llvm.Context, typ llvm.Type) llvm.Attribute {
	id := llvm.AttributeKindID("sret")
	return ctx.CreateTypeAttribute(id, typ)
}

func funcInlineHint(ctx llvm.Context) llvm.Attribute {
	return ctx.CreateEnumAttribute(llvm.AttributeKindID("inlinehint"), 0)
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

func (p *Transformer) transformFuncType(ctx llvm.Context, info *FuncInfo) (llvm.Type, map[int]llvm.Attribute) {
	var paramTypes []llvm.Type
	var returnType llvm.Type
	attrs := make(map[int]llvm.Attribute)
	switch info.Return.Kind {
	case AttrPointer:
		returnType = ctx.VoidType()
		paramTypes = append(paramTypes, info.Return.Type1)
		attrs[1] = sretAttribute(ctx, info.Return.Type)
	case AttrWidthType:
		returnType = info.Return.Type1
	case AttrWidthType2:
		returnType = ctx.StructType([]llvm.Type{info.Return.Type1, info.Return.Type2}, false)
	default:
		returnType = info.Return.Type1
	}

	for _, ti := range info.Params {
		switch ti.Kind {
		case AttrVoid:
			// skip
		case AttrNone, AttrWidthType:
			paramTypes = append(paramTypes, ti.Type1)
		case AttrPointer:
			paramTypes = append(paramTypes, ti.Type1)
			if p.sys.SupportByVal() {
				attrs[len(paramTypes)] = byvalAttribute(ctx, ti.Type)
			}
		case AttrWidthType2:
			paramTypes = append(paramTypes, ti.Type1, ti.Type2)
		case AttrExtract:
			subs := ti.Type.StructElementTypes()
			paramTypes = append(paramTypes, subs...)
		}
	}
	return llvm.FunctionType(returnType, paramTypes, info.Type.IsFunctionVarArg()), attrs
}

func (p *Transformer) transformFunc(m llvm.Module, fn llvm.Value) bool {
	ctx := m.Context()
	info := p.GetFuncInfo(ctx, fn.GlobalValueType())
	if !info.HasWrap() {
		return false
	}
	nft, attrs := p.transformFuncType(ctx, &info)
	fname := fn.Name()
	fn.SetName("")
	nfn := llvm.AddFunction(m, fname, nft)
	for i, attr := range attrs {
		nfn.AddAttributeAtIndex(i, attr)
	}
	nfn.SetLinkage(fn.Linkage())
	nfn.SetFunctionCallConv(fn.FunctionCallConv())
	for _, attr := range fn.GetFunctionAttributes() {
		nfn.AddAttributeAtIndex(-1, attr)
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
			nv = b.CreateLoad(ti.Type, params[index], "")
			// replace %0 to %2
			if p.optimize {
				replaceAllocaInstrs(fn.Param(i), params[index])
			}
		case AttrWidthType:
			iptr := llvm.CreateAlloca(b, ti.Type1)
			b.CreateStore(params[index], iptr)
			ptr := b.CreateBitCast(iptr, llvm.PointerType(ti.Type, 0), "")
			nv = b.CreateLoad(ti.Type, ptr, "")
			if p.optimize {
				replaceAllocaInstrs(fn.Param(i), ptr)
			}
		case AttrWidthType2:
			typ := ctx.StructType([]llvm.Type{ti.Type1, ti.Type2}, false)
			iptr := llvm.CreateAlloca(b, typ)
			b.CreateStore(params[index], b.CreateStructGEP(typ, iptr, 0, ""))
			index++
			b.CreateStore(params[index], b.CreateStructGEP(typ, iptr, 1, ""))
			ptr := b.CreateBitCast(iptr, llvm.PointerType(ti.Type, 0), "")
			nv = b.CreateLoad(ti.Type, ptr, "")
			if p.optimize {
				replaceAllocaInstrs(fn.Param(i), ptr)
			}
		case AttrExtract:
			nsubs := ti.Type.StructElementTypesCount()
			nv = llvm.Undef(ti.Type)
			for i := 0; i < nsubs; i++ {
				nv = b.CreateInsertValue(nv, params[index], i, "")
				index++
			}
			fn.Param(i).ReplaceAllUsesWith(nv)
			continue
		}
		fn.Param(i).ReplaceAllUsesWith(nv)
		index++
	}

	if info.Return.Kind >= AttrPointer {
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
			var rv llvm.Value
			switch info.Return.Kind {
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
				// See: https://github.com/goplus/llgo/issues/1608
				b.CreateStore(ret, params[0])
				rv = b.CreateRetVoid()
			case AttrWidthType:
				if p.optimize {
					if load := ret.IsALoadInst(); !load.IsNil() {
						iptr := b.CreateBitCast(ret.Operand(0), llvm.PointerType(nft.ReturnType(), 0), "")
						rv = b.CreateRet(b.CreateLoad(nft.ReturnType(), iptr, ""))
						break
					}
				}
				ptr := llvm.CreateAlloca(b, info.Return.Type)
				b.CreateStore(ret, ptr)
				iptr := b.CreateBitCast(ptr, llvm.PointerType(nft.ReturnType(), 0), "")
				rv = b.CreateRet(b.CreateLoad(nft.ReturnType(), iptr, ""))
			case AttrWidthType2:
				ptr := llvm.CreateAlloca(b, info.Return.Type)
				b.CreateStore(ret, ptr)
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
	info := p.GetFuncInfo(ctx, call.CalledFunctionType())
	if !info.HasWrap() {
		return false
	}
	nft, attrs := p.transformFuncType(ctx, &info)
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
	var nparams []llvm.Value
	for i := 0; i < operandCount; i++ {
		param := call.Operand(i)
		ti := info.Params[i]
		switch ti.Kind {
		default:
			nparams = append(nparams, param)
		case AttrVoid:
			// none
		case AttrPointer:
			// Do not pass a load's source pointer directly. The source memory may
			// be modified between the load and this call; pass the loaded value.
			ptr := createAlloca(ti.Type)
			b.CreateStore(param, ptr)
			nparams = append(nparams, ptr)
		case AttrWidthType:
			ptr := createAlloca(ti.Type)
			b.CreateStore(param, ptr)
			iptr := b.CreateBitCast(ptr, llvm.PointerType(ti.Type1, 0), "")
			nparams = append(nparams, b.CreateLoad(ti.Type1, iptr, ""))
		case AttrWidthType2:
			ptr := createAlloca(ti.Type)
			b.CreateStore(param, ptr)
			typ := ctx.StructType([]llvm.Type{ti.Type1, ti.Type2}, false) // {i8,i64}
			iptr := b.CreateBitCast(ptr, llvm.PointerType(typ, 0), "")
			nparams = append(nparams, b.CreateLoad(ti.Type1, b.CreateStructGEP(typ, iptr, 0, ""), ""))
			nparams = append(nparams, b.CreateLoad(ti.Type2, b.CreateStructGEP(typ, iptr, 1, ""), ""))
		case AttrExtract:
			nsubs := ti.Type.StructElementTypesCount()
			for i := 0; i < nsubs; i++ {
				nparams = append(nparams, b.CreateExtractValue(param, i, ""))
			}
		}
	}

	updateCallAttr := func(call llvm.Value) {
		for i, attr := range attrs {
			call.AddCallSiteAttribute(i, attr)
		}
	}

	var instr llvm.Value
	switch info.Return.Kind {
	case AttrVoid:
		instr = llvm.CreateCall(b, nft, nfn, nparams)
		updateCallAttr(instr)
	case AttrPointer:
		ret := createAlloca(info.Return.Type)
		call := llvm.CreateCall(b, nft, nfn, append([]llvm.Value{ret}, nparams...))
		updateCallAttr(call)
		instr = b.CreateLoad(info.Return.Type, ret, "")
	case AttrWidthType, AttrWidthType2:
		ret := llvm.CreateCall(b, nft, nfn, nparams)
		updateCallAttr(ret)
		ptr := createAlloca(nft.ReturnType())
		b.CreateStore(ret, ptr)
		pret := b.CreateBitCast(ptr, llvm.PointerType(info.Return.Type, 0), "")
		instr = b.CreateLoad(info.Return.Type, pret, "")
	default:
		instr = llvm.CreateCall(b, nft, nfn, nparams)
		updateCallAttr(instr)
	}
	call.ReplaceAllUsesWith(instr)
	call.EraseFromParentAsInstruction()
	return true
}

func (p *Transformer) transformFuncCall(m llvm.Module, fn llvm.Value) {
	u := fn.FirstUse()
	ctx := m.Context()
	for !u.IsNil() {
		if call := u.User().IsACallInst(); !call.IsNil() {
			n := call.OperandsCount()
			for i := 0; i < n; i++ {
				op := call.Operand(i)
				if op == fn {
					continue
				}
				if gv := op.IsAGlobalValue(); !gv.IsNil() {
					if ft := gv.GlobalValueType(); ft.TypeKind() == llvm.FunctionTypeKind {
						if p.isCFunc(gv.Name()) {
							continue
						}
						if p.isWrapFunctionType(ctx, ft) {
							if wrap, ok := p.transformCallbackFunc(m, gv); ok {
								call.SetOperand(i, wrap)
							}
						}
					}
				}
			}
		}
		u = u.NextUse()
	}
}

func (p *Transformer) transformCallbackFunc(m llvm.Module, fn llvm.Value) (wrap llvm.Value, ok bool) {
	ctx := m.Context()
	info := p.GetFuncInfo(ctx, fn.GlobalValueType())
	if !info.HasWrap() {
		return fn, false
	}

	nft, attrs := p.transformFuncType(ctx, &info)

	fname := fn.Name()
	wrapName := "__llgo_cdecl$" + fname
	if wrapFunc := m.NamedFunction(wrapName); !wrapFunc.IsNil() {
		return wrapFunc, true
	}
	wrapFunc := llvm.AddFunction(m, wrapName, nft)
	wrapFunc.SetLinkage(llvm.LinkOnceAnyLinkage)
	wrapFunc.AddFunctionAttr(funcInlineHint(ctx))

	for i, attr := range attrs {
		wrapFunc.AddAttributeAtIndex(i, attr)
	}

	b := ctx.NewBuilder()
	block := ctx.AddBasicBlock(wrapFunc, "entry")
	b.SetInsertPointAtEnd(block)

	var nparams []llvm.Value
	params := wrapFunc.Params()
	index := 0
	if info.Return.Kind == AttrPointer {
		index++
	}
	for _, ti := range info.Params {
		switch ti.Kind {
		default:
		case AttrVoid:
			// none
		case AttrPointer:
			nparams = append(nparams, b.CreateLoad(ti.Type, params[index], ""))
		case AttrWidthType:
			iptr := llvm.CreateAlloca(b, ti.Type1)
			b.CreateStore(params[index], iptr)
			ptr := b.CreateBitCast(iptr, llvm.PointerType(ti.Type, 0), "")
			nparams = append(nparams, b.CreateLoad(ti.Type, ptr, ""))
		case AttrWidthType2:
			typ := ctx.StructType([]llvm.Type{ti.Type1, ti.Type2}, false)
			iptr := llvm.CreateAlloca(b, typ)
			b.CreateStore(params[index], b.CreateStructGEP(typ, iptr, 0, ""))
			index++
			b.CreateStore(params[index], b.CreateStructGEP(typ, iptr, 1, ""))
			ptr := b.CreateBitCast(iptr, llvm.PointerType(ti.Type, 0), "")
			nparams = append(nparams, b.CreateLoad(ti.Type, ptr, ""))
		case AttrExtract:
			nsubs := ti.Type.StructElementTypesCount()
			nv := llvm.Undef(ti.Type)
			for i := 0; i < nsubs; i++ {
				nv = b.CreateInsertValue(nv, params[index], i, "")
				index++
			}
			nparams = append(nparams, nv)
			continue
		}
		index++
	}

	switch info.Return.Kind {
	case AttrVoid:
		llvm.CreateCall(b, info.Type, fn, nparams)
		b.CreateRetVoid()
	case AttrPointer:
		ret := llvm.CreateCall(b, info.Type, fn, nparams)
		b.CreateStore(ret, params[0])
		b.CreateRetVoid()
	case AttrWidthType, AttrWidthType2:
		ret := llvm.CreateCall(b, info.Type, fn, nparams)
		ptr := llvm.CreateAlloca(b, info.Return.Type)
		b.CreateStore(ret, ptr)
		returnType := nft.ReturnType()
		iptr := b.CreateBitCast(ptr, llvm.PointerType(returnType, 0), "")
		b.CreateRet(b.CreateLoad(returnType, iptr, ""))
	default:
		ret := llvm.CreateCall(b, info.Type, fn, nparams)
		b.CreateRet(ret)
	}
	return wrapFunc, true
}

func (p *Transformer) callMemcpy(m llvm.Module, ctx llvm.Context, b llvm.Builder, dst llvm.Value, src llvm.Value, size int) llvm.Value {
	memcpy := p.getMemcpy(m, ctx)
	sz := llvm.ConstInt(ctx.IntType(p.prog.PointerSize()*8), uint64(size), false)
	return b.CreateCall(memcpy.GlobalValueType(), memcpy, []llvm.Value{
		dst, src, sz, llvm.ConstInt(ctx.Int1Type(), 0, false),
	}, "")
}

func (p *Transformer) getMemcpy(m llvm.Module, ctx llvm.Context) llvm.Value {
	memcpy := m.NamedFunction("llvm.memcpy")
	if !memcpy.IsNil() {
		return memcpy
	}
	ftyp := llvm.FunctionType(ctx.VoidType(), []llvm.Type{
		llvm.PointerType(ctx.Int8Type(), 0),
		llvm.PointerType(ctx.Int8Type(), 0),
		ctx.IntType(p.prog.PointerSize() * 8),
		ctx.Int1Type(),
	}, false)
	memcpy = llvm.AddFunction(m, "llvm.memcpy", ftyp)
	memcpy.SetFunctionCallConv(llvm.CCallConv)
	memcpy.AddFunctionAttr(funcNoUnwind(ctx))
	return memcpy
}

func replaceAllocaInstrs(param llvm.Value, nv llvm.Value) {
	u := param.FirstUse()
	var storeInstrs []llvm.Value
	for !u.IsNil() {
		if user := u.User().IsAStoreInst(); !user.IsNil() && user.Operand(0) == param {
			storeInstrs = append(storeInstrs, user)
		}
		u = u.NextUse()
	}
	for _, instr := range storeInstrs {
		if alloc := instr.Operand(1).IsAAllocaInst(); !alloc.IsNil() {
			skips := make(map[llvm.Value]bool)
			next := llvm.NextInstruction(alloc)
			for !next.IsNil() && next != instr {
				skips[next] = true
				next = llvm.NextInstruction(next)
			}
			var uses []llvm.Value
			u := alloc.FirstUse()
			for !u.IsNil() {
				if v := u.User(); !skips[v] {
					uses = append(uses, v)
				}
				u = u.NextUse()
			}
			for _, use := range uses {
				n := use.OperandsCount()
				for i := 0; i < n; i++ {
					if use.Operand(i) == alloc {
						use.SetOperand(i, nv)
					}
				}
			}
		}
	}
}
