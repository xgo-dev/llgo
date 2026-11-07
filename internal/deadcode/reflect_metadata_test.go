package deadcode

import (
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/internal/meta"
)

func TestReflectMethodMetadataPreservesChildVisibility(t *testing.T) {
	for _, reachable := range []bool{false, true} {
		b := meta.NewBuilder()
		main, lookup := b.Sym("main.main"), b.Sym("pkg.lookup")
		owner, child, grandchild := b.Sym("_llgo_T"), b.Sym("_llgo_U"), b.Sym("_llgo_V")
		makeSig, nextSig, callSig := b.Sym("_llgo_func$Make"), b.Sym("_llgo_func$Next"), b.Sym("_llgo_func$Call")
		b.AddOrdinaryEdge(main, owner)
		b.AddIfaceUse(main, owner)
		if reachable {
			b.AddOrdinaryEdge(main, lookup)
		}
		b.MarkReflectMethodMetadata(lookup)
		// TypeOf(T{}).Method(0).Type.Out(0) exposes U, then U.Next's
		// parameter/result exposes V. Neither Make nor Next must be callable.
		b.AddMethodSlot(owner, "Make", makeSig, b.Sym("T.Make$ifn"), b.Sym("T.Make"))
		b.AddTypeChild(makeSig, child)
		b.AddOrdinaryEdge(makeSig, child)
		b.AddMethodSlot(child, "Next", nextSig, b.Sym("U.Next$ifn"), b.Sym("U.Next"))
		b.AddTypeChild(nextSig, grandchild)
		b.AddOrdinaryEdge(nextSig, grandchild)
		b.AddMethodSlot(grandchild, "Call", callSig, b.Sym("V.Call$ifn"), b.Sym("V.Call"))
		b.AddNamedMethodUse(main, "Call")
		pm, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		summary := newSummary(t, pm)
		analysis := AnalyzeProgram(summary, []string{"main.main"})
		want := map[string][]int{}
		if reachable {
			want["_llgo_V"] = []int{0}
		}
		if !reflect.DeepEqual(analysis.LiveSlots, want) {
			t.Fatalf("reachable=%v: live slots = %v, want %v", reachable, analysis.LiveSlots, want)
		}
		for _, typ := range []string{"_llgo_U", "_llgo_V"} {
			if got := analysis.CanDropMethodMetadata(typ); got == reachable {
				t.Errorf("reachable=%v: CanDropMethodMetadata(%q) = %v", reachable, typ, got)
			}
		}
		pm.Close()
	}
}
