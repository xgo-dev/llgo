package ssa

import (
	"strings"
	"testing"
)

func TestReflectSitesKeepIndependentDemands(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	prog.EnableDeadcodeDrop(true)
	pkg := prog.NewPackageEx("pkg", "example.com/pkg", true)
	first := pkg.recordReflectMethodSite("example.com/pkg.lookup")
	second := pkg.recordReflectMethodSite("example.com/pkg.lookup")
	if first == "" || first == second {
		t.Fatalf("lookup identities must be distinct: %q, %q", first, second)
	}
	clone := prog.NewPackageEx("other", "example.com/other", true)
	clone.recordReflectMethodSite("example.com/other.unrelated")
	if got := clone.recordReflectMethodSite("example.com/pkg.lookup"); got != first {
		t.Fatalf("cloned body identity depends on unrelated compilation order: %q, want %q", got, first)
	}
	pkg.MarkReflectMethod("example.com/pkg.lookup")
	pm, err := pkg.metaBuilder.Build()
	if err != nil {
		t.Fatal(err)
	}
	defer pm.Close()
	if got := pm.ReflectMethodSites(); len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("callsite identities lost in metadata: %v", got)
	}
	if !strings.Contains(pm.String(), "[Reflect]\n    example.com/pkg.lookup\n") {
		t.Fatalf("callsite refinement erased independent generic demand:\n%s", pm.String())
	}
}
