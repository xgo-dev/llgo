//go:build !llgo

/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package build

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestManifestBuilder_Build(t *testing.T) {
	m := newManifestBuilder()
	m.env.Goarch = "arm64"
	m.env.Goos = "darwin"
	m.common.AbiMode = "2"
	m.pkg.PkgPath = "example.com/foo"

	content := m.Build()
	data, err := decodeManifest(content)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}

	if data.Env.Goarch != "arm64" || data.Env.Goos != "darwin" {
		t.Fatalf("env section mismatch: %+v", data.Env)
	}
	if data.Common.AbiMode != "2" {
		t.Fatalf("common section mismatch: %+v", data.Common)
	}
	if data.Package.PkgPath != "example.com/foo" {
		t.Fatalf("package section mismatch: %+v", data.Package)
	}
}

func TestManifestBuilder_BuildSorting(t *testing.T) {
	m := newManifestBuilder()
	// Add in reverse order
	m.env.Vars = orderedStringMap{"Z_KEY": "z", "A_KEY": "a", "M_KEY": "m"}
	m.pkg.RewriteVars = orderedStringMap{"Z": "1", "A": "2"}

	content := m.Build()
	data, err := decodeManifest(content)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}
	// env vars are sorted by key
	keys := []string{}
	for k := range data.Env.Vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if strings.Join(keys, ",") != "A_KEY,M_KEY,Z_KEY" {
		t.Fatalf("env vars not sorted: %v", keys)
	}
	rvKeys := []string{}
	for k := range data.Package.RewriteVars {
		rvKeys = append(rvKeys, k)
	}
	sort.Strings(rvKeys)
	if strings.Join(rvKeys, ",") != "A,Z" {
		t.Fatalf("rewrite vars not sorted: %v", rvKeys)
	}
}

func TestFingerprintManifestMatchesSHA256(t *testing.T) {
	content := "env:\n  GOOS: linux\npackage:\n  pkg_path: example.com/pkg\n"
	sum := sha256.Sum256([]byte(content))
	want := hex.EncodeToString(sum[:])
	if got := fingerprintManifest(content); got != want {
		t.Fatalf("fingerprintManifest() = %q, want %q", got, want)
	}
	if got := fingerprintManifest(""); got != hex.EncodeToString(sha256.New().Sum(nil)) {
		t.Fatalf("fingerprintManifest(empty) = %q", got)
	}
}

func TestManifestBuilder_Fingerprint(t *testing.T) {
	m := newManifestBuilder()
	m.env.Goos = "linux"
	m.env.Goarch = "amd64"
	m.pkg.PkgPath = "test/pkg"

	fp1 := fingerprintManifest(m.Build())
	fp2 := fingerprintManifest(m.Build())

	if fp1 != fp2 {
		t.Error("fingerprint not stable")
	}

	// Fingerprint should be 64 hex characters (sha256)
	if len(fp1) != 64 {
		t.Errorf("fingerprint length = %d, want 64", len(fp1))
	}
}

func TestManifestBuilder_FingerprintDeterminism(t *testing.T) {
	// Different order of adding, same fingerprint
	m1 := newManifestBuilder()
	m1.env.Vars = orderedStringMap{"A": "1", "B": "2"}
	m1.common.BuildTags = []string{"10", "20"}

	m2 := newManifestBuilder()
	m2.env.Vars = orderedStringMap{"B": "2", "A": "1"}
	m2.common.BuildTags = []string{"20", "10"}

	if fingerprintManifest(m1.Build()) != fingerprintManifest(m2.Build()) {
		t.Error("order should not affect fingerprint")
	}

	if m1.Build() != m2.Build() {
		t.Error("order should not affect build output")
	}
}

func TestManifestBuilder_FingerprintDifferentValues(t *testing.T) {
	m1 := newManifestBuilder()
	m1.env.Vars = orderedStringMap{"KEY": "value1"}

	m2 := newManifestBuilder()
	m2.env.Vars = orderedStringMap{"KEY": "value2"}

	if fingerprintManifest(m1.Build()) == fingerprintManifest(m2.Build()) {
		t.Error("different values should produce different fingerprints")
	}
}

func TestManifestBuilder_EmptySections(t *testing.T) {
	m := newManifestBuilder()
	content := m.Build()

	// Empty sections should not be written
	expected := ``
	if content != expected {
		t.Errorf("unexpected empty manifest:\ngot:\n%s\nwant:\n%s", content, expected)
	}

	// Should still produce a valid fingerprint
	fp := fingerprintManifest(m.Build())
	if len(fp) != 64 {
		t.Errorf("fingerprint length = %d, want 64", len(fp))
	}
}

func TestDigestBytes(t *testing.T) {
	data := []byte("hello world")
	hash := digestBytes(data)

	// Known sha256 of "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Errorf("digestBytes = %s, want %s", hash, expected)
	}

	// Empty data
	emptyHash := digestBytes([]byte{})
	if len(emptyHash) != 64 {
		t.Errorf("empty hash length = %d, want 64", len(emptyHash))
	}
}

func TestDigestFiles(t *testing.T) {
	td := t.TempDir()

	// Create test files
	file1 := filepath.Join(td, "a.go")
	file2 := filepath.Join(td, "b.go")
	content1 := []byte("package a")
	content2 := []byte("package b")

	if err := os.WriteFile(file1, content1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, content2, 0644); err != nil {
		t.Fatal(err)
	}

	// Test with files in reverse order (should be sorted)
	list, err := digestFiles([]string{file2, file1})
	if err != nil {
		t.Fatalf("digestFiles: %v", err)
	}

	// Should be sorted by path
	if len(list) != 2 || list[0].Path != file1 || list[1].Path != file2 {
		t.Errorf("structured list not sorted: %+v", list)
	}
	info1, err := os.Stat(file1)
	if err != nil {
		t.Fatal(err)
	}
	info2, err := os.Stat(file2)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].Size != info1.Size() || list[0].ModTime != info1.ModTime().UnixNano() {
		t.Errorf("metadata mismatch for %s: %+v vs %v", file1, list[0], info1)
	}
	if list[1].Size != info2.Size() || list[1].ModTime != info2.ModTime().UnixNano() {
		t.Errorf("metadata mismatch for %s: %+v vs %v", file2, list[1], info2)
	}
}

func TestDigestFiles_Empty(t *testing.T) {
	list, err := digestFiles([]string{})
	if err != nil {
		t.Fatalf("digestFiles: %v", err)
	}
	if list != nil {
		t.Errorf("digestFiles empty list = %#v, want nil", list)
	}
}

func TestDigestFiles_Determinism(t *testing.T) {
	td := t.TempDir()

	file1 := filepath.Join(td, "x.go")
	file2 := filepath.Join(td, "y.go")
	if err := os.WriteFile(file1, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("y"), 0644); err != nil {
		t.Fatal(err)
	}

	// Different order should produce same result
	result1, _ := digestFiles([]string{file1, file2})
	result2, _ := digestFiles([]string{file2, file1})

	if !reflect.DeepEqual(result1, result2) {
		t.Errorf("order should not affect digestFiles result: %+v vs %+v", result1, result2)
	}
}

func TestDigestFilesWithOverlay(t *testing.T) {
	td := t.TempDir()

	// Create a real file
	realFile := filepath.Join(td, "real.go")
	realContent := []byte("real content")
	if err := os.WriteFile(realFile, realContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Create overlay
	overlayFile := filepath.Join(td, "overlay.go")
	overlayContent := []byte("overlay content")
	overlay := map[string][]byte{
		overlayFile: overlayContent,
	}

	list, err := digestFilesWithOverlay([]string{overlayFile, realFile}, overlay)
	if err != nil {
		t.Fatalf("digestFilesWithOverlay: %v", err)
	}

	hashOverlay := digestBytes(overlayContent)

	// Should be sorted by path
	if len(list) != 2 {
		t.Fatalf("expected 2 digests, got %d", len(list))
	}

	if list[0].Path != overlayFile || list[0].OverlayHash != hashOverlay || list[0].Size != int64(len(overlayContent)) || list[0].ModTime != 0 {
		t.Errorf("overlay digest mismatch: %+v", list[0])
	}
	infoReal, err := os.Stat(realFile)
	if err != nil {
		t.Fatal(err)
	}
	if list[1].Path != realFile || list[1].Size != infoReal.Size() || list[1].ModTime != infoReal.ModTime().UnixNano() {
		t.Errorf("real file digest mismatch: %+v vs %v", list[1], infoReal)
	}
}

func TestDigestFilesWithOverlay_Empty(t *testing.T) {
	list, err := digestFilesWithOverlay([]string{}, nil)
	if err != nil {
		t.Fatalf("digestFilesWithOverlay: %v", err)
	}
	if list != nil {
		t.Errorf("digestFilesWithOverlay empty list = %#v, want nil", list)
	}
}

func TestManifestBuilder_SpecialCharacters(t *testing.T) {
	m := newManifestBuilder()
	m.env.Vars = orderedStringMap{"PATH": "/usr/bin:/usr/local/bin"}
	m.pkg.RewriteVars = orderedStringMap{"FLAGS": "-X main.version=1.0.0 -ldflags=-s"}

	content := m.Build()
	data, err := decodeManifest(content)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}
	if data.Env == nil || data.Env.Vars["PATH"] != "/usr/bin:/usr/local/bin" {
		t.Errorf("PATH not preserved: %+v", data.Env)
	}
	if data.Package == nil || data.Package.RewriteVars["FLAGS"] != "-X main.version=1.0.0 -ldflags=-s" {
		t.Errorf("FLAGS not preserved: %+v", data.Package)
	}
}

func TestManifestBuilder_MultipleValues(t *testing.T) {
	m := newManifestBuilder()
	m.env.Vars = orderedStringMap{"KEY": "value1"}
	m.env.Vars = m.env.Vars.Add("KEY", "value2")

	content := m.Build()
	data, err := decodeManifest(content)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}
	if data.Env == nil || data.Env.Vars["KEY"] != "value2" {
		t.Fatalf("duplicate env should keep last value, got %+v", data.Env.Vars)
	}
}
