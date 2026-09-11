package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/xgo-dev/llvm"
)

type wasmReflectBridgePair struct {
	call Function
	make Function
}

// wasmReflectBridge returns package-local bridges shared by every function
// type with the same lowered WebAssembly signature and GC-root extraction
// shape. Signedness, names, and other source-only distinctions deliberately do
// not create more code.
func (p Package) wasmReflectBridge(sig *types.Signature) wasmReflectBridgePair {
	rawSig := p.Prog.gocvt.cvtFunc(sig, nil)
	key := p.Prog.wasmReflectBridgeShape(rawSig)
	if bridges, ok := p.wasmReflectBridges[key]; ok {
		return bridges
	}
	if p.wasmReflectBridges == nil {
		p.wasmReflectBridges = make(map[string]wasmReflectBridgePair)
	}
	id := compactWasmBridgeID(len(p.wasmReflectBridges))
	bridges := wasmReflectBridgePair{
		call: p.newWasmReflectCallBridge(rawSig, "$c"+id),
		make: p.newWasmReflectMakeBridge(sig, "$m"+id),
	}
	p.wasmReflectBridges[key] = bridges
	return bridges
}

func compactWasmBridgeID(value int) string {
	return compactWasmBridgeUint64(uint64(value))
}

func compactWasmBridgeUint64(value uint64) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	base := uint64(len(alphabet))
	var reversed [12]byte
	i := len(reversed)
	for {
		i--
		reversed[i] = alphabet[value%base]
		value = value/base - 1
		if value == ^uint64(0) {
			return string(reversed[i:])
		}
	}
}

func (p Program) wasmReflectBridgeShape(sig *types.Signature) string {
	var shape strings.Builder
	appendWasmLLVMTypeShape(&shape, p.toLLVMFuncBackground(sig, InC))
	shape.WriteByte('/')
	for i := 0; i < sig.Params().Len(); i++ {
		p.appendWasmRootShape(&shape, p.Type(sig.Params().At(i).Type(), InC))
		shape.WriteByte(';')
	}
	return shape.String()
}

func appendWasmLLVMTypeShape(shape *strings.Builder, typ llvm.Type) {
	switch typ.TypeKind() {
	case llvm.VoidTypeKind:
		shape.WriteByte('v')
	case llvm.FloatTypeKind:
		shape.WriteByte('f')
	case llvm.DoubleTypeKind:
		shape.WriteByte('d')
	case llvm.X86_FP80TypeKind:
		shape.WriteString("x80")
	case llvm.FP128TypeKind:
		shape.WriteString("f128")
	case llvm.PPC_FP128TypeKind:
		shape.WriteString("p128")
	case llvm.IntegerTypeKind:
		shape.WriteByte('i')
		shape.WriteString(strconv.Itoa(typ.IntTypeWidth()))
	case llvm.FunctionTypeKind:
		shape.WriteByte('(')
		for _, param := range typ.ParamTypes() {
			appendWasmLLVMTypeShape(shape, param)
			shape.WriteByte(',')
		}
		if typ.IsFunctionVarArg() {
			shape.WriteByte('*')
		}
		shape.WriteByte(')')
		appendWasmLLVMTypeShape(shape, typ.ReturnType())
	case llvm.StructTypeKind:
		if typ.IsStructPacked() {
			shape.WriteByte('<')
		} else {
			shape.WriteByte('{')
		}
		for _, elem := range typ.StructElementTypes() {
			appendWasmLLVMTypeShape(shape, elem)
			shape.WriteByte(',')
		}
		if typ.IsStructPacked() {
			shape.WriteByte('>')
		} else {
			shape.WriteByte('}')
		}
	case llvm.ArrayTypeKind:
		shape.WriteByte('[')
		shape.WriteString(strconv.Itoa(typ.ArrayLength()))
		shape.WriteByte(':')
		appendWasmLLVMTypeShape(shape, typ.ElementType())
		shape.WriteByte(']')
	case llvm.PointerTypeKind:
		shape.WriteByte('p')
		shape.WriteString(strconv.Itoa(typ.PointerAddressSpace()))
	case llvm.VectorTypeKind:
		shape.WriteByte('V')
		shape.WriteString(strconv.Itoa(typ.VectorSize()))
		shape.WriteByte(':')
		appendWasmLLVMTypeShape(shape, typ.ElementType())
	default:
		panic(fmt.Sprintf("ssa: unsupported LLVM type in WebAssembly reflection bridge: %v", typ.TypeKind()))
	}
}

func (p Program) appendWasmRootShape(shape *strings.Builder, typ Type) {
	switch typ.kind {
	case vkPtr, vkMap, vkChan:
		shape.WriteByte('r')
	case vkString, vkSlice:
		shape.WriteByte(byte('a' + typ.kind))
		shape.WriteString("0r")
	case vkEface, vkIface:
		shape.WriteByte('i')
		shape.WriteString("1r")
	case vkClosure:
		// Closures and interfaces are both two-word Wasm values with their
		// second word rooted, but their first words use different function-table
		// and linear-memory storage conversions.
		shape.WriteByte('c')
		shape.WriteString("1r")
	case vkStruct, vkTuple:
		shape.WriteByte('{')
		var count int
		switch raw := typ.raw.Type.Underlying().(type) {
		case *types.Struct:
			count = raw.NumFields()
		case *types.Tuple:
			count = raw.Len()
		}
		for i := 0; i < count; i++ {
			p.appendWasmRootShape(shape, p.Field(typ, i))
			shape.WriteByte(',')
		}
		shape.WriteByte('}')
	case vkArray:
		raw := typ.raw.Type.Underlying().(*types.Array)
		shape.WriteByte('[')
		shape.WriteString(strconv.FormatInt(raw.Len(), 10))
		shape.WriteByte(':')
		p.appendWasmRootShape(shape, p.Index(typ))
		shape.WriteByte(']')
	default:
		shape.WriteByte('-')
	}
}

// newWasmReflectCallBridge adapts arrays of addresses to the exact lowered
// LLVM signature. A closure or interface method has an explicit environment
// parameter, so both indirect-call forms are emitted.
func (p Package) newWasmReflectCallBridge(rawSig *types.Signature, name string) Function {
	ptr := types.Typ[types.UnsafePointer]
	ptrs := types.NewPointer(ptr)
	params := types.NewTuple(
		types.NewParam(token.NoPos, nil, "fn", ptr),
		types.NewParam(token.NoPos, nil, "env", ptr),
		types.NewParam(token.NoPos, nil, "prefix", types.Typ[types.Bool]),
		types.NewParam(token.NoPos, nil, "args", ptrs),
		types.NewParam(token.NoPos, nil, "results", ptrs),
	)
	fn := p.NewFunc(name, types.NewSignatureType(nil, nil, nil, params, nil, false), InC)
	fn.impl.SetLinkage(llvm.InternalLinkage)
	b := fn.MakeBody(3)
	defer b.Dispose()
	prog := p.Prog
	args := make([]Expr, rawSig.Params().Len())
	for i := range args {
		typ := prog.Type(rawSig.Params().At(i).Type(), InC)
		args[i] = b.Load(b.Convert(prog.Pointer(typ), b.wasmReflectSlot(fn.Param(3), i)))
	}
	b.If(fn.Param(2), fn.Block(1), fn.Block(2))
	for _, prefix := range []bool{true, false} {
		entrySig, callArgs, block := rawSig, args, 2
		if prefix {
			entrySig = FuncAddCtx(types.NewParam(token.NoPos, nil, "env", ptr), rawSig)
			callArgs = append([]Expr{fn.Param(1)}, args...)
			block = 1
		}
		b.SetBlock(fn.Block(block))
		entry := Expr{fn.Param(0).impl, prog.FuncDecl(entrySig, InC)}
		ret := b.Call(entry, callArgs...)
		for i := 0; i < rawSig.Results().Len(); i++ {
			value := ret
			if rawSig.Results().Len() > 1 {
				value = b.Extract(ret, i)
			}
			b.Store(b.Convert(prog.Pointer(value.Type), b.wasmReflectSlot(fn.Param(4), i)), value)
		}
		b.Return()
	}
	b.EndBuild()
	return fn
}

func (b Builder) wasmReflectSlot(base Expr, i int) Expr {
	prog := b.Prog
	storage := prog.storageType(prog.VoidPtr())
	slot := llvm.CreateInBoundsGEP(b.impl, storage, base.impl, []llvm.Value{
		llvm.ConstInt(prog.Int().ll, uint64(i), false),
	})
	return b.Load(Expr{slot, prog.Pointer(prog.VoidPtr())})
}

// newWasmReflectMakeBridge is a normal environment-bearing Go entry. Pointer
// arguments are published as roots before reflection can allocate or suspend.
func (p Package) newWasmReflectMakeBridge(sig *types.Signature, name string) Function {
	ptr := types.Typ[types.UnsafePointer]
	fn := p.NewEnvFunc(name, sig, InGo, types.NewParam(token.NoPos, nil, "env", ptr), false)
	fn.impl.SetLinkage(llvm.InternalLinkage)
	b := fn.MakeBody(1)
	defer b.Dispose()
	prog := p.Prog
	rawSig := fn.raw.Type.(*types.Signature)
	rootValues := []Expr{fn.Env()}
	for i := 0; i < rawSig.Params().Len(); i++ {
		rootValues = append(rootValues, b.GCRootPointers(fn.Param(i))...)
	}
	if prog.GCRootsEnabled() {
		roots := fn.NewGCRoots(len(rootValues))
		for i, value := range rootValues {
			b.SetGCRoot(roots[i], value)
		}
	}
	args, _ := b.wasmReflectFrame(rawSig.Params(), func(i int) Expr { return fn.Param(i) })
	results, slots := b.wasmReflectFrame(rawSig.Results(), nil)
	callbackSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(
		types.NewParam(token.NoPos, nil, "env", ptr),
		types.NewParam(token.NoPos, nil, "args", types.NewPointer(ptr)),
		types.NewParam(token.NoPos, nil, "results", types.NewPointer(ptr)),
	), nil, false)
	callback := b.Load(b.Convert(prog.Pointer(prog.VoidPtr()), fn.Env()))
	b.Call(Expr{callback.impl, prog.FuncDecl(callbackSig, InC)}, fn.Env(), args, results)
	ret := make([]Expr, len(slots))
	for i, slot := range slots {
		ret[i] = b.Load(slot)
	}
	b.Return(ret...)
	b.EndBuild()
	return fn
}

func (b Builder) wasmReflectFrame(tuple *types.Tuple, value func(int) Expr) (Expr, []Expr) {
	prog := b.Prog
	if tuple.Len() == 0 {
		return prog.Nil(prog.Pointer(prog.VoidPtr())), nil
	}
	storage := prog.storageType(prog.VoidPtr())
	frame := Expr{
		llvm.CreateArrayAlloca(b.impl, storage, prog.IntVal(uint64(tuple.Len()), prog.Int()).impl),
		prog.Pointer(prog.VoidPtr()),
	}
	slots := make([]Expr, tuple.Len())
	for i := range slots {
		typ := prog.Type(tuple.At(i).Type(), InC)
		slots[i] = b.AllocaT(typ)
		initial := prog.Nil(typ)
		if value != nil {
			initial = value(i)
		}
		b.Store(slots[i], initial)
		address := llvm.CreateInBoundsGEP(b.impl, storage, frame.impl, []llvm.Value{
			llvm.ConstInt(prog.Int().ll, uint64(i), false),
		})
		b.Store(Expr{address, prog.Pointer(prog.VoidPtr())}, b.Convert(prog.VoidPtr(), slots[i]))
	}
	return frame, slots
}
