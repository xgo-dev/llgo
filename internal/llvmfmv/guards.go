package llvmfmv

import (
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
	if !cond.IsACallInst().IsNil() && taken {
		return queries[cond.CalledValue()]
	}
	if cond.IsAInstruction().IsNil() {
		return 0
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
			return a | b
		}
		return a & b
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
func (a *wideABI) intrinsicGuards(inst, fn llvm.Value) uint64 {
	if hint := inst.Metadata(a.mod.Context().MDKindID(PreferFeatureMD)); !hint.IsNil() {
		if operands := hint.MDNodeOperands(); len(operands) == 1 && operands[0].IsAMDString() {
			return predicateBits[operands[0].MDString()]
		}
	}
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

func blockGuards(fn llvm.Value, queries map[llvm.Value]uint64) map[llvm.BasicBlock]uint64 {
	entry := fn.EntryBasicBlock()
	known := map[llvm.BasicBlock]uint64{entry: 0}
	queue := []llvm.BasicBlock{entry}
	for len(queue) != 0 {
		block := queue[0]
		queue = queue[1:]
		term := block.LastInstruction()
		for i := 0; i < term.SuccessorsCount(); i++ {
			f := known[block]
			if term.InstructionOpcode() == llvm.Br && term.SuccessorsCount() == 2 {
				f |= positiveGuard(term.Operand(0), i == 0, queries)
			}
			next := term.Successor(i)
			if prior, ok := known[next]; ok {
				f &= prior
				if f == prior {
					continue
				}
			}
			known[next] = f
			queue = append(queue, next)
		}
	}
	return known
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

func isaWidth(bits uint64) int {
	switch {
	case bits&zmmPredicates != 0:
		return 512
	case bits&ymmPredicates != 0:
		return 256
	}
	return 0
}

func (p profile) suffix() string {
	if p.width == 0 {
		return guardSuffix(p.isa)
	}
	name := "ymm"
	if p.width == 512 {
		name = "zmm"
	}
	names := strings.TrimPrefix(guardSuffix(p.isa), symbolPrefix+"guard_")
	if names != "" {
		name += "_" + names
	}
	return symbolPrefix + "guard_" + name
}

func (p profile) attributes(fn llvm.Value) {
	if p.width != 0 {
		enableWidth(fn, p.width)
	}
	if p.isa == 0 {
		return
	}
	features := stringAttribute(fn, "target-features")
	for _, pred := range x86Predicates {
		if pred.runtimeBit&p.isa != 0 {
			if features != "" {
				features += ","
			}
			features += pred.features
		}
	}
	fn.AddTargetDependentFunctionAttr("target-features", features)
}

func (a *wideABI) operation(inst, callee llvm.Value) profile {
	typ := inst.Type()
	var p profile
	switch inst.InstructionOpcode() {
	case llvm.Call, llvm.Invoke:
		if callee.IsNil() || callee.IntrinsicID() == 0 {
			p.width = a.callWidth(inst)
			return p
		}
		p.isa = a.intrinsicGuards(inst, callee)
		if typ.TypeKind() != llvm.VectorTypeKind && inst.ArgOperandsCount() != 0 {
			typ = inst.Operand(0).Type()
		}
	case llvm.Store, llvm.ExtractElement, llvm.ICmp, llvm.FCmp:
		typ = inst.Operand(0).Type()
	}
	if typ.TypeKind() == llvm.VectorTypeKind {
		switch w := a.data.TypeSizeInBits(typ); {
		case w > 256:
			p.width = 512
		case w > 128:
			p.width = 256
		}
		// AVX2 also supplies packed integer arithmetic for YMM registers.
		if p.width != 0 && typ.ElementType().TypeKind() == llvm.IntegerTypeKind {
			switch inst.InstructionOpcode() {
			case llvm.Add, llvm.Sub, llvm.Mul, llvm.Shl, llvm.LShr, llvm.AShr, llvm.ICmp:
				p.isa |= featureAVX2
			}
		}
	}
	return p
}
