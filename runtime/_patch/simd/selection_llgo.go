//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd

import "runtime"

// Validate at the query boundary, after the standard simd initialization.
// Source-patch files need not run their init functions after upstream files.
// Upstream midway_common.go maps both absent hardware (archMaxVectorSize == 0)
// and GODEBUG=simd=0 to maxVectorSize=128, emulated=true. Width describes the
// logical vector even in emulation; zero is only the hardware eligibility result.
func checkMidwayWidth() {
	if maxVectorSize != 128 && (runtime.GOARCH != "amd64" || maxVectorSize != 256) {
		panic("LLGo portable SIMD selected an unsupported vector width")
	}
}

func VectorBitSize() int {
	checkMidwayWidth()
	return maxVectorSize
}

func Emulated() bool {
	checkMidwayWidth()
	return emulated
}
