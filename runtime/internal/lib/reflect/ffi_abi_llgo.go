//go:build llgo

package reflect

import "runtime"

const ffiCallSliceAsTriple = false

// Match the empty-parameter rules in internal/cabi. Microsoft's x64 ABI
// retains an integer slot; its ARM64 ABI, AAPCS and SysV64 omit empty values.
const ffiSkipEmptyArgs = runtime.GOARCH == "arm64" ||
	runtime.GOARCH == "amd64" && runtime.GOOS != "windows" ||
	runtime.GOARCH == "arm" || runtime.GOARCH == "riscv64" || runtime.GOARCH == "wasm"
