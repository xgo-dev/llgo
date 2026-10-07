// Package llvmfmv implements early, CPU-guarded LLVM function specialization.
// Its C++ implementation is compiled into the compiler, independently of LTO.
package llvmfmv

/*
#cgo CXXFLAGS: -std=c++17 -fno-rtti
#include <stdlib.h>
#include "fmv.h"
*/
import "C"

import (
	"errors"
	"unsafe"

	"github.com/xgo-dev/llvm"
)

// Run preserves baseline entries and creates AVX2 implementations of supported
// SIMD128 functions. It must run before ABI lowering and target optimization,
// including in O0 and ModeGen builds. Running it again is harmless.
func Run(mod llvm.Module) error {
	message := C.llgoRunSIMDFMV(unsafe.Pointer(mod.C))
	if message == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(message))
	return errors.New(C.GoString(message))
}
