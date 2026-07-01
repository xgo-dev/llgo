//go:build !llgo
// +build !llgo

package build

import (
	"github.com/goplus/llgo/internal/pclntab"
	llvm "github.com/xgo-dev/llvm"
)

// emitPCLnFindFuncBuckets is the LLVM materialization layer for the Go-style
// findfunctab data produced by internal/pclntab. Keep the algorithm in that
// package; this function should only translate buckets into IR constants.
func emitPCLnFindFuncBuckets(mod llvm.Module, symbol string, buckets []pclntab.FindFuncBucket) llvm.Value {
	ctx := mod.Context()
	i8Type := ctx.Int8Type()
	i32Type := ctx.Int32Type()
	subType := llvm.ArrayType(i8Type, pclntab.FindFuncSubbucket)
	bucketType := ctx.StructType([]llvm.Type{i32Type, subType}, false)
	arrayType := llvm.ArrayType(bucketType, len(buckets))
	values := make([]llvm.Value, 0, len(buckets))
	for _, bucket := range buckets {
		subs := make([]llvm.Value, 0, len(bucket.Subbuckets))
		for _, sub := range bucket.Subbuckets {
			subs = append(subs, llvm.ConstInt(i8Type, uint64(sub), false))
		}
		values = append(values, llvm.ConstNamedStruct(bucketType, []llvm.Value{
			llvm.ConstInt(i32Type, uint64(bucket.Idx), false),
			llvm.ConstArray(i8Type, subs),
		}))
	}
	global := llvm.AddGlobal(mod, arrayType, symbol)
	global.SetInitializer(llvm.ConstArray(bucketType, values))
	global.SetGlobalConstant(true)
	global.SetUnnamedAddr(true)
	global.SetAlignment(4)
	return global
}
