package llvmfmv

import (
	"fmt"

	"github.com/xgo-dev/llvm"
)

// Width authorizes register use. Only isa bits tested by this entry can fold
// the corresponding Go queries, even when a wider instruction set implies them.
type profile struct {
	width int
	isa   uint64
}

type localVersion struct {
	profile
	witness uint64 // any one of these effective queries guarantees the width
}

type functionPlan struct {
	fn       llvm.Value
	versions []localVersion // ascending priority, with the baseline implicit
	outline  bool           // a guarded family outside the common x86 levels
}

func planVersions(a *wideABI, baseline string) error {
	mod := a.mod
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

	var plans []functionPlan
	for _, fn := range functions {
		if fn.IsDeclaration() || !supported(fn) {
			continue
		}
		view := folds.view(fn)
		var contexts map[llvm.BasicBlock]uint64
		if candidates[fn] {
			contexts = blockGuards(view, queries)
		}
		plan := a.planFunction(fn, view, baseline, contexts)
		if len(plan.versions) == 0 && !plan.outline {
			continue
		}
		if err := a.validateOutlines(view, baseline, queries, machineFeatures(fn, baseline)); err != nil {
			return err
		}
		plans = append(plans, plan)
		if len(plan.versions) == 0 {
			continue
		}
		// Every generated name uses a distinct suffix of this function.
		// Check the entire input before changing any public definition.
		names := []string{symbolPrefix + "guard_base", resolverSuffix, slotSuffix}
		for _, v := range plan.versions {
			names = append(names, v.suffix())
		}
		for _, suffix := range names {
			if name := fn.Name() + suffix; !mod.NamedValue(name).IsNil() {
				return fmt.Errorf("SIMD FMV symbol collision: %s", name)
			}
		}
	}
	folds.commit()
	for _, fn := range functions {
		if fn.IntrinsicID() == 0 {
			a.applyRegisterABI(fn, baseline)
		}
	}
	if len(plans) != 0 {
		source := newSourceInfo(mod)
		for _, plan := range plans {
			plan.emit(a, source, queries, baseline)
		}
	}
	return nil
}

func (p functionPlan) emit(a *wideABI, source *sourceInfo, queries map[llvm.Value]uint64, baseline string) {
	if len(p.versions) == 0 {
		a.outlineBody(p.fn, baseline, queries, machineFeatures(p.fn, baseline))
		return
	}
	clone := func(suffix string, cap profile) llvm.Value {
		v := p.fn.CloneFunction()
		v.SetName(p.fn.Name() + suffix)
		v.SetLinkage(llvm.InternalLinkage)
		v.RemoveStringAttributeAtIndex(-1, InlineEntryAttribute)
		cap.attributes(v)
		source.clone(p.fn, v)

		if cap.isa != 0 {
			var fold []llvm.Value
			for bb := v.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
				for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
					if !i.IsACallInst().IsNil() && queries[i.CalledValue()]&cap.isa != 0 {
						fold = append(fold, i)
					}
				}
			}
			foldTrue(v, fold)
		}
		a.outlineBody(v, baseline, queries, machineFeatures(p.fn, baseline)|cap.isa)
		return v
	}
	plan := dispatchPlan{baseline: clone(symbolPrefix+"guard_base", profile{})}
	for _, v := range p.versions {
		body := clone(v.suffix(), v.profile)
		plan.choices = append(plan.choices, dispatchChoice{bits: v.isa, anyOf: v.witness, fn: body})
	}
	emitDispatch(a.mod, source, p.fn, plan)
}
