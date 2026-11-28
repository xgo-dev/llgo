//go:build !windows && !wasm && !nogc && !(baremetal && llgo.baremetal.stdio)

package runtime

import (
	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

func callbackABI(*abi.FuncType) ffi.ABI { return ffi.DefaultABI }
