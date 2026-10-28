//go:build llgo

package reflect

import "runtime"

const ffiCallSliceAsTriple = false

// Match empty-parameter lowering in internal/cabi. Windows/amd64 models an
// empty aggregate as an i32, and 386 has position-dependent rules; neither
// can use the register-free mapping used by the targets below.
const ffiOmitEmptyArgs = runtime.GOARCH == "amd64" && runtime.GOOS != "windows" ||
	runtime.GOARCH == "arm64" || runtime.GOARCH == "arm" ||
	runtime.GOARCH == "riscv64" || runtime.GOARCH == "riscv32" || runtime.GOARCH == "wasm"
