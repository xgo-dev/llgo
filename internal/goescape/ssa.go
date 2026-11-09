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
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

type object struct {
	location *location
	reason   string
	typ      types.Type
}

type functionAnalysis struct {
	a             *analyzer
	g             *graph
	fn            *ssa.Function
	scopes        *scopes
	values        map[ssa.Value]*location
	tuples        map[ssa.Value][]*location
	objects       map[ssa.Instruction]*object
	params        []*location
	results       []*location
	sinks         map[string]*location
	bindings      map[ssa.Value]ssa.Value
	contextDepth  int
	contextBudget *int
	borrowProofs  map[*ssa.Convert]bool
}

func (a *analyzer) analyzeFunction(fn *ssa.Function) (Summary, map[ssa.Instruction]Decision) {
	budget := 64
	return a.analyzeBoundFunction(fn, nil, 0, &budget)
}

func (a *analyzer) analyzeBoundFunction(fn *ssa.Function, bindings map[ssa.Value]ssa.Value, depth int, budget *int) (Summary, map[ssa.Instruction]Decision) {
	if a.scopes[fn] == nil {
		a.scopes[fn] = functionScopes(fn)
	}
	f := &functionAnalysis{a: a, g: newGraph(), fn: fn,
		scopes: a.scopes[fn], values: make(map[ssa.Value]*location),
		tuples: make(map[ssa.Value][]*location), objects: make(map[ssa.Instruction]*object), bindings: bindings, contextDepth: depth, contextBudget: budget}
	for _, param := range fn.Params {
		l := f.value(param)
		l.param = true
		f.params = append(f.params, l)
	}
	for _, param := range fn.FreeVars {
		l := f.value(param)
		l.param = true
		f.params = append(f.params, l)
	}
	for i := 0; i < fn.Signature.Results().Len(); i++ {
		l := f.g.location("result", f.scopes.invocation)
		l.result = i + 1
		f.results = append(f.results, l)
	}
	// Create storage before visiting uses: phi backedges and block ordering
	// must not change whether a store resolves to a known local object.
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			typ := a.objectType(instr)
			if typ == nil {
				continue
			}
			reason := ""
			if _, channel := instr.(*ssa.MakeChan); channel {
				reason = "channel storage"
			}
			f.newObject(instr, typ, reason)
			if _, slice := instr.(*ssa.MakeSlice); slice {
				// Dynamic heap fallbacks may retain pointer elements.
				f.g.flow(&f.g.heap, f.objects[instr].location, 0)
			}
		}
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			f.instruction(instr)
		}
	}
	complete := f.g.solve()
	// First prove that a byte conversion is neither retained nor mutated.
	// Then add its borrowed-data edge and solve again, so source lifetimes
	// and exported effects also describe the no-copy lowering. A failed or
	// revoked proof retains the ordinary copying/heap fallback.
	borrowed := false
	if complete {
		for site, obj := range f.objects {
			if conversion, ok := site.(*ssa.Convert); ok && f.borrowableConversion(conversion) && obj.location.attrs&(escapes|mutates) == 0 {
				f.flow(f.value(conversion), conversion.X, 0)
				borrowed = true
			}
		}
		if borrowed {
			complete = f.g.solve()
		}
	}
	if complete && bindings == nil {
		complete = f.applyStackBudget()
	}
	summary := emptySummary(fn)
	for i, param := range f.params {
		p := &summary.Parameters[i]
		p.Heap, p.Mutator, p.Callee = Effect(param.leaks.heap), Effect(param.leaks.mutator), Effect(param.leaks.callee)
		for j, effect := range param.leaks.results {
			p.Results[j] = Effect(effect)
		}
	}
	decisions := make(map[ssa.Instruction]Decision, len(f.objects))
	for site, obj := range f.objects {
		heap := obj.location.attrs&escapes != 0
		reason := obj.reason
		if reason == "" && heap {
			reason = obj.location.reason
		}
		if !complete {
			heap, reason = true, "analysis work limit"
		}
		readOnly := false
		if conversion, ok := site.(*ssa.Convert); ok {
			readOnly = complete && !heap && obj.location.attrs&mutates == 0 && f.borrowableConversion(conversion)
		}
		decisions[site] = Decision{Heap: heap, Reason: reason, ReadOnly: readOnly}
	}
	if !complete {
		// Partial proofs cannot be exported: callers must also retain anything
		// reachable through arguments when this function exceeded the budget.
		summary = unknownSummary(fn)
		if bindings == nil {
			a.result.stackBytes[fn] = 0
		}
	}
	return summary, decisions
}

func (f *functionAnalysis) applyStackBudget() bool {
	config := f.a.config
	if config.MaxStackBytes == 0 || config.StackLayout == nil {
		return true
	}
	var used uint64
	forced := false
	// SSA order makes selection independent of map iteration and worker count.
	for _, block := range f.fn.Blocks {
		for _, site := range block.Instrs {
			obj := f.objects[site]
			if obj == nil || obj.location.attrs&escapes != 0 {
				continue
			}
			layout := config.StackLayout
			if alloc, ok := site.(*ssa.Alloc); ok && !alloc.Heap {
				layout = config.LocalStackLayout
				if layout == nil {
					continue
				}
			}
			if conversion, ok := site.(*ssa.Convert); ok && f.borrowableConversion(conversion) && obj.location.attrs&mutates == 0 {
				continue // A borrowed view reserves no character buffer.
			}
			size, alignment := layout(site, obj.typ)
			if size == 0 {
				continue
			}
			reason := "function stack budget"
			if alignment == 0 || alignment&(alignment-1) != 0 {
				reason = "unknown stack alignment"
			} else if padding := -used & (alignment - 1); padding <= config.MaxStackBytes-used && size <= config.MaxStackBytes-used-padding {
				used += padding + size
				continue
			}
			obj.reason, obj.location.reason = reason, reason
			f.g.flow(&f.g.heap, obj.location, -1)
			forced = true
		}
	}
	f.a.result.stackBytes[f.fn] = used
	// A rejected container can retain other objects and caller parameters.
	// Propagate that fact before exporting summaries or placement decisions.
	return !forced || f.g.solve()
}

func (f *functionAnalysis) sink(reason string) *location {
	if f.sinks == nil {
		f.sinks = make(map[string]*location)
	}
	if l := f.sinks[reason]; l != nil {
		return l
	}
	l := f.g.location(reason, nil)
	l.attrs, l.reason = escapes|persists|mutates|calls, reason
	f.sinks[reason] = l
	return l
}

func (f *functionAnalysis) value(v ssa.Value) *location {
	if v == nil {
		return nil
	}
	if l := f.values[v]; l != nil {
		return l
	}
	l := f.g.location(v.Name(), f.scopes.valueScope(v))
	f.values[v] = l
	return l
}

func (f *functionAnalysis) tuple(v ssa.Value, index int) *location {
	parts, ok := f.tuples[v]
	if !ok {
		typ := v.Type().Underlying().(*types.Tuple)
		parts = make([]*location, typ.Len())
		for i := range parts {
			parts[i] = f.g.location(v.Name()+".part", f.scopes.valueScope(v))
		}
		f.tuples[v] = parts
	}
	return parts[index]
}

func (f *functionAnalysis) flow(dst *location, src ssa.Value, depth int) {
	if src != nil && dst != nil && containsPointers(src.Type()) {
		f.g.flow(dst, f.value(src), depth)
	}
}

func (f *functionAnalysis) newObject(site ssa.Instruction, typ types.Type, reason string) {
	value := site.(ssa.Value)
	l := f.g.location(value.Name()+".storage", f.scopes.valueScope(value))
	if f.scopes.uncertain[site.Block()] {
		reason = "irreducible or over-budget control flow"
	}
	if f.a.config.ForceHeap != nil {
		if forced := f.a.config.ForceHeap(site, typ); forced != "" {
			reason = forced
		}
	}
	if f.bindings != nil && f.a.result.Decision(site).Heap {
		// The generic body is emitted once. A specialized summary must never
		// promise that a pointer stored in its actual heap objects stays local.
		reason = "generic callee heap storage"
	}
	f.objects[site] = &object{location: l, reason: reason, typ: typ}
	l.reason = reason
	f.g.flow(f.value(value), l, -1)
	if reason != "" {
		f.g.flow(&f.g.heap, l, -1)
	}
}

func (f *functionAnalysis) indirectInterface(typ types.Type) bool {
	if f.a.config.IndirectInterface != nil {
		return f.a.config.IndirectInterface(typ)
	}
	return !directInterface(typ)
}

func directInterface(typ types.Type) bool {
	switch typ := typ.Underlying().(type) {
	case *types.Pointer, *types.Chan, *types.Map:
		return true
	case *types.Basic:
		return typ.Kind() == types.UnsafePointer
	case *types.Array:
		return typ.Len() == 1 && directInterface(typ.Elem())
	case *types.Struct:
		return typ.NumFields() == 1 && directInterface(typ.Field(0).Type())
	}
	return false
}

func containsPointers(typ types.Type) bool {
	switch typ := typ.Underlying().(type) {
	case *types.Basic:
		return typ.Kind() == types.UnsafePointer || typ.Kind() == types.String
	case *types.Array:
		return typ.Len() != 0 && containsPointers(typ.Elem())
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if containsPointers(typ.Field(i).Type()) {
				return true
			}
		}
		return false
	case *types.Tuple:
		for i := 0; i < typ.Len(); i++ {
			if containsPointers(typ.At(i).Type()) {
				return true
			}
		}
		return false
	}
	return true
}

func (f *functionAnalysis) instruction(instr ssa.Instruction) {
	switch v := instr.(type) {
	case *ssa.Alloc, *ssa.MakeSlice, *ssa.MakeMap, *ssa.MakeChan:
		// Address edges and placement constraints were added in the first pass.
	case *ssa.Phi:
		for _, edge := range v.Edges {
			f.flow(f.value(v), edge, 0)
		}
	case *ssa.Store:
		f.store(v.Addr, v.Val)
	case *ssa.UnOp:
		if v.Op == token.MUL {
			f.flow(f.value(v), v.X, 1)
		}
		// Channel contents have already escaped at send sites.
	case *ssa.FieldAddr:
		f.flow(f.value(v), v.X, 0)
	case *ssa.IndexAddr:
		f.flow(f.value(v), v.X, 0)
	case *ssa.Field:
		f.flow(f.value(v), v.X, 0)
	case *ssa.Index:
		f.flow(f.value(v), v.X, 0)
	case *ssa.Slice:
		f.flow(f.value(v), v.X, 0)
	case *ssa.SliceToArrayPointer:
		f.flow(f.value(v), v.X, 0)
	case *ssa.ChangeType:
		f.flow(f.value(v), v.X, 0)
	case *ssa.ChangeInterface:
		f.flow(f.value(v), v.X, 0)
	case *ssa.Convert:
		f.convert(v, v.X)
	case *ssa.MultiConvert:
		f.convert(v, v.X)
	case *ssa.MakeInterface:
		if obj := f.objects[v]; obj != nil {
			f.flow(obj.location, v.X, 0)
		} else {
			f.flow(f.value(v), v.X, 0)
		}
	case *ssa.TypeAssert:
		dst := f.value(v)
		if v.CommaOk {
			dst = f.tuple(v, 0)
		}
		depth := 0
		if _, intf := v.AssertedType.Underlying().(*types.Interface); !intf && f.indirectInterface(v.AssertedType) {
			depth = 1
		}
		if containsPointers(v.AssertedType) {
			f.flow(dst, v.X, depth)
		}
	case *ssa.Extract:
		f.g.flow(f.value(v), f.tuple(v.Tuple, v.Index), 0)
	case *ssa.MakeClosure:
		fn, _ := v.Fn.(*ssa.Function)
		for i, binding := range v.Bindings {
			depth := 0
			if fn != nil && f.a.result.CaptureByValue(fn.FreeVars[i]) {
				depth = 1
			}
			f.flow(f.objects[v].location, binding, depth)
		}
	case *ssa.Call:
		f.call(v, &v.Call)
	case *ssa.Defer:
		// Range-over-function bodies can enqueue into the caller's defer
		// stack. That owner may outlive this synthesized invocation.
		f.retainCall(&v.Call, v.Block(), v.DeferStack != nil)
		f.call(v, &v.Call)
	case *ssa.Go:
		f.retainCall(&v.Call, v.Block(), true)
		f.call(v, &v.Call)
	case *ssa.Return:
		for i, result := range v.Results {
			f.flow(f.results[i], result, 0)
		}
	case *ssa.Panic:
		f.flow(&f.g.heap, v.X, 0)
	case *ssa.Send:
		f.flow(&f.g.heap, v.X, 0)
	case *ssa.MapUpdate:
		f.flow(&f.g.heap, v.Key, 0)
		f.flow(&f.g.heap, v.Value, 0)
	case *ssa.Select:
		for _, state := range v.States {
			f.flow(&f.g.heap, state.Send, 0)
		}
	case *ssa.BinOp, *ssa.Lookup, *ssa.Range, *ssa.Next:
		// Comparisons produce scalars. Map/channel elements are heap-retained;
		// no local address can first escape through a read of those containers.
	case *ssa.Jump, *ssa.If, *ssa.RunDefers, *ssa.DebugRef:
	default:
		// A future SSA opcode must never accidentally acquire a noescape proof.
		for _, obj := range f.objects {
			obj.reason = "unsupported SSA instruction"
			f.g.flow(&f.g.heap, obj.location, -1)
		}
		for _, operand := range instr.Operands(nil) {
			f.flow(&f.g.heap, *operand, 0)
		}
	}
}

func (f *functionAnalysis) convert(dst ssa.Value, src ssa.Value) {
	if site, ok := dst.(ssa.Instruction); ok && f.objects[site] != nil {
		// String/slice conversions copy scalar contents into fresh storage;
		// retaining the copy does not retain the source backing array.
		return
	}
	if containsPointers(src.Type()) && !containsPointers(dst.Type()) {
		// In particular unsafe.Pointer -> uintptr must not hide an address
		// from later calls, arithmetic, assembly or native code.
		f.flow(&f.g.heap, src, 0)
		return
	}
	f.flow(f.value(dst), src, 0)
}

func stringType(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.String
}

func byteStringView(dst, src types.Type) bool {
	s, ok := dst.Underlying().(*types.Slice)
	if !ok || !stringType(src) {
		return false
	}
	b, ok := s.Elem().Underlying().(*types.Basic)
	return ok && b.Kind() == types.Byte
}

func copiedStringStorage(dst, src types.Type) bool {
	if stringType(dst) {
		if b, ok := src.Underlying().(*types.Basic); ok {
			return b.Info()&types.IsInteger != 0
		}
		_, ok := src.Underlying().(*types.Slice)
		return ok
	}
	_, slice := dst.Underlying().(*types.Slice)
	return slice && stringType(src)
}

// localStorage resolves stores only through syntactically known local bases.
// Unknown aliases use the heap sink; a visible pointer type is not ownership.
func (f *functionAnalysis) localStorage(addr ssa.Value, seen map[ssa.Value]bool, found map[*location]bool) bool {
	if seen[addr] {
		return true
	}
	seen[addr] = true
	if instr, ok := addr.(ssa.Instruction); ok {
		if obj := f.objects[instr]; obj != nil {
			switch addr.(type) {
			case *ssa.Alloc, *ssa.MakeSlice:
				found[obj.location] = true
				return true
			}
		}
	}
	switch v := addr.(type) {
	case *ssa.FieldAddr:
		return f.localStorage(v.X, seen, found)
	case *ssa.IndexAddr:
		return f.localStorage(v.X, seen, found)
	case *ssa.Slice:
		return f.localStorage(v.X, seen, found)
	case *ssa.SliceToArrayPointer:
		return f.localStorage(v.X, seen, found)
	case *ssa.ChangeType:
		return f.localStorage(v.X, seen, found)
	case *ssa.Phi:
		for _, edge := range v.Edges {
			if !f.localStorage(edge, seen, found) {
				return false
			}
		}
		return true
	}
	return false
}

func (f *functionAnalysis) store(addr, value ssa.Value) {
	f.flow(&f.g.mutator, addr, 0)
	found := make(map[*location]bool)
	if !f.localStorage(addr, make(map[ssa.Value]bool), found) || len(found) == 0 {
		f.flow(&f.g.heap, value, 0)
		return
	}
	for location := range found {
		f.flow(location, value, 0)
	}
}

func (f *functionAnalysis) retainCall(call *ssa.CallCommon, block *ssa.BasicBlock, async bool) {
	dst := f.sink("deferred or asynchronous use")
	if !async && f.scopes.blocks[block] == f.scopes.invocation {
		dst = f.g.location("defer arguments", f.scopes.invocation)
		dst.attrs = persists
	}
	f.flow(dst, call.Value, 0)
	for _, arg := range call.Args {
		f.flow(dst, arg, 0)
	}
}

func (f *functionAnalysis) call(site ssa.CallInstruction, call *ssa.CallCommon) {
	if builtin, ok := call.Value.(*ssa.Builtin); ok {
		f.builtin(site, call, builtin.Name())
		return
	}
	var results []*location
	if value := site.Value(); value != nil {
		if tuple, ok := value.Type().Underlying().(*types.Tuple); ok {
			for i := 0; i < tuple.Len(); i++ {
				results = append(results, f.tuple(value, i))
			}
		} else {
			results = append(results, f.value(value))
		}
	}
	resolved := *call
	if value := f.boundValue(call.Value); value != call.Value {
		resolved.Value = value
		// Keep capture effects attached to the formal callback as well. Its
		// caller composes these with the actual environment's generic summary.
		if !call.IsInvoke() {
			f.flow(&f.g.callee, call.Value, 0)
		}
	}
	targets := callTargets(&resolved)
	if len(targets) == 0 {
		// Calling a function value does not retain its environment object,
		// but an unknown body may publish any of its captured pointers.
		f.flow(&f.g.callee, call.Value, 0)
		// A directly called function parameter exports the invocation effect;
		// its caller can compose the actual callback's capture summary. Other
		// unresolved values still need conservative capture publication here.
		switch call.Value.(type) {
		case *ssa.Parameter, *ssa.FreeVar:
			if call.IsInvoke() {
				f.flow(f.sink("unknown call"), call.Value, 1)
			}
		default:
			f.flow(f.sink("unknown call"), call.Value, 1)
		}
		if call.IsInvoke() {
			f.flow(f.sink("interface call"), call.Value, 0)
		}
		for _, arg := range call.Args {
			f.flow(f.sink("unknown call"), arg, 0)
		}
		return
	}
	for _, target := range targets {
		if call.IsInvoke() && resolved.Value != call.Value {
			if f.indirectInterface(target.args[0].Type()) {
				target.receiverDepth = 1
			}
			target.args[0] = call.Value
		}
		f.callTarget(target, results)
	}
}

func (f *functionAnalysis) callTarget(target callTarget, results []*location) {
	callee, args := target.fn, target.args
	summary := f.contextSummary(callee, args)
	if len(args) != len(summary.Parameters) {
		for _, arg := range args {
			f.flow(&f.g.heap, arg, 0)
		}
		return
	}
	for i, arg := range args {
		p := summary.Parameters[i]
		if i == 0 && target.receiverDepth != 0 {
			add := func(effect Effect) Effect {
				if effect != 0 && effect < 65535 {
					effect++
				}
				return effect
			}
			p.Heap, p.Mutator, p.Callee = add(p.Heap), add(p.Mutator), add(p.Callee)
			p.Results = append([]Effect(nil), p.Results...)
			for j := range p.Results {
				p.Results[j] = add(p.Results[j])
			}
		}
		if p.Heap != 0 {
			f.effect(f.sink("retained by call to "+callee.String()), arg, p.Heap)
		}
		f.effect(&f.g.mutator, arg, p.Mutator)
		f.effect(&f.g.callee, arg, p.Callee)
		if p.Callee != 0 && (p.Callee != 1 || !f.callback(arg)) {
			f.flow(&f.g.heap, arg, int(p.Callee))
		}
		for j, effect := range p.Results {
			if j < len(results) {
				f.effect(results[j], arg, effect)
			}
		}
	}
}

func (f *functionAnalysis) effect(dst *location, arg ssa.Value, effect Effect) {
	if effect != 0 {
		f.flow(dst, arg, int(effect)-1)
	}
}

func (f *functionAnalysis) builtin(site ssa.CallInstruction, call *ssa.CallCommon, name string) {
	switch name {
	case "append":
		if len(call.Args) == 2 {
			if result := site.Value(); result != nil {
				f.flow(f.value(result), call.Args[0], 0)
			}
			// The destination can acquire a new heap backing array. Only the
			// source's elements are retained, never the source slice itself.
			f.flow(&f.g.heap, call.Args[0], 1)
			f.flow(&f.g.heap, call.Args[1], 1)
			f.flow(&f.g.mutator, call.Args[0], 0)
			return
		}
	case "copy":
		if len(call.Args) == 2 {
			f.flow(&f.g.heap, call.Args[1], 1)
			f.flow(&f.g.mutator, call.Args[0], 0)
			return
		}
	case "clear", "delete", "close":
		f.flow(&f.g.mutator, call.Args[0], 0)
		return
	case "min", "max":
		if result := site.Value(); result != nil {
			for _, arg := range call.Args {
				f.flow(f.value(result), arg, 0)
			}
		}
		return
	case "len", "cap", "real", "imag", "complex", "print", "println", "recover":
		return
	case "panic":
		f.flow(&f.g.heap, call.Args[0], 0)
		return
	}
	for _, arg := range call.Args {
		f.flow(&f.g.heap, arg, 0)
	}
}
