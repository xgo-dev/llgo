/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package llvmgen

import (
	"go/token"
	"github.com/xgo-dev/llgo/types"

	"github.com/xgo-dev/llvm"
)

type wasmReflectBridgePair struct {
	call Function
	make Function
}

// wasmReflectBridge returns package-local bridges shared by every function
// type with the same Go and native call conversions and reflection frame layout.
func (p Package) wasmReflectBridge(sig *types.Signature) wasmReflectBridgePair {
	rawSig := p.Prog.gocvt.cvtFunc(sig, nil)
	goType := p.Prog.toLLVMFunc(rawSig)
	nativeType := p.Prog.toLLVMFuncBackground(sig, CDecl)
	key, hasAttrs := p.Prog.wasmCallShape(rawSig, sig, goType, nativeType)
	if bridges, ok := p.wasmReflectBridges[key]; ok {
		return bridges
	}
	if p.wasmReflectBridges == nil {
		p.wasmReflectBridges = make(map[string]wasmReflectBridgePair)
	}
	nativeCall := nativeType != goType || p.Prog.hasWasmABIPadding(nativeType.ReturnType()) || hasAttrs
	id := compactWasmBridgeID(len(p.wasmReflectBridges))
	bridges := wasmReflectBridgePair{
		call: p.newWasmReflectCallBridge(rawSig, sig, nativeCall, "$c"+id),
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

// newWasmReflectCallBridge adapts arrays of addresses to the exact lowered
// Go signature, with an optional leading context for closures and methods.
// Emit a separate raw C call only when its prototype or conversions differ.
func (p Package) newWasmReflectCallBridge(goSig, nativeSig *types.Signature, nativeCall bool, name string) Function {
	ptr := types.Typ[types.UnsafePointer]
	ptrs := types.NewPointer(ptr)
	params := types.NewTuple(
		types.NewParam(token.NoPos, nil, "fn", ptr),
		types.NewParam(token.NoPos, nil, "env", ptr),
		types.NewParam(token.NoPos, nil, "prefix", types.Typ[types.Bool]),
		types.NewParam(token.NoPos, nil, "native", types.Typ[types.Bool]),
		types.NewParam(token.NoPos, nil, "args", ptrs),
		types.NewParam(token.NoPos, nil, "results", ptrs),
	)
	// reflect invokes this bridge through a Go funcval, including its bool arguments.
	fn := p.NewFunc(name, types.NewSignatureType(nil, nil, nil, params, nil, false), GoDecl)
	fn.impl.SetLinkage(llvm.InternalLinkage)
	blocks := 3
	if nativeCall {
		blocks += 2
	}
	b := fn.MakeBody(blocks)
	defer b.Dispose()
	prog := p.Prog
	loadArgs := func(sig *types.Signature) []Expr {
		args := make([]Expr, sig.Params().Len())
		for i := range args {
			// Reflect supplies addresses into Go storage, including for a C
			// callee whose physical uintptr parameter has to be narrowed later.
			typ := prog.rawType(sig.Params().At(i).Type())
			args[i] = b.Load(b.Convert(prog.Pointer(typ), b.wasmReflectSlot(fn.Param(4), i)))
		}
		return args
	}
	invoke := func(entry Type, args []Expr) {
		ret := b.Call(Expr{fn.Param(0).impl, entry}, args...)
		for i := 0; i < goSig.Results().Len(); i++ {
			value := ret
			if goSig.Results().Len() > 1 {
				value = b.Extract(ret, i)
			}
			b.Store(b.Convert(prog.Pointer(value.Type), b.wasmReflectSlot(fn.Param(5), i)), value)
		}
		b.Return()
	}
	if nativeCall {
		b.If(fn.Param(3), fn.Block(3), fn.Block(4))
		b.SetBlock(fn.Block(3))
		invoke(prog.rawType(nativeSig), loadArgs(nativeSig))
		b.SetBlock(fn.Block(4))
	}
	args := loadArgs(goSig)
	b.If(fn.Param(2), fn.Block(1), fn.Block(2))
	b.SetBlock(fn.Block(1))
	envSig := FuncAddCtx(types.NewParam(token.NoPos, nil, "env", ptr), goSig)
	invoke(prog.FuncDecl(envSig, GoDecl), append([]Expr{fn.Param(1)}, args...))
	b.SetBlock(fn.Block(2))
	invoke(prog.FuncDecl(goSig, GoDecl), args)
	b.EndBuild()
	return fn
}

func (b Builder) wasmReflectSlot(base Expr, i int) Expr {
	prog := b.Prog
	storage := prog.storageType(prog.VoidPtr())
	// The Go-word index is a compile-time constant and is folded into the GEP
	// offset before physical Memory32 address lowering. Only dynamic addresses
	// need physicalPointerIndex conversion.
	slot := llvm.CreateInBoundsGEP(b.impl, storage, base.impl, []llvm.Value{
		llvm.ConstInt(prog.Int().ll, uint64(i), false),
	})
	return b.Load(Expr{slot, prog.Pointer(prog.VoidPtr())})
}

// newWasmReflectMakeBridge is a normal environment-bearing Go entry. Pointer
// arguments are published as roots before reflection can allocate or suspend.
func (p Package) newWasmReflectMakeBridge(sig *types.Signature, name string) Function {
	ptr := types.Typ[types.UnsafePointer]
	fn := p.NewEnvFunc(name, sig, GoDecl, types.NewParam(token.NoPos, nil, "env", ptr), false)
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
	b.Call(Expr{callback.impl, prog.FuncDecl(callbackSig, CDecl)}, fn.Env(), args, results)
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
		typ := prog.rawType(tuple.At(i).Type())
		slots[i] = b.AllocaT(typ)
		initial := prog.Nil(typ)
		if value != nil {
			initial = value(i)
		}
		b.Store(slots[i], initial)
		// As in wasmReflectSlot, this constant Go-word index is folded before
		// Memory32 physical address lowering.
		address := llvm.CreateInBoundsGEP(b.impl, storage, frame.impl, []llvm.Value{
			llvm.ConstInt(prog.Int().ll, uint64(i), false),
		})
		b.Store(Expr{address, prog.Pointer(prog.VoidPtr())}, b.Convert(prog.VoidPtr(), slots[i]))
	}
	return frame, slots
}
