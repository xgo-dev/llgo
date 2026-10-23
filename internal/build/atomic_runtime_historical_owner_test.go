package build

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
)

type historicalAtomicProfile struct {
	version, path string
	entries       int
}

func historicalAtomicRows(t *testing.T, entries int) []runtimeAtomicRow {
	t.Helper()
	var rows []runtimeAtomicRow
	for _, row := range runtimeAtomicRows() {
		if entries < 49 && row.name == "Xchg8" {
			continue
		}
		if entries == 42 && (row.family == "and" || row.family == "or") {
			continue
		}
		rows = append(rows, row)
	}
	if len(rows) != entries {
		t.Fatal("the independent historical atomic inventory changed", entries, len(rows))
	}
	return rows
}

func historicalAtomicFixture(t *testing.T, profile historicalAtomicProfile, mutation string) (string, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "src", filepath.FromSlash(profile.path))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	var assembly, declarations strings.Builder
	declarations.WriteString("package atomic\nimport \"unsafe\"\n")
	for _, row := range historicalAtomicRows(t, profile.entries) {
		name, declaration := row.name, row.declaration()
		if row.name == "Load8" {
			switch mutation {
			case "missing-text":
				name = ""
			case "unknown-same-count":
				name = "FutureAtomic"
			case "wrong-signature":
				declaration = strings.ReplaceAll(declaration, "uint8", "uint64")
			case "existing-sdk-body":
				declaration += " { return *ptr }"
			}
		}
		fmt.Fprintln(&declarations, declaration)
		if name == "Or32" || name == "And32" {
			// Go1.22/1.23 use MOVD for a four-byte return at +16. Ownership
			// must replace the whole reviewed family, not authorize that spill.
			fmt.Fprintf(&assembly, "TEXT ·%s(SB),4,$0-20\nMOVD R2,ret+16(FP)\nRET\n", name)
		} else if name != "" {
			fmt.Fprintf(&assembly, "TEXT ·%s(SB),4,$0\nRET\n", name)
		}
	}
	assemblyLanguageTestWrite(t, dir, "atomic_arm64.s", assembly.String())
	assemblyLanguageTestWrite(t, dir, "stubs.go", declarations.String())
	assemblyLanguageTestWrite(t, dir, "atomic_arm64.go", "package atomic\n")
	assemblyLanguageTestWrite(t, dir, "existing.go", "package atomic\nfunc ExistingSDKBody() uint32 { return 7 }\n")
	assemblyLanguageTestWrite(t, dir, "other_arm64.s", "TEXT ·Other(SB),4,$0\nRET\n")
	return root, dir
}

func TestRuntimeAtomicARM64HistoricalSDKOwnership(t *testing.T) {
	for _, profile := range []historicalAtomicProfile{
		{"go1.20.14", "runtime/internal/atomic", 42},
		{"go1.21.13", "runtime/internal/atomic", 42},
		{"go1.22.2", "runtime/internal/atomic", 48},
		{"go1.23.4", "internal/runtime/atomic", 48},
		{"go1.24.0", "internal/runtime/atomic", 49},
	} {
		t.Run(profile.version, func(t *testing.T) {
			root, dir := historicalAtomicFixture(t, profile, "")
			context := sourcePatchBuildContext{goos: "linux", goarch: "arm64", goversion: profile.version}
			overlay, files, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), root, context)
			if err != nil {
				t.Fatal(err)
			}
			if string(overlay[filepath.Join(dir, "atomic_arm64.s")]) != "// replaced by LLGo source patch\n" ||
				len(files[profile.path]) == 0 {
				t.Fatal("the complete selected historical ARM64 family has no Go body owner")
			}
			var sources [][]byte
			for name, source := range overlay {
				if filepath.Dir(name) == dir && strings.HasPrefix(filepath.Base(name), "z_llgo_patch_") {
					sources = append(sources, source)
				}
			}
			bodies, err := atomicSourcePatchFunctions(sources)
			if err != nil {
				t.Fatal(err)
			}
			public := 0
			for name := range bodies {
				if ast.IsExported(name) {
					public++
				}
			}
			if public != profile.entries {
				t.Fatal("the owner injected or omitted historical public APIs", public, profile.entries)
			}
			for _, row := range historicalAtomicRows(t, profile.entries) {
				declaration := runtimeAtomicFunctions(t, []byte("package atomic\nimport \"unsafe\"\n"+row.declaration()))[row.name]
				body := bodies[row.name]
				if body == nil || body.Body == nil || len(body.Body.List) == 0 ||
					runtimeAtomicSignature(t, body) != runtimeAtomicSignature(t, declaration) {
					t.Fatal("the historical wrapper lacks its exact typed Go body", row.name)
				}
			}
			for _, name := range []string{"existing.go", "other_arm64.s"} {
				if _, changed := overlay[filepath.Join(dir, name)]; changed {
					t.Fatal("whole-package suppression consumed an unrelated SDK source", name)
				}
			}
			repeated, _, err := buildSourcePatchOverlayForGOROOT(overlay, env.LLGoRuntimeDir(), root, context)
			if err != nil || !reflect.DeepEqual(repeated, overlay) {
				t.Fatal("reusing the intact historical source view changed ownership", err)
			}
		})
	}
}

func TestRuntimeAtomicARM64HistoricalSDKRejectsIncompleteOwners(t *testing.T) {
	for _, profile := range []historicalAtomicProfile{
		{"go1.20.14", "runtime/internal/atomic", 42},
		{"go1.22.2", "runtime/internal/atomic", 48},
		{"go1.23.4", "internal/runtime/atomic", 48},
	} {
		for _, mutation := range []string{"missing-text", "unknown-same-count", "wrong-signature", "existing-sdk-body"} {
			t.Run(profile.version+"/"+mutation, func(t *testing.T) {
				root, _ := historicalAtomicFixture(t, profile, mutation)
				_, _, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), root, sourcePatchBuildContext{
					goos: "linux", goarch: "arm64", goversion: profile.version,
				})
				if err == nil {
					t.Fatal("an incomplete historical atomic source family was silently accepted")
				}
			})
		}
	}
}

func TestRuntimeAtomicARM64HistoricalSDKRetainsOtherArchitectures(t *testing.T) {
	profile := historicalAtomicProfile{"go1.22.2", "runtime/internal/atomic", 48}
	root, _ := historicalAtomicFixture(t, profile, "")
	for _, target := range []struct{ goos, goarch string }{
		{"linux", "amd64"}, {"linux", "386"}, {"linux", "arm"}, {"js", "wasm"},
	} {
		t.Run(target.goos+"/"+target.goarch, func(t *testing.T) {
			overlay, files, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), root, sourcePatchBuildContext{
				goos: target.goos, goarch: target.goarch, goversion: profile.version,
			})
			if err != nil {
				t.Fatal(err)
			}
			// Other runtime patches remain legitimate; only the two atomic
			// package inventories must be untouched on these architectures.
			for _, packagePath := range []string{"runtime/internal/atomic", "internal/runtime/atomic"} {
				if len(files[packagePath]) != 0 {
					t.Fatal("the historical ARM64 alias injected another architecture's atomic owner", packagePath)
				}
				dir := filepath.Join(root, "src", filepath.FromSlash(packagePath))
				for name := range overlay {
					if filepath.Dir(name) == dir {
						t.Fatal("the historical ARM64 alias changed another architecture's atomic source", name)
					}
				}
			}
		})
	}
}
