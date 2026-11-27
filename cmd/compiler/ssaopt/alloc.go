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
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/abipolicy"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// AllocPass computes compiler allocation placement, GC scanning (NoScan),
// entry-block hoisting, dead-allocation, and recover-volatile policies for
// every *ssa.Alloc instruction in a function.
//
// Pointer presence comes from llgo/types (Type.HasPtrData); the actual NoScan
// allocation policy lives here rather than in root llgo/ssa so runtime pragmas
// such as //go:uintptrescapes are honored consistently.
type AllocPass struct{}

func (AllocPass) Name() string { return "alloc" }

func (AllocPass) Run(fn *ssa.Function, cfg Config, plan *FunctionPlan) error {
	if fn == nil || len(fn.Blocks) == 0 {
		return nil
	}
	isInit := fn.Synthetic == "package initializer" || fn.Name() == "init"
	var allocs map[*ssa.Alloc]AllocPlan
	for _, blk := range fn.Blocks {
		inLoop := blk.Index >= 0 && blk.Index < len(plan.InLoopBlock) && plan.InLoopBlock[blk.Index]
		for _, ins := range blk.Instrs {
			alloc, ok := ins.(*ssa.Alloc)
			if !ok {
				continue
			}
			ptr, ok := alloc.Type().(*types.Pointer)
			if !ok || ptr.Elem() == nil {
				continue
			}
			elem := ptr.Elem()
			patchedElem := patchEscapeType(elem, cfg.PatchTypes)
			dead := hasZeroNonDebugReferrers(alloc)
			heap := decideAllocHeap(alloc, patchedElem, cfg)
			uintptrEsc := heap && UintptrEscapesBackingArray(alloc)
			noScan := !elem.HasPtrData() && !patchedElem.HasPtrData() && !uintptrEsc
			hoistToEntry := true
			if isInit {
				hoistToEntry = inLoop
			}
			recoverVolatile := !heap && fn.Recover != nil
			if allocs == nil {
				allocs = make(map[*ssa.Alloc]AllocPlan)
			}
			allocs[alloc] = AllocPlan{
				Heap:            heap,
				NoScan:          noScan,
				UintptrEscapes:  uintptrEsc,
				HoistToEntry:    hoistToEntry,
				Dead:            dead,
				RecoverVolatile: recoverVolatile,
			}
		}
	}
	plan.Allocs = allocs
	return nil
}

func decideAllocHeap(alloc *ssa.Alloc, elem types.Type, cfg Config) bool {
	if FreshNilAppendAllocation(alloc) {
		return true
	}
	heap := alloc.Heap
	if cfg.Escape != nil {
		if decision, ok := cfg.Escape.LookupDecision(alloc); ok {
			heap = decision.Heap
		}
	}
	if !heap && cfg.TypeCtx != nil && uint64(cfg.TypeCtx.Sizeof(elem)) > abipolicy.MaxStackVarSize {
		heap = true
	}
	return heap
}

func hasZeroNonDebugReferrers(v ssa.Value) bool {
	refs := v.Referrers()
	if refs == nil {
		return false
	}
	for _, ref := range *refs {
		if _, isDbg := ref.(*ssa.DebugRef); !isDbg {
			return false
		}
	}
	return true
}

// UintptrEscapesBackingArray identifies compiler-created variadic storage whose
// uintptr elements retain pointers for a //go:uintptrescapes call. Such storage
// must remain scanned while a go/defer call is pending, even though uintptr has
// no pointers in its ordinary type layout.
func UintptrEscapesBackingArray(alloc *ssa.Alloc) bool {
	if alloc == nil || alloc.Comment != "varargs" {
		return false
	}
	ptr, ok := alloc.Type().(*types.Pointer)
	if !ok {
		return false
	}
	array, ok := ptr.Elem().Underlying().(*types.Array)
	if !ok || basicKindOf(array.Elem()) != types.Uintptr {
		return false
	}
	seen := make(map[ssa.Value]bool)
	var usedByMarkedCall func(ssa.Value) bool
	usedByMarkedCall = func(value ssa.Value) bool {
		if value == nil || seen[value] {
			return false
		}
		seen[value] = true
		refs := value.Referrers()
		if refs == nil {
			return false
		}
		for _, ref := range *refs {
			if call, ok := ref.(ssa.CallInstruction); ok {
				common := call.Common()
				if common.Signature().Variadic() && len(common.Args) != 0 &&
					common.Args[len(common.Args)-1] == value &&
					HasUintptrEscapesDirective(common.StaticCallee()) {
					return true
				}
			}
			switch ref := ref.(type) {
			case *ssa.Slice:
				if usedByMarkedCall(ref) {
					return true
				}
			case *ssa.ChangeType:
				if usedByMarkedCall(ref) {
					return true
				}
			case *ssa.Phi:
				if usedByMarkedCall(ref) {
					return true
				}
			}
		}
		return false
	}
	return usedByMarkedCall(alloc)
}

// HasUintptrEscapesDirective reports whether fn (or its generic origin /
// transparent method wrapper target) carries //go:uintptrescapes.
func HasUintptrEscapesDirective(fn *ssa.Function) bool {
	seen := make(map[*ssa.Function]bool)
	var visit func(*ssa.Function) bool
	visit = func(fn *ssa.Function) bool {
		if fn == nil || seen[fn] {
			return false
		}
		seen[fn] = true
		if origin := fn.Origin(); origin != nil && visit(origin) {
			return true
		}
		if hasFuncDirective(fn, "go:uintptrescapes") {
			return true
		}
		if !strings.HasPrefix(fn.Synthetic, "wrapper for ") &&
			!strings.HasPrefix(fn.Synthetic, "thunk for ") &&
			!strings.HasPrefix(fn.Synthetic, "bound method wrapper for ") {
			return false
		}
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if call, ok := instr.(*ssa.Call); ok && visit(call.Call.StaticCallee()) {
					return true
				}
			}
		}
		return false
	}
	return visit(fn)
}

func basicKindOf(typ types.Type) types.BasicKind {
	if b, ok := types.Unalias(typ).Underlying().(*types.Basic); ok {
		return b.Kind()
	}
	return types.Invalid
}

// computeLoopBlocks uses Tarjan's strongly connected components algorithm to
// identify blocks that participate in any CFG cycle (including self-loops).
func computeLoopBlocks(blks []*ssa.BasicBlock) []bool {
	n := len(blks)
	inLoop := make([]bool, n)
	if n == 0 {
		return inLoop
	}
	index := make([]int, n)
	lowlink := make([]int, n)
	onStack := make([]bool, n)
	for i := range index {
		index[i] = -1
	}
	var stack []int
	nextIndex := 0
	var strongConnect func(v int)
	strongConnect = func(v int) {
		index[v] = nextIndex
		lowlink[v] = nextIndex
		nextIndex++
		stack = append(stack, v)
		onStack[v] = true
		for _, succ := range blks[v].Succs {
			w := succ.Index
			if w < 0 || w >= n {
				continue
			}
			if index[w] == -1 {
				strongConnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] {
				if index[w] < lowlink[v] {
					lowlink[v] = index[w]
				}
			}
		}
		if lowlink[v] == index[v] {
			var comp []int
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			if len(comp) > 1 {
				for _, id := range comp {
					inLoop[id] = true
				}
			} else if len(comp) == 1 {
				u := comp[0]
				for _, succ := range blks[u].Succs {
					if succ.Index == u {
						inLoop[u] = true
						break
					}
				}
			}
		}
	}
	for i := 0; i < n; i++ {
		if index[i] == -1 {
			strongConnect(i)
		}
	}
	return inLoop
}
