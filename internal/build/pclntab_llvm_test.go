//go:build !llgo
// +build !llgo

package build

import (
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/pclntab"
	llvm "github.com/xgo-dev/llvm"
)

func TestEmitPCLnFindFuncBuckets(t *testing.T) {
	llvm.InitializeAllTargets()
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("pclntab-test")
	defer mod.Dispose()

	buckets := []pclntab.FindFuncBucket{
		{Idx: 0, Subbuckets: [16]uint8{0, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}},
		{Idx: 3, Subbuckets: [16]uint8{0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
	}
	emitPCLnFindFuncBuckets(mod, "__llgo_findfunctab", buckets)
	ir := mod.String()
	for _, want := range []string{
		`@__llgo_findfunctab = unnamed_addr constant [2 x { i32, [16 x i8] }]`,
		`{ i32 0, [16 x i8] c"\00\01\02`,
		`{ i32 3, [16 x i8] c"\00\00\01`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("IR missing %q:\n%s", want, ir)
		}
	}
}
