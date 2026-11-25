package callgraph_test

import (
	"go/constant"
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/analysis/callgraph"
	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

func TestAnalyzeLiveAndCachedPackages(t *testing.T) {
	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	pkg := ssa.NewPackage("example.com/app", "app", spec)
	voidSig := types.NewSignature(nil, nil, nil, false)
	i64 := types.Typ[types.Int64]

	mainFn := pkg.NewFunction("example.com/app.main", voidSig)
	b := ssa.NewBuilder(mainFn.NewBasicBlock("entry"))
	b.CallDirect("example.com/dep.Helper", types.Go, nil)
	clo := b.MakeClosure(voidSig, "example.com/app.cb", nil)
	b.CallClosure(clo, nil)
	iface := types.NewInterfaceType([]*types.Func{types.NewFunc(0, nil, "Run", voidSig)}, nil)
	b.CallInterface(b.Const(iface, nil), "Run", 0, nil)
	b.Return()

	cbFn := pkg.NewFunction("example.com/app.cb", voidSig)
	ssa.NewBuilder(cbFn.NewBasicBlock("entry")).Return()

	deadFn := pkg.NewFunction("example.com/app.dead", voidSig)
	ssa.NewBuilder(deadFn.NewBasicBlock("entry")).Return()

	// Cached dependency package metadata (never parsed into ssa.Package!).
	depMeta := &exportdata.PackageMeta{
		Summary: ssa.PackageSummary{
			Path:         "example.com/dep",
			DefinedFuncs: []string{"example.com/dep.Helper", "example.com/dep.Runner.Run", "example.com/dep.Unused"},
		},
		CallEdges: []exportdata.CallEdge{
			{Caller: "example.com/dep.Helper", Callee: "example.com/dep.Leaf", Kind: uint8(callgraph.EdgeDirect)},
		},
		LiveTypes: []string{"example.com/dep.Runner"},
	}

	g := callgraph.Analyze([]*ssa.Package{pkg}, []*exportdata.PackageMeta{depMeta}, []string{"example.com/app.main"})
	for _, want := range []string{
		"example.com/app.main",
		"example.com/app.cb",
		"example.com/dep.Helper",
		"example.com/dep.Leaf",
		"example.com/dep.Runner.Run",
	} {
		if !g.Reachable[want] {
			t.Fatalf("expected %s to be reachable", want)
		}
	}
	if g.Reachable["example.com/app.dead"] || g.Reachable["example.com/dep.Unused"] {
		t.Fatalf("dead functions marked reachable: %+v", g.Reachable)
	}
	if !g.InterfaceCalls["Run"] || !g.LiveTypes["example.com/dep.Runner"] {
		t.Fatalf("missing interface/live-type facts: %+v", g)
	}
	_ = constant.MakeInt64(0)
	_ = i64
	edges := callgraph.ExtractEdges(pkg)
	if len(edges) != 3 || !reflect.DeepEqual(edges, callgraph.ExtractEdges(pkg)) {
		t.Fatalf("ExtractEdges non-deterministic or wrong count: %+v", edges)
	}
}
