package abi

import (
	"testing"

	"github.com/xgo-dev/llgo/internal/abi/abitest"
	"github.com/xgo-dev/llvm"
)

func TestLowerLargeAggregatesFunctionProperties(t *testing.T) {
	abitest.CheckFunctionProperties(t, "[65537 x i8]", func(mod llvm.Module) {
		td := llvm.NewTargetData("e-p:64:64-i64:64-n8:16:32:64-S128")
		defer td.Dispose()
		LowerLargeAggregates(td, mod, AggregateLoweringConfig{GoWordSize: 8})
	})
}
