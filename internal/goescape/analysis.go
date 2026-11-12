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

package goescape

import (
	"go/types"
	"sort"

	"golang.org/x/tools/go/ssa"
)

// Version participates in package fingerprints, including compiler API users
// that do not supply a development compiler hash. Bump when semantic contracts
// or placement proofs change incompatibly with previously emitted archives.
const Version = 16

// Config supplies the backend's representation and resolved call contracts.
// Callbacks run synchronously during Analyze. Results do not retain them.
type Config struct {
	// Cache optionally persists complete, validated facts for this program.
	Cache *Cache

	// Inspect must reject bodies replaced by linkname, source patches, native
	// declarations or intrinsics unless the replacement itself is being analyzed.
	Inspect func(*ssa.Function) bool
	// External provides a validated summary for an uninspectable function.
	// Missing contracts conservatively publish every argument.
	External func(*ssa.Function) (Summary, bool)
	// ExternalKey identifies the validated component that compiled an opaque
	// dependency. It keeps callers' component keys stable when only bodies
	// have been replaced by exported contracts.
	ExternalKey func(*ssa.Function) string
	// IndirectInterface describes the actual interface data representation.
	// The default uses two-word Go function values, as LLGo does.
	IndirectInterface func(types.Type) bool
	// ForceHeap runs before solving, so pointers inside forced-heap storage
	// also escape. An empty reason permits ordinary lifetime analysis.
	ForceHeap func(ssa.Instruction, types.Type) string
	// StackLayout describes newly introduced stack storage. Zero size needs
	// no reservation; alignment must be a positive power of two.
	StackLayout func(ssa.Instruction, types.Type) (size, alignment uint64)
	// LocalStackLayout accounts for newly required root slots of storage
	// already local in Go SSA. The original object bytes are not new storage.
	LocalStackLayout func(ssa.Instruction, types.Type) (size, alignment uint64)
	// MaxStackBytes bounds new storage per function, before final call effects.
	// Zero leaves the target's existing per-object policy unchanged.
	MaxStackBytes uint64
}

// Effect describes a minimum indirection depth. Zero means absent; a positive
// value encodes depth+1. Keeping this encoding makes the zero Summary useful
// as the initial element of the recursive-group fixed point.
type Effect uint16

type Parameter struct {
	Heap, Mutator, Callee Effect
	Results               []Effect
}

// Summary parameters are receivers/ordinary parameters followed by free vars.
// Exported copies may be serialized; Analyze validates imported dimensions.
type Summary struct {
	Parameters []Parameter
}

type Decision struct {
	Heap   bool
	Reason string
	// ReadOnly permits a proven byte/string conversion to borrow its input.
	ReadOnly bool
}

// Result is immutable after Analyze returns and may be read by backend workers.
type Result struct {
	objects       map[ssa.Instruction]Decision
	summaries     map[*ssa.Function]Summary
	stackBytes    map[*ssa.Function]uint64
	captures      map[*ssa.FreeVar]bool
	componentKeys map[*ssa.Function]string
}

// CaptureByValue reports whether lowering must store the captured value rather
// than its variable's address. The shared SSA itself is never rewritten.
func (r *Result) CaptureByValue(v *ssa.FreeVar) bool {
	return r != nil && r.captures[v]
}

func (r *Result) ComponentKey(fn *ssa.Function) string {
	if r == nil {
		return ""
	}
	return r.componentKeys[fn]
}

// StackBytes is a conservative reservation for the function's promoted Go
// objects. Lowering helpers can reserve additional storage from the same budget.
func (r *Result) StackBytes(fn *ssa.Function) uint64 {
	if r == nil {
		return 0
	}
	return r.stackBytes[fn]
}

func (r *Result) Decision(site ssa.Instruction) Decision {
	if decision, ok := r.LookupDecision(site); ok {
		return decision
	}
	return Decision{Heap: true, Reason: "missing analysis"}
}

// LookupDecision distinguishes analyzed objects from bodies that were excluded
// because their symbols use another ABI or have a replacement implementation.
func (r *Result) LookupDecision(site ssa.Instruction) (Decision, bool) {
	if r != nil {
		decision, ok := r.objects[site]
		return decision, ok
	}
	return Decision{}, false
}

func (r *Result) Summary(fn *ssa.Function) (Summary, bool) {
	if r == nil {
		return Summary{}, false
	}
	summary, ok := r.summaries[fn]
	return cloneSummary(summary), ok
}

func cloneSummary(s Summary) Summary {
	ret := Summary{Parameters: append([]Parameter(nil), s.Parameters...)}
	for i := range ret.Parameters {
		ret.Parameters[i].Results = append([]Effect(nil), s.Parameters[i].Results...)
	}
	return ret
}

func emptySummary(fn *ssa.Function) Summary {
	s := Summary{Parameters: make([]Parameter, parameterCount(fn))}
	for i := range s.Parameters {
		s.Parameters[i].Results = make([]Effect, fn.Signature.Results().Len())
	}
	return s
}

func parameterCount(fn *ssa.Function) int {
	n := fn.Signature.Params().Len() + len(fn.FreeVars)
	if fn.Signature.Recv() != nil {
		n++
	}
	return n
}

func unknownSummary(fn *ssa.Function) Summary {
	s := emptySummary(fn)
	for i := range s.Parameters {
		s.Parameters[i].Heap = 1
	}
	return s
}

func validSummary(fn *ssa.Function, s Summary) bool {
	if len(s.Parameters) != parameterCount(fn) {
		return false
	}
	for _, p := range s.Parameters {
		if len(p.Results) != fn.Signature.Results().Len() {
			return false
		}
	}
	return true
}

// ValidSummary checks the dimensions of a body-independent imported contract.
// In particular, an unbuilt dependency may not have SSA Params yet.
func ValidSummary(fn *ssa.Function, s Summary) bool { return validSummary(fn, s) }

func mergeSummary(dst *Summary, src Summary) (changed bool) {
	merge := func(a *Effect, b Effect) {
		if b != 0 && (*a == 0 || b < *a) {
			*a = b
			changed = true
		}
	}
	for i := range dst.Parameters {
		a, b := &dst.Parameters[i], src.Parameters[i]
		merge(&a.Heap, b.Heap)
		merge(&a.Mutator, b.Mutator)
		merge(&a.Callee, b.Callee)
		for j := range a.Results {
			merge(&a.Results[j], b.Results[j])
		}
	}
	return
}

type analyzer struct {
	config    Config
	result    *Result
	bodies    map[*ssa.Function]bool
	functions map[*ssa.Function]bool
	scopes    map[*ssa.Function]*scopes
	deps      map[*ssa.Function][]*ssa.Function
	// Only completed SCCs may supply contextual proofs. Generic placement
	// remains authoritative even when a particular caller has purer arguments.
	solved map[*ssa.Function]bool
}

func (a *analyzer) inspect(fn *ssa.Function) bool {
	return fn != nil && len(fn.Blocks) != 0 && (a.config.Inspect == nil || a.config.Inspect(fn))
}

func (a *analyzer) summary(fn *ssa.Function) Summary {
	if s, ok := a.result.summaries[fn]; ok {
		return s
	}
	if a.config.External != nil {
		if s, ok := a.config.External(fn); ok && validSummary(fn, s) {
			return s
		}
	}
	return unknownSummary(fn)
}

// Analyze follows inspectable static callees and nested functions from roots.
// Each call SCC reaches a monotone summary fixed point before its callers.
func Analyze(roots []*ssa.Function, config Config) *Result {
	a := &analyzer{config: config, result: &Result{
		objects: make(map[ssa.Instruction]Decision), summaries: make(map[*ssa.Function]Summary), stackBytes: make(map[*ssa.Function]uint64), captures: make(map[*ssa.FreeVar]bool), componentKeys: make(map[*ssa.Function]string),
	}, bodies: make(map[*ssa.Function]bool), functions: make(map[*ssa.Function]bool), scopes: make(map[*ssa.Function]*scopes), deps: make(map[*ssa.Function][]*ssa.Function), solved: make(map[*ssa.Function]bool)}
	var discover func(*ssa.Function)
	discover = func(fn *ssa.Function) {
		if fn == nil || a.functions[fn] {
			return
		}
		a.functions[fn] = true
		if !a.inspect(fn) {
			return
		}
		a.bodies[fn] = true
		for i, v := range fn.FreeVars {
			a.result.captures[v] = immutableCapture(fn, i)
		}
		a.result.summaries[fn] = emptySummary(fn)
		for _, callee := range a.dependencies(fn) {
			discover(callee)
		}
	}
	for _, root := range roots {
		discover(root)
	}
	cacheBase, cacheFunctions := a.cacheIdentity()
	cacheKeys := make(map[*ssa.Function]string)
	cacheRecords := make(map[*ssa.Function]cacheFunction)
	for _, item := range cacheFunctions {
		cacheRecords[item.fn] = item
		if !a.bodies[item.fn] {
			cacheKeys[item.fn] = item.key
		}
	}
	for _, group := range a.functionComponents() {
		cacheKey, records := a.componentCache(cacheBase, cacheRecords, cacheKeys, group)
		if cached := a.loadCache(cacheKey, records); cached != nil {
			for fn, summary := range cached.summaries {
				a.result.summaries[fn] = summary
				a.result.stackBytes[fn] = cached.StackBytes(fn)
				a.solved[fn] = true
				a.result.componentKeys[fn] = cacheKey
				cacheKeys[fn] = cacheKey
			}
			for site, decision := range cached.objects {
				a.result.objects[site] = decision
			}
			continue
		}
		for iteration := 0; ; iteration++ {
			changed := false
			for _, fn := range group {
				s, decisions := a.analyzeFunction(fn)
				previous := a.result.summaries[fn]
				changed = mergeSummary(&previous, s) || changed
				a.result.summaries[fn] = previous
				for site, decision := range decisions {
					a.result.objects[site] = decision
				}
			}
			if !changed {
				// No summary changed during this pass, so its storage decisions
				// already use the converged effects. Do not rebuild once more.
				break
			}
			if iteration >= 63 {
				// Bound pathological recursive graphs. Publishing arguments is
				// conservative even when a smaller fixed point was possible.
				for _, fn := range group {
					a.result.summaries[fn] = unknownSummary(fn)
				}
				// Only the bounded fallback needs another solve: the final
				// decisions must observe the newly conservative call effects.
				for _, fn := range group {
					_, decisions := a.analyzeFunction(fn)
					for site, decision := range decisions {
						a.result.objects[site] = decision
					}
				}
				break
			}
		}
		for _, fn := range group {
			a.solved[fn] = true
			a.result.componentKeys[fn] = cacheKey
			cacheKeys[fn] = cacheKey
		}
		a.storeCache(cacheKey, records)
	}
	return a.result
}

func (a *analyzer) dependencies(fn *ssa.Function) []*ssa.Function {
	if deps, ok := a.deps[fn]; ok {
		return deps
	}
	ret := append([]*ssa.Function(nil), fn.AnonFuncs...)
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if value, ok := instr.(*ssa.MakeInterface); ok {
				methods := fn.Prog.MethodSets.MethodSet(value.X.Type())
				for i := 0; i < methods.Len(); i++ {
					if method := fn.Prog.MethodValue(methods.At(i)); method != nil {
						ret = append(ret, method)
					}
				}
			}
			for _, operand := range instr.Operands(nil) {
				if operand != nil {
					if fn, ok := (*operand).(*ssa.Function); ok {
						ret = append(ret, fn)
					}
				}
			}
			if call, ok := instr.(ssa.CallInstruction); ok {
				for _, target := range callTargets(call.Common()) {
					ret = append(ret, target.fn)
				}
			}
		}
	}
	a.deps[fn] = ret
	return ret
}

func (a *analyzer) functionComponents() [][]*ssa.Function {
	index, low := make(map[*ssa.Function]int), make(map[*ssa.Function]int)
	active := make(map[*ssa.Function]bool)
	var stack []*ssa.Function
	var groups [][]*ssa.Function
	next := 1
	var visit func(*ssa.Function)
	visit = func(fn *ssa.Function) {
		index[fn], low[fn] = next, next
		next++
		stack = append(stack, fn)
		active[fn] = true
		for _, callee := range a.dependencies(fn) {
			if !a.bodies[callee] {
				continue
			}
			if index[callee] == 0 {
				visit(callee)
				if low[callee] < low[fn] {
					low[fn] = low[callee]
				}
			} else if active[callee] && index[callee] < low[fn] {
				low[fn] = index[callee]
			}
		}
		if low[fn] == index[fn] {
			var group []*ssa.Function
			for {
				i := len(stack) - 1
				member := stack[i]
				stack = stack[:i]
				active[member] = false
				group = append(group, member)
				if member == fn {
					break
				}
			}
			groups = append(groups, group)
		}
	}
	var roots []*ssa.Function
	for fn := range a.bodies {
		roots = append(roots, fn)
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].String() < roots[j].String() })
	for _, fn := range roots {
		if index[fn] == 0 {
			visit(fn)
		}
	}
	return groups
}
