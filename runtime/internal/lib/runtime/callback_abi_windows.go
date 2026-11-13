package runtime

import (
	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

func callbackABI(ft *abi.FuncType) ffi.ABI {
	if ft.IsStdcall() {
		return ffi.WindowsStdcallABI
	}
	return ffi.DefaultABI
}
