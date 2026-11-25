package llvmfmv

import (
	"strings"

	"github.com/xgo-dev/llvm"
)

// A direct vector in the machine signature fixes the register width for both
// definitions and calls, including calls through a Go function value. Ordinary
// aggregate ABI lowering has already handled stack and indirect arguments.
type wideABI struct {
	mod    llvm.Module
	widths map[llvm.Type]int
	data   llvm.TargetData
	abi    x86ABI
	source *sourceInfo
}

func (a *wideABI) signature(ft llvm.Type) int {
	if w, ok := a.widths[ft]; ok {
		return w
	}
	width := a.valueWidth(ft.ReturnType())
	for _, t := range ft.ParamTypes() {
		width = max(width, a.valueWidth(t))
	}
	a.widths[ft] = width
	return width
}

// Look through LLVM register tuples that survive aggregate lowering. Pointers
// (including sret and byval) already have a width-independent machine ABI.
func (a *wideABI) valueWidth(t llvm.Type) int {
	switch t.TypeKind() {
	case llvm.VectorTypeKind:
		switch a.data.TypeSizeInBits(t) {
		case 256:
			return 256
		case 512:
			return 512
		}
	case llvm.ArrayTypeKind:
		if t.ArrayLength() != 0 {
			return a.valueWidth(t.ElementType())
		}
	case llvm.StructTypeKind:
		width := 0
		if !t.IsStructOpaque() {
			for _, field := range t.StructElementTypes() {
				width = max(width, a.valueWidth(field))
			}
		}
		return width
	}
	return 0
}

func wideCall(inst llvm.Value) bool {
	if inst.IsACallInst().IsNil() && inst.IsAInvokeInst().IsNil() {
		return false
	}
	callee := inst.CalledValue()
	return callee.IsAInlineAsm().IsNil() && (callee.IsAFunction().IsNil() || callee.IntrinsicID() == 0)
}

func (a *wideABI) calls(fn llvm.Value, visit func(llvm.Value, int)) {
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if wideCall(inst) {
				if width := a.signature(inst.CalledFunctionType()); width != 0 {
					visit(inst, width)
				}
			}
		}
	}
}

func targetWidth(fn llvm.Value, baseline string) int {
	width := 0
	switch baseline {
	case "v3":
		width = 256
	case "v4":
		width = 512
	}
	for _, f := range strings.Split(stringAttribute(fn, "target-features"), ",") {
		switch f {
		case "+avx", "+avx2":
			width = max(width, 256)
		case "+avx512f":
			width = 512
		case "-avx512f":
			width = min(width, 256)
		case "-avx":
			width = 0
		}
	}
	return width
}

func enableWidth(fn llvm.Value, width int) {
	feature := "+avx"
	if width == 512 {
		feature = "+avx512f"
	}
	if old := stringAttribute(fn, "target-features"); old != "" {
		feature = old + "," + feature
	}
	fn.AddTargetDependentFunctionAttr("target-features", feature)
}

// The narrow implementation can still take every scalar path. Only reaching
// a call that requires unavailable registers violates the archsimd precondition.
func (a *wideABI) removeWiderCalls(fn llvm.Value, width int) {
	b := a.mod.Context().NewBuilder()
	defer b.Dispose()
	a.calls(fn, func(call llvm.Value, need int) {
		if need > width {
			b.SetInsertPointBefore(call)
			b.CreateIntrinsic(a.mod.Context().VoidType(), llvm.LookupIntrinsicID("llvm.trap"), nil, "")
		}
	})
	fn.RemoveUnreachableBlocks()
}

// Several independent Go predicates guarantee the registers for a YMM call.
func ymmChoice(fn llvm.Value) dispatchChoice {
	var bits uint64
	for _, p := range x86Predicates {
		if p.registerBits >= 256 {
			bits |= p.runtimeBit
		}
	}
	return dispatchChoice{bits: bits, fn: fn, anyBit: true}
}

func applyWideABI(mod llvm.Module, baseline string) {
	a := wideABI{
		mod: mod, widths: make(map[llvm.Type]int),
		data: llvm.NewTargetData(mod.DataLayout()), abi: x86ABIFor(mod),
	}
	defer a.data.Dispose()
	// Clones and dispatchers created below already have the finalized call ABI.
	last := mod.LastFunction()
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IntrinsicID() == 0 {
			a.lower(fn, baseline)
		}
		if fn == last {
			break
		}
	}
}

func (a *wideABI) lower(fn llvm.Value, baseline string) {
	width := a.signature(fn.GlobalValueType())
	floor := targetWidth(fn, baseline)
	if width > floor {
		enableWidth(fn, width)
		floor = width
	}
	if width != 0 {
		a.abi.applyWideCallingConvention(fn)
	}
	var ymm, zmm bool
	a.calls(fn, func(call llvm.Value, need int) {
		a.abi.applyWideCallingConvention(call)
		if need > floor {
			switch need {
			case 256:
				ymm = true
			case 512:
				zmm = true
			}
		}
	})
	if !ymm && !zmm {
		return
	}

	// A scalar caller can reach a wider call conditionally, including through
	// a function value. Keep the caller's signature fixed in every version.
	if a.source == nil {
		a.source = newSourceInfo(a.mod)
	}
	body := func(suffix string) llvm.Value {
		v := fn.CloneFunction()
		v.SetName(fn.Name() + symbolPrefix + suffix)
		v.SetLinkage(llvm.InternalLinkage)
		a.source.clone(fn, v)
		return v
	}
	var choices []dispatchChoice
	if ymm {
		v := body("ymm")
		enableWidth(v, 256)
		if zmm {
			a.removeWiderCalls(v, 256)
		}
		choices = append(choices, ymmChoice(v))
	}
	if zmm {
		v := body("zmm")
		enableWidth(v, 512)
		// Go requires its combined AVX512 feature for 512-bit vectors.
		choices = append(choices, dispatchChoice{bits: predicateBits[X86Query("AVX512")], fn: v})
	}
	base := body("width_base")
	a.removeWiderCalls(base, floor)
	dispatch(a.mod, a.source, fn, base, choices)
}
