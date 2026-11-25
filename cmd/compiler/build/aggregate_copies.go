package build

import (
	"github.com/xgo-dev/llgo/cmd/compiler/abi"
	"github.com/xgo-dev/llvm"
)

func lowerAggregateCopies(td llvm.TargetData, mod llvm.Module, config abi.AggregateLoweringConfig) int {
	return abi.LowerAggregateCopies(td, mod, config)
}
