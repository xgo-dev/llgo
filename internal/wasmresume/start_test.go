package wasmresume

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestStartEntryRejectsIncompatibleDeclaration(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("incompatible-start")
	defer mod.Dispose()
	targetData := llvm.NewTargetData("e-m:e-p:32:32-i64:64-n32:64-S128")
	defer targetData.Dispose()

	voidFn := llvm.FunctionType(ctx.VoidType(), nil, false)
	fn := llvm.AddFunction(mod, "leaf", voidFn)
	markFunction(ctx, fn)
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	builder.CreateRetVoid()
	llvm.AddFunction(mod, startEntryPrefix+fn.Name(), voidFn)

	if _, err := lowerPrototype(mod, targetData); err == nil ||
		!strings.Contains(err.Error(), "incompatible resumable start entry") {
		t.Fatalf("lowerPrototype error = %v", err)
	}
}
