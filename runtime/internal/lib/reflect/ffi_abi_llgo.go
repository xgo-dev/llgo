//go:build llgo

package reflect

import "runtime"

const ffiCallSliceAsTriple = false

// Match the empty-parameter rules in internal/cabi. Microsoft's x64 ABI
// retains an integer slot; its ARM64 ABI, AAPCS and SysV64 omit empty values.
const ffiSkipEmptyArgs = runtime.GOARCH == "arm64" ||
	(runtime.GOARCH == "amd64" && runtime.GOOS != "windows") ||
	runtime.GOARCH == "arm" || runtime.GOARCH == "riscv32" ||
	runtime.GOARCH == "riscv64" || runtime.GOARCH == "wasm"

// HostLayout tail padding is part of LLVM storage and, on most targets, of
// the callable ABI. Windows/386 CABI strips those Go field wrappers, so
// libffi must describe the unpadded layout: padding to typ.Size_ would turn
// an 8-byte register return such as (HostTail, byte) into a 12-byte sret.
const ffiHostLayoutTailPadding = runtime.GOOS != "windows" || runtime.GOARCH != "386"
