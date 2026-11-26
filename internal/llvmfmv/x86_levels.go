package llvmfmv

import "github.com/xgo-dev/llvm"

// OS-enabled register state is detected before applying CPU feature overrides.
// These bits select a machine ABI and never imply a Go predicate.
const (
	machineYMM = 1 << 17
	machineZMM = 1 << 18
)

var x86RegisterLevels = []struct {
	bits      int
	witnesses uint64
	machine   uint64
}{
	{256, ymmPredicates, machineYMM | machineZMM},
	{512, zmmPredicates, machineZMM},
}

// A single guarded instruction family can keep its hot loop in one body.
// Mixed families use these common x86 combinations. The remaining independent
// operations retain their source guards and can execute in local ISA leaves.
func x86VersionMasks(required uint64) []uint64 {
	if required == 0 {
		return []uint64{0}
	}
	if required&(required-1) == 0 {
		return []uint64{0, required}
	}
	fma := predicateBits["x86.fma"]
	avx512 := predicateBits["x86.avx512"]
	masks := []uint64{0}
	for _, level := range []uint64{featureAVX2, featureAVX2 | fma, featureAVX2 | fma | avx512} {
		if bits := required & level; bits != masks[len(masks)-1] {
			masks = append(masks, bits)
		}
	}
	return masks
}

// Machine capabilities can be used for lowering. Only guarantees from the
// GOAMD64 baseline also make the corresponding effective Go queries constant.
func machineFeatures(fn llvm.Value, baseline string) uint64 {
	features := midwayFloor(fn)
	if baseline == "v3" || baseline == "v4" {
		features |= featureAVX2 | featureAVX | predicateBits["x86.fma"]
	}
	if baseline == "v4" {
		features |= predicateBits["x86.avx512"]
	}
	return features
}

// Collect local requirements after the baseline's guaranteed queries have
// been folded. Calls contribute only their machine signature.
func (a *wideABI) planFunction(fn, view llvm.Value, baseline string, contexts map[llvm.BasicBlock]uint64) functionPlan {
	floor := a.registerFloor(fn, baseline)
	var widths [2]bool // YMM, ZMM
	var calls [2]bool  // fixed register calls still need their ABI under GODEBUG
	var isa uint64
	for bb := view.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		facts := contexts[bb]
		for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
			op := a.operation(i, directGuardCallee(i))
			switch op.width {
			case 512:
				widths[1] = true
				if a.callWidth(i) != 0 {
					calls[1] = true
				} else {
					widths[0] = true // Generic operations can benefit from YMMs too.
				}
			case 256:
				widths[0] = true
				if a.callWidth(i) != 0 {
					calls[0] = true
				}
			}
			isa |= op.isa & facts
			// AVX authorizes the YMM carrier. Packed integer arithmetic also
			// benefits from AVX2 in a pointer kernel with no source query.
			if floor < 256 && op.width == 256 && op.isa == featureAVX2 && i.InstructionOpcode() != llvm.Call && i.InstructionOpcode() != llvm.Invoke {
				isa |= featureAVX2
			}
			// AVX2 guarded vector expressions can also use VEX instructions
			// for 128-bit values. A pointer-only call has no such work.
			if facts&featureAVX2 != 0 && a.localVectorWork(i) {
				isa |= featureAVX2
			}
		}
	}
	plan := functionPlan{fn: fn}
	isa &^= machineFeatures(fn, baseline)
	masks := x86VersionMasks(isa)
	plan.outline = isa&^masks[len(masks)-1] != 0
	for _, bits := range masks {
		if bits != 0 {
			plan.versions = append(plan.versions, localVersion{profile: profile{isa: bits}})
		}
		for i, level := range x86RegisterLevels {
			if widths[i] && level.bits > max(floor, isaWidth(bits)) {
				witnesses := level.witnesses
				if calls[i] {
					witnesses |= level.machine
				}
				plan.versions = append(plan.versions, localVersion{profile{level.bits, bits}, witnesses})
			}
		}
	}
	return plan
}
