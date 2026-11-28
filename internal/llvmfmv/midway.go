package llvmfmv

import "github.com/xgo-dev/llvm"

const midwayFloorAttr = "llgo.cpu.midway-floor"

// Midway's amd64 dispatcher selected the hardware width using AVX, AVX2,
// or the Go AVX512 bundle. Record its instruction capabilities separately
// from the independently observable Go query results.
func MarkMidwayFloor(fn llvm.Value, width int) {
	var name string
	switch width {
	case 128:
		name = "x86.avx"
	case 256:
		name = "x86.avx2"
	case 512:
		name = "x86.avx512"
	default:
		return
	}
	if stringAttribute(fn, midwayFloorAttr) == name {
		return
	}
	fn.AddTargetDependentFunctionAttr(midwayFloorAttr, name)
	bits := predicateBits[name]
	if width == 512 {
		bits |= predicateBits["x86.avx2"]
	}
	profile{isa: bits}.attributes(fn)
}

func midwayFloor(fn llvm.Value) uint64 {
	switch stringAttribute(fn, midwayFloorAttr) {
	case "x86.avx":
		return predicateBits["x86.avx"]
	case "x86.avx2":
		return predicateBits["x86.avx"] | predicateBits["x86.avx2"]
	case "x86.avx512":
		return predicateBits["x86.avx"] | predicateBits["x86.avx2"] | predicateBits["x86.avx512"]
	}
	return 0
}
