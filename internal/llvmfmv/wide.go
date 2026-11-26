package llvmfmv

import (
	"strings"

	"github.com/xgo-dev/llvm"
)

// A direct vector in the machine signature fixes the register width for both
// definitions and calls, including calls through a Go function value. Ordinary
// aggregate ABI lowering has already handled stack and indirect arguments.
type wideABI struct {
	mod      llvm.Module
	widths   map[llvm.Type]int
	outlines map[outlineKey]llvm.Value
	data     llvm.TargetData
	abi      x86ABI
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

func (a *wideABI) callWidth(inst llvm.Value) int {
	if inst.IsACallInst().IsNil() && inst.IsAInvokeInst().IsNil() {
		return 0
	}
	callee := inst.CalledValue()
	if !callee.IsAInlineAsm().IsNil() || (!callee.IsAFunction().IsNil() && callee.IntrinsicID() != 0) {
		return 0
	}
	return a.signature(inst.CalledFunctionType())
}

func (a *wideABI) calls(fn llvm.Value, visit func(llvm.Value, int)) {
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if width := a.callWidth(inst); width != 0 {
				visit(inst, width)
			}
		}
	}
}

func baselineWidth(baseline string) int {
	switch baseline {
	case "v3":
		return 256
	case "v4":
		return 512
	}
	return 0
}

func targetWidth(fn llvm.Value, baseline string) int {
	width := baselineWidth(baseline)
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

func registerPredicateBits(width int) uint64 {
	var bits uint64
	for _, p := range x86Predicates {
		if p.registerBits >= width {
			bits |= p.runtimeBit
		}
	}
	return bits
}

var (
	ymmPredicates = registerPredicateBits(256)
	zmmPredicates = registerPredicateBits(512)
)

func newWideABI(mod llvm.Module) *wideABI {
	return &wideABI{
		mod: mod, widths: make(map[llvm.Type]int),
		data: llvm.NewTargetData(mod.DataLayout()), abi: x86ABIFor(mod),
	}
}

// A wide argument or result is a caller/callee ABI contract. It establishes
// the same register floor on every implementation reached through this entry.
func (a *wideABI) registerFloor(fn llvm.Value, baseline string) int {
	return max(targetWidth(fn, baseline), a.signature(fn.GlobalValueType()))
}

// Apply the fixed platform ABI independently of implementation selection.
// CloneFunction preserves these conventions; new forwarding calls use the
// enclosing function's convention in tailReturn.
func (a *wideABI) applyRegisterABI(fn llvm.Value, baseline string) {
	width := a.signature(fn.GlobalValueType())
	if width != 0 {
		if width > targetWidth(fn, baseline) {
			enableWidth(fn, width)
		}
		a.abi.applyWideCallingConvention(fn)
	}
	if a.abi == x86Win64 {
		a.calls(fn, func(call llvm.Value, _ int) {
			a.abi.applyWideCallingConvention(call)
		})
	}
}
