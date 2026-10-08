// Package llvmattr defines LLVM attribute encodings and audited runtime contracts.
// Keep ABI transport and semantic models on the same encoding. The definitions
// follow LLVM 22's llvm/Support/ModRef.h and llvm/lib/IR/Attributes.cpp; the tests
// check them against the linked LLVM IR parser rather than duplicating the math.
package llvmattr

const (
	memoryBitsPerLocation        = 2
	memoryRef             uint64 = 1
	memoryMod             uint64 = 2
)

// LLVM's IRMemLocation order, including the target-specific state locations.
const (
	argMem = iota
	inaccessibleMem
	errnoMem
	otherMem
	targetMem0
	targetMem1
)

const (
	MemoryNone     uint64 = 0
	MemoryArgRead         = memoryRef << (memoryBitsPerLocation * argMem)
	MemoryArgWrite        = memoryMod << (memoryBitsPerLocation * argMem)
	// MemoryRead is memory(read), including target-specific state. Omitting the
	// target locations would silently impose additional memory(none) contracts.
	MemoryRead = MemoryArgRead |
		memoryRef<<(memoryBitsPerLocation*inaccessibleMem) |
		memoryRef<<(memoryBitsPerLocation*errnoMem) |
		memoryRef<<(memoryBitsPerLocation*otherMem) |
		memoryRef<<(memoryBitsPerLocation*targetMem0) |
		memoryRef<<(memoryBitsPerLocation*targetMem1)
)

const (
	allocSizeIndexBits = 32
	// AllocSizeNoCount is LLVM's sentinel for an absent element-count argument.
	AllocSizeNoCount = ^uint32(0)
)

// AllocSize encodes zero-based parameter indices, not byte sizes. Pass
// AllocSizeNoCount for count to express a single-argument allocsize attribute.
func AllocSize(element, count uint32) uint64 {
	return uint64(element)<<allocSizeIndexBits | uint64(count)
}

func AllocSizeArgs(value uint64) (element, count uint32) {
	return uint32(value >> allocSizeIndexBits), uint32(value)
}
