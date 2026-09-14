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

package ssa

import (
	"go/types"

	"github.com/xgo-dev/llvm"
)

const (
	gcRootChainName         = "llvm_gc_root_chain"
	gcRootSJLJReplayingName = "llvm_gc_root_sjlj_replaying"
)

// EnableGCRoots controls compiler-maintained GC roots.
func (p Program) EnableGCRoots(enable bool) {
	p.enableGCRoots = enable
}

// GCRootsEnabled reports whether compiler-maintained GC roots are enabled.
func (p Program) GCRootsEnabled() bool {
	return p.enableGCRoots
}

// NewGCRoots reserves count pointer roots in one compiler-maintained frame.
// It must be called once, before the function emits a return.
func (p Function) NewGCRoots(count int) []Expr {
	if count <= 0 {
		return nil
	}
	if !p.gcRootPrev.IsNil() {
		panic("ssa: GC roots already reserved")
	}
	b := p.NewBuilder()
	defer b.Dispose()
	entry := p.Block(0)

	prog := p.Prog
	voidPtr := prog.tyVoidPtr()
	rootArrayType := llvm.ArrayType(voidPtr, count)
	frameType := prog.ctx.StructType([]llvm.Type{voidPtr, voidPtr, rootArrayType}, false)
	originalEntry := entry.first
	prologue := prog.ctx.InsertBasicBlock(originalEntry, "gcroot.entry")
	initialize := prog.ctx.InsertBasicBlock(originalEntry, "gcroot.init")
	b.impl.SetInsertPointAtEnd(prologue)
	frame := llvm.CreateAlloca(b.impl, frameType)

	chain := p.gcRootChain()
	prev := llvm.CreateLoad(b.impl, voidPtr, chain)
	nextSlot := llvm.CreateStructGEP(b.impl, frameType, frame, 0)
	reentered := llvm.CreateICmp(b.impl, llvm.IntEQ, prev, frame)
	sjljReplaying := llvm.CreateLoad(b.impl, prog.Bool().ll, p.gcRootSJLJReplaying())
	// SJLJ/Asyncify replays discarded function entries on the way back to a
	// setjmp. Their stack slots must be reused without publishing dead frames.
	reusingFrame := b.impl.CreateOr(reentered, sjljReplaying, "")

	frameMap := p.newGCRootMap(count)
	roots := make([]Expr, count)
	rootArray := llvm.CreateStructGEP(b.impl, frameType, frame, 2)
	zero := llvm.ConstInt(prog.tyInt32(), 0, false)
	for i := range roots {
		index := llvm.ConstInt(prog.tyInt32(), uint64(i), false)
		root := llvm.CreateInBoundsGEP(b.impl, rootArrayType, rootArray, []llvm.Value{zero, index})
		roots[i] = Expr{root, prog.Pointer(prog.VoidPtr())}
	}
	b.impl.CreateCondBr(reusingFrame, originalEntry, initialize)

	b.impl.SetInsertPointAtEnd(initialize)
	b.impl.CreateStore(prev, nextSlot)
	b.impl.CreateStore(frameMap, llvm.CreateStructGEP(b.impl, frameType, frame, 1))
	for _, root := range roots {
		b.impl.CreateStore(llvm.ConstNull(voidPtr), root.impl)
	}
	b.impl.CreateStore(frame, chain)
	b.impl.CreateBr(originalEntry)
	p.gcRootFrame = Expr{frame, prog.VoidPtr()}
	p.gcRootPrev = Expr{nextSlot, prog.Pointer(prog.VoidPtr())}
	return roots
}

// SetGCRoot publishes value through a root created by NewGCRoots.
func (b Builder) SetGCRoot(root, value Expr) {
	b.Store(root, b.Convert(b.Prog.VoidPtr(), value))
}

func (p Function) gcRootChain() llvm.Value {
	global := p.Pkg.mod.NamedGlobal(gcRootChainName)
	if global.IsNil() {
		global = llvm.AddGlobal(p.Pkg.mod, p.Prog.tyVoidPtr(), gcRootChainName)
	}
	global.SetInitializer(llvm.ConstNull(p.Prog.tyVoidPtr()))
	global.SetLinkage(llvm.LinkOnceAnyLinkage)
	global.SetAlignment(p.Prog.PointerSize())
	return global
}

func (p Function) gcRootSJLJReplaying() llvm.Value {
	global := p.Pkg.mod.NamedGlobal(gcRootSJLJReplayingName)
	if global.IsNil() {
		global = llvm.AddGlobal(p.Pkg.mod, p.Prog.Bool().ll, gcRootSJLJReplayingName)
	}
	global.SetInitializer(llvm.ConstNull(p.Prog.Bool().ll))
	global.SetLinkage(llvm.LinkOnceAnyLinkage)
	global.SetAlignment(1)
	return global
}

// currentGCRootChain loads the compiler-maintained chain at the call site.
// Runtime helpers must not discover this through a Go call: their own root
// frame is already linked by then and would be mistaken for the caller frame.
func (b Builder) currentGCRootChain() Expr {
	chain := b.Func.gcRootChain()
	return Expr{llvm.CreateLoad(b.impl, b.Prog.tyVoidPtr(), chain), b.Prog.VoidPtr()}
}

func (p Function) newGCRootMap(count int) llvm.Value {
	prog := p.Prog
	mapType := prog.ctx.StructType([]llvm.Type{prog.tyInt32(), prog.tyInt32()}, false)
	name := p.Name() + "$gcmap"
	global := llvm.AddGlobal(p.Pkg.mod, mapType, name)
	global.SetInitializer(llvm.ConstNamedStruct(mapType, []llvm.Value{
		llvm.ConstInt(prog.tyInt32(), uint64(count), false),
		llvm.ConstInt(prog.tyInt32(), 0, false),
	}))
	global.SetGlobalConstant(true)
	global.SetLinkage(llvm.InternalLinkage)
	global.SetAlignment(4)
	return global
}

func (p Function) endGCRoots(b Builder) {
	if p.gcRootPrev.IsNil() {
		return
	}
	chain := p.gcRootChain()
	for block := p.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		term := block.LastInstruction()
		if term.IsNil() || term.InstructionOpcode() != llvm.Ret {
			continue
		}
		b.impl.SetInsertPointBefore(term)
		current := llvm.CreateLoad(b.impl, p.Prog.tyVoidPtr(), chain)
		prev := llvm.CreateLoad(b.impl, p.Prog.tyVoidPtr(), p.gcRootPrev.impl)
		// A helper replayed during a non-local return can finish without ever
		// linking its frame. Only the actual chain head owns a pop operation.
		linked := llvm.CreateICmp(b.impl, llvm.IntEQ, current, p.gcRootFrame.impl)
		restored := llvm.CreateSelect(b.impl, linked, prev, current)
		b.impl.CreateStore(restored, chain)
	}
}

// GCRootCount reports how many heap pointers typ contributes to a root frame.
func (p Program) GCRootCount(typ Type) int {
	switch typ.kind {
	case vkPtr, vkString, vkSlice, vkMap, vkEface, vkIface, vkClosure, vkChan:
		return 1
	case vkStruct:
		raw := typ.raw.Type.Underlying().(*types.Struct)
		count := 0
		for i := 0; i < raw.NumFields(); i++ {
			count += p.GCRootCount(p.Field(typ, i))
		}
		return count
	case vkArray:
		raw := typ.raw.Type.Underlying().(*types.Array)
		return int(raw.Len()) * p.GCRootCount(p.Index(typ))
	case vkTuple:
		raw := typ.raw.Type.Underlying().(*types.Tuple)
		count := 0
		for i := 0; i < raw.Len(); i++ {
			count += p.GCRootCount(p.Field(typ, i))
		}
		return count
	default:
		return 0
	}
}

// GCRootPointers extracts the heap pointers represented by value.
func (b Builder) GCRootPointers(value Expr) []Expr {
	var roots []Expr
	b.appendGCRootPointers(&roots, value)
	return roots
}

func (b Builder) appendGCRootPointers(roots *[]Expr, value Expr) {
	switch value.Type.kind {
	case vkPtr, vkMap, vkChan:
		*roots = append(*roots, b.Convert(b.Prog.VoidPtr(), value))
	case vkString:
		*roots = append(*roots, b.Convert(b.Prog.VoidPtr(), b.StringData(value)))
	case vkSlice:
		*roots = append(*roots, b.Convert(b.Prog.VoidPtr(), b.SliceData(value)))
	case vkEface, vkIface:
		*roots = append(*roots, b.InterfaceData(value))
	case vkClosure:
		data := llvm.CreateExtractValue(b.impl, value.impl, 1)
		*roots = append(*roots, Expr{data, b.Prog.VoidPtr()})
	case vkStruct, vkTuple:
		var count int
		switch raw := value.Type.raw.Type.Underlying().(type) {
		case *types.Struct:
			count = raw.NumFields()
		case *types.Tuple:
			count = raw.Len()
		}
		for i := 0; i < count; i++ {
			b.appendGCRootPointers(roots, b.Field(value, i))
		}
	case vkArray:
		raw := value.Type.raw.Type.Underlying().(*types.Array)
		elem := b.Prog.Index(value.Type)
		for i := 0; i < int(raw.Len()); i++ {
			part := llvm.CreateExtractValue(b.impl, value.impl, i)
			b.appendGCRootPointers(roots, Expr{part, elem})
		}
	}
}

// ClosureContextParam returns the hidden closure context parameter.
func (p Function) ClosureContextParam() Expr {
	if p.env == nil {
		return Nil
	}
	return Expr{p.impl.Param(0), p.env}
}
