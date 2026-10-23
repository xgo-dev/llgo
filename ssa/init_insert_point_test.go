package ssa

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestAfterInitInsertPoint(t *testing.T) {
	for _, test := range []struct {
		name string
		args bool
		init bool
	}{
		{name: "dependency.init", init: true},
		{name: "consumer.init$hasPatch", init: true},
		{name: "consumer.init#1"},
		{name: "consumer.init$hasPatchExtra"},
		{name: "consumer.init$hasPatch#1"},
		{name: "consumer.initialize"},
		{name: "dependency.init", args: true},
		{name: "consumer.init$hasPatch", args: true},
	} {
		name := test.name
		if test.args {
			name += "/with_argument"
		}
		t.Run(name, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			mod := ctx.NewModule("initialization")
			defer mod.Dispose()
			builder := ctx.NewBuilder()
			defer builder.Dispose()

			fnType := llvm.FunctionType(ctx.VoidType(), nil, false)
			fn := llvm.AddFunction(mod, "package.init", fnType)
			block := ctx.AddBasicBlock(fn, "entry")
			builder.SetInsertPointAtEnd(block)
			guard := llvm.AddGlobal(mod, ctx.Int1Type(), "guard")
			guard.SetInitializer(llvm.ConstNull(ctx.Int1Type()))
			builder.CreateStore(llvm.ConstInt(ctx.Int1Type(), 1, false), guard)

			var types []llvm.Type
			var args []llvm.Value
			if test.args {
				types = []llvm.Type{ctx.Int32Type()}
				args = []llvm.Value{llvm.ConstInt(ctx.Int32Type(), 7, false)}
			}
			calleeType := llvm.FunctionType(ctx.VoidType(), types, false)
			callee := llvm.AddFunction(mod, test.name, calleeType)
			call := builder.CreateCall(calleeType, callee, args, "")
			ret := builder.CreateRetVoid()
			want := call
			if test.init {
				want = ret
			}
			if got := instrAfterInit(block); got != want {
				t.Fatalf("insertion point = %s, want %s", got.String(), want.String())
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
