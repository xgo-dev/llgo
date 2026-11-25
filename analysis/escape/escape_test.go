package escape_test

import (
	"go/constant"
	"testing"

	"github.com/xgo-dev/llgo/analysis/escape"
	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

func TestAnalyzePackageIntraproceduralAndCrossPackage(t *testing.T) {
	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	pkg := ssa.NewPackage("example.com/esc", "esc", spec)
	i64 := types.Typ[types.Int64]
	ptrI64 := types.NewPointerWithKind(i64, types.GoPtr)
	gSlot := pkg.NewGlobal("esc.Sink", ptrI64)

	// 1. Local non-escaping alloc -> StackAllocs=1, HeapAllocs=0.
	localSig := types.NewSignature(nil, nil, types.NewTuple(types.NewParam(0, nil, "", i64)), false)
	localFn := pkg.NewFunction("esc.LocalOnly", localSig)
	lb := ssa.NewBuilder(localFn.NewBasicBlock("entry"))
	a := lb.Alloc(i64, false, true)
	lb.Store(a, lb.Const(i64, constant.MakeInt64(7)))
	v := lb.Load(a, i64)
	lb.Return(v)

	// 2. Returned alloc -> HeapAllocs=1; param returned -> ParamToReturn.
	retSig := types.NewSignature(
		nil,
		types.NewTuple(types.NewParam(0, nil, "p", ptrI64)),
		types.NewTuple(types.NewParam(0, nil, "", ptrI64)),
		false,
	)
	retFn := pkg.NewFunction("esc.ReturnAlloc", retSig)
	rb := ssa.NewBuilder(retFn.NewBasicBlock("entry"))
	ha := rb.Alloc(i64, true, true)
	rb.Store(ha, rb.Load(retFn.Params[0], i64))
	rb.Return(ha)

	// 3. Stores param into global -> ParamToHeap.
	sinkSig := types.NewSignature(nil, types.NewTuple(types.NewParam(0, nil, "p", ptrI64)), nil, false)
	sinkFn := pkg.NewFunction("esc.LeakToGlobal", sinkSig)
	sb := ssa.NewBuilder(sinkFn.NewBasicBlock("entry"))
	sb.Store(&ssa.GlobalRef{Global: gSlot}, sinkFn.Params[0])
	sb.Return()

	// 4. Calls imported cached summary (non-escaping param vs escaping param).
	callerSig := types.NewSignature(
		nil,
		types.NewTuple(types.NewParam(0, nil, "a", ptrI64), types.NewParam(0, nil, "b", ptrI64)),
		nil,
		false,
	)
	callerFn := pkg.NewFunction("esc.CallImported", callerSig)
	cb := ssa.NewBuilder(callerFn.NewBasicBlock("entry"))
	cb.CallDirect("dep.Inspect", types.Go, nil, callerFn.Params[0], callerFn.Params[1])
	cb.Return()

	imports := map[string]*exportdata.EscapeSummary{
		"dep": {
			PkgPath: "dep",
			Funcs: []exportdata.FuncEscape{
				{Name: "dep.Inspect", Params: []exportdata.ParamEscapeFlags{exportdata.ParamNoEscape, exportdata.ParamToHeap}},
			},
		},
	}

	sum := escape.AnalyzePackage(pkg, imports)
	byName := make(map[string]exportdata.FuncEscape)
	for _, f := range sum.Funcs {
		byName[f.Name] = f
	}
	if f := byName["esc.LocalOnly"]; f.StackAllocs != 1 || f.HeapAllocs != 0 {
		t.Fatalf("LocalOnly = %+v, want StackAllocs=1 HeapAllocs=0", f)
	}
	if f := byName["esc.ReturnAlloc"]; f.HeapAllocs != 1 || f.StackAllocs != 0 {
		t.Fatalf("ReturnAlloc = %+v, want HeapAllocs=1", f)
	}
	if f := byName["esc.LeakToGlobal"]; f.Params[0]&exportdata.ParamToHeap == 0 {
		t.Fatalf("LeakToGlobal param 0 should escape to heap: %+v", f)
	}
	if f := byName["esc.CallImported"]; f.Params[0] != exportdata.ParamNoEscape || f.Params[1]&exportdata.ParamToHeap == 0 {
		t.Fatalf("CallImported summary mismatch: %+v", f)
	}
}
