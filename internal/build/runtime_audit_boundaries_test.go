package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeAuditExternalProviders(t *testing.T) {
	a := runtimeAuditFixture(t)
	report := RuntimeOwnership{Symbols: []RuntimeSymbol{
		{Name: "runtime.local", Linkname: "runtime.implementation", Owner: "external-boundary"},
		{Name: "runtime.implementation", Kind: "func", Body: true, Source: "implementation.go:1"},
		{Name: "runtime.native", Linkname: "C.getenv", Owner: "external-boundary"},
		{Name: "runtime.newNative", Linkname: "C.llgo_new_native_hook", Owner: "external-boundary"},
		{Name: "runtime.missing", Linkname: "runtime.newGoHook", Owner: "external-boundary"},
		{Name: "runtime.coverage", Linkname: "internal/coverage/rtcov.AddMeta", Owner: "external-boundary"},
		{Name: "runtime.coverageAlias", Linkname: "internal/coverage/rtcov.AddMeta", Owner: "external-boundary"},
	}}
	dir := filepath.Join(a.goroot, "src/internal/coverage/rtcov")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rtcov.go"), []byte("package rtcov\nfunc AddMeta() {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := a.externalBoundaries(&report); err != nil {
		t.Fatal(err)
	}
	if len(report.ExternalBoundaries) != 4 || len(report.UnresolvedBoundaries) != 2 {
		t.Fatalf("qualified: %+v; unresolved: %+v", report.ExternalBoundaries, report.UnresolvedBoundaries)
	}
	if got := report.ExternalBoundaries[2]; got.Kind != "go" || got.Provider != "internal/coverage/rtcov.AddMeta" ||
		len(got.Sources) != 1 || !strings.HasPrefix(got.Sources[0], "$GOROOT/src/internal/coverage/rtcov/") {
		t.Fatalf("lost selected GOROOT provider: %+v", got)
	}
	if report.UnresolvedBoundaries[0].Symbol != "C.llgo_new_native_hook" || report.UnresolvedBoundaries[1].Symbol != "runtime.newGoHook" {
		t.Fatalf("new hooks were not rejected: %+v", report.UnresolvedBoundaries)
	}
}

func TestRuntimeNativeOwnersAreUnique(t *testing.T) {
	providers, err := runtimeNativeProviders(runtimeNativeOwners)
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range runtimeNativeOwners {
		if owner.owner == "" {
			t.Fatal("missing native provider")
		}
		for _, symbol := range strings.Fields(owner.symbols) {
			if got := providers[symbol]; got != owner {
				t.Fatalf("provider for %s = %+v, want %+v", symbol, got, owner)
			}
		}
	}
}

func TestRuntimeNativeProvidersRejectDuplicates(t *testing.T) {
	for _, symbol := range []string{"C.shared", "stdcall.Shared"} {
		for _, sameEntry := range []bool{false, true} {
			owners := []runtimeNativeOwner{
				{symbol + " C.first", "first adapter", "first.c"},
				{"C.other\n" + symbol, "second adapter", "second.c"},
			}
			if sameEntry {
				owners = []runtimeNativeOwner{{symbol + "\n" + symbol, "same adapter", "same.c"}}
			}
			providers, err := runtimeNativeProviders(owners)
			if err == nil || !strings.Contains(err.Error(), symbol) || !strings.Contains(err.Error(), "multiple entries") {
				t.Fatalf("duplicate %s (same entry: %v): providers=%v, error=%v", symbol, sameEntry, providers, err)
			}
			if providers != nil {
				t.Fatalf("returned partial providers after duplicate %s: %v", symbol, providers)
			}
		}
	}
}

func TestRuntimeAuditRejectsBrokenProviders(t *testing.T) {
	for _, tc := range []struct{ name, target, source, want string }{
		{"missing-native-source", "C.llgo_debugtrap", "", "debugtrap.c"},
		{"malformed-go-source", "provider.Hook", "package provider\nfunc (", "expected"},
		{"conflicting-go-linknames", "provider.Hook", "package provider\nimport _ \"unsafe\"\n//go:linkname Hook one\n//go:linkname Hook two\nfunc Hook() {}", "conflicting linkname"},
		{"bodyless-go-provider", "provider.Hook", "package provider\nfunc Hook()", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := runtimeAuditFixture(t)
			if tc.source != "" {
				dir := filepath.Join(a.goroot, "src/provider")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "provider.go"), []byte(tc.source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			report := RuntimeOwnership{Symbols: []RuntimeSymbol{{
				Name: "runtime.hook", Linkname: tc.target, Owner: "external-boundary",
			}}}
			err := a.externalBoundaries(&report)
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error = %v, want %s", err, tc.want)
				}
			} else if err != nil || len(report.UnresolvedBoundaries) != 1 || report.UnresolvedBoundaries[0].Symbol != tc.target {
				t.Fatalf("bodyless provider was not rejected: %v, %+v", err, report.UnresolvedBoundaries)
			}
			if len(report.ExternalBoundaries) != 0 {
				t.Fatalf("qualified a broken provider: %+v", report.ExternalBoundaries)
			}
		})
	}
}
