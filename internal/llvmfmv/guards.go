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
	method       string
	runtimeBit   uint64
	registerBits int // widest x86 register guaranteed by this predicate
	features     string
}

// The runtime publishes one bit per effective Go predicate, including the
// combined HasAVX && HasAES result for AVXAES. Keep the bits in sync with
// runtime/_patch/internal/cpu/{init,archsimd_go127}_llgo.go.
var x86Predicates = []x86Predicate{
	{"AVX2", featureAVX2, 256, "+avx,+avx2"}, {"AVX", featureAVX, 256, "+avx"},
	{"AVXAES", 8, 256, "+avx,+aes"}, {"VAES", 16, 256, "+avx,+vaes"},
	{"AVX512VAES", 32, 512, "+avx512f,+avx512vl,+vaes"}, {"AVX512GFNI", 64, 512, "+avx512f,+gfni"},
	{"SHA", 128, 0, "+sha"}, {"AVX512", 256, 512, "+avx512f,+avx512cd,+avx512bw,+avx512dq,+avx512vl"},
	{"FMA", 1 << 9, 256, "+avx,+fma"}, {"AVX512BITALG", 1 << 10, 512, "+avx512f,+avx512bitalg"},
	{"AVX512VBMI", 1 << 11, 512, "+avx512f,+avx512vbmi"}, {"AVX512VBMI2", 1 << 12, 512, "+avx512f,+avx512vbmi2"},
	{"AVX512VNNI", 1 << 13, 512, "+avx512f,+avx512vnni"}, {"AVX512VPCLMULQDQ", 1 << 14, 512, "+avx512f,+vpclmulqdq"},
	{"AVX512VPOPCNTDQ", 1 << 15, 512, "+avx512f,+avx512vpopcntdq"}, {"AVXVNNI", 1 << 16, 256, "+avx,+avxvnni"},
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
		m[X86Query(p.method)] = p.runtimeBit
	}
	return m
}()

// internal/cpu excludes these baseline features from GODEBUG overrides.
func guaranteedQuery(name, baseline string) bool {
	if baseline != "v3" && baseline != "v4" {
		return false
	}
	switch name {
	case "x86.avx", AVX2Query, "x86.fma":
		return true
	case "x86.avx512":
		return baseline == "v4"
	}
	return false
}

// Plan using the simplified CFG, but keep the original instructions intact
// until every symbol needed by guarded specialization has been checked.
// Only functions reached during guard discovery need a temporary body.
type baselineFolds struct {
	calls   map[llvm.Value][]llvm.Value
	queries map[llvm.Value]bool
	views   map[llvm.Value]llvm.Value
}

func foldTrue(fn llvm.Value, calls []llvm.Value) {
	for _, call := range calls {
		call.ReplaceAllUsesWith(llvm.ConstInt(call.Type(), 1, false))
		call.EraseFromParentAsInstruction()
	}
	simplifyGuards(fn)
}

func (f *baselineFolds) view(fn llvm.Value) llvm.Value {
	if len(f.calls[fn]) == 0 {
		return fn
	}
	if v := f.views[fn]; !v.IsNil() {
		return v
	}
	v := fn.CloneFunction()
	var calls []llvm.Value
	for bb := v.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if !inst.IsACallInst().IsNil() && f.queries[inst.CalledValue()] {
				calls = append(calls, inst)
			}
		}
	}
	foldTrue(v, calls)
	f.views[fn] = v
	return v
}

func (f *baselineFolds) commit() {
	for fn, calls := range f.calls {
		foldTrue(fn, calls)
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

// An opaque call can return a vector without doing any local vector work.
// LLVM intrinsics are lowered in the caller and can use its target features.
func localVectorWork(inst llvm.Value) bool {
	if inst.Type().TypeKind() != llvm.VectorTypeKind {
		return false
	}
	switch inst.InstructionOpcode() {
	case llvm.Call, llvm.Invoke:
		callee := inst.CalledValue().IsAFunction()
		return !callee.IsNil() && callee.IntrinsicID() != 0
	}
	return true
}

// Collect evidence of vector work along direct calls in this module. Start
// from functions with a feature query and propagate back from SIMD operations
// to their callers. The two walks handle cycles without relying on DFS order.
func guardedWork(candidates map[llvm.Value]bool, folds *baselineFolds, avx2 *avx2Plan) map[llvm.Value]bool {
	work := make(map[llvm.Value]bool)
	seen := make(map[llvm.Value]bool)
	callers := make(map[llvm.Value][]llvm.Value)
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
		if fn.IsDeclaration() || !supported(fn) || (avx2 != nil && !avx2.baselines[fn].IsNil()) {
			continue
		}
		for bb := folds.view(fn).FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if localVectorWork(inst) {
					work[fn] = true
				}
				if callee := directGuardCallee(inst); !callee.IsNil() {
					callers[callee] = append(callers[callee], fn)
					if !seen[callee] {
						visit = append(visit, callee)
					}
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
		for _, caller := range callers[fn] {
			if !work[caller] {
				work[caller] = true
				visit = append(visit, caller)
			}
		}
	}
	return work
}

// guardContexts intersects known true predicates at joins to identify which
// positive guards protect vector work in this function.
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
			if localVectorWork(inst) || work[directGuardCallee(inst)] {
				sets[mask] = true
				break
			}
		}
	}
	return dispatchMasks(sets)
}

// Reserve room for a baseline body and a resolver as well as the roots and
// local helpers. At exhaustion, calls through the existing ABI remain legal.
const maxGuardBodies = 64
const maxGuardChoices = 15

// Independent guard sites may both execute. A matching union must follow its
// subsets in the resolver so it can cache a body with every proven feature.
func dispatchMasks(sets map[uint64]bool) []uint64 {
	var all uint64
	unions := []uint64{0}
	seen := map[uint64]bool{0: true}
	// Four independent predicates already yield fifteen nonempty combinations.
	// Above the limit, keep up to fourteen observed masks and their full union.
	const maxUnions = maxGuardChoices + 1
	for mask := range sets {
		all |= mask
		if len(unions) > maxUnions {
			continue
		}
		for _, prior := range unions {
			union := prior | mask
			if !seen[union] {
				seen[union] = true
				unions = append(unions, union)
			}
		}
	}
	if len(unions) <= maxUnions {
		for _, mask := range unions[1:] {
			sets[mask] = true
		}
	} else {
		sets[all] = true
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
	if len(ordered) > maxGuardChoices {
		ordered = append(ordered[:maxGuardChoices-1], all)
	}
	return ordered
}

type guardRoot struct {
	fn, base llvm.Value
	masks    []uint64
	choices  []dispatchChoice
}

func (r *guardRoot) avx2Only() bool {
	return len(r.masks) == 1 && r.masks[0] == featureAVX2
}

type guardKey struct {
	fn   llvm.Value
	mask uint64
}

func guardSuffix(mask uint64) string {
	var names []string
	for _, p := range x86Predicates {
		if mask&p.runtimeBit != 0 {
			names = append(names, strings.ToLower(p.method))
		}
	}
	return symbolPrefix + "guard_" + strings.Join(names, "_")
}

func runGuardedFeatures(mod llvm.Module, avx2 *avx2Plan, baseline string) error {
	folds := baselineFolds{calls: make(map[llvm.Value][]llvm.Value), queries: make(map[llvm.Value]bool), views: make(map[llvm.Value]llvm.Value)}
	defer func() {
		for _, v := range folds.views {
			v.EraseFromParentAsFunction()
		}
	}()
	queries := make(map[llvm.Value]uint64)
	candidates := make(map[llvm.Value]bool)
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		name := checkedQuery(fn)
		bit := predicateBits[name]
		guaranteed := guaranteedQuery(name, baseline)
		if bit == 0 {
			continue
		}
		if guaranteed {
			folds.queries[fn] = true
		} else {
			queries[fn] = bit
		}
		for use := fn.FirstUse(); !use.IsNil(); use = use.NextUse() {
			call := use.User()
			if !call.IsACallInst().IsNil() && call.CalledValue() == fn {
				parent := call.InstructionParent().Parent()
				if guaranteed {
					folds.calls[parent] = append(folds.calls[parent], call)
					continue
				}
				// An AVX2 root's staged baseline and variant now own its code.
				if avx2 != nil && !avx2.baselines[parent].IsNil() {
					continue
				}
				if supported(parent) {
					candidates[parent] = true
				}
			}
		}
	}
	if len(candidates) == 0 {
		folds.commit()
		avx2.commit(mod, nil)
		return nil
	}
	work := guardedWork(candidates, &folds, avx2)
	var roots []guardRoot
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if !candidates[fn] {
			continue
		}
		masks := guardContexts(folds.view(fn), queries, work)
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
		folds.commit()
		avx2.commit(mod, nil)
		return nil
	}
	ctx := mod.Context()
	// Stage private bodies until every name needed by the simplified call
	// graph is known. A collision can be discovered in a recursive callee.
	var copies []struct{ original, body llvm.Value }
	newBody := func(fn llvm.Value, name string, mask uint64) llvm.Value {
		v := folds.view(fn).CloneFunction()
		v.SetName(name)
		v.SetLinkage(llvm.InternalLinkage)
		v.RemoveStringAttributeAtIndex(-1, EntryAttribute)
		v.RemoveStringAttributeAtIndex(-1, InlineEntryAttribute)
		copies = append(copies, struct{ original, body llvm.Value }{fn, v})
		if mask == 0 {
			return v
		}
		features := stringAttribute(fn, "target-features")
		for _, p := range x86Predicates {
			if mask&p.runtimeBit != 0 {
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
		foldTrue(v, fold)
		return v
	}
	clones := make(map[guardKey]llvm.Value)
	// Materialize all copies before replacing any public entry. Local helper
	// copies are shared by callers with identical proven predicates. Across a
	// translation unit boundary, the ordinary Go carrier and entry remain valid.
	var helperBudget int
	var currentRoot llvm.Value
	var specializeLocal func(llvm.Value, uint64) (llvm.Value, error)
	specializeLocal = func(fn llvm.Value, mask uint64) (llvm.Value, error) {
		key := guardKey{fn, mask}
		if v, ok := clones[key]; ok {
			return v, nil
		}
		// A promised AVX2 entry is already staged and folded. Use it for
		// this exact mask, including when reached through a local helper.
		var v llvm.Value
		if avx2 != nil && mask == featureAVX2 {
			v = avx2.variants[fn]
		}
		canonical := !v.IsNil()
		if !canonical {
			if fn != currentRoot {
				if helperBudget == 0 {
					return fn, nil
				}
				helperBudget--
			}
			name := fn.Name() + guardSuffix(mask)
			if !mod.NamedValue(name).IsNil() {
				return llvm.Value{}, fmt.Errorf("SIMD FMV symbol collision: %s", name)
			}
			v = newBody(fn, name, mask)
		}
		clones[key] = v

		for bb := v.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				target := directGuardCallee(inst)
				if !work[target] {
					continue
				}
				callee, err := specializeLocal(target, mask)
				if err != nil {
					return llvm.Value{}, err
				}
				// Staged AVX2 callers are redirected together once all copies exist.
				if !canonical {
					inst.SetOperand(inst.OperandsCount()-1, callee)
				}
			}
		}
		return v, nil
	}
	for i := range roots {
		root := &roots[i]
		currentRoot = root.fn
		helperBudget = maxGuardBodies - len(root.masks) - 2
		// Allocate helper bodies to the widest choices first. Keep the resolver's
		// select chain in ascending order so matching supersets still win.
		root.choices = make([]dispatchChoice, len(root.masks))
		for j := len(root.masks) - 1; j >= 0; j-- {
			mask := root.masks[j]
			v, err := specializeLocal(root.fn, mask)
			if err != nil {
				for _, staged := range copies {
					discardStagedBody(staged.body)
				}
				return err
			}
			root.choices[j] = dispatchChoice{bits: mask, fn: v}
		}
		root.base = newBody(root.fn, root.fn.Name()+symbolPrefix+"guard_base", 0)
	}
	byFunction := make(map[llvm.Value]*guardRoot, len(roots))
	for i := range roots {
		byFunction[roots[i].fn] = &roots[i]
	}
	// Reuse existing local bodies in the canonical AVX2 call graph, including
	// the externally callable AVX2 entries. Allocate no additional clones.
	if avx2 != nil {
		for key, body := range clones {
			// A root with other choices must select its own best body.
			root := byFunction[key.fn]
			if key.mask == featureAVX2 && (root == nil || root.avx2Only()) {
				avx2.redirectStagedCalls(key.fn, body, avx2.variants)
			}
		}
		for _, root := range roots {
			if root.avx2Only() {
				avx2.redirectStagedCalls(root.fn, root.base, avx2.baselines)
			}
		}
	}
	folds.commit()
	source := newSourceInfo(mod)
	sites := avx2.commit(mod, source)
	for _, staged := range copies {
		original := staged.original
		// The copy predates publication of its AVX2 parent's new PC-site IDs.
		if avx2 != nil && !avx2.origins[original].IsNil() {
			original = avx2.origins[original]
		}
		source.clone(original, staged.body)
	}
	// Keep the canonical AVX2 entry callable from separately compiled code.
	// For public calls, cache the physical target directly in the outer slot.
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	for _, site := range sites {
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
		dispatch(mod, source, root.fn, root.base, root.choices)
	}
	return nil
}
