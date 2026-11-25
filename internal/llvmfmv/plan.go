package llvmfmv

import (
	"fmt"

	"github.com/xgo-dev/llvm"
)

// A profile is a machine-code contract. Only isa bits also establish the
// corresponding exact Go queries. A register width can have many witnesses.
type profile struct {
	width int
	isa   uint64
}

func (p profile) join(q profile) profile { return profile{max(p.width, q.width), p.isa | q.isa} }

type plannedCall struct {
	target llvm.Value
	facts  guardFacts
}

// All decisions for a function refer to the same input CFG. A scalar call
// carrying a proven feature requests a callee entry, not a caller clone.
type functionPlan struct {
	fn       llvm.Value
	floor    int
	knownISA uint64
	work     profile
	calls    []plannedCall // one per call/invoke, in IR order
	masks    map[uint64]bool
	widths   map[int]uint64 // width -> disjunction of runtime witnesses
	ordinary bool
}

func (f *functionPlan) project(p profile) profile {
	p.isa &= f.work.isa &^ f.knownISA
	p.width = min(p.width, f.work.width)
	if p.width <= max(f.floor, isaWidth(p.isa)) {
		p.width = 0
	}
	return p
}

func (f *functionPlan) observe(p profile, facts guardFacts, abiCall bool) {
	f.work = f.work.join(p)
	if common := p.isa & facts.common; common != 0 {
		if f.masks == nil {
			f.masks = make(map[uint64]bool)
		}
		f.masks[common] = true
		f.ordinary = false
	}
	if p.width <= f.floor {
		return
	}
	width, witness := p.width, facts.ymm
	if width == 512 {
		witness = facts.zmm
	}
	if witness != 0 {
		f.ordinary = false
	} else if abiCall {
		choice := ordinaryWidthChoice(width)
		witness = choice.anyOf | choice.bits
	} else if width == 512 && facts.ymm != 0 && f.floor < 256 {
		width, witness = 256, facts.ymm
		f.ordinary = false
	}
	if witness != 0 {
		if f.widths == nil {
			f.widths = make(map[int]uint64)
		}
		f.widths[width] |= witness
	}
}

type versionKey struct {
	fn llvm.Value
	profile
}
type bodyPlan struct {
	key     versionKey
	targets []versionKey
	value   llvm.Value
}
type entryChoice struct {
	key         versionKey
	bits, anyOf uint64
}
type entryPlan struct {
	key     versionKey
	choices []entryChoice
	value   llvm.Value
}
type versionPlan struct {
	abi        *wideABI
	canonical  *avx2Plan
	funcs      map[llvm.Value]*functionPlan
	entries    map[versionKey]*entryPlan
	entryOrder []*entryPlan
	bodies     map[versionKey]*bodyPlan
	bodyOrder  []*bodyPlan
}

func (p *versionPlan) entry(k versionKey) {
	if p.entries[k] != nil {
		return
	}
	e := &entryPlan{key: k}
	p.entries[k] = e
	p.entryOrder = append(p.entryOrder, e)
}

func (p *versionPlan) body(k versionKey) {
	if p.bodies[k] != nil {
		return
	}
	b := &bodyPlan{key: k}
	p.bodies[k] = b
	p.bodyOrder = append(p.bodyOrder, b)
	f := p.funcs[k.fn]
	for _, call := range f.calls {
		target := call.target
		// The call site can establish a stronger target contract than the caller's entry. The callee only inherits capabilities its work can use.
		incoming := k.profile
		incoming.isa |= f.knownISA
		incoming.isa |= call.facts.common
		incoming.width = max(k.width, f.floor, isaWidth(incoming.isa))
		if call.facts.zmm != 0 {
			incoming.width = 512
		} else if call.facts.ymm != 0 {
			incoming.width = max(256, incoming.width)
		}
		if p.canonical != nil && incoming.isa&featureAVX2 != 0 {
			if v := p.canonical.variants[target]; !v.IsNil() {
				target = v
			}
		}
		dest := versionKey{fn: target}
		if callee := p.funcs[target]; callee != nil && callee.work != (profile{}) {
			dest.profile = callee.project(incoming)
			if dest.profile != (profile{}) || !p.isCanonicalRoot(target) {
				p.entry(dest)
			}
		}
		b.targets = append(b.targets, dest)
	}
}

// Enumerate combinations of actual ISA requirements. Each width adds a single
// alternative regardless of how many source predicates can establish it.
func (p *versionPlan) expand(e *entryPlan) {
	k := e.key
	f := p.funcs[k.fn]
	p.body(k)
	for _, bits := range append([]uint64{0}, dispatchMasks(f.masks)...) {
		if bits != 0 && bits&^k.isa == 0 {
			continue
		}
		for _, width := range []int{0, 256, 512} {
			witness := f.widths[width]
			if width != 0 && (witness == 0 || width <= max(k.width, isaWidth(k.isa|bits))) {
				continue
			}
			if width == 0 && bits == 0 {
				continue
			}
			next := versionKey{k.fn, f.project(k.profile.join(profile{width, bits}))}
			if next == k {
				continue
			}
			p.body(next)
			e.choices = append(e.choices, entryChoice{next, bits &^ k.isa, witness})
		}
	}
}

// The public entry for these functions is owned by the canonical AVX2 ABI.
// A local caller that proves a different capability may still request a
// private entry from the original with its AVX2 condition intact.
func (p *versionPlan) isCanonicalRoot(fn llvm.Value) bool {
	return p.canonical != nil && p.canonical.roots[fn]
}

// collect reads requirements at operations and calls. Propagation through
// direct helpers reaches a fixed point even for recursive call graphs.
func (p *versionPlan) collect(functions []llvm.Value, folds *baselineFolds, queries map[llvm.Value]uint64, candidates map[llvm.Value]bool, baseline string) {
	reachable := make(map[llvm.Value]bool)
	var visit []llvm.Value
	if p.canonical != nil {
		for _, v := range p.canonical.variants {
			visit = append(visit, v)
		}
	}
	width := baselineWidth(baseline)
	floors := make(map[llvm.Value]int)
	for _, fn := range functions {
		// A public wide signature can be called from a different module. Its
		// register floor is sufficient evidence for direct calls in its own body.
		if fn.IsDeclaration() {
			continue
		}
		floors[fn] = p.abi.registerFloor(fn, baseline)
		if floors[fn] > width {
			visit = append(visit, fn)
		}
	}
	for fn := range candidates {
		visit = append(visit, fn)
	}
	for len(visit) > 0 {
		fn := visit[len(visit)-1]
		visit = visit[:len(visit)-1]
		if reachable[fn] || fn.IsDeclaration() || !supported(fn) {
			continue
		}
		reachable[fn] = true
		for bb := folds.view(fn).FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
				if target := directGuardCallee(i); !target.IsNil() && !target.IsDeclaration() && !reachable[target] {
					visit = append(visit, target)
				}
			}
		}
	}
	parents := make(map[llvm.Value][]*functionPlan)
	for _, fn := range functions {
		if fn.IsDeclaration() {
			continue
		}
		f := &functionPlan{fn: fn, floor: floors[fn], ordinary: true}
		if p.canonical != nil && p.canonical.variants[p.canonical.origins[fn]] == fn {
			f.knownISA = featureAVX2
		}
		view := folds.view(fn)
		var contexts map[llvm.BasicBlock]guardFacts
		if candidates[fn] {
			contexts = blockGuards(view, queries)
		}
		// Functions outside the guarded graph need only the static call ABI scan.
		if !reachable[fn] {
			p.abi.calls(view, func(_ llvm.Value, w int) { f.observe(profile{width: w}, guardFacts{}, true) })
			if len(f.widths) == 0 {
				continue
			}
		} else {
			for bb := view.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
				facts := contexts[bb]
				for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
					target := directGuardCallee(i)
					call := i.InstructionOpcode() == llvm.Call || i.InstructionOpcode() == llvm.Invoke
					op := p.abi.operation(i, target)
					f.observe(op, facts, call && op.width != 0 && (target.IsNil() || target.IntrinsicID() == 0))
					// A promised entry may live in another package. Its body is
					// unavailable here, but a caller can still forward the AVX2 proof.
					if p.canonical != nil && !p.canonical.variants[target].IsNil() {
						f.work.isa |= featureAVX2
					}
					if call {
						f.calls = append(f.calls, plannedCall{target, facts})
						if !target.IsNil() {
							parents[target] = append(parents[target], f)
						}
					}
				}
			}
		}
		p.funcs[fn] = f
		if f.work != (profile{}) {
			visit = append(visit, fn)
		}
	}
	for len(visit) > 0 {
		target := visit[len(visit)-1]
		visit = visit[:len(visit)-1]
		for _, caller := range parents[target] {
			if work := caller.work.join(p.funcs[target].work); work != caller.work {
				caller.work = work
				visit = append(visit, caller.fn)
			}
		}
	}
	for _, fn := range functions {
		if f := p.funcs[fn]; f != nil && f.work != (profile{}) && !p.isCanonicalRoot(fn) {
			p.entry(versionKey{fn: fn})
		}
	}
	for i := 0; i < len(p.entryOrder); i++ {
		p.expand(p.entryOrder[i])
	}
}

func planVersions(abi *wideABI, avx2 *avx2Plan, baseline string) error {
	mod := abi.mod
	avx2.materialize()
	var functions []llvm.Value
	queries := map[llvm.Value]uint64{}
	candidates := map[llvm.Value]bool{}
	folds := baselineFolds{calls: map[llvm.Value][]llvm.Value{}, queries: map[llvm.Value]bool{}, views: map[llvm.Value]llvm.Value{}}
	defer func() {
		for _, v := range folds.views {
			v.EraseFromParentAsFunction()
		}
	}()
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		functions = append(functions, fn)
		name := checkedQuery(fn)
		bit := predicateBits[name]
		if bit == 0 {
			continue
		}
		guaranteed := guaranteedQuery(name, baseline)
		if guaranteed {
			folds.queries[fn] = true
		} else {
			queries[fn] = bit
		}
		for use := fn.FirstUse(); !use.IsNil(); use = use.NextUse() {
			c := use.User()
			if c.IsACallInst().IsNil() || c.CalledValue() != fn {
				continue
			}
			parent := c.InstructionParent().Parent()
			if guaranteed {
				folds.calls[parent] = append(folds.calls[parent], c)
			} else {
				candidates[parent] = true
			}
		}
	}
	p := versionPlan{abi: abi, canonical: avx2, funcs: map[llvm.Value]*functionPlan{}, entries: map[versionKey]*entryPlan{}, bodies: map[versionKey]*bodyPlan{}}
	p.collect(functions, &folds, queries, candidates, baseline)
	return p.emit(functions, &folds, queries, baseline)
}

func (p *versionPlan) bodyName(k versionKey) string {
	f := p.funcs[k.fn]
	if k.profile == (profile{}) {
		if len(p.entries[k].choices) == 0 {
			return k.fn.Name()
		}
		suffix := "guard_base"
		if f.ordinary {
			suffix = "width_base"
		}
		return k.fn.Name() + symbolPrefix + suffix
	}
	if f.ordinary && k.isa == 0 && len(f.widths) > 0 {
		if k.width == 512 {
			return k.fn.Name() + symbolPrefix + "zmm"
		}
		return k.fn.Name() + symbolPrefix + "ymm"
	}
	return k.fn.Name() + k.suffix()
}

func (p *versionPlan) entryName(e *entryPlan) string {
	if e.key.profile == (profile{}) {
		return e.key.fn.Name()
	}
	return p.bodyName(e.key) + "_entry"
}

func (p *versionPlan) validate() error {
	mod := p.abi.mod
	reserved := map[string]bool{}
	check := func(name string) error {
		if reserved[name] || !mod.NamedValue(name).IsNil() {
			return fmt.Errorf("SIMD FMV symbol collision: %s", name)
		}
		reserved[name] = true
		return nil
	}
	for _, b := range p.bodyOrder {
		if name := p.bodyName(b.key); name != b.key.fn.Name() {
			if err := check(name); err != nil {
				return err
			}
		}
	}
	for _, e := range p.entryOrder {
		if len(e.choices) == 0 {
			continue
		}
		name := p.entryName(e)
		if name != e.key.fn.Name() {
			if err := check(name); err != nil {
				return err
			}
		}
		for _, suffix := range []string{slotSuffix, resolverSuffix} {
			if err := check(name + suffix); err != nil {
				return err
			}
		}
	}
	return nil
}

func privateBody(fn llvm.Value, name string, cap profile) llvm.Value {
	v := fn.CloneFunction()
	v.SetName(name)
	v.SetLinkage(llvm.InternalLinkage)
	v.RemoveStringAttributeAtIndex(-1, EntryAttribute)
	v.RemoveStringAttributeAtIndex(-1, InlineEntryAttribute)
	cap.attributes(v)
	return v
}

func (p *versionPlan) emit(functions []llvm.Value, folds *baselineFolds, queries map[llvm.Value]uint64, baseline string) error {
	if err := p.validate(); err != nil {
		return err
	}
	folds.commit()
	for _, fn := range functions {
		if fn.IntrinsicID() == 0 {
			p.abi.applyRegisterABI(fn, baseline)
		}
	}
	if p.canonical == nil && len(p.bodies) == 0 {
		return nil
	}
	mod := p.abi.mod
	source := newSourceInfo(mod)
	clone := func(k versionKey, name string) llvm.Value {
		v := privateBody(k.fn, name, k.profile)
		original := k.fn
		if p.canonical != nil && !p.canonical.origins[original].IsNil() {
			original = p.canonical.origins[original]
		}
		source.clone(original, v)
		return v
	}
	// Allocate all targets before rewriting recursive edges or replacing entries.
	for _, b := range p.bodyOrder {
		name := p.bodyName(b.key)
		b.value = b.key.fn
		if name != b.key.fn.Name() {
			b.value = clone(b.key, name)
		}
	}
	plans := map[llvm.Value]dispatchPlan{}
	for _, e := range p.entryOrder {
		body := p.bodies[e.key].value
		e.value = body
		if len(e.choices) == 0 {
			continue
		}
		e.value = e.key.fn
		if e.key.profile != (profile{}) {
			e.value = clone(e.key, p.entryName(e))
		}
		plan := dispatchPlan{baseline: body}
		for _, c := range e.choices {
			plan.choices = append(plan.choices, dispatchChoice{bits: c.bits, anyOf: c.anyOf, fn: p.bodies[c.key].value})
		}
		plans[e.value] = plan
	}
	for _, b := range p.bodyOrder {
		n := 0
		var fold []llvm.Value
		if len(b.targets) != 0 {
			for bb := b.value.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
				for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
					if i.InstructionOpcode() != llvm.Call && i.InstructionOpcode() != llvm.Invoke {
						continue
					}
					dst := b.targets[n]
					n++
					if !i.IsACallInst().IsNil() && queries[i.CalledValue()]&b.key.isa != 0 {
						fold = append(fold, i)
					}
					target := dst.fn
					if e := p.entries[dst]; e != nil {
						target = e.value
					}
					if !target.IsNil() && target != i.CalledValue() {
						i.SetOperand(i.OperandsCount()-1, target)
					}
				}
			}
		}
		if len(fold) > 0 {
			foldTrue(b.value, fold)
		}
		f := p.funcs[b.key.fn]
		if len(f.widths) > 0 {
			p.abi.removeWiderCalls(b.value, p.abi.registerFloor(b.value, baseline))
		}
	}
	p.canonical.commit(mod, source, plans)
	for _, e := range p.entryOrder {
		if len(e.choices) != 0 {
			emitDispatch(mod, source, e.value, plans[e.value], nil)
		}
	}
	return nil
}
