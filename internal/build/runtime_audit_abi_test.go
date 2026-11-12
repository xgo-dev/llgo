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

func TestRuntimeAuditWasmLayouts(t *testing.T) {
	for _, tc := range []struct {
		profile, goos, triple string
		tags                  []string
		pointerBytes          uint64
	}{
		{"j32", "js", "wasm32-unknown-emscripten", []string{"llgo.wasm.emscripten"}, 4},
		{"j64", "js", "wasm64-unknown-emscripten", []string{"llgo.wasm.emscripten", "llgo.wasm.emscripten.memory64"}, 8},
		{"w32", "wasip1", "wasm32-unknown-wasip1", []string{"llgo.wasi_threads", "llgo.wasm.workers"}, 4},
	} {
		t.Run(tc.profile, func(t *testing.T) {
			report, err := AuditRuntimeSources(RuntimeAuditConfig{
				RuntimeDir: env.LLGoRuntimeDir(), GOOS: tc.goos, GOARCH: "wasm",
				Tags: append([]string{"llgo.wasm.gc.linear"}, tc.tags...),
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Missing) != 0 || len(report.UnresolvedBoundaries) != 0 || len(report.DynamicCompilerUses) != 0 {
				t.Fatalf("runtime boundaries: missing=%v, external=%v, dynamic=%v", report.Missing, report.UnresolvedBoundaries, report.DynamicCompilerUses)
			}
			// All hosted profiles use Go64 storage. Physical C addresses are
			// narrower on J32/W32; matching those widths would hide this ABI split.
			got := report.ABI
			if got.WasmProfile != tc.profile || got.LLGoTarget != tc.triple || got.GoWordBytes != 8 || got.NativePointerBytes != tc.pointerBytes ||
				got.LLGoType.Size != 64 || got.GoType.Size != 48 || got.LLGoFuncValueBytes != 8 || got.GoFuncValueBytes != 8 {
				t.Fatalf("profile lost: %+v", got)
			}
			if got.GoRuntimeASMReady || len(got.Blockers) == 0 {
				t.Fatalf("matching function sizes must not waive the remaining runtime ABI boundaries: %+v", got)
			}
		})
	}
}
