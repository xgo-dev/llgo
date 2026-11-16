package meta_test

import (
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/internal/deadcode"
	"github.com/xgo-dev/llgo/internal/meta"
)

func TestReflectMethodSiteKeepsUnresolvedCallsConservative(t *testing.T) {
	for _, tc := range []struct {
		name    string
		names   map[string][]string
		generic bool
		want    []int
	}{
		{"all sites resolved", map[string][]string{"site.0": {"BeforeSave"}, "site.1": {"AfterSave"}}, false, []int{0, 1}},
		{"one site unresolved", map[string][]string{"site.0": {"BeforeSave"}}, false, []int{0, 1, 2}},
		{"empty set unresolved", map[string][]string{"site.0": {"BeforeSave"}, "site.1": {}}, false, []int{0, 1, 2}},
		{"independent generic demand", map[string][]string{"site.0": {"BeforeSave"}, "site.1": {"AfterSave"}}, true, []int{0, 1, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := meta.NewBuilder()
			main, typ := b.Sym("main.main"), b.Sym("_llgo_Model")
			b.AddOrdinaryEdge(main, typ)
			b.AddIfaceUse(main, typ)
			b.AddReflectMethodSite(main, b.Sym("site.0"))
			b.AddReflectMethodSite(main, b.Sym("site.1"))
			if tc.generic {
				b.MarkReflect(main)
			}
			for _, name := range []string{"BeforeSave", "AfterSave", "Unrelated"} {
				b.AddMethodSlot(typ, name, b.Sym("_llgo_func$"+name), b.Sym(name+"$ifn"), b.Sym(name+"$tfn"))
			}
			pm, err := b.Build()
			if err != nil {
				t.Fatal(err)
			}
			defer pm.Close()
			g, err := meta.NewGlobalSummaryWithReflectNames([]*meta.PackageMeta{pm}, tc.names)
			if err != nil {
				t.Fatal(err)
			}
			got := deadcode.Analyze(g, []string{"main.main"})["_llgo_Model"]
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("live slots = %v, want %v", got, tc.want)
			}
		})
	}
}
