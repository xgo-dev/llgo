package deadcode_test

import (
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/analysis/deadcode"
	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

func TestDeadcodeEliminationWithSnapshots(t *testing.T) {
	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	pkg := ssa.NewPackage("example.com/dce", "dce", spec)
	i64 := types.Typ[types.Int64]
	voidSig := types.NewSignature(nil, nil, nil, false)

	liveG := pkg.NewGlobal("dce.LiveG", i64)
	_ = pkg.NewGlobal("dce.DeadG", i64)

	mainFn := pkg.NewFunction("dce.main", voidSig)
	mb := ssa.NewBuilder(mainFn.NewBasicBlock("entry"))
	_ = mb.Load(&ssa.GlobalRef{Global: liveG}, i64)
	mb.CallDirect("dce.used", types.Go, nil)
	mb.Return()

	usedFn := pkg.NewFunction("dce.used", voidSig)
	ssa.NewBuilder(usedFn.NewBasicBlock("entry")).Return()

	exportedFn := pkg.NewFunction("dce.exported", voidSig)
	exportedFn.ExportName = "c_exported"
	ssa.NewBuilder(exportedFn.NewBasicBlock("entry")).Return()

	deadFn := pkg.NewFunction("dce.dead", voidSig)
	ssa.NewBuilder(deadFn.NewBasicBlock("entry")).Return()

	snap := &exportdata.LinkSnapshot{
		PkgPath:      "example.com/dce",
		MethodByName: []string{"String"},
	}

	res := deadcode.Analyze([]*ssa.Package{pkg}, nil, []*exportdata.LinkSnapshot{snap}, []string{"dce.main"})
	if !reflect.DeepEqual(res.ReachableFuncs, []string{"dce.exported", "dce.main", "dce.used"}) {
		t.Fatalf("ReachableFuncs = %v", res.ReachableFuncs)
	}
	if !reflect.DeepEqual(res.DeadFuncs, []string{"dce.dead"}) {
		t.Fatalf("DeadFuncs = %v", res.DeadFuncs)
	}
	if !reflect.DeepEqual(res.ReachableGlobals, []string{"dce.LiveG"}) || !reflect.DeepEqual(res.DeadGlobals, []string{"dce.DeadG"}) {
		t.Fatalf("Globals reachability mismatch: live=%v dead=%v", res.ReachableGlobals, res.DeadGlobals)
	}
	if !reflect.DeepEqual(res.ReachableMethods, []string{"String"}) {
		t.Fatalf("ReachableMethods = %v", res.ReachableMethods)
	}
}
