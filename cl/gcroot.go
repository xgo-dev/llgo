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

package cl

import (
	"go/token"
	"go/types"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llgo/internal/gcrootplan"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func (p *context) prepareGCRoots(fn *ssa.Function, hasClosureContext bool) {
	p.gcRoots = nil
	p.gcClosureRoot = llssa.Nil
	if !p.prog.GCRootsEnabled() {
		return
	}

	planned := gcrootplan.Plan(fn, func(value ssa.Value) bool {
		switch value.(type) {
		case *ssa.FreeVar:
			return false
		}
		if call, ok := value.(*ssa.Call); ok {
			if callee, ok := call.Call.Value.(*ssa.Function); ok {
				if _, _, kind := p.funcOf(callee); kind == llgoFuncAddr {
					// llgo.funcAddr has an unsafe.Pointer signature but lowers
					// to a code pointer, not a Go heap pointer.
					return false
				}
			}
		}
		if !mayContainGCRoot(value.Type()) {
			return false
		}
		typ := p.type_(value.Type(), llssa.GoDecl)
		return p.prog.GCRootCount(typ) != 0
	}, p.isGCSafepoint)
	for value := range uintptrEscapesRoots(fn) {
		planned[value] = struct{}{}
	}
	if p.safepointEntry {
		for _, param := range fn.Params {
			if !mayContainGCRoot(param.Type()) {
				continue
			}
			typ := p.type_(param.Type(), llssa.GoDecl)
			if p.prog.GCRootCount(typ) != 0 {
				planned[param] = struct{}{}
			}
		}
	}
	if p.prog.DeadcodeDropEnabled() && p.prog.Target() != nil && p.prog.Target().GOARCH == "wasm" {
		owners := make(map[ssa.Value]struct{}, len(planned))
		for value := range planned {
			value = gcRootOwner(value)
			switch value.(type) {
			case *ssa.Const, *ssa.Global, *ssa.Function, *ssa.FreeVar:
				// Static storage needs no heap root; free variables are kept
				// by the separately rooted closure context.
				continue
			}
			owners[value] = struct{}{}
		}
		planned = owners
	}
	counts := make(map[ssa.Value]int, len(planned))
	total := 0
	count := func(value ssa.Value) {
		if _, ok := planned[value]; !ok {
			return
		}
		typ := p.type_(value.Type(), llssa.GoDecl)
		n := p.prog.GCRootCount(typ)
		if basicKind(value.Type()) == types.Uintptr {
			// Only pragma-designated uintptr parameters enter planned. Keep this
			// single-slot reservation paired with publishGCRoot's direct SetGCRoot:
			// uintptr has no ordinary pointer layout for GCRootPointers to expand.
			n = 1
		}
		if n != 0 {
			counts[value] = n
			total += n
		}
	}
	for _, param := range fn.Params {
		count(param)
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if value, ok := instr.(ssa.Value); ok {
				count(value)
			}
		}
	}
	hasClosureRoot := hasClosureContext && p.functionHasGCSafepoint(fn)
	if hasClosureRoot {
		total++
	}
	allSlots := p.fn.NewGCRoots(total)
	next := 0
	roots := make(map[ssa.Value][]llssa.Expr, len(counts))
	assign := func(value ssa.Value) {
		if n := counts[value]; n != 0 {
			roots[value] = allSlots[next : next+n]
			next += n
		}
	}
	for _, param := range fn.Params {
		assign(param)
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if value, ok := instr.(ssa.Value); ok {
				assign(value)
			}
		}
	}
	p.gcRoots = roots
	if hasClosureRoot {
		p.gcClosureRoot = allSlots[next]
	}
}

func mayContainGCRoot(typ types.Type) bool {
	if typ == nil {
		return false
	}
	switch typ := types.Unalias(typ).Underlying().(type) {
	case *types.Basic:
		return typ.Kind() == types.UnsafePointer || typ.Info()&types.IsString != 0
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature, *types.Interface:
		return true
	case *types.Array:
		return mayContainGCRoot(typ.Elem())
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if mayContainGCRoot(typ.Field(i).Type()) {
				return true
			}
		}
	case *types.Tuple:
		// SSA tuple results are immediately projected into Extract values. Root
		// those projections instead; a tuple can also contain the invalid
		// placeholder type used for an unrequested range key or value.
		return false
	}
	return false
}

func (p *context) initGCRoots(b llssa.Builder, fn *ssa.Function) {
	if len(p.gcRoots) == 0 && p.gcClosureRoot.IsNil() {
		return
	}
	b.SetBlockEx(p.fn.Block(0), llssa.AtEnd, true)
	for i, param := range fn.Params {
		if _, ok := p.gcRoots[param]; ok {
			p.publishGCRoot(b, param, b.Param(i))
		}
	}
	if !p.gcClosureRoot.IsNil() {
		b.SetGCRoot(p.gcClosureRoot, p.fn.ClosureContextParam())
	}
}

func functionHasGCSafepoint(fn *ssa.Function) bool {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if gcSafepoint(instr) {
				return true
			}
		}
	}
	return false
}

func (p *context) functionHasGCSafepoint(fn *ssa.Function) bool {
	if p.safepointEntry {
		return true
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if p.isGCSafepoint(instr) {
				return true
			}
		}
	}
	return false
}

func (p *context) isGCSafepoint(instr ssa.Instruction) bool {
	if load, ok := instr.(*ssa.UnOp); ok && load.Op == token.MUL {
		// ABI lowering may snapshot these loads with AllocU after this root
		// plan is built. Keep the predicate in sync with abi.ShouldSnapshotAggregateLoad.
		size := p.prog.SizeOf(p.type_(load.Type(), llssa.GoDecl))
		ptrSize := p.prog.PointerSize()
		switch t := load.Type().Underlying().(type) {
		case *types.Array:
			if llabi.ShouldSnapshotAggregateLoad(true, int(t.Len()), false, size, ptrSize) {
				return true
			}
		case *types.Struct:
			if llabi.ShouldSnapshotAggregateLoad(false, 0, true, size, ptrSize) {
				return true
			}
		}
	}
	return gcSafepoint(instr) || p.isCooperativeSafepoint(instr)
}

// gcSafepoint mirrors the operations whose LLGo lowering can call the runtime.
// Unknown instructions stay conservative.
func gcSafepoint(instr ssa.Instruction) bool {
	switch instr := instr.(type) {
	case *ssa.Phi, *ssa.DebugRef, *ssa.Extract, *ssa.Field, *ssa.FieldAddr,
		*ssa.Index, *ssa.IndexAddr, *ssa.If, *ssa.Jump, *ssa.Return,
		*ssa.Slice, *ssa.SliceToArrayPointer, *ssa.Store, *ssa.ChangeType:
		return false
	case *ssa.BinOp:
		return gcBinOpSafepoint(instr)
	case *ssa.UnOp:
		return instr.Op == token.ARROW
	case *ssa.Convert:
		return gcConversionSafepoint(instr.X.Type(), instr.Type())
	case *ssa.Call:
		if builtin, ok := instr.Call.Value.(*ssa.Builtin); ok {
			switch builtin.Name() {
			case "cap", "complex", "imag", "len", "real":
				return false
			}
		}
		return true
	default:
		return true
	}
}

func gcBinOpSafepoint(instr *ssa.BinOp) bool {
	switch basicKind(instr.X.Type()) {
	case types.String, types.UntypedString:
		return true
	}
	_, isInterface := types.Unalias(instr.X.Type()).Underlying().(*types.Interface)
	return isInterface
}

func gcConversionSafepoint(src, dst types.Type) bool {
	return isStringOrSlice(src) || isStringOrSlice(dst)
}

func isStringOrSlice(typ types.Type) bool {
	switch typ := types.Unalias(typ).Underlying().(type) {
	case *types.Slice:
		return true
	case *types.Basic:
		return typ.Info()&types.IsString != 0
	default:
		return false
	}
}

func basicKind(typ types.Type) types.BasicKind {
	if basic, ok := types.Unalias(typ).Underlying().(*types.Basic); ok {
		return basic.Kind()
	}
	return types.Invalid
}

func (p *context) publishGCRoot(b llssa.Builder, value ssa.Value, expr llssa.Expr) {
	slots, ok := p.gcRoots[value]
	if !ok || expr.IsNil() {
		return
	}
	if basicKind(value.Type()) == types.Uintptr {
		// prepareGCRoots reserves exactly one slot for pragma-designated uintptr
		// values even though their ordinary GC root count is zero.
		b.SetGCRoot(slots[0], expr)
		return
	}
	roots := b.GCRootPointers(expr)
	if len(roots) != len(slots) {
		panic("cl: inconsistent GC root layout")
	}
	for i, root := range roots {
		b.SetGCRoot(slots[i], root)
	}
}

// gcRootOwner follows views into the same allocation. Root slots are published
// at their SSA definition and never cleared until function return, so keeping
// the owner also keeps every derived interior pointer alive.
func gcRootOwner(value ssa.Value) ssa.Value {
	for {
		switch v := value.(type) {
		case *ssa.FieldAddr:
			value = v.X
		case *ssa.IndexAddr:
			value = v.X
		case *ssa.Slice:
			// LLGo can replace a synthetic makeslice array with a fresh
			// allocation at this slice and omit the array entirely. Keep
			// that result rooted even if its SSA input looks like an owner.
			if alloc, ok := v.X.(*ssa.Alloc); ok && alloc.Comment == "makeslice" {
				return value
			}
			value = v.X
		case *ssa.SliceToArrayPointer:
			value = v.X
		case *ssa.ChangeType:
			// Function/aggregate conversions can have a different lowered
			// ABI. Only these views preserve their backing allocation.
			switch v.Type().Underlying().(type) {
			case *types.Pointer, *types.Slice:
			default:
				if kind := basicKind(v.Type()); kind != types.String && kind != types.UnsafePointer {
					return value
				}
			}
			value = v.X
		case *ssa.Convert:
			pointer := func(t types.Type) bool {
				_, ptr := t.Underlying().(*types.Pointer)
				return ptr || basicKind(t) == types.UnsafePointer
			}
			if !pointer(v.Type()) || !pointer(v.X.Type()) {
				return value
			}
			value = v.X
		default:
			return value
		}
	}
}
