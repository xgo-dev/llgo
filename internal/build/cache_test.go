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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizePkgPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github.com/user/repo", filepath.Join("github.com", "user", "repo")},
		{"example.com/pkg", filepath.Join("example.com", "pkg")},
		{"simple", "simple"},
		{"github.com/user/pkg.v2", filepath.Join("github.com", "user", "pkg.v2")},
		{"", "_"},
		{"a//b", filepath.Join("a", "_", "b")},
		{"a/./b", filepath.Join("a", "_", "b")},
		{"a/../b", filepath.Join("a", "_", "b")},
		{"a/pkg+plus", filepath.Join("a", sanitizeComponent("pkg+plus"))},
	}

	for _, tt := range tests {
		got := sanitizePkgPath(tt.input)
		if got != tt.want {
			t.Errorf("sanitizePkgPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestJoinCachePath(t *testing.T) {
	sep := string(os.PathSeparator)
	roots := []string{
		"",
		".",
		filepath.Join(sep, "tmp", "root"),
		filepath.Join(sep, "tmp", "root") + sep,
		sep,
	}
	for _, root := range roots {
		got := joinCachePath(root, "target", filepath.Join("pkg", "sub"))
		want := filepath.Join(root, "target", filepath.Join("pkg", "sub"))
		if got != want {
			t.Fatalf("joinCachePath(%q) = %q, want %q", root, got, want)
		}
	}
}

func TestCacheManager_PackagePaths(t *testing.T) {
	// Override cache root for testing
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	cm := newCacheManager()
	paths := cm.PackagePaths("arm64-darwin", "github.com/user/pkg", "abc123")

	expectedDir := filepath.Join(td, "build", "arm64-darwin", "github.com", "user", "pkg")
	if paths.Dir != expectedDir {
		t.Errorf("Dir = %q, want %q", paths.Dir, expectedDir)
	}

	expectedArchive := filepath.Join(expectedDir, "abc123.a")
	if paths.Archive != expectedArchive {
		t.Errorf("Archive = %q, want %q", paths.Archive, expectedArchive)
	}

	expectedManifest := filepath.Join(expectedDir, "abc123.manifest")
	if paths.Manifest != expectedManifest {
		t.Errorf("Manifest = %q, want %q", paths.Manifest, expectedManifest)
	}
}

func TestCacheManager_EnsureDir(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	cm := newCacheManager()
	paths := cm.PackagePaths("x86_64-linux", "test/pkg", "fp123")

	if err := cm.EnsureDir(paths); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}

	if _, err := os.Stat(paths.Dir); err != nil {
		t.Errorf("directory not created: %v", err)
	}

	// Should be idempotent
	if err := cm.EnsureDir(paths); err != nil {
		t.Errorf("EnsureDir second call: %v", err)
	}
}

func TestWriteManifest(t *testing.T) {
	td := t.TempDir()
	path := filepath.Join(td, "subdir", "test.manifest")
	content := "Env:\nGOOS=linux\n"

	if err := writeManifest(path, content); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	// Read back
	got, err := readManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	if got != content {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestWriteManifest_Atomic(t *testing.T) {
	td := t.TempDir()
	path := filepath.Join(td, "test.manifest")

	// Write first version
	if err := writeManifest(path, "version1"); err != nil {
		t.Fatalf("WriteManifest v1: %v", err)
	}

	// Write second version (should replace atomically)
	if err := writeManifest(path, "version2"); err != nil {
		t.Fatalf("WriteManifest v2: %v", err)
	}

	got, _ := readManifest(path)
	if got != "version2" {
		t.Errorf("content = %q, want version2", got)
	}

	// No temp files should remain
	entries, _ := os.ReadDir(td)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestReadManifest_NotExist(t *testing.T) {
	_, err := readManifest("/nonexistent/file.manifest")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestCacheManager_CacheExists(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	cm := newCacheManager()
	paths := cm.PackagePaths("arm64-darwin", "test/pkg", "fp123")

	// Initially should not exist
	if cm.cacheExists(paths) {
		t.Error("cache should not exist initially")
	}

	// Create directory and files
	if err := cm.EnsureDir(paths); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(paths.Archive, []byte("archive"), 0644)

	// Still should not exist (manifest missing)
	if cm.cacheExists(paths) {
		t.Error("cache should not exist without manifest")
	}

	// Create manifest
	os.WriteFile(paths.Manifest, []byte("manifest"), 0644)

	// Now should exist
	if !cm.cacheExists(paths) {
		t.Error("cache should exist with both files")
	}
}

func TestTargetTriple(t *testing.T) {
	tests := []struct {
		goos, goarch, llvmTarget, abi string
		want                          string
	}{
		{"darwin", "arm64", "arm64-apple-darwin", "", "arm64-apple-darwin"},
		{"linux", "amd64", "x86_64-unknown-linux-gnu", "", "x86_64-unknown-linux-gnu"},
		{"darwin", "arm64", "", "", "arm64-darwin"},
		{"linux", "amd64", "", "gnu", "amd64-linux-gnu"},
		{"windows", "amd64", "x86_64-pc-windows-msvc", "msvc", "x86_64-pc-windows-msvc-msvc"},
	}

	for _, tt := range tests {
		got := targetTriple(tt.goos, tt.goarch, tt.llvmTarget, tt.abi)
		if got != tt.want {
			t.Errorf("TargetTriple(%q, %q, %q, %q) = %q, want %q",
				tt.goos, tt.goarch, tt.llvmTarget, tt.abi, got, tt.want)
		}
	}
}

func TestCacheManager_CleanPackageCache(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	cm := newCacheManager()
	paths := cm.PackagePaths("arm64-darwin", "test/pkg", "fp123")

	// Create cache
	cm.EnsureDir(paths)
	os.WriteFile(paths.Archive, []byte("archive"), 0644)
	os.WriteFile(paths.Manifest, []byte("manifest"), 0644)

	// Clean
	if err := cm.cleanPackageCache("arm64-darwin", "test/pkg"); err != nil {
		t.Fatalf("CleanPackageCache: %v", err)
	}

	// Should not exist
	if cm.cacheExists(paths) {
		t.Error("cache should be cleaned")
	}
}

func TestCacheManager_CleanAllCache(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	cm := newCacheManager()

	// Create multiple caches
	paths1 := cm.PackagePaths("arm64-darwin", "pkg1", "fp1")
	paths2 := cm.PackagePaths("x86_64-linux", "pkg2", "fp2")

	cm.EnsureDir(paths1)
	cm.EnsureDir(paths2)
	os.WriteFile(paths1.Archive, []byte("1"), 0644)
	os.WriteFile(paths2.Archive, []byte("2"), 0644)

	// Clean all
	if err := cm.cleanAllCache(); err != nil {
		t.Fatalf("CleanAllCache: %v", err)
	}

	// Check root is removed
	if _, err := os.Stat(filepath.Join(td, "build")); !os.IsNotExist(err) {
		t.Error("cache root should be removed")
	}
}

func TestCacheManager_ListCachedPackages(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	cm := newCacheManager()

	// Initially empty
	fps, err := cm.listCachedPackages("arm64-darwin", "test/pkg")
	if err != nil {
		t.Fatalf("ListCachedPackages: %v", err)
	}
	if len(fps) != 0 {
		t.Errorf("expected empty list, got %v", fps)
	}

	// Create some caches
	paths1 := cm.PackagePaths("arm64-darwin", "test/pkg", "fp1")
	paths2 := cm.PackagePaths("arm64-darwin", "test/pkg", "fp2")
	cm.EnsureDir(paths1)
	os.WriteFile(paths1.Archive, []byte("1"), 0644)
	os.WriteFile(paths2.Archive, []byte("2"), 0644)

	fps, err = cm.listCachedPackages("arm64-darwin", "test/pkg")
	if err != nil {
		t.Fatalf("ListCachedPackages: %v", err)
	}
	if len(fps) != 2 {
		t.Errorf("expected 2 fingerprints, got %d", len(fps))
	}
}

func TestCacheManager_Stats(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	cm := newCacheManager()

	// Create some caches
	paths1 := cm.PackagePaths("arm64-darwin", "pkg1", "fp1")
	paths2 := cm.PackagePaths("arm64-darwin", "pkg2", "fp2")
	cm.EnsureDir(paths1)
	cm.EnsureDir(paths2)

	content1 := []byte("archive content 1")
	content2 := []byte("archive content 2 longer")
	os.WriteFile(paths1.Archive, content1, 0644)
	os.WriteFile(paths2.Archive, content2, 0644)
	os.WriteFile(paths1.Manifest, []byte("m1"), 0644)
	os.WriteFile(paths2.Manifest, []byte("m2"), 0644)

	stats, err := cm.stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	if stats.TotalPackages != 2 {
		t.Errorf("TotalPackages = %d, want 2", stats.TotalPackages)
	}

	expectedSize := int64(len(content1) + len(content2) + 4) // +4 for manifests
	if stats.TotalSize != expectedSize {
		t.Errorf("TotalSize = %d, want %d", stats.TotalSize, expectedSize)
	}
}

func TestCacheManager_Stats_Empty(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	cm := newCacheManager()

	// Stats on empty cache should not error
	stats, err := cm.stats()
	if err != nil {
		t.Fatalf("Stats on empty: %v", err)
	}

	if stats.TotalPackages != 0 || stats.TotalSize != 0 {
		t.Errorf("expected empty stats, got packages=%d size=%d",
			stats.TotalPackages, stats.TotalSize)
	}
}
