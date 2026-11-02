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
	"strconv"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

const (
	gcRootChainName          = "llvm_gc_root_chain"
	gcRootSJLJReplayingName  = "llvm_gc_root_sjlj_replaying"
	threadLocalGCRootChain   = env.LLGoRuntimePkg + "/internal/gcroot.currentRootChain"
	threadLocalSJLJReplaying = env.LLGoRuntimePkg + "/internal/gcroot.sjljReplaying"
)

// EnableGCRoots controls compiler-maintained GC roots.
func (p Program) EnableGCRoots(enable bool) {
	p.enableGCRoots = enable
}

// GCRootsEnabled reports whether compiler-maintained GC roots are enabled.
func (p Program) GCRootsEnabled() bool {
	return p.enableGCRoots
}

// EnableThreadLocalGCRoots gives each native worker an independent compiler
// root chain and SJLJ replay flag. The runtime provides matching TLS globals.
func (p Program) EnableThreadLocalGCRoots(enable bool) {
	p.threadLocalGCRoots = enable
}

// ThreadLocalGCRootsEnabled reports whether compiler roots are worker-local.
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

	prog := p.Prog
	rootValueType := prog.VoidPtr()
	rootStorageType := prog.storageType(rootValueType)
	rootArrayType := llvm.ArrayType(rootStorageType, count)
	frameType := prog.ctx.StructType([]llvm.Type{rootStorageType, rootStorageType, rootArrayType}, false)
	originalEntry := entry.first
	prologue := prog.ctx.InsertBasicBlock(originalEntry, "gcroot.entry")
	b.impl.SetInsertPointAtEnd(prologue)
	frame := llvm.CreateAlloca(b.impl, frameType)
	nextSlot := llvm.CreateStructGEP(b.impl, frameType, frame, 0)
	frameMap := p.newGCRootMap(count)
	roots := make([]Expr, count)
	rootArray := llvm.CreateStructGEP(b.impl, frameType, frame, 2)
	zero := llvm.ConstInt(prog.tyInt32(), 0, false)
	rootSlotType := prog.Pointer(rootValueType)
	for i := range roots {
		index := llvm.ConstInt(prog.tyInt32(), uint64(i), false)
		root := llvm.CreateInBoundsGEP(b.impl, rootArrayType, rootArray, []llvm.Value{zero, index})
		roots[i] = Expr{root, rootSlotType}
	}
	if p.outlineGCRoots() {
		helper := p.gcRootHelper(true, count)
		b.impl.CreateCall(helper.GlobalValueType(), helper, []llvm.Value{frame, frameMap}, "")
	} else {
		chain := Expr{p.gcRootChain(), rootSlotType}
		prev := b.Load(chain)
		reentered := llvm.CreateICmp(b.impl, llvm.IntEQ, prev.impl, frame)
		sjljReplaying := llvm.CreateLoad(b.impl, prog.Bool().ll, p.gcRootSJLJReplaying())
		// SJLJ replays discarded function entries on the way back to a setjmp.
		// Their stack slots must be reused without publishing dead frames.
		reusingFrame := b.impl.CreateOr(reentered, sjljReplaying, "")
		initialize := prog.ctx.InsertBasicBlock(originalEntry, "gcroot.init")
		b.impl.CreateCondBr(reusingFrame, originalEntry, initialize)
		b.impl.SetInsertPointAtEnd(initialize)
		b.Store(Expr{nextSlot, rootSlotType}, prev)
		b.Store(Expr{llvm.CreateStructGEP(b.impl, frameType, frame, 1), rootSlotType}, Expr{frameMap, rootValueType})
		for _, root := range roots {
			b.Store(root, prog.Nil(rootValueType))
		}
		b.Store(chain, Expr{frame, rootValueType})
	}
	b.impl.CreateBr(originalEntry)
	p.gcRootFrame = Expr{frame, rootValueType}
	p.gcRootPrev = Expr{nextSlot, rootSlotType}
	return roots
}

// SetGCRoot publishes value through a root created by NewGCRoots.
func (b Builder) SetGCRoot(root, value Expr) {
	b.Store(root, b.Convert(b.Prog.VoidPtr(), value))
}

func (p Function) gcRootChain() llvm.Value {
	name := gcRootChainName
	if p.Prog.threadLocalGCRoots {
		name = threadLocalGCRootChain
	}
	global := p.Pkg.mod.NamedGlobal(name)
	if global.IsNil() {
		storageType := p.Prog.storageType(p.Prog.VoidPtr())
		global = llvm.AddGlobal(p.Pkg.mod, storageType, name)
	}
	global.SetInitializer(llvm.ConstNull(global.GlobalValueType()))
	global.SetLinkage(llvm.LinkOnceAnyLinkage)
	global.SetThreadLocal(p.Prog.threadLocalGCRoots)
	global.SetAlignment(int(p.Prog.AlignOf(p.Prog.VoidPtr())))
	return global
}

func (p Function) gcRootSJLJReplaying() llvm.Value {
	name := gcRootSJLJReplayingName
	if p.Prog.threadLocalGCRoots {
		name = threadLocalSJLJReplaying
	}
	global := p.Pkg.mod.NamedGlobal(name)
	if global.IsNil() {
		global = llvm.AddGlobal(p.Pkg.mod, p.Prog.Bool().ll, name)
	}
	global.SetInitializer(llvm.ConstNull(p.Prog.Bool().ll))
	global.SetLinkage(llvm.LinkOnceAnyLinkage)
	global.SetThreadLocal(p.Prog.threadLocalGCRoots)
	global.SetAlignment(1)
	return global
}

// currentGCRootChain loads the compiler-maintained chain at the call site.
// Runtime helpers must not discover this through a Go call: their own root
// frame is already linked by then and would be mistaken for the caller frame.
func (b Builder) currentGCRootChain() Expr {
	chain := Expr{b.Func.gcRootChain(), b.Prog.Pointer(b.Prog.VoidPtr())}
	return b.Load(chain)
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
	// Only the root count is observed, never the map address. Let LLVM
	// merge identical immutable maps, including across LTO modules.
	global.SetUnnamedAddr(true)
	global.SetLinkage(llvm.InternalLinkage)
	global.SetAlignment(4)
	return global
}

func (p Function) endGCRoots(b Builder) {
	if p.gcRootPrev.IsNil() {
		return
	}
	var helper llvm.Value
	if p.outlineGCRoots() {
		helper = p.gcRootHelper(false, 0)
	}
	chain := p.gcRootChain()
	for block := p.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		term := block.LastInstruction()
		if term.IsNil() || term.InstructionOpcode() != llvm.Ret {
			continue
		}
		b.impl.SetInsertPointBefore(term)
		if !helper.IsNil() {
			b.impl.CreateCall(helper.GlobalValueType(), helper, []llvm.Value{p.gcRootFrame.impl}, "")
			continue
		}
		chainSlot := Expr{chain, p.Prog.Pointer(p.Prog.VoidPtr())}
		current := b.Load(chainSlot)
		prev := b.Load(p.gcRootPrev)
		// A helper replayed during a non-local return can finish without ever
		// linking its frame. Only the actual chain head owns a pop operation.
		linked := llvm.CreateICmp(b.impl, llvm.IntEQ, current.impl, p.gcRootFrame.impl)
		restored := llvm.CreateSelect(b.impl, linked, prev.impl, current.impl)
		b.Store(chainSlot, Expr{restored, p.Prog.VoidPtr()})
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
		*roots = append(*roots, b.Convert(b.Prog.VoidPtr(), b.getField(value, 1)))
	case vkStruct, vkTuple:
		var count int
		switch raw := value.Type.raw.Type.Underlying().(type) {
		case *types.Struct:
			count = raw.NumFields()
		case *types.Tuple:
			count = raw.Len()
		}
		for i := 0; i < count; i++ {
			if b.Prog.GCRootCount(b.Prog.Field(value.Type, i)) != 0 {
				b.appendGCRootPointers(roots, b.Field(value, i))
			}
		}
	case vkArray:
		raw := value.Type.raw.Type.Underlying().(*types.Array)
		elem := b.Prog.Index(value.Type)
		if b.Prog.GCRootCount(elem) == 0 {
			return
		}
		for i := 0; i < int(raw.Len()); i++ {
			part := llvm.CreateExtractValue(b.impl, value.impl, i)
			part = b.fromStorageValue(elem, part)
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

// Outlining trades two small helper calls for repeated frame bookkeeping.
// Keep it within dev Wasm DCE at Oz: small hot functions pay measurable
// call overhead, so Os and speed/debug builds retain inline bookkeeping.
func (p Function) outlineGCRoots() bool {
	target := p.Prog.Target()
	if !p.Prog.DeadcodeDropEnabled() || target == nil || target.GOARCH != "wasm" {
		return false
	}
	return target.effectiveOptLevel() == optlevel.Oz
}

// These LLVM-only helpers cannot allocate, yield or unwind. They use the same
// stored-pointer layout and SjLj replay/head guards as inline root frames.
// In particular, initialization must finish before publishing the chain head.
func (p Function) gcRootHelper(push bool, count int) llvm.Value {
	name := "__llgo_gcroot_pop"
	if push {
		name = "__llgo_gcroot_push_" + strconv.Itoa(count)
	}
	if p.Prog.threadLocalGCRoots {
		name += "_tls"
	}
	mod := p.Pkg.mod
	fn := mod.NamedFunction(name)
	if !fn.IsNil() {
		return fn
	}
	prog := p.Prog
	ptr := prog.VoidPtr()
	params := []llvm.Type{ptr.ll}
	if push {
		params = append(params, ptr.ll)
	}
	fn = llvm.AddFunction(mod, name, llvm.FunctionType(prog.ctx.VoidType(), params, false))
	fn.SetLinkage(llvm.LinkOnceODRLinkage)
	fn.SetVisibility(llvm.HiddenVisibility)
	fn.AddFunctionAttr(prog.ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
	fn.AddFunctionAttr(prog.ctx.CreateEnumAttribute(llvm.AttributeKindID("nounwind"), 0))
	b := p.NewBuilder()
	defer b.Dispose()
	entry := prog.ctx.AddBasicBlock(fn, "entry")
	b.impl.SetInsertPointAtEnd(entry)
	stored := prog.storageType(ptr)
	frameType := prog.ctx.StructType([]llvm.Type{stored, stored, llvm.ArrayType(stored, 0)}, false)
	frame := fn.Param(0)
	chain := Expr{p.gcRootChain(), prog.Pointer(ptr)}
	current := b.Load(chain)
	previous := Expr{llvm.CreateStructGEP(b.impl, frameType, frame, 0), prog.Pointer(ptr)}
	same := llvm.CreateICmp(b.impl, llvm.IntEQ, current.impl, frame)
	if !push {
		prev := b.Load(previous)
		selected := llvm.CreateSelect(b.impl, same, prev.impl, current.impl)
		b.Store(chain, Expr{selected, ptr})
		b.impl.CreateRetVoid()
		return fn
	}
	replay := llvm.CreateLoad(b.impl, prog.Bool().ll, p.gcRootSJLJReplaying())
	initialize := prog.ctx.AddBasicBlock(fn, "initialize")
	done := prog.ctx.AddBasicBlock(fn, "done")
	b.impl.CreateCondBr(b.impl.CreateOr(same, replay, ""), done, initialize)
	b.impl.SetInsertPointAtEnd(initialize)
	b.Store(previous, current)
	b.Store(Expr{llvm.CreateStructGEP(b.impl, frameType, frame, 1), prog.Pointer(ptr)}, Expr{fn.Param(1), ptr})
	// Specialize the byte count so small frames clear with fixed stores,
	// avoiding a dynamic memset call on every function entry.
	size := llvm.ConstInt(prog.ctx.Int64Type(), uint64(count)*prog.SizeOf(ptr), false)
	memset := mod.NamedFunction("llvm.memset.p0.i64")
	if memset.IsNil() {
		memset = llvm.AddFunction(mod, "llvm.memset.p0.i64", llvm.FunctionType(prog.ctx.VoidType(), []llvm.Type{ptr.ll, prog.ctx.Int8Type(), prog.ctx.Int64Type(), prog.ctx.Int1Type()}, false))
	}
	array := llvm.CreateStructGEP(b.impl, frameType, frame, 2)
	b.impl.CreateCall(memset.GlobalValueType(), memset, []llvm.Value{array, llvm.ConstInt(prog.ctx.Int8Type(), 0, false), size, llvm.ConstInt(prog.ctx.Int1Type(), 0, false)}, "")
	b.Store(chain, Expr{frame, ptr})
	b.impl.CreateBr(done)
	b.impl.SetInsertPointAtEnd(done)
	b.impl.CreateRetVoid()
	return fn
}
