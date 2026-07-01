//go:build !llgo
// +build !llgo

package build

import (
	"strings"
	"testing"

	llvm "github.com/xgo-dev/llvm"
)

func TestBuildPCLnFindFuncBucketsLookup(t *testing.T) {
	ftab := []pclnFuncTabEntry{
		{entryOff: 0, funcOff: 11},
		{entryOff: 16, funcOff: 22},
		{entryOff: 64, funcOff: 33},
		{entryOff: 4096, funcOff: 44},
		{entryOff: 4352, funcOff: 55},
		{entryOff: 8192, funcOff: 0}, // sentinel
	}
	buckets, err := buildPCLnFindFuncBuckets(ftab, 8192)
	if err != nil {
		t.Fatalf("buildPCLnFindFuncBuckets: %v", err)
	}
	if got, want := len(buckets), 2; got != want {
		t.Fatalf("bucket count = %d, want %d", got, want)
	}
	for _, tt := range []struct {
		pc   uint32
		want int
	}{
		{pc: 0, want: 0},
		{pc: 15, want: 0},
		{pc: 16, want: 1},
		{pc: 63, want: 1},
		{pc: 64, want: 2},
		{pc: 4095, want: 2},
		{pc: 4096, want: 3},
		{pc: 4351, want: 3},
		{pc: 4352, want: 4},
		{pc: 8191, want: 4},
	} {
		if got := pclnLookupFuncIndex(ftab, buckets, tt.pc); got != tt.want {
			t.Fatalf("lookup(%d) = %d, want %d", tt.pc, got, tt.want)
		}
	}
}

func TestBuildPCLnFindFuncBucketsRejectsOverflow(t *testing.T) {
	ftab := make([]pclnFuncTabEntry, 0, 302)
	for i := 0; i < 301; i++ {
		ftab = append(ftab, pclnFuncTabEntry{entryOff: uint32(i), funcOff: uint32(i + 1)})
	}
	ftab = append(ftab, pclnFuncTabEntry{entryOff: pclnFuncTabBucketSize, funcOff: 0})
	if _, err := buildPCLnFindFuncBuckets(ftab, pclnFuncTabBucketSize); err == nil {
		t.Fatal("expected subbucket overflow error")
	}
}

func TestEmitPCLnFindFuncBuckets(t *testing.T) {
	llvm.InitializeAllTargets()
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("pclntab-test")
	defer mod.Dispose()

	buckets := []pclnFindFuncBucket{
		{idx: 0, subbuckets: [16]uint8{0, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}},
		{idx: 3, subbuckets: [16]uint8{0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
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
