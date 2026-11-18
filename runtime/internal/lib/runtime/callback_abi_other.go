//go:build !windows && !nogc

package runtime

import (
	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

func callbackABI(*abi.FuncType) ffi.ABI { return ffi.DefaultABI }
