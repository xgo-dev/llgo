package llvmfmv

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"

	"github.com/xgo-dev/llvm"
)

// Predicate bits describe the Go query after GODEBUG. LLVM capabilities
// describe instructions the CPU can execute. Implied capabilities never
// establish the truth of another independently observable Go predicate.
type x86Predicate struct {
	method   string
	bit      uint64
	features string
}

// Bit positions match runtime/_patch/internal/cpu/{init,archsimd_go127}_llgo.go.
var x86Predicates = []x86Predicate{
	{"AVX2", 2, "+avx,+avx2"}, {"AVX", 4, "+avx"},
	{"AVXAES", 8, "+avx,+aes"}, {"VAES", 16, "+avx,+vaes"},
	{"AVX512VAES", 32, "+avx512f,+avx512vl,+vaes"}, {"AVX512GFNI", 64, "+avx512f,+gfni"},
	{"SHA", 128, "+sha"}, {"AVX512", 256, "+avx512f,+avx512cd,+avx512bw,+avx512dq,+avx512vl"},
	{"FMA", 1 << 9, "+avx,+fma"}, {"AVX512BITALG", 1 << 10, "+avx512bitalg"},
	{"AVX512VBMI", 1 << 11, "+avx512vbmi"}, {"AVX512VBMI2", 1 << 12, "+avx512vbmi2"},
	{"AVX512VNNI", 1 << 13, "+avx512vnni"}, {"AVX512VPCLMULQDQ", 1 << 14, "+avx512f,+vpclmulqdq"},
	{"AVX512VPOPCNTDQ", 1 << 15, "+avx512vpopcntdq"}, {"AVXVNNI", 1 << 16, "+avxvnni"},
}

func X86Query(method string) string {
	for _, p := range x86Predicates {
		if method == p.method {
			return "x86." + strings.ToLower(method)
		}
	}
	return ""
}

var predicateBits = func() map[string]uint64 {
	m := make(map[string]uint64)
	for _, p := range x86Predicates {
		m[X86Query(p.method)] = p.bit
	}
	return m
}()

// internal/cpu excludes these baseline features from GODEBUG overrides.
func foldBaselineQueries(mod llvm.Module, baseline string) {
	var calls []llvm.Value
	parents := make(map[llvm.Value]bool)
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		switch checkedQuery(fn) {
		case "x86.avx", AVX2Query, "x86.fma":
		case "x86.avx512":
			if baseline != "v4" {
				continue
			}
		default:
			continue
		}
		for use := fn.FirstUse(); !use.IsNil(); use = use.NextUse() {
			call := use.User()
			if !call.IsACallInst().IsNil() && call.CalledValue() == fn {
				calls = append(calls, call)
				parents[call.InstructionParent().Parent()] = true
			}
		}
	}
	for _, call := range calls {
		call.ReplaceAllUsesWith(llvm.ConstInt(mod.Context().Int1Type(), 1, false))
		call.EraseFromParentAsInstruction()
	}
	for fn := range parents {
		simplifyGuards(fn)
	}
}

// positiveGuard returns predicates established on the given branch edge.
// LLVM commonly represents Go's negated query as xor i1 %query, true.
func positiveGuard(cond llvm.Value, taken bool, queries map[llvm.Value]uint64) uint64 {
	if !cond.IsACallInst().IsNil() {
		if taken {
			return queries[cond.CalledValue()]
		}
		return 0
	}
	if cond.IsAInstruction().IsNil() {
		return 0
	}
	switch cond.InstructionOpcode() {
	case llvm.Xor:
		for i := 0; i < 2; i++ {
			c := cond.Operand(i).IsAConstantInt()
			if !c.IsNil() && c.ZExtValue() == 1 {
				return positiveGuard(cond.Operand(1-i), !taken, queries)
			}
		}
	case llvm.And:
		if taken {
			return positiveGuard(cond.Operand(0), true, queries) | positiveGuard(cond.Operand(1), true, queries)
		}
	}
	return 0
}

// directGuardCallee excludes indirect calls even when a function address also
// appears as an ordinary argument to the same instruction.
func directGuardCallee(inst llvm.Value) llvm.Value {
	if inst.IsACallInst().IsNil() && inst.IsAInvokeInst().IsNil() {
		return llvm.Value{}
	}
	return inst.CalledValue().IsAFunction()
}

// Collect evidence of vector work along direct calls in this module. Start
// from functions with a feature query and propagate back from SIMD operations
// to their callers. The two walks handle cycles without relying on DFS order.
func guardedWork(candidates map[llvm.Value]bool) map[llvm.Value]bool {
	work := make(map[llvm.Value]bool)
	seen := make(map[llvm.Value]bool)
	var visit []llvm.Value
	for fn := range candidates {
		visit = append(visit, fn)
	}
	for len(visit) != 0 {
		fn := visit[len(visit)-1]
		visit = visit[:len(visit)-1]
		if seen[fn] {
			continue
		}
		seen[fn] = true
		if hasAttribute(fn, EntryAttribute) || strings.HasSuffix(fn.Name(), variantSuffix) {
			work[fn] = true
		}
		if fn.IsDeclaration() || !supported(fn) {
			continue
		}
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if inst.Type().TypeKind() == llvm.VectorTypeKind {
					work[fn] = true
				}
				if callee := directGuardCallee(inst); !callee.IsNil() && !seen[callee] {
					visit = append(visit, callee)
				}
			}
		}
	}
	for fn := range work {
		visit = append(visit, fn)
	}
	for len(visit) != 0 {
		fn := visit[len(visit)-1]
		visit = visit[:len(visit)-1]
		for use := fn.FirstUse(); !use.IsNil(); use = use.NextUse() {
			inst := use.User()
			if directGuardCallee(inst) != fn {
				continue
			}
			caller := inst.InstructionParent().Parent()
			if seen[caller] && !work[caller] && supported(caller) {
				work[caller] = true
				visit = append(visit, caller)
			}
		}
	}
	return work
}

// guardContexts intersects known true predicates at joins. This builds
// versions for the positive guard combinations present in the CFG, without
// enumerating the power set of the runtime feature bitmap.
func guardContexts(fn llvm.Value, queries map[llvm.Value]uint64, work map[llvm.Value]bool) []uint64 {
	entry := fn.EntryBasicBlock()
	known := map[llvm.BasicBlock]uint64{entry: 0}
	queue := []llvm.BasicBlock{entry}
	for len(queue) != 0 {
		block := queue[0]
		queue = queue[1:]
		term := block.LastInstruction()
		for i := 0; i < term.SuccessorsCount(); i++ {
			mask := known[block]
			if term.InstructionOpcode() == llvm.Br && term.SuccessorsCount() == 2 {
				mask |= positiveGuard(term.Operand(0), i == 0, queries)
			}
			next := term.Successor(i)
			if prior, ok := known[next]; ok {
				mask &= prior
				if mask == prior {
					continue
				}
			}
			known[next] = mask
			queue = append(queue, next)
		}
	}
	sets := map[uint64]bool{}
	for block, mask := range known {
		if mask == 0 {
			continue
		}
		for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if inst.Type().TypeKind() == llvm.VectorTypeKind || work[directGuardCallee(inst)] {
				sets[mask] = true
				break
			}
		}
	}
	ordered := make([]uint64, 0, len(sets))
	for mask := range sets {
		ordered = append(ordered, mask)
	}
	sort.Slice(ordered, func(i, j int) bool {
		a, b := bits.OnesCount64(ordered[i]), bits.OnesCount64(ordered[j])
		if a != b {
			return a < b
		}
		return ordered[i] < ordered[j]
	})
	return ordered
}

type guardRoot struct {
	fn, base llvm.Value
	masks    []uint64
	choices  []dispatchChoice
}
type guardKey struct {
	fn   llvm.Value
	mask uint64
}

func guardSuffix(mask uint64) string {
	var names []string
	for _, p := range x86Predicates {
		if mask&p.bit != 0 {
			names = append(names, strings.ToLower(p.method))
		}
	}
	return symbolPrefix + "guard_" + strings.Join(names, "_")
}

func runGuardedFeatures(mod llvm.Module, avx2 []dispatchSite) error {
	arch, _, _ := strings.Cut(mod.Target(), "-")
	if arch != "x86_64" && arch != "amd64" && arch != "x86_64h" {
		return nil
	}
	const processed = attributePrefix + "guarded"
	queries := make(map[llvm.Value]uint64)
	candidates := make(map[llvm.Value]bool)
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		bit := predicateBits[checkedQuery(fn)]
		if bit == 0 || bit == featureAVX2 {
			continue
		}
		queries[fn] = bit
		for use := fn.FirstUse(); !use.IsNil(); use = use.NextUse() {
			call := use.User()
			if !call.IsACallInst().IsNil() && call.CalledValue() == fn {
				parent := call.InstructionParent().Parent()
				if !hasAttribute(parent, processed) && supported(parent) {
					candidates[parent] = true
				}
			}
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	work := guardedWork(candidates)
	var roots []guardRoot
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if !candidates[fn] {
			continue
		}
		masks := guardContexts(fn, queries, work)
		if len(masks) == 0 {
			continue
		}
		for _, suffix := range []string{symbolPrefix + "guard_base", slotSuffix, resolverSuffix} {
			if !mod.NamedValue(fn.Name() + suffix).IsNil() {
				return fmt.Errorf("SIMD FMV symbol collision: %s%s", fn.Name(), suffix)
			}
		}
		roots = append(roots, guardRoot{fn: fn, masks: masks})
	}
	if len(roots) == 0 {
		return nil
	}
	ctx := mod.Context()
	if global := mod.NamedValue(featureSymbol); !global.IsNil() && (global.IsAGlobalVariable().IsNil() || global.GlobalValueType() != ctx.Int64Type()) {
		return fmt.Errorf("SIMD FMV symbol collision: %s", featureSymbol)
	}
	// Stage private bodies until every name needed by the simplified call
	// graph is known. A collision can be discovered in a recursive callee.
	var copies []struct{ original, body llvm.Value }
	newBody := func(fn llvm.Value, name string) llvm.Value {
		v := fn.CloneFunction()
		v.SetName(name)
		v.SetLinkage(llvm.InternalLinkage)
		v.AddFunctionAttr(ctx.CreateStringAttribute(processed, ""))
		v.RemoveStringAttributeAtIndex(-1, EntryAttribute)
		v.RemoveStringAttributeAtIndex(-1, InlineEntryAttribute)
		copies = append(copies, struct{ original, body llvm.Value }{fn, v})
		return v
	}
	clones := make(map[guardKey]llvm.Value)
	// Materialize all copies before replacing any public entry. Local helper
	// copies are shared by callers with identical proven predicates. Across a
	// translation unit boundary, the ordinary Go carrier and entry remain valid.
	var specializeLocal func(llvm.Value, uint64) (llvm.Value, error)
	specializeLocal = func(fn llvm.Value, mask uint64) (llvm.Value, error) {
		key := guardKey{fn, mask}
		if v, ok := clones[key]; ok {
			return v, nil
		}
		name := fn.Name() + guardSuffix(mask)
		if !mod.NamedValue(name).IsNil() {
			return llvm.Value{}, fmt.Errorf("SIMD FMV symbol collision: %s", name)
		}
		v := newBody(fn, name)
		clones[key] = v
		features := stringAttribute(fn, "target-features")
		for _, p := range x86Predicates {
			if mask&p.bit != 0 {
				if features != "" {
					features += ","
				}
				features += p.features
			}
		}
		v.AddFunctionAttr(ctx.CreateStringAttribute("target-features", features))
		var fold []llvm.Value
		for bb := v.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if !inst.IsACallInst().IsNil() && queries[inst.CalledValue()]&mask != 0 {
					fold = append(fold, inst)
				}
			}
		}
		for _, q := range fold {
			q.ReplaceAllUsesWith(llvm.ConstInt(ctx.Int1Type(), 1, false))
			q.EraseFromParentAsInstruction()
		}
		simplifyGuards(v)
		for bb := v.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				target := directGuardCallee(inst)
				if !work[target] || target.IsDeclaration() || !supported(target) {
					continue
				}
				callee, err := specializeLocal(target, mask)
				if err != nil {
					return llvm.Value{}, err
				}
				inst.SetOperand(inst.OperandsCount()-1, callee)
			}
		}
		return v, nil
	}
	for i := range roots {
		root := &roots[i]
		for _, mask := range root.masks {
			v, err := specializeLocal(root.fn, mask)
			if err != nil {
				for _, copy := range copies {
					copy.body.ReplaceAllUsesWith(llvm.Undef(copy.body.Type()))
					copy.body.EraseFromParentAsFunction()
				}
				return err
			}
			root.choices = append(root.choices, dispatchChoice{mask, v})
		}
		root.base = newBody(root.fn, root.fn.Name()+symbolPrefix+"guard_base")
	}
	source := newSourceInfo(mod)
	for _, copy := range copies {
		source.clone(copy.original, copy.body)
	}
	// Keep the canonical AVX2 entry callable from separately compiled code.
	// For public calls, cache the physical target directly in the outer slot.
	byFunction := make(map[llvm.Value]*guardRoot, len(roots))
	for i := range roots {
		byFunction[roots[i].fn] = &roots[i]
	}
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	for _, site := range avx2 {
		builder.SetInsertPointBefore(site.selection)
		// LLVM select operands are condition, true (AVX2), false (baseline).
		for operand := 1; operand <= 2; operand++ {
			root := byFunction[site.selection.Operand(operand)]
			if root == nil {
				continue
			}
			target := selectDispatchBody(builder, site.features, root.base, root.choices)
			site.selection.SetOperand(operand, target)
			if operand == 2 {
				call := site.uninitializedCall
				call.SetOperand(call.OperandsCount()-1, root.base)
			}
		}
	}
	for _, root := range roots {
		root.fn.AddFunctionAttr(ctx.CreateStringAttribute(processed, ""))
		dispatch(mod, source, root.fn, root.base, root.choices)
	}
	return nil
}
