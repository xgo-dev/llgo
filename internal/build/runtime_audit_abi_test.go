package build

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
)

func TestRuntimeABIBlockersFollowLayouts(t *testing.T) {
	for _, tc := range []struct {
		name                string
		size, align, offset uint64
		fields              int
		funcSize            uint64
		wantType, wantFunc  bool
	}{
		{"matching-layouts", 16, 8, 8, 2, 8, false, false},
		{"descriptor-size", 24, 8, 8, 2, 8, true, false},
		{"descriptor-align", 16, 4, 8, 2, 8, true, false},
		{"descriptor-offset", 16, 8, 4, 2, 8, true, false},
		{"descriptor-fields", 16, 8, 8, 1, 8, true, false},
		{"funcval-size", 16, 8, 8, 2, 16, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			abi := &RuntimeABI{
				GoType: RuntimeTypeLayout{Size: 16, Align: 8, Fields: []RuntimeFieldLayout{
					{Name: "first", Offset: 0, Size: 8}, {Name: "second", Offset: 8, Size: 8},
				}},
				LLGoType: RuntimeTypeLayout{Size: tc.size, Align: tc.align, Fields: []RuntimeFieldLayout{
					{Name: "first", Offset: 0, Size: 8}, {Name: "second", Offset: tc.offset, Size: 8},
				}},
				GoFuncValueBytes: 8, LLGoFuncValueBytes: tc.funcSize,
			}
			abi.LLGoType.Fields = abi.LLGoType.Fields[:tc.fields]
			blockers := runtimeABIBlockers(abi)
			for _, check := range []struct {
				prefix string
				want   bool
			}{
				{"Go and LLGo type descriptor", tc.wantType},
				{"Go and LLGo function-value sizes", tc.wantFunc},
			} {
				if got := slices.ContainsFunc(blockers, func(b string) bool { return strings.HasPrefix(b, check.prefix) }); got != check.want {
					t.Fatalf("blocker %q present = %v, want %v: %v", check.prefix, got, check.want, blockers)
				}
			}
			for _, gate := range []string{"Go ABIInternal qualification", "Go g0", "Go stack growth"} {
				if !slices.ContainsFunc(blockers, func(b string) bool { return strings.Contains(b, gate) }) {
					t.Fatalf("matching layout must not waive compatibility gate %q: %v", gate, blockers)
				}
			}
		})
	}
}

func TestRuntimeMetadataRejectsInvalidBoundary(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(env.LLGoRuntimeDir(), "abi/type.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, source, want string }{
		{"new-dependency", strings.Replace(string(src), "type Type struct {", "type Type struct {\n Future FutureMetadata", 1), "undefined: FutureMetadata"},
		{"malformed-source", "package abi\nfunc (", "expected"},
		{"non-struct-descriptor", "package abi\nimport \"unsafe\"\ntype Type unsafe.Pointer", "Type is not a struct"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "type.go")
			if err := os.WriteFile(path, []byte(tc.source), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := runtimeMetadataType(path, "Type"); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid metadata error = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestRuntimeAuditMemory64Layout(t *testing.T) {
	report, err := AuditRuntimeSources(RuntimeAuditConfig{
		RuntimeDir: env.LLGoRuntimeDir(), GOOS: "js", GOARCH: "wasm",
		Tags: []string{"llgo.wasm.gc.linear", "llgo.wasm.emscripten", "llgo.wasm.emscripten.memory64"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Missing) != 0 || len(report.UnresolvedBoundaries) != 0 || len(report.DynamicCompilerUses) != 0 {
		t.Fatalf("memory64 runtime boundaries: missing=%v, external=%v, dynamic=%v", report.Missing, report.UnresolvedBoundaries, report.DynamicCompilerUses)
	}
	// gc's Wasm port uses 64-bit words; Memory64 selects that physical width
	// for LLGo too, while the descriptor and function-value layouts still differ.
	if got := report.ABI; got.LLGoTarget != "wasm64-unknown-emscripten" || got.LLGoType.Size != 72 || got.GoType.Size != 48 || got.LLGoFuncValueBytes != 16 {
		t.Fatalf("memory64 profile lost: %+v", got)
	}
}
