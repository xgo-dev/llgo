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

	"github.com/goplus/llgo/internal/env"
	"github.com/xgo-dev/llvm"
)

const (
	gcRootChainName      = "llvm_gc_root_chain"
	threadLocalRootChain = env.LLGoRuntimePkg + "/internal/gcroot.currentRootChain"
)

// EnableGCRoots controls compiler-maintained GC roots.
func (p Program) EnableGCRoots(enable bool) {
	p.enableGCRoots = enable
}

// GCRootsEnabled reports whether compiler-maintained GC roots are enabled.
func (p Program) GCRootsEnabled() bool {
	return p.enableGCRoots
}

// EnableThreadLocalGCRoots gives each native thread an independent compiler
// root chain. The runtime must provide a matching native TLS variable.
func (p Program) EnableThreadLocalGCRoots(enable bool) {
	p.threadLocalGCRoots = enable
}

// ThreadLocalGCRootsEnabled reports whether root chains are thread-local.
func (p Program) ThreadLocalGCRootsEnabled() bool {
	return p.threadLocalGCRoots
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
	if entry.first.FirstInstruction().IsNil() {
		b.SetBlockEx(entry, AtEnd, false)
	} else {
		b.SetBlockEx(entry, AtStart, false)
	}

	prog := p.Prog
	voidPtr := prog.tyVoidPtr()
	rootArrayType := llvm.ArrayType(voidPtr, count)
	frameType := prog.ctx.StructType([]llvm.Type{voidPtr, voidPtr, rootArrayType}, false)
	frame := llvm.CreateAlloca(b.impl, frameType)

	chain := p.gcRootChain()
	prevWord := llvm.CreateLoad(b.impl, prog.Uintptr().ll, chain)
	prev := llvm.CreateIntToPtr(b.impl, prevWord, voidPtr)
	b.impl.CreateStore(prev, llvm.CreateStructGEP(b.impl, frameType, frame, 0))

	frameMap := p.newGCRootMap(count)
	b.impl.CreateStore(frameMap, llvm.CreateStructGEP(b.impl, frameType, frame, 1))

	roots := make([]Expr, count)
	rootArray := llvm.CreateStructGEP(b.impl, frameType, frame, 2)
	zero := llvm.ConstInt(prog.tyInt32(), 0, false)
	for i := range roots {
		index := llvm.ConstInt(prog.tyInt32(), uint64(i), false)
		root := llvm.CreateInBoundsGEP(b.impl, rootArrayType, rootArray, []llvm.Value{zero, index})
		b.impl.CreateStore(llvm.ConstNull(voidPtr), root)
		roots[i] = Expr{root, prog.Pointer(prog.VoidPtr())}
	}
	b.impl.CreateStore(llvm.CreatePtrToInt(b.impl, frame, prog.Uintptr().ll), chain)
	p.gcRootPrev = Expr{prevWord, prog.Uintptr()}
	return roots
}

// SetGCRoot publishes value through a root created by NewGCRoots.
func (b Builder) SetGCRoot(root, value Expr) {
	b.Store(root, b.Convert(b.Prog.VoidPtr(), value))
}

func (p Function) gcRootChain() llvm.Value {
	name := gcRootChainName
	if p.Prog.threadLocalGCRoots {
		name = threadLocalRootChain
	}
	global := p.Pkg.mod.NamedGlobal(name)
	if global.IsNil() {
		global = llvm.AddGlobal(p.Pkg.mod, p.Prog.Uintptr().ll, name)
	}
	global.SetInitializer(llvm.ConstNull(p.Prog.Uintptr().ll))
	global.SetLinkage(llvm.LinkOnceAnyLinkage)
	global.SetThreadLocal(p.Prog.threadLocalGCRoots)
	global.SetAlignment(p.Prog.PointerSize())
	return global
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
		b.impl.CreateStore(p.gcRootPrev.impl, chain)
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
	if p.base == 0 {
		return Nil
	}
	return Expr{p.impl.Param(0), p.params[0]}
}
