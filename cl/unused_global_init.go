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
	"crypto/sha256"
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"maps"
	"slices"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// UnusedGlobalInits describes package initializations whose destination is never
// accessed by the program. It is computed before parallel package lowering and
// is immutable afterwards. CacheKeys must participate in each package's cache key.
type UnusedGlobalInits struct {
	names  map[*ssa.Package][]string
	instrs map[*ssa.Package][]ssa.Instruction
	keys   map[*ssa.Package][]string
}

// CacheKeys identifies the exact instruction plan, including discarded blank
// initializers that have no named destination. Alternate packages have separate
// SSA identities, so their keys must also be included in the compiled module.
func (p *UnusedGlobalInits) CacheKeys(pkg *ssa.Package) []string {
	if p == nil {
		return nil
	}
	return p.keys[pkg]
}

// Names returns dead-global names for diagnostics; use CacheKeys for cache identity.
func (p *UnusedGlobalInits) Names(pkg *ssa.Package) []string {
	if p == nil {
		return nil
	}
	return p.names[pkg]
}

// FindUnusedGlobalInits deliberately examines all functions, including unreachable
// methods and external entry points. The caller must preserve globals exposed
// through aliases or native code, and must not apply this plan to a library.
// Removing dead stores and map operations leaves operand evaluation in place,
// including calls, channel receives, and expressions that may panic. A fixed
// point also removes now-unused global loads. Pointer-bearing globals remain
// GC roots unless their initializer graph is provably private and constant.
func FindUnusedGlobalInits(prog *ssa.Program, preserve func(*ssa.Global) bool) *UnusedGlobalInits {
	plan := &UnusedGlobalInits{names: make(map[*ssa.Package][]string), instrs: make(map[*ssa.Package][]ssa.Instruction), keys: make(map[*ssa.Package][]string)}
	uses := make(map[*ssa.Global][]ssa.Instruction)
	for fn := range ssautil.AllFunctions(prog) {
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if _, debug := instr.(*ssa.DebugRef); debug {
					continue
				}
				for _, operand := range instr.Operands(nil) {
					if global, ok := (*operand).(*ssa.Global); ok {
						uses[global] = append(uses[global], instr)
					}
				}
			}
		}
	}
	// Export/native visibility is immutable across the fixed point.
	preserved := make(map[*ssa.Global]bool, len(uses))
	for global := range uses {
		preserved[global] = preserve(global)
	}
	proof := globalInitRootProof{uses: uses, preserved: maps.Clone(preserved), checked: make(map[*ssa.Global]bool), safe: make(map[*ssa.Global]bool)}
	keepRoots := make(map[*ssa.Global]bool)
	for global := range uses {
		// A global is a GC root even if no SSA instruction reads it. Calls
		// in its initializer can return objects with registered finalizers,
		// including pointers nested in aggregates, maps and closure contexts.
		// Keep these roots until their lifetime can be proved unobservable.
		if mayContainGCRoot(global.Type().Underlying().(*types.Pointer).Elem()) && !proof.global(global) {
			keepRoots[global] = true
		}
	}
	for global := range keepRoots {
		preserved[global] = true
	}
	removed := make(map[ssa.Instruction]bool)
	deadGlobals := make(map[*ssa.Global]bool)
	var inits []ssa.Instruction
	for _, pkg := range prog.AllPackages() {
		if init := pkg.Func("init"); init != nil && init.Synthetic == "package initializer" {
			for _, block := range init.Blocks {
				inits = append(inits, block.Instrs...)
			}
		}
	}
	live := func(refs []ssa.Instruction) []ssa.Instruction {
		var result []ssa.Instruction
		for _, ref := range refs {
			if _, debug := ref.(*ssa.DebugRef); !debug && !removed[ref] {
				result = append(result, ref)
			}
		}
		return result
	}
	for changed := true; changed; {
		changed = false
		remove := func(instr ssa.Instruction) {
			if !removed[instr] {
				removed[instr] = true
				changed = true
			}
		}
		for global, refs := range uses {
			if deadGlobals[global] || preserved[global] {
				continue
			}
			init := global.Pkg.Func("init")
			if init == nil || init.Synthetic != "package initializer" {
				continue
			}
			refs = live(refs)
			valid := true
			for _, ref := range refs {
				store, ok := ref.(*ssa.Store)
				if !ok || store.Addr != global || store.Parent() != init {
					valid = false
					break
				}
			}
			if !valid {
				continue
			}
			deadGlobals[global] = true
			plan.names[global.Pkg] = append(plan.names[global.Pkg], global.String())
			for _, ref := range refs {
				remove(ref)
			}
		}
		for _, instr := range inits {
			if removed[instr] {
				continue
			}
			value, ok := instr.(ssa.Value)
			if !ok || value.Referrers() == nil {
				continue
			}
			refs := live(*value.Referrers())
			switch value := value.(type) {
			case *ssa.UnOp:
				// Loads from ordinary Go globals cannot fault. Removing unused
				// loads exposes chains such as B = A after B's store is removed.
				global, ok := value.X.(*ssa.Global)
				if ok && value.Op == token.MUL && len(refs) == 0 && !preserved[global] {
					remove(instr)
				}
			case *ssa.MakeMap:
				if !discardableMap(value) {
					continue
				}
				valid := true
				for _, ref := range refs {
					update, ok := ref.(*ssa.MapUpdate)
					if !ok || update.Map != value || update.Key == value || update.Value == value {
						valid = false
						break
					}
				}
				if valid {
					remove(instr)
					for _, ref := range refs {
						remove(ref)
					}
				}
			}
		}
	}
	for instr := range removed {
		plan.instrs[instr.Parent().Pkg] = append(plan.instrs[instr.Parent().Pkg], instr)
		if value, ok := instr.(ssa.Value); ok && value.Referrers() != nil {
			for _, ref := range *value.Referrers() {
				if _, debug := ref.(*ssa.DebugRef); debug {
					plan.instrs[instr.Parent().Pkg] = append(plan.instrs[instr.Parent().Pkg], ref)
				}
			}
		}
	}

	for _, names := range plan.names {
		slices.Sort(names)
	}
	// Every removal above belongs to a synthetic package initializer: stores
	// check their parent explicitly, and other instructions come from inits.
	for pkg := range plan.instrs {
		hash := sha256.New()
		for _, block := range pkg.Func("init").Blocks {
			for i, instr := range block.Instrs {
				if removed[instr] {
					fmt.Fprintf(hash, "%d:%d;", block.Index, i)
				}
			}
		}
		plan.keys[pkg] = []string{fmt.Sprintf("%s:%x", pkg.Pkg.Path(), hash.Sum(nil))}
	}
	return plan
}

// A constant allocation can still acquire a finalizer through an alias. Loads
// from another global are safe only if that root cannot be replaced or escape
// through its address. Any remaining load keeps the source root alive in the
// fixed point; removing its last load requires the same proof for its contents.
type globalInitRootProof struct {
	uses      map[*ssa.Global][]ssa.Instruction
	preserved map[*ssa.Global]bool
	checked   map[*ssa.Global]bool
	safe      map[*ssa.Global]bool
}

func (p *globalInitRootProof) global(global *ssa.Global) bool {
	if p.preserved[global] || p.checked[global] {
		return p.safe[global]
	}
	p.checked[global] = true // Cycles fail closed.
	init := global.Pkg.Func("init")
	if init == nil || init.Synthetic != "package initializer" {
		return false
	}
	var initializer *ssa.Store
	for _, ref := range p.uses[global] {
		switch ref := ref.(type) {
		case *ssa.Store:
			if ref.Addr != global || ref.Parent() != init || initializer != nil {
				return false
			}
			initializer = ref
		case *ssa.UnOp:
			if ref.Op != token.MUL || ref.X != global {
				return false
			}
		default:
			return false
		}
	}
	p.safe[global] = initializer == nil || p.value(initializer.Val, initializer)
	return p.safe[global]
}

func (p *globalInitRootProof) value(value ssa.Value, consumer ssa.Instruction) bool {
	switch value := value.(type) {
	case *ssa.Const:
		return true
	case *ssa.UnOp:
		if global, ok := value.X.(*ssa.Global); ok && value.Op == token.MUL {
			return p.global(global)
		}
	case *ssa.MakeMap:
		if !discardableMap(value) {
			return false
		}
		// Without referrers, the consumer check below fails closed.
		refs, _ := nonDebugReferrers(value)
		seenConsumer := false
		for _, ref := range refs {
			if ref == consumer {
				seenConsumer = true
				continue
			}
			update, ok := ref.(*ssa.MapUpdate)
			if !ok || update.Map != value || update.Key == value || update.Value == value ||
				!p.value(update.Key, update) || !p.value(update.Value, update) {
				return false
			}
		}
		return seenConsumer
	}
	// Reuse the static initializer checks: every allocation must have a single
	// terminal use, and its private fields/backing arrays must contain constants.
	store, ok := consumer.(*ssa.Store)
	if !ok || store.Val != value {
		return false
	}
	var stores []staticInitStore
	var instrs []ssa.Instruction
	if collectStaticInitLeaf(store, nil, &stores, &instrs, nil) {
		return staticInitRootStores(stores)
	}
	if load, ok := value.(*ssa.UnOp); ok && load.Op == token.MUL {
		if alloc, ok := load.X.(*ssa.Alloc); ok && !alloc.Heap {
			return collectAllocStores(alloc, load, store, nil, &stores, &instrs, make(map[*ssa.Alloc]bool)) && staticInitRootStores(stores)
		}
	}
	return false
}

func staticInitRootStores(stores []staticInitStore) bool {
	for _, store := range stores {
		// SSA free variables do not describe implicit //llgo:env contexts.
		// Their final ABI is resolved later during static initializer lowering.
		if store.function != nil || (store.slice != nil && !staticInitRootStores(store.slice.stores)) ||
			(store.pointer != nil && !staticInitRootStores(store.pointer.stores)) {
			return false
		}
	}
	return true
}

// Map construction and writes can be removed only when they cannot introduce a
// language-level panic beyond the separately evaluated operands.
func discardableMap(value *ssa.MakeMap) bool {
	if hint := value.Reserve; hint != nil {
		c, ok := hint.(*ssa.Const)
		if !ok || c.Value == nil || c.Value.Kind() != constant.Int || constant.Sign(c.Value) < 0 {
			return false
		}
		if n, exact := constant.Int64Val(c.Value); !exact || n > maxStaticInitArrayElements {
			return false
		}
	}
	return mapKeyCannotPanic(value.Type().Underlying().(*types.Map).Key())
}

func mapKeyCannotPanic(typ types.Type) bool {
	switch typ := typ.Underlying().(type) {
	case *types.Interface:
		return false
	case *types.Array:
		return mapKeyCannotPanic(typ.Elem())
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if !mapKeyCannotPanic(typ.Field(i).Type()) {
				return false
			}
		}
	}
	return true
}

func (p *context) collectUnusedGlobalInits(pkg *ssa.Package) {
	if p.options.UnusedGlobalInits == nil {
		return
	}
	p.ensureStaticInitState()
	for _, instr := range p.options.UnusedGlobalInits.instrs[pkg] {
		p.staticInitInstrs[instr] = none{}
	}
}
