package dcepass

import "github.com/xgo-dev/llvm"

// MarkNoInlineFunctions prevents function-scoped Go semantic facts from moving
// into callers during an experimental feedback link. The final rewritten link
// does not need this restriction; it exists only while feedback is represented
// at function granularity rather than by instruction-level DemandIDs.
func MarkNoInlineFunctions(mod llvm.Module, names []string) int {
	if mod.IsNil() {
		return 0
	}
	kind := llvm.AttributeKindID("noinline")
	attr := mod.Context().CreateEnumAttribute(kind, 0)
	marked := 0
	for _, name := range names {
		fn := mod.NamedFunction(name)
		if fn.IsNil() || fn.IsDeclaration() || !fn.GetEnumFunctionAttribute(kind).IsNil() {
			continue
		}
		fn.AddFunctionAttr(attr)
		marked++
	}
	return marked
}

// UnmarkNoInlineFunctions removes the temporary feedback barrier before the
// final ThinLTO link so normal cross-package inlining is available again.
func UnmarkNoInlineFunctions(mod llvm.Module, names []string) int {
	if mod.IsNil() {
		return 0
	}
	kind := llvm.AttributeKindID("noinline")
	unmarked := 0
	for _, name := range names {
		fn := mod.NamedFunction(name)
		if fn.IsNil() || fn.GetEnumFunctionAttribute(kind).IsNil() {
			continue
		}
		fn.RemoveEnumFunctionAttribute(kind)
		unmarked++
	}
	return unmarked
}

// DeadNoInlineFunctionsFromModules returns noinline candidate functions that
// are not reachable from roots in the post-optimization LLVM global-reference
// graph.
//
// This is an intentionally small ThinLTO feedback prototype. It scans function
// instructions and global initializers across all backend modules. Following
// references instead of checking which definitions remain avoids treating a
// function kept alive by the initial ThinLTO index as semantically live after
// optimization deleted its last caller. Requiring noinline makes function-level
// feedback sound: semantic facts cannot have moved into a live caller. A future
// instruction-level DemandID design can remove this restriction.
func DeadNoInlineFunctionsFromModules(mods []llvm.Module, roots, candidates []string) map[string]struct{} {
	return DeadNoInlineFunctionsFromModulesWithDefinitions(mods, roots, candidates, nil)
}

// DeadNoInlineFunctionsFromModulesWithDefinitions is the feedback scanner used
// by the build pipeline. knownDefinitions contains candidate functions that
// were present and marked noinline before the ThinLTO link. A noinline
// function can be absent from every optimized module when ThinLTO deletes its
// whole body; those known definitions are therefore dead as well. Candidates
// outside knownDefinitions retain the conservative behavior of the legacy
// scanner and must still appear in an optimized module with a noinline
// attribute before they can be reported.
func DeadNoInlineFunctionsFromModulesWithDefinitions(mods []llvm.Module, roots, candidates []string, knownDefinitions map[string]struct{}) map[string]struct{} {
	edges := make(map[string]map[string]struct{})
	noInlineDefinitions := make(map[string]struct{})
	noInlineKind := llvm.AttributeKindID("noinline")
	for _, mod := range mods {
		if mod.IsNil() {
			continue
		}
		for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
			if fn.IsDeclaration() || fn.Name() == "" {
				continue
			}
			if !fn.GetEnumFunctionAttribute(noInlineKind).IsNil() {
				noInlineDefinitions[fn.Name()] = struct{}{}
			}
			for _, block := range fn.BasicBlocks() {
				for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
					addGlobalReferences(edges, fn.Name(), inst)
				}
			}
		}
		for global := mod.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
			if global.IsDeclaration() || global.Name() == "" || global.Initializer().IsNil() {
				continue
			}
			addGlobalReferences(edges, global.Name(), global.Initializer())
		}
	}

	reachable := make(map[string]struct{})
	queue := append([]string(nil), roots...)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if _, seen := reachable[name]; seen {
			continue
		}
		reachable[name] = struct{}{}
		for next := range edges[name] {
			queue = append(queue, next)
		}
	}

	dead := make(map[string]struct{})
	for _, name := range candidates {
		_, known := knownDefinitions[name]
		if _, eligible := noInlineDefinitions[name]; !eligible && !known {
			continue
		}
		if _, live := reachable[name]; !live {
			dead[name] = struct{}{}
		}
	}
	return dead
}

func addGlobalReferences(edges map[string]map[string]struct{}, from string, value llvm.Value) {
	seen := make(map[llvm.Value]struct{})
	var visit func(llvm.Value)
	visit = func(current llvm.Value) {
		if current.IsNil() {
			return
		}
		if _, ok := seen[current]; ok {
			return
		}
		seen[current] = struct{}{}
		if global := current.IsAGlobalValue(); !global.IsNil() {
			name := global.Name()
			if name != "" && name != from {
				out := edges[from]
				if out == nil {
					out = make(map[string]struct{})
					edges[from] = out
				}
				out[name] = struct{}{}
			}
			return
		}
		for i := 0; i < current.OperandsCount(); i++ {
			visit(current.Operand(i))
		}
	}
	visit(value)
}
