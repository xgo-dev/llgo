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

package ssaopt

import (
	"go/token"

	"github.com/xgo-dev/llgo/cmd/compiler/abipolicy"
	"github.com/xgo-dev/llgo/cmd/compiler/gcrootplan"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// GCRootPass computes explicit GC root slot reservations for every SSA value
// live across a GC safepoint, unifying pure-Go gcrootplan and
// //go:uintptrescapes root tracking into FunctionPlan.
//
// It must run after SafepointPass so cooperative safepoint backedges and
// entry-point polls are already reflected in FunctionPlan.
type GCRootPass struct{}

func (GCRootPass) Name() string { return "gcroot" }

func (GCRootPass) Run(fn *ssa.Function, cfg Config, plan *FunctionPlan) error {
	if !cfg.GCRoots || fn == nil || len(fn.Blocks) == 0 {
		return nil
	}
	plan.GCRootPlanned = true

	typeCtx := cfg.TypeCtx
	if typeCtx == nil {
		typeCtx = types.NewContext()
	}
	spec := typeCtx.Spec()
	ptrSize := spec.PtrSize
	if ptrSize <= 0 {
		ptrSize = spec.WordSize
	}
	if ptrSize <= 0 {
		ptrSize = 8
	}

	rootCount := func(t types.Type) int {
		if !MayContainGCRoot(t) {
			return 0
		}
		return GCRootCount(patchEscapeType(t, cfg.PatchTypes), typeCtx)
	}

	isSafepoint := func(instr ssa.Instruction) bool {
		if load, ok := instr.(*ssa.UnOp); ok && load.Op == token.MUL {
			patched := patchEscapeType(load.Type(), cfg.PatchTypes)
			size := uint64(typeCtx.Sizeof(patched))
			switch t := types.Unalias(patched).Underlying().(type) {
			case *types.Array:
				if abipolicy.ShouldSnapshotAggregateLoad(true, int(t.Len()), false, size, ptrSize) {
					return true
				}
			case *types.Struct:
				if abipolicy.ShouldSnapshotAggregateLoad(false, 0, true, size, ptrSize) {
					return true
				}
			}
		}
		if GCSafepoint(instr) {
			return true
		}
		_, isCoop := plan.SafepointBackedges[instr]
		return isCoop
	}

	planned := gcrootplan.Plan(fn, func(value ssa.Value) bool {
		if _, isFreeVar := value.(*ssa.FreeVar); isFreeVar {
			return false
		}
		if call, ok := value.(*ssa.Call); ok {
			if callee, ok := call.Call.Value.(*ssa.Function); ok && callee.Intrinsic() == ssa.IntrinsicFuncAddr {
				// llgo.funcAddr has an unsafe.Pointer signature but lowers
				// to a code pointer, not a Go heap pointer.
				return false
			}
		}
		return rootCount(value.Type()) != 0
	}, isSafepoint)
	if planned == nil {
		planned = make(map[ssa.Value]struct{})
	}
	for value := range UintptrEscapesRoots(fn) {
		planned[value] = struct{}{}
	}
	if plan.SafepointEntry {
		for _, param := range fn.Params {
			if rootCount(param.Type()) != 0 {
				planned[param] = struct{}{}
			}
		}
	}
	if spec.GOARCH == "wasm" {
		owners := make(map[ssa.Value]struct{}, len(planned))
		for value := range planned {
			owner := GCRootOwner(value)
			switch owner.(type) {
			case *ssa.Const, *ssa.Global, *ssa.Function, *ssa.FreeVar:
				// Static storage needs no heap root; free variables are kept
				// by the separately rooted closure context.
				continue
			}
			if _, rooted := planned[owner]; rooted || cfg.DeadcodeDrop {
				value = owner
			}
			owners[value] = struct{}{}
		}
		planned = owners
	}

	var counts map[ssa.Value]int
	recordCount := func(value ssa.Value) {
		if _, ok := planned[value]; !ok {
			return
		}
		n := rootCount(value.Type())
		if basicKindOf(value.Type()) == types.Uintptr {
			// Only pragma-designated uintptr parameters enter planned.
			n = 1
		}
		if n != 0 {
			if counts == nil {
				counts = make(map[ssa.Value]int, len(planned))
			}
			counts[value] = n
		}
	}
	for _, param := range fn.Params {
		recordCount(param)
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if value, ok := instr.(ssa.Value); ok {
				recordCount(value)
			}
		}
	}

	plan.GCRoots = counts
	plan.GCClosureRoot = plan.SafepointEntry || functionHasSafepoint(fn, isSafepoint)
	return nil
}

// UintptrEscapesRoots preserves the pointer semantics of //go:uintptrescapes
// at both ends of a call. Ordinary uintptr values remain integers, not roots.
func UintptrEscapesRoots(fn *ssa.Function) map[ssa.Value]struct{} {
	roots := make(map[ssa.Value]struct{})
	if fn == nil {
		return roots
	}
	directives := make(map[*ssa.Function]bool)
	hasDirective := func(callee *ssa.Function) bool {
		value, ok := directives[callee]
		if !ok {
			value = HasUintptrEscapesDirective(callee)
			directives[callee] = value
		}
		return value
	}
	if hasDirective(fn) {
		for _, param := range fn.Params {
			if basicKindOf(param.Type()) == types.Uintptr {
				roots[param] = struct{}{}
			}
		}
	}
	seen := make(map[ssa.Value]bool)
	var source func(ssa.Value)
	source = func(value ssa.Value) {
		if value == nil || seen[value] {
			return
		}
		seen[value] = true
		switch value := value.(type) {
		case *ssa.Convert:
			if basicKindOf(value.Type()) == types.Uintptr && basicKindOf(value.X.Type()) == types.UnsafePointer {
				roots[value.X] = struct{}{}
			}
		case *ssa.ChangeType:
			source(value.X)
		case *ssa.Phi:
			for _, edge := range value.Edges {
				source(edge)
			}
		case *ssa.Slice:
			source(value.X)
		case *ssa.Alloc:
			if refs := value.Referrers(); refs != nil {
				for _, ref := range *refs {
					index, ok := ref.(*ssa.IndexAddr)
					if !ok || index.X != value {
						continue
					}
					if uses := index.Referrers(); uses != nil {
						for _, use := range *uses {
							if store, ok := use.(*ssa.Store); ok && store.Addr == index {
								source(store.Val)
							}
						}
					}
				}
			}
		}
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(ssa.CallInstruction)
			if !ok || !hasDirective(call.Common().StaticCallee()) {
				continue
			}
			args := call.Common().Args
			for i, arg := range args {
				if basicKindOf(arg.Type()) == types.Uintptr {
					source(arg)
				} else if call.Common().Signature().Variadic() && i == len(args)-1 {
					if slice, ok := arg.Type().Underlying().(*types.Slice); ok && basicKindOf(slice.Elem()) == types.Uintptr {
						source(arg)
					}
				}
			}
		}
	}
	return roots
}

// MayContainGCRoot reports whether Go values of typ may hold traceable heap pointers.
func MayContainGCRoot(typ types.Type) bool {
	if typ == nil {
		return false
	}
	switch typ := types.Unalias(typ).Underlying().(type) {
	case *types.Basic:
		return typ.Kind() == types.UnsafePointer || typ.Info()&types.IsString != 0
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature, *types.Interface:
		return true
	case *types.Array:
		return MayContainGCRoot(typ.Elem())
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if MayContainGCRoot(typ.Field(i).Type()) {
				return true
			}
		}
	case *types.Tuple:
		return false
	}
	return false
}

// FunctionHasGCSafepoint reports whether fn contains any instruction that may
// trigger a runtime GC safepoint.
func FunctionHasGCSafepoint(fn *ssa.Function) bool {
	return functionHasSafepoint(fn, GCSafepoint)
}

func functionHasSafepoint(fn *ssa.Function, isSafepoint func(ssa.Instruction) bool) bool {
	if fn == nil {
		return false
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if isSafepoint(instr) {
				return true
			}
		}
	}
	return false
}

// GCSafepoint mirrors the operations whose LLGo lowering can call the runtime.
// Unknown instructions stay conservative.
func GCSafepoint(instr ssa.Instruction) bool {
	switch instr := instr.(type) {
	case *ssa.Phi, *ssa.DebugRef, *ssa.Extract, *ssa.Field, *ssa.FieldAddr,
		*ssa.Index, *ssa.IndexAddr, *ssa.If, *ssa.Jump, *ssa.Return,
		*ssa.Slice, *ssa.SliceToArrayPointer, *ssa.Store, *ssa.ChangeType:
		return false
	case *ssa.BinOp:
		switch basicKindOf(instr.X.Type()) {
		case types.String, types.UntypedString:
			return true
		}
		_, isInterface := types.Unalias(instr.X.Type()).Underlying().(*types.Interface)
		return isInterface
	case *ssa.UnOp:
		return instr.Op == token.ARROW
	case *ssa.Convert:
		return GCConversionSafepoint(instr.X.Type(), instr.Type())
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

// GCConversionSafepoint reports whether converting src to dst may allocate or
// call into the runtime.
func GCConversionSafepoint(src, dst types.Type) bool {
	return isStringOrSliceType(src) || isStringOrSliceType(dst)
}

func isStringOrSliceType(typ types.Type) bool {
	switch typ := types.Unalias(typ).Underlying().(type) {
	case *types.Slice:
		return true
	case *types.Basic:
		return typ.Info()&types.IsString != 0
	default:
		return false
	}
}

// GCRootOwner follows interior views back to the owning allocation.
func GCRootOwner(value ssa.Value) ssa.Value {
	for {
		switch v := value.(type) {
		case *ssa.FieldAddr:
			value = v.X
		case *ssa.IndexAddr:
			value = v.X
		case *ssa.Slice:
			if alloc, ok := v.X.(*ssa.Alloc); ok && alloc.Comment == "makeslice" {
				return value
			}
			value = v.X
		case *ssa.SliceToArrayPointer:
			value = v.X
		case *ssa.ChangeType:
			switch v.Type().Underlying().(type) {
			case *types.Pointer, *types.Slice:
			default:
				if kind := basicKindOf(v.Type()); kind != types.String && kind != types.UnsafePointer {
					return value
				}
			}
			value = v.X
		case *ssa.Convert:
			pointer := func(t types.Type) bool {
				_, ptr := t.Underlying().(*types.Pointer)
				return ptr || basicKindOf(t) == types.UnsafePointer
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
