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
	"go/types"
	"strings"
	"sync"

	"golang.org/x/tools/go/ssa"
)

func (p *context) canTailRecoverAlias(call *ssa.CallCommon) bool {
	target := p.prog.Target()
	return target != nil && target.GOARCH != "wasm" && recoverAliasTailPosition(p.goFn, call)
}

// recoverAliasTailPosition proves that an SSA-generated adapter immediately
// returns the forwarded result. Other wrappers retain their scoped cleanup.
// Keep composite results conservative: lowering may add result validation or
// storage work after their physical call.
func recoverAliasTailPosition(fn *ssa.Function, call *ssa.CallCommon) bool {
	if fn == nil || call == nil || fn.Recover != nil || !(strings.HasPrefix(fn.Synthetic, "wrapper for ") ||
		strings.HasPrefix(fn.Synthetic, "thunk for ") || strings.HasPrefix(fn.Synthetic, "bound method wrapper for ")) {
		return false
	}
	results := fn.Signature.Results()
	if results.Len() > 1 {
		return false
	}
	if results.Len() == 1 {
		switch typ := results.At(0).Type().Underlying().(type) {
		case *types.Basic:
			if typ.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat) == 0 {
				return false
			}
		case *types.Pointer:
		default:
			return false
		}
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if _, deferred := instr.(*ssa.Defer); deferred {
				return false
			}
		}
	}
	for _, block := range fn.Blocks {
		for i, instr := range block.Instrs {
			invoke, ok := instr.(*ssa.Call)
			if !ok || &invoke.Call != call || i+2 != len(block.Instrs) {
				continue
			}
			ret, ok := block.Instrs[i+1].(*ssa.Return)
			return ok && (len(ret.Results) == 0 || len(ret.Results) == 1 && ret.Results[0] == invoke)
		}
	}
	return false
}

// recoverFacts caches recover classification for one LLGo compilation. The
// build driver constructs all dependency Go SSA before consulting its package
// archive cache, so these facts remain available when a dependency backend is
// skipped on a cache hit. Keeping them here also avoids encoding frontend
// semantics in LLVM function attributes, which do not describe closure values.
type recoverFacts struct {
	mu     sync.RWMutex
	uses   map[*ssa.Function]bool
	scopes map[*ssa.Function]bool
}

func newRecoverFacts() *recoverFacts {
	return &recoverFacts{
		uses:   make(map[*ssa.Function]bool),
		scopes: make(map[*ssa.Function]bool),
	}
}

func (c *CallerTracking) recoverAnalysis() *recoverFacts {
	if c.recover == nil {
		c.recover = newRecoverFacts()
	}
	return c.recover
}

func (p *context) recoverAnalysis() *recoverFacts {
	if p.recoverFacts == nil {
		p.recoverFacts = newRecoverFacts()
	}
	return p.recoverFacts
}

func (r *recoverFacts) precompute(pkgs []*ssa.Package) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for _, member := range pkg.Members {
			if fn, ok := member.(*ssa.Function); ok {
				r.needsRecoverScopeLocked(fn)
			}
		}
	}
}

func (p *context) functionUsesRecover(fn *ssa.Function) bool {
	return p.recoverAnalysis().functionUsesRecover(fn)
}

func (r *recoverFacts) functionUsesRecover(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	r.mu.RLock()
	uses, ok := r.uses[fn]
	r.mu.RUnlock()
	if ok {
		return uses
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.functionUsesRecoverLocked(fn)
}

func (r *recoverFacts) functionUsesRecoverLocked(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	if uses, ok := r.uses[fn]; ok {
		return uses
	}
	uses := false
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(ssa.CallInstruction)
			if !ok {
				continue
			}
			builtin, ok := call.Common().Value.(*ssa.Builtin)
			if ok && builtin.Name() == "recover" {
				uses = true
				break
			}
		}
		if uses {
			break
		}
	}
	r.uses[fn] = uses
	return uses
}

func (p *context) needsRecoverScope(fn *ssa.Function) bool {
	return p.recoverAnalysis().needsRecoverScope(fn)
}

func (r *recoverFacts) needsRecoverScope(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	r.mu.RLock()
	needs, ok := r.scopes[fn]
	r.mu.RUnlock()
	if ok {
		return needs
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.needsRecoverScopeLocked(fn)
}

func (r *recoverFacts) needsRecoverScopeLocked(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	if needs, ok := r.scopes[fn]; ok {
		return needs
	}

	seen := make(map[*ssa.Function]bool)
	work := []*ssa.Function{fn}
	for len(work) > 0 {
		last := len(work) - 1
		current := work[last]
		work = work[:last]
		if current == nil || seen[current] {
			continue
		}
		if needs, ok := r.scopes[current]; ok {
			if needs {
				r.scopes[fn] = true
				return true
			}
			continue
		}
		seen[current] = true
		if r.functionUsesRecoverLocked(current) {
			r.scopes[fn] = true
			return true
		}
		if len(current.Blocks) == 0 {
			// Source-backed builds normally have every dependency body even
			// when its archive is cached. Be conservative for declarations from
			// isolated/allow-no-body compilation modes.
			r.scopes[fn] = true
			return true
		}
		if !isRecoverTransparentWrapper(current) {
			continue
		}
		for _, block := range current.Blocks {
			for _, instr := range block.Instrs {
				call, ok := instr.(ssa.CallInstruction)
				if !ok {
					continue
				}
				common := call.Common()
				if common.Method != nil {
					// An interface wrapper can dispatch to a recover-capable method.
					r.scopes[fn] = true
					return true
				}
				work = append(work, common.StaticCallee())
			}
		}
	}

	// A fully explored negative graph is safe to cache for every visited
	// function, including cycles of synthetic wrappers.
	for current := range seen {
		r.scopes[current] = false
	}
	return false
}

func isRecoverTransparentWrapper(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	// These are the Go SSA forms of compiler-generated forwarding frames.
	// Treating them as transparent mirrors the standard runtime's
	// abi.FuncIDWrapper rule in gorecover.
	return strings.HasPrefix(fn.Synthetic, "wrapper for ") ||
		strings.HasPrefix(fn.Synthetic, "thunk for ") ||
		strings.HasPrefix(fn.Synthetic, "bound method wrapper for ") ||
		isMidwayRecoverDispatcher(fn)
}

// Midway retains the source entry as a dispatcher, so it is not an SSA
// synthetic wrapper. Its sibling names contain compiler-only identifiers;
// ordinary Go source cannot declare or call them. Check the forwarding calls
// as well, excluding the retained trapping entries with dependent signatures.
func isMidwayRecoverDispatcher(fn *ssa.Function) bool {
	origin := fn.Origin()
	if origin == nil {
		origin = fn
	}
	if origin.Pkg == nil {
		return false
	}
	// The baseline variants are mandatory. Every additional variant present
	// in this package must be forwarded to as well.
	pending := make(map[*ssa.Function]bool)
	for _, variant := range []struct {
		suffix   string
		required bool
	}{{"@simd0", true}, {"@simd128", true}, {"@simd256", false}} {
		callee := origin.Pkg.Func(origin.Name() + variant.suffix)
		if callee == nil {
			if variant.required {
				return false
			}
			continue
		}
		pending[callee] = true
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			callee := call.Common().StaticCallee()
			if callee == nil {
				continue
			}
			if base := callee.Origin(); base != nil {
				callee = base
			}
			delete(pending, callee)
		}
	}
	return len(pending) == 0
}
