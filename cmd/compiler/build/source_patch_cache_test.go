/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSourcePatchCacheCodecRoundTrip(t *testing.T) {
	rec := &sourcePatchCacheRecord{
		Dirs: []depfileDirStat{
			{Path: "/dir1", Exists: true, Hash: "hash1"},
			{Path: "/dir2", Exists: false},
		},
		Files: []sourcePatchFileStat{
			{Path: "/file1.go", Size: 1234, ModTime: 5678},
		},
		Overlay: map[string][]byte{
			"/file1.go": []byte("package p"),
		},
		PkgFiles: map[string][]string{
			"pkg": {"/file1.go"},
		},
	}

	data := marshalSourcePatchCache(rec)
	decoded, err := unmarshalSourcePatchCache(data)
	if err != nil {
		t.Fatalf("unmarshalSourcePatchCache failed: %v", err)
	}

	if !reflect.DeepEqual(rec, decoded) {
		t.Fatalf("mismatch:\ngot  %+v\nwant %+v", decoded, rec)
	}
}

func TestSourcePatchCacheValidationAndErrors(t *testing.T) {
	// 1. Nil validation
	if validateSourcePatchCache(nil) {
		t.Error("expected false for nil record")
	}

	// 2. Bad magic
	if _, err := unmarshalSourcePatchCache([]byte("BADMAGIC123")); err == nil {
		t.Error("expected error for bad magic")
	}

	// 3. Short data
	if _, err := unmarshalSourcePatchCache([]byte("LLGO")); err == nil {
		t.Error("expected error for short data")
	}

	// 4. Trailing bytes
	valid := marshalSourcePatchCache(&sourcePatchCacheRecord{})
	trailing := append(valid, "extra"...)
	if _, err := unmarshalSourcePatchCache(trailing); err == nil {
		t.Error("expected error for trailing bytes")
	}

	// 5. Validation on actual files/dirs
	tmpDir := t.TempDir()
	fPath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(fPath, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(fPath)
	if err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(tmpDir)
	dHash := hashDirEntries(entries)

	validRec := &sourcePatchCacheRecord{
		Dirs: []depfileDirStat{
			{Path: tmpDir, Exists: true, Hash: dHash},
		},
		Files: []sourcePatchFileStat{
			{Path: fPath, Size: fi.Size(), ModTime: fi.ModTime().UnixNano()},
		},
	}
	if !validateSourcePatchCache(validRec) {
		t.Error("expected validRec to validate successfully")
	}

	// Tamper file size
	invalidRec := &sourcePatchCacheRecord{
		Dirs: validRec.Dirs,
		Files: []sourcePatchFileStat{
			{Path: fPath, Size: fi.Size() + 10, ModTime: fi.ModTime().UnixNano()},
		},
	}
	if validateSourcePatchCache(invalidRec) {
		t.Error("expected invalidRec to fail validation on size mismatch")
	}

	// Non-existent dir validation
	missingDir := filepath.Join(tmpDir, "nonexistent")
	missingDirRec := &sourcePatchCacheRecord{
		Dirs: []depfileDirStat{
			{Path: missingDir, Exists: false},
		},
	}
	if !validateSourcePatchCache(missingDirRec) {
		t.Error("expected missing dir to validate as non-existent")
	}
}
