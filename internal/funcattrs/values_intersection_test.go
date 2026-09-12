package funcattrs

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestDirectRangeIntersectionOptimization(t *testing.T) {
	attrs, sig, err := parseTest(t, "//llgo:attr result(0) range(-4,20) nonnegative\nfunc F() int32 { return 0 }")
	if err != nil {
		t.Fatal(err)
	}
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	m := ctx.NewModule("intersection")
	defer m.Dispose()
	ft := llvm.FunctionType(ctx.Int32Type(), nil, false)
	f := llvm.AddFunction(m, "F", ft)
	if err = Apply(ctx, f, sig, attrs, 0, 64); err != nil {
		t.Fatal(err)
	}
	caller := llvm.AddFunction(m, "caller", llvm.FunctionType(ctx.Int1Type(), nil, false))
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(caller, "entry"))
	value := llvm.CreateCall(b, ft, f, nil)
	lower := b.CreateICmp(llvm.IntSGE, value, llvm.ConstInt(ctx.Int32Type(), 0, false), "lower")
	upper := b.CreateICmp(llvm.IntSLT, value, llvm.ConstInt(ctx.Int32Type(), 20, false), "upper")
	b.CreateRet(b.CreateAnd(lower, upper, "within"))
	if err = MaterializeValueContracts(m); err != nil {
		t.Fatal(err)
	}
	if err = llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	options := llvm.NewPassBuilderOptions()
	defer options.Dispose()
	if err = m.RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
		t.Fatal(err)
	}
	// A native range attribute can hold only one range. Both source facts
	// must reach optimization even if writing one native attribute replaced
	// the other; neither fact alone proves this entire condition.
	result := m.NamedFunction("caller").String()
	if !strings.Contains(result, "ret i1 true") || !strings.Contains(result, "call") {
		t.Fatalf("intersection did not simplify while retaining the external call:\n%s", result)
	}
}
