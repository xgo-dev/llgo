package meta

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestReflectMethodSiteRoundTrip(t *testing.T) {
	b := NewBuilder()
	caller, site := b.Sym("pkg.lookup"), b.Sym("pkg.lookup.reflectsite.0")
	b.AddReflectMethodSite(caller, site)
	pm, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	defer pm.Close()
	var data bytes.Buffer
	if _, err := pm.WriteTo(&data); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "site.meta")
	if err := os.WriteFile(path, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if local := opened.funcDemands(caller); len(local) != 1 ||
		local[0].Kind != DemandReflectMethodSite || local[0].Target != uint32(site) || local[0].Extra != 0 {
		t.Fatalf("callsite identity lost after mmap: %+v", local)
	}
	if got := opened.String(); got != "[ReflectMethodSite]\npkg.lookup:\n    pkg.lookup.reflectsite.0\n\n" {
		t.Fatalf("callsite demand format: %q", got)
	}
	for _, names := range []map[string][]string{nil, {"pkg.lookup.reflectsite.0": {"BeforeSave", "AfterSave"}}} {
		g, err := NewGlobalSummaryWithReflectNames([]*PackageMeta{opened}, names)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"reflect"}
		if names != nil {
			want = []string{"name:AfterSave", "name:BeforeSave"}
		}
		if got := reflectSiteDemands(t, g, "pkg.lookup"); !reflect.DeepEqual(got, want) {
			t.Fatalf("translated roundtrip demand: got %v, want %v", got, want)
		}
	}
	old := bytes.Clone(data.Bytes())
	binary.LittleEndian.PutUint32(old[4:8], 2)
	if _, err := newPackageMeta(old); err == nil {
		t.Fatal("accepted v2 metadata with unknown callsite-demand semantics")
	}
	invalid := bytes.Clone(data.Bytes())
	record := pm.demandOff + 4 + (pm.nsyms+1)*4
	binary.LittleEndian.PutUint32(invalid[record+4:], pm.nsyms)
	if _, err := newPackageMeta(invalid); err == nil || !strings.Contains(err.Error(), "invalid target symbol") {
		t.Fatalf("invalid callsite target accepted: %v", err)
	}
}

func TestReflectMethodSiteRefinementIsolation(t *testing.T) {
	b := NewBuilder()
	caller := b.Sym("pkg.lookup")
	b.AddReflectMethodSite(caller, b.Sym("pkg.lookup.site0"))
	b.AddReflectMethodSite(caller, b.Sym("pkg.lookup.site1"))
	generic := b.Sym("pkg.generic")
	b.AddReflectMethodSite(generic, b.Sym("pkg.lookup.site0"))
	b.MarkReflect(generic)
	b.MarkReflectMethodMetadata(generic)
	pm, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	defer pm.Close()
	for _, tc := range []struct {
		name  string
		names map[string][]string
		want  []string
	}{
		{"no proof", nil, []string{"reflect", "reflect"}},
		{"empty proofs", map[string][]string{"pkg.lookup.site0": {}, "pkg.lookup.site1": nil}, []string{"reflect", "reflect"}},
		{"only one site proved", map[string][]string{"pkg.lookup.site0": {"BeforeSave"}}, []string{"name:BeforeSave", "reflect"}},
		{"union from cloned calls", map[string][]string{"pkg.lookup.site0": {"BeforeSave", "AfterSave", "BeforeSave"}, "pkg.lookup.site1": {"BeforeCreate"}}, []string{"name:AfterSave", "name:BeforeCreate", "name:BeforeSave"}},
		{"empty literal is a name", map[string][]string{"pkg.lookup.site0": {""}, "pkg.lookup.site1": {"BeforeSave"}}, []string{"name:", "name:BeforeSave"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, err := NewGlobalSummaryWithReflectNames([]*PackageMeta{pm}, tc.names)
			if err != nil {
				t.Fatal(err)
			}
			if got := reflectSiteDemands(t, g, "pkg.lookup"); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("lookup demand: got %v, want %v", got, tc.want)
			}
			generic := reflectSiteDemands(t, g, "pkg.generic")
			if !slices.Contains(generic, "reflect") || !slices.Contains(generic, "metadata") {
				t.Fatalf("refining a site erased an independent demand: %v", generic)
			}
		})
	}
	plain, err := NewGlobalSummary([]*PackageMeta{pm})
	if err != nil {
		t.Fatal(err)
	}
	if got := reflectSiteDemands(t, plain, "pkg.lookup"); !reflect.DeepEqual(got, []string{"reflect", "reflect"}) {
		t.Fatalf("ordinary constructor must conservatively handle sites: %v", got)
	}
}

func TestReflectMethodSiteCrossPackageIdentityAndOwnedNames(t *testing.T) {
	var pkgs []*PackageMeta
	for _, caller := range []string{"pkg.a", "pkg.b"} {
		b := NewBuilder()
		if caller == "pkg.b" {
			b.Sym("different local symbol numbering")
		}
		b.AddReflectMethodSite(b.Sym(caller), b.Sym("shared.lookup.site"))
		pm, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		defer pm.Close()
		pkgs = append(pkgs, pm)
	}
	names := map[string][]string{"shared.lookup.site": {"BeforeSave", "AfterSave", "BeforeSave"}}
	g, err := NewGlobalSummaryWithReflectNames(pkgs, names)
	if err != nil {
		t.Fatal(err)
	}
	// The summary translates lazily; mutation by its caller must not change
	// either a previously queried function or one queried for the first time.
	want := []string{"name:AfterSave", "name:BeforeSave"}
	if got := reflectSiteDemands(t, g, "pkg.a"); !reflect.DeepEqual(got, want) {
		t.Fatalf("first copy: %v", got)
	}
	names["shared.lookup.site"][0] = "Mutated"
	delete(names, "shared.lookup.site")
	for _, caller := range []string{"pkg.a", "pkg.b"} {
		if got := reflectSiteDemands(t, g, caller); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s lost cloned-name union or changed through caller map: %v", caller, got)
		}
	}
}

func TestReflectMethodSites(t *testing.T) {
	if got := (*PackageMeta)(nil).ReflectMethodSites(); got != nil {
		t.Fatalf("nil package sites = %v", got)
	}
	for _, hasDemands := range []bool{false, true} {
		b := NewBuilder()
		fn := b.Sym("pkg.fn")
		if hasDemands {
			b.MarkReflect(fn)
		}
		pm, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		if got := pm.ReflectMethodSites(); got != nil {
			t.Fatalf("package without sites: %v", got)
		}
		pm.Close()
	}
	b := NewBuilder()
	a, z := b.Sym("lookup.a"), b.Sym("lookup.z")
	first, second := b.Sym("pkg.first"), b.Sym("pkg.second")
	b.AddReflectMethodSite(first, z)
	b.AddReflectMethodSite(first, a)
	b.AddReflectMethodSite(second, z)
	b.MarkReflect(second)
	pm, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	defer pm.Close()
	got := pm.ReflectMethodSites()
	want := []string{"lookup.a", "lookup.z"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sites = %v, want %v", got, want)
	}
	// Emulate backing-storage invalidation without dereferencing an unmapped
	// region: returned strings must not borrow the metadata's wire bytes.
	for i := pm.strOff; i < pm.symOff; i++ {
		pm.raw[i] = 'x'
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("site names alias package backing storage: %v", got)
	}
}

func reflectSiteDemands(t *testing.T, g *GlobalSummary, caller string) []string {
	t.Helper()
	sym, ok := g.LookupSymbol(caller)
	if !ok || !g.HasFacts(sym) {
		t.Fatalf("missing caller facts: %s", caller)
	}
	var out []string
	for _, demand := range g.FuncDemands(sym) {
		switch demand.Kind {
		case DemandReflectMethod:
			out = append(out, "reflect")
		case DemandReflectMethodMetadata:
			out = append(out, "metadata")
		case DemandNamedMethod:
			out = append(out, "name:"+g.Name(demand.MethodName))
		default:
			t.Fatalf("untranslated/unexpected callsite demand: %+v", demand)
		}
	}
	slices.Sort(out)
	return out
}
