package inline_test

import (
	"go/constant"
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/analysis/inline"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

func TestAnalyzePackageInlineBudgetsAndDisqualifiers(t *testing.T) {
	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	pkg := ssa.NewPackage("example.com/inl", "inl", spec)
	i64 := types.Typ[types.Int64]
	addSig := types.NewSignature(
		nil,
		types.NewTuple(types.NewParam(0, nil, "a", i64), types.NewParam(0, nil, "b", i64)),
		types.NewTuple(types.NewParam(0, nil, "", i64)),
		false,
	)

	// Small leaf function -> inlineable.
	addFn := pkg.NewFunction("inl.Add", addSig)
	ab := ssa.NewBuilder(addFn.NewBasicBlock("entry"))
	ab.Return(ab.BinOp(token.ADD, i64, addFn.Params[0], addFn.Params[1]))

	// NoInline -> rejected.
	noInl := pkg.NewFunction("inl.NoInl", addSig)
	noInl.NoInline = true
	nb := ssa.NewBuilder(noInl.NewBasicBlock("entry"))
	nb.Return(nb.Const(i64, constant.MakeInt64(0)))

	// Recursive -> rejected.
	recFn := pkg.NewFunction("inl.Rec", addSig)
	rb := ssa.NewBuilder(recFn.NewBasicBlock("entry"))
	rb.Return(rb.CallDirect("inl.Rec", types.Go, i64, recFn.Params[0], recFn.Params[1]))

	// Defer -> rejected.
	voidSig := types.NewSignature(nil, nil, nil, false)
	defFn := pkg.NewFunction("inl.WithDefer", voidSig)
	db := ssa.NewBuilder(defFn.NewBasicBlock("entry"))
	db.Defer(db.MakeClosure(voidSig, "inl.cleanup", nil), false)
	db.RunDefers()
	db.Return()

	// Over budget -> rejected.
	bigFn := pkg.NewFunction("inl.Big", voidSig)
	bb := ssa.NewBuilder(bigFn.NewBasicBlock("entry"))
	for i := 0; i < 6; i++ {
		bb.CallDirect("inl.ext", types.Go, nil)
	}
	bb.Return()

	sum := inline.AnalyzePackage(pkg, inline.DefaultBudget)
	byName := make(map[string]struct {
		ok     bool
		reason string
	})
	for _, f := range sum.Funcs {
		byName[f.Name] = struct {
			ok     bool
			reason string
		}{f.Inlineable, f.Reason}
	}
	if !byName["inl.Add"].ok {
		t.Fatalf("Add should be inlineable: %+v", byName["inl.Add"])
	}
	if byName["inl.NoInl"].ok || byName["inl.NoInl"].reason != "noinline" {
		t.Fatalf("NoInl = %+v", byName["inl.NoInl"])
	}
	if byName["inl.Rec"].ok || byName["inl.Rec"].reason != "recursive" {
		t.Fatalf("Rec = %+v", byName["inl.Rec"])
	}
	if byName["inl.WithDefer"].ok || byName["inl.WithDefer"].reason != "defer" {
		t.Fatalf("WithDefer = %+v", byName["inl.WithDefer"])
	}
	if byName["inl.Big"].ok || byName["inl.Big"].reason != "over-budget" {
		t.Fatalf("Big = %+v", byName["inl.Big"])
	}
}
