/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package simd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const inventoryFixture = `//go:build goexperiment.simd

package archsimd

type Vec struct { values [4]uint32 }
type Alias = Vec
type X86Features struct{}
type ARM64Features struct{}
var X86 X86Features
var ARM64 ARM64Features
const Width = 128
func (X86Features) AVX2() bool { return false }
func (ARM64Features) PMULL() bool { return false }
func (v Vec) Add(x Vec) Vec
func (v *Vec) Store(x *[4]uint32)
func Load(x *[4]uint32) Vec
func Params(first, second Vec) (left, right Vec) { return first, second }
func Wrapper(x *[4]uint32) Vec { return Load(x) }
func hidden(x Vec) Vec
func (v Vec) hidden() Vec
`

func inventoryRoot(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte("go1.27.0\ntime test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, text := range files {
		inventoryWrite(t, root, name, text)
	}
	return root
}

func inventoryWrite(t *testing.T, root, name, text string) {
	t.Helper()
	path := filepath.Join(root, "src", "simd", "archsimd", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func inventoryScan(t *testing.T, root string) *Inventory {
	t.Helper()
	inv, err := ScanInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func inventoryEntry(t *testing.T, target InventoryTarget, key string) InventoryEntry {
	t.Helper()
	for _, entry := range target.Entries {
		if entry.Key == inventoryPackage+"."+key {
			return entry
		}
	}
	t.Fatalf("missing %s API %s", target.GOARCH, key)
	return InventoryEntry{}
}

func TestInventorySourceClassificationAndProfiles(t *testing.T) {
	root := inventoryRoot(t, map[string]string{
		"common.go":          inventoryFixture,
		"wide_amd64.go":      "package archsimd\ntype Wide [8]uint64\n",
		"neon_arm64.go":      "package archsimd\nfunc Neon(x Vec) Vec\n",
		"wasm.go":            "//go:build goexperiment.simd && wasm && wasip1\n\npackage archsimd\nfunc Wasm(x Vec) Vec\n",
		"new_api_windows.go": "package archsimd\nfunc WindowsOnly()\n",
		"test_test.go":       "package archsimd\nfunc TestOnly() {}\n",
		"ignored.go":         "//go:build ignore\n\npackage main\nfunc GeneratedTool() {}\n",
		"feature.go":         "//go:build amd64.v3 || goexperiment.somefutureexperiment\n\npackage archsimd\nfunc HostTagLeak()\n",
		"_gen/tool.go":       "package main\nfunc Tool() {}\n",
	})
	inv := inventoryScan(t, root)
	if len(inv.Sources) != 7 || len(inv.Targets) != 3 {
		t.Fatalf("sources/targets = %d/%d", len(inv.Sources), len(inv.Targets))
	}
	if inv.ReferenceRevision != InventoryReferenceRevision || len(inv.SourceSHA256) != 64 {
		t.Fatalf("missing source provenance: %+v", inv)
	}
	for _, target := range inv.Targets {
		for _, tc := range []struct{ key, kind, body, status string }{
			{"Vec", "type", "not-applicable", "unverified"},
			{"Alias", "type", "not-applicable", "unverified"},
			{"Width", "constant", "not-applicable", "unverified"},
			{"X86", "variable", "not-applicable", "unverified"},
			{"X86Features.AVX2", "cpu-query", "present", "unverified"},
			{"ARM64Features.PMULL", "cpu-query", "present", "unverified"},
			{"Vec.Add", "intrinsic-candidate", "absent", "unimplemented"},
			{"Vec.Store", "intrinsic-candidate", "absent", "unimplemented"},
			{"Wrapper", "go-body", "present", "unverified"},
		} {
			entry := inventoryEntry(t, target, tc.key)
			if entry.Kind != tc.kind || entry.Body != tc.body || entry.LLGoStatus != tc.status {
				t.Errorf("%s %s = %+v", target.GOARCH, tc.key, entry)
			}
			if entry.Source != "common.go" || entry.Line < 3 {
				t.Errorf("missing location for %+v", entry)
			}
		}
		if got := inventoryEntry(t, target, "Params").Signature; got != "func Params(Vec, Vec) (Vec, Vec)" {
			t.Errorf("parameter names leaked into signature: %q", got)
		}
		if got := inventoryEntry(t, target, "Vec.Store").Signature; got != "func (*Vec) Store(*[4]uint32)" {
			t.Errorf("pointer receiver signature = %q", got)
		}
		for _, entry := range target.Entries {
			for _, absent := range []string{"hidden", "WindowsOnly", "TestOnly", "GeneratedTool", "HostTagLeak", "Tool"} {
				if strings.HasSuffix(entry.Key, "."+absent) {
					t.Errorf("excluded API leaked into %s: %s", target.GOARCH, entry.Key)
				}
			}
		}
		switch target.GOARCH {
		case "amd64":
			inventoryEntry(t, target, "Wide")
		case "arm64":
			inventoryEntry(t, target, "Neon")
		case "wasm":
			inventoryEntry(t, target, "Wasm")
		}
	}
	if another := inventoryScan(t, root); !reflect.DeepEqual(inv, another) {
		t.Fatal("repeated scans are not deterministic")
	}
}

func TestInventoryDriftSeparatesAPIAndSource(t *testing.T) {
	root := inventoryRoot(t, map[string]string{"api.go": inventoryFixture})
	before := inventoryScan(t, root)
	changed := strings.Replace(inventoryFixture, "func Load(x *[4]uint32) Vec", "func Load(x *[8]uint32) Vec", 1)
	changed = strings.Replace(changed, "func (v Vec) Add(x Vec) Vec", "func (v Vec) Add(x Vec) Vec { return x }", 1)
	changed = strings.Replace(changed, "type Alias = Vec\n", "", 1)
	changed += "\nfunc NewAPI(x Vec) Vec\n"
	inventoryWrite(t, root, "api.go", changed)
	after := inventoryScan(t, root)
	changes := CompareInventories(before, after)
	for _, target := range after.Targets {
		for _, expected := range []struct{ key, kind string }{
			{"Alias", "removed"}, {"NewAPI", "added"}, {"Load", "signature"},
			{"Vec.Add", "body"}, {"Vec.Add", "classification"}, {"Vec.Add", "llgo-status"},
		} {
			found := false
			for _, change := range changes {
				if change.Target == target.GOOS+"/"+target.GOARCH &&
					change.Key == inventoryPackage+"."+expected.key && change.Kind == expected.kind {
					found = true
				}
			}
			if !found {
				t.Errorf("missing %s %s %s in %+v", target.GOARCH, expected.kind, expected.key, changes)
			}
		}
	}
	if len(CompareInventorySummaries(before.Summary(), after.Summary())) == 0 {
		t.Fatal("compact baseline did not detect API drift")
	}
	// Equivalent parameter grouping/names and comments change bytes, not APIs.
	changed = strings.Replace(inventoryFixture, "first, second Vec) (left, right Vec", "a Vec, b Vec) (r Vec, s Vec", 1)
	changed = "// new source comment\n" + changed
	inventoryWrite(t, root, "api.go", changed)
	after = inventoryScan(t, root)
	for _, change := range CompareInventories(before, after) {
		if change.Target != "" {
			t.Errorf("source-only change reported as API change: %+v", change)
		}
	}
	if before.SourceSHA256 == after.SourceSHA256 {
		t.Fatal("source byte change was lost")
	}
	// Even an API introduced behind an unaudited OS constraint changes the
	// source baseline and requires profile review; it cannot be silently lost.
	inventoryWrite(t, root, "api.go", inventoryFixture)
	inventoryWrite(t, root, "future_windows.go", "package archsimd\nfunc FutureWindowsAPI()\n")
	future := inventoryScan(t, root)
	if before.Summary().Targets[0].APISHA256 != future.Summary().Targets[0].APISHA256 || before.SourceSHA256 == future.SourceSHA256 {
		t.Fatal("excluded new source did not preserve profiled API and change source identity")
	}
	// A body-only change to a private helper remains visible in source hashes.
	inventoryWrite(t, root, "api.go", inventoryFixture+"\nfunc helper() int { return 1 }\n")
	private := inventoryScan(t, root)
	if before.Summary().Targets[0].APISHA256 != private.Summary().Targets[0].APISHA256 || before.SourceSHA256 == private.SourceSHA256 {
		t.Fatal("private helper change did not preserve API and change source identity")
	}
}

func TestInventoryRejectsInvalidSources(t *testing.T) {
	for _, tc := range []struct{ name, version, source, extra string }{
		{"wrong-version", "go1.28.0", inventoryFixture, ""},
		{"wrong-package", "go1.27.0", "package other\ntype V int", ""},
		{"syntax", "go1.27.0", "package archsimd\ntype V struct {", ""},
		{"duplicate", "go1.27.0", inventoryFixture, "package archsimd\nfunc Load(x int) int"},
		{"empty", "go1.27.0", "package archsimd\nfunc hidden() {}", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := inventoryRoot(t, map[string]string{"api.go": tc.source})
			if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte(tc.version), 0o644); err != nil {
				t.Fatal(err)
			}
			if tc.extra != "" {
				inventoryWrite(t, root, "extra.go", tc.extra)
			}
			if _, err := ScanInventory(root); err == nil {
				t.Fatal("invalid inventory accepted")
			}
		})
	}
}

func TestInventoryImportedSignatureIdentity(t *testing.T) {
	source := "package archsimd\nimport v \"example.net/vectors\"\nfunc Convert(v.Vector) v.Vector\n"
	root := inventoryRoot(t, map[string]string{"api.go": source})
	before := inventoryScan(t, root)
	// A local import alias is not part of the public signature.
	inventoryWrite(t, root, "api.go", strings.ReplaceAll(strings.ReplaceAll(source, "import v ", "import renamed "), "v.Vector", "renamed.Vector"))
	for _, change := range CompareInventories(before, inventoryScan(t, root)) {
		if change.Target != "" {
			t.Errorf("import alias rename reported as API drift: %+v", change)
		}
	}
	// The same alias with a different defining package changes the signature.
	inventoryWrite(t, root, "api.go", strings.ReplaceAll(source, "example.net/vectors", "example.org/vectors"))
	changes := CompareInventories(before, inventoryScan(t, root))
	count := 0
	for _, change := range changes {
		if change.Kind == "signature" && change.Key == inventoryPackage+".Convert" {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("imported signature drift not detected on each target: %+v", changes)
	}
}

func TestInventoryMatchesPinnedGo127(t *testing.T) {
	// The pinned release-branch revision says go1.27.1; the Go 1.27.0
	// toolchain contains the same archsimd package bytes. Summary comparison
	// is deliberately package-content-based across the Go 1.27 release line.
	inv := inventoryScan(t, runtime.GOROOT())
	data, err := os.ReadFile("testdata/go1.27-inventory-summary.json")
	if err != nil {
		t.Fatal(err)
	}
	var baseline InventorySummary
	if err := json.Unmarshal(data, &baseline); err != nil {
		t.Fatal(err)
	}
	if changes := CompareInventorySummaries(baseline, inv.Summary()); len(changes) != 0 {
		t.Fatalf("Go 1.27 archsimd source drift; generate full inventories from both revisions before updating the baseline: %+v", changes)
	}
	for _, target := range inv.Targets {
		inventoryEntry(t, target, "Int8x16.Add")
		inventoryEntry(t, target, "Uint64x2.SetElem")
		inventoryEntry(t, target, "LoadInt8x16Part")
		inventoryEntry(t, target, "X86Features.AVX2")
		inventoryEntry(t, target, "ARM64Features.PMULL")
		if target.GOARCH == "amd64" {
			inventoryEntry(t, target, "Float64x8.Add")
		}
	}
}
