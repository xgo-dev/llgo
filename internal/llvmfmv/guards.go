package llvmfmv

import (
	"fmt"
	"slices"
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

type guardFacts struct {
	common uint64 // every bit is known true
	zmm    uint64 // at least one of these ZMM guards is known true, or zero
}

func (a guardFacts) and(b guardFacts) guardFacts {
	return guardFacts{common: a.common | b.common, zmm: a.zmm | b.zmm}
}

// At a join only common predicates can be folded. Alternate paths can still
// establish the same register width with different effective Go predicates.
func (a guardFacts) or(b guardFacts) guardFacts {
	f := guardFacts{common: a.common & b.common}
	if a.zmm != 0 && b.zmm != 0 {
		f.zmm = a.zmm | b.zmm
	}
	return f
}

// positiveGuard returns predicates established on the given branch edge.
// LLVM commonly represents Go's negated query as xor i1 %query, true.
func positiveGuard(cond llvm.Value, taken bool, queries map[llvm.Value]uint64) guardFacts {
	if !cond.IsACallInst().IsNil() && taken {
		mask := queries[cond.CalledValue()]
		return guardFacts{common: mask, zmm: mask & zmmPredicates}
	}
	if cond.IsAInstruction().IsNil() {
		return guardFacts{}
	}
	switch op := cond.InstructionOpcode(); op {
	case llvm.Xor:
		for i := 0; i < 2; i++ {
			c := cond.Operand(i).IsAConstantInt()
			if !c.IsNil() && c.ZExtValue() == 1 {
				return positiveGuard(cond.Operand(1-i), !taken, queries)
			}
		}
	case llvm.And, llvm.Or:
		a := positiveGuard(cond.Operand(0), taken, queries)
		b := positiveGuard(cond.Operand(1), taken, queries)
		// Both operands are true for a taken AND, or false for an untaken OR.
		if (op == llvm.And) == taken {
			return a.and(b)
		}
		return a.or(b)
	}
	return guardFacts{}
}

// directGuardCallee excludes indirect calls even when a function address also
// appears as an ordinary argument to the same instruction.
func directGuardCallee(inst llvm.Value) llvm.Value {
	if inst.IsACallInst().IsNil() && inst.IsAInvokeInst().IsNil() {
		return llvm.Value{}
	}
	return inst.CalledValue().IsAFunction()
}

// A wide register argument or result is SIMD work at the call site, even
// when the operand is a constant or the callee is indirect or external.
func (a *wideABI) localVectorWork(inst llvm.Value) bool {
	switch inst.InstructionOpcode() {
	case llvm.Call, llvm.Invoke:
		if a.callWidth(inst) != 0 {
			return true
		}
		callee := inst.CalledValue().IsAFunction()
		return inst.Type().TypeKind() == llvm.VectorTypeKind && !callee.IsNil() && callee.IntrinsicID() != 0
	}
	return inst.Type().TypeKind() == llvm.VectorTypeKind
}

// Generic LLVM intrinsics that can use an optional instruction even when the
// vector fits the baseline. Instruction-specific x86 leaves already carry their
// own target features; their caller emits SIMDRequireX86Features with its mask.
func (a *wideABI) intrinsicGuards(fn llvm.Value) uint64 {
	name := fn.Name()
	switch {
	case strings.HasPrefix(name, "llvm.fma."), strings.HasPrefix(name, "llvm.fmuladd."):
		return predicateBits["x86.fma"]
	case strings.HasPrefix(name, "llvm.clmul."):
		return predicateBits["x86.avx512vpclmulqdq"]
	case strings.HasPrefix(name, "llvm.experimental.vector.compress."), strings.HasPrefix(name, "llvm.masked.expandload."):
		if a.data.TypeSizeInBits(fn.GlobalValueType().ReturnType().ElementType()) < 32 {
			return predicateBits["x86.avx512vbmi2"]
		}
		return predicateBits["x86.avx512"]
	case strings.HasPrefix(name, "llvm.ctpop."):
		t := fn.GlobalValueType().ReturnType()
		if t.TypeKind() == llvm.VectorTypeKind {
			if t.ElementType().IntTypeWidth() < 32 {
				return predicateBits["x86.avx512bitalg"]
			}
			return predicateBits["x86.avx512vpopcntdq"]
		}
	}
	return 0
}

// instructionGuards identifies work that benefits from the guard. Ordinary
// narrow vector arithmetic can use the baseline implementation on amd64.
func (a *wideABI) instructionGuards(inst, callee llvm.Value, floor int) uint64 {
	var needs uint64
	typ := inst.Type()
	switch inst.InstructionOpcode() {
	case llvm.Call, llvm.Invoke:
		if callee.IsNil() || callee.IntrinsicID() == 0 {
			if !callee.IsNil() && callee.Name() == "github.com/xgo-dev/llgo/runtime/internal/runtime.SIMDRequireX86Features" && inst.ArgOperandsCount() == 1 {
				if arg := inst.Operand(0).IsAConstantInt(); !arg.IsNil() && arg.Type().IntTypeWidth() == 64 {
					return arg.ZExtValue()
				}
			}
			width := a.callWidth(inst)
			if width <= floor {
				return 0
			}
			if width == 512 {
				return zmmPredicates
			}
			return ymmPredicates
		}
		needs = a.intrinsicGuards(callee)
		if typ.TypeKind() != llvm.VectorTypeKind && inst.ArgOperandsCount() != 0 {
			typ = inst.Operand(0).Type()
		}
	case llvm.Store, llvm.ExtractElement, llvm.ICmp, llvm.FCmp:
		typ = inst.Operand(0).Type()
	}
	if typ.TypeKind() == llvm.VectorTypeKind {
		width := a.data.TypeSizeInBits(typ)
		if width > 128 && floor < 256 {
			needs |= featureAVX | featureAVX2
		}
		if width > 256 && floor < 512 {
			needs |= zmmPredicates
		}
	}
	return needs
}

// Summarize reachable local helpers before choosing root masks. Propagate newly
// discovered requirements to a fixed point, including through recursive calls.
func guardedWork(candidates map[llvm.Value]bool, folds *baselineFolds, avx2 *avx2Plan, abi *wideABI, baseline string) map[llvm.Value]uint64 {
	work := make(map[llvm.Value]uint64)
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
		floor := targetWidth(fn, baseline)
		var needs uint64
		for bb := folds.view(fn).FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				callee := directGuardCallee(inst)
				needs |= abi.instructionGuards(inst, callee, floor)
				if !callee.IsNil() {
					callers[callee] = append(callers[callee], fn)
					if !seen[callee] {
						visit = append(visit, callee)
					}
				}
			}
		}
		if needs != 0 {
			work[fn] = needs
		}
	}
	for fn := range work {
		visit = append(visit, fn)
	}
	for len(visit) != 0 {
		fn := visit[len(visit)-1]
		visit = visit[:len(visit)-1]
		for _, caller := range callers[fn] {
			if work[fn]&^work[caller] != 0 {
				work[caller] |= work[fn]
				visit = append(visit, caller)
			}
		}
	}
	return work
}

// Keep common Go predicates at joins. A() || B() can also specialize guarded
// vector work when either predicate independently supplies ZMM registers.
func guardContexts(fn llvm.Value, queries map[llvm.Value]uint64, requirements func(llvm.Value) uint64) []uint64 {
	entry := fn.EntryBasicBlock()
	known := map[llvm.BasicBlock]guardFacts{entry: {}}
	queue := []llvm.BasicBlock{entry}
	for len(queue) != 0 {
		block := queue[0]
		queue = queue[1:]
		term := block.LastInstruction()
		for i := 0; i < term.SuccessorsCount(); i++ {
			f := known[block]
			if term.InstructionOpcode() == llvm.Br && term.SuccessorsCount() == 2 {
				f = f.and(positiveGuard(term.Operand(0), i == 0, queries))
			}
			next := term.Successor(i)
			if prior, ok := known[next]; ok {
				f = f.or(prior)
				if f == prior {
					continue
				}
			}
			known[next] = f
			queue = append(queue, next)
		}
	}
	sets := map[uint64]bool{}
	for block, f := range known {
		if f.common == 0 && f.zmm == 0 {
			continue
		}
		for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			needs := requirements(inst)
			common := f.common & needs
			if common != 0 {
				sets[common] = true
			}
			if alternatives := f.zmm & needs; alternatives != 0 && common&zmmPredicates == 0 {
				for _, p := range x86Predicates {
					if alternatives&p.runtimeBit != 0 {
						sets[common|p.runtimeBit] = true
					}
				}
			}
		}
	}
	return dispatchMasks(sets)
}

// Independent guarded regions can both execute. Build the distinct unions of
// their masks so the resolver can select every enabled region together.
func dispatchMasks(sets map[uint64]bool) []uint64 {
	unions := []uint64{0}
	seen := map[uint64]bool{0: true}
	for mask := range sets {
		for _, prior := range unions {
			union := prior | mask
			if !seen[union] {
				seen[union] = true
				unions = append(unions, union)
			}
		}
	}
	// Every strict superset has a larger unsigned value and wins the select.
	choices := unions[1:]
	slices.Sort(choices)
	return choices
}

type guardRoot struct {
	fn, base llvm.Value
	masks    []uint64
	choices  []dispatchChoice
}

// A direct call to a clone is valid when the caller covers every guard choice.
// dispatchMasks places the union of all choices last.
func (r *guardRoot) needsDispatch(known uint64) bool {
	return r != nil && r.masks[len(r.masks)-1]&^known != 0
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

func runGuardedFeatures(abi *wideABI, avx2 *avx2Plan, baseline string) error {
	mod := abi.mod
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
	work := guardedWork(candidates, &folds, avx2, abi, baseline)
	var roots []guardRoot
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if !candidates[fn] {
			continue
		}
		floor := targetWidth(fn, baseline)
		masks := guardContexts(folds.view(fn), queries, func(inst llvm.Value) uint64 {
			callee := directGuardCallee(inst)
			return abi.instructionGuards(inst, callee, floor) | work[callee]
		})
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
	byFunction := make(map[llvm.Value]*guardRoot, len(roots))
	for i := range roots {
		byFunction[roots[i].fn] = &roots[i]
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
	// copies are shared by callers with identical proven predicates.
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
				calleeMask := mask & work[target]
				if calleeMask == 0 || byFunction[target].needsDispatch(mask) {
					continue
				}
				callee, err := specializeLocal(target, calleeMask)
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
		root.choices = make([]dispatchChoice, len(root.masks))
		for j, mask := range root.masks {
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
	// Reuse existing local bodies in the canonical AVX2 call graph, including
	// the externally callable AVX2 entries. Allocate no additional clones.
	if avx2 != nil {
		for key, body := range clones {
			// A root with other choices must select its own best body.
			root := byFunction[key.fn]
			if key.mask == featureAVX2 && !root.needsDispatch(featureAVX2) {
				avx2.redirectStagedCalls(key.fn, body, avx2.variants)
			}
		}
		for _, root := range roots {
			if !root.needsDispatch(featureAVX2) {
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
