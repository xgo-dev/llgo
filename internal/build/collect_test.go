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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/goplus/llgo/internal/crosscompile"
	"github.com/goplus/llgo/internal/packages"
	gopackages "golang.org/x/tools/go/packages"
)

func TestCollectFingerprint(t *testing.T) {
	td := t.TempDir()

	// Create a test file
	goFile := filepath.Join(td, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &context{
		conf: &packages.Config{},
		buildConf: &Config{
			Goos:      "darwin",
			Goarch:    "arm64",
			BuildMode: BuildModeExe,
			Tags:      "test",
		},
		crossCompile: crosscompile.Export{
			LLVMTarget: "arm64-apple-darwin",
		},
	}

	pkg := &aPackage{
		Package: &packages.Package{
			PkgPath: "example.com/test",
			GoFiles: []string{goFile},
		},
	}

	if err := ctx.collectFingerprint(pkg); err != nil {
		t.Fatalf("collectFingerprint: %v", err)
	}

	// Check fingerprint is generated
	if pkg.Fingerprint == "" {
		t.Error("fingerprint should not be empty")
	}
	if len(pkg.Fingerprint) != 64 {
		t.Errorf("fingerprint length = %d, want 64", len(pkg.Fingerprint))
	}
	if want := fingerprintManifest(pkg.Manifest); pkg.Fingerprint != want {
		t.Errorf("fingerprint = %s, want hash of manifest %s", pkg.Fingerprint, want)
	}

	data, err := decodeManifest(pkg.Manifest)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}
	if data.Env == nil || data.Common == nil || data.Package == nil {
		t.Fatal("manifest sections should not be empty")
	}
	if data.Env.Goos != "darwin" {
		t.Error("manifest should contain GOOS = darwin")
	}
	if data.Package.PkgPath != "example.com/test" {
		t.Error("manifest should contain PKG_PATH")
	}
}

func TestParseManifestMetadata(t *testing.T) {
	m := newManifestBuilder()
	m.env.Goos = "linux"
	m.pkg.PkgPath = "example.com/lib"
	meta, err := parseManifestMetadata(m.Build())
	if err != nil {
		t.Fatalf("parseManifestMetadata no metadata: %v", err)
	}
	if len(meta.LinkArgs) != 0 || meta.NeedRt || meta.NeedPyInit {
		t.Fatalf("no metadata parse = %+v, want empty", meta)
	}

	content, err := buildManifestYAML(manifestData{Metadata: &manifestMetadata{
		LinkArgs:   []string{"-lm", "-lpthread"},
		NeedRt:     true,
		NeedPyInit: true,
	}})
	if err != nil {
		t.Fatalf("buildManifestYAML: %v", err)
	}
	meta, err = parseManifestMetadata(content)
	if err != nil {
		t.Fatalf("parseManifestMetadata yaml metadata: %v", err)
	}
	if strings.Join(meta.LinkArgs, " ") != "-lm -lpthread" || !meta.NeedRt || !meta.NeedPyInit {
		t.Fatalf("yaml metadata parse = %+v", meta)
	}

	simpleContent := "env:\n    GOOS: linux\nmetadata:\n    need_rt: true\n    need_py_init: true\n"
	meta, err = parseManifestMetadata(simpleContent)
	if err != nil {
		t.Fatalf("parseManifestMetadata simple metadata: %v", err)
	}
	if len(meta.LinkArgs) != 0 || !meta.NeedRt || !meta.NeedPyInit {
		t.Fatalf("simple metadata parse = %+v", meta)
	}
	if simple, ok := parseSimpleManifestMetadata("metadata:\n    need_rt: true\n    need_py_init: false\n"); !ok || !simple.NeedRt || simple.NeedPyInit {
		t.Fatalf("parseSimpleManifestMetadata = %+v, %v", simple, ok)
	}
	if simple, ok := parseSimpleManifestMetadata("metadata:\n    link_args:\n        - -lm\n        - -Wl,--gc-sections\n"); !ok || strings.Join(simple.LinkArgs, " ") != "-lm -Wl,--gc-sections" {
		t.Fatalf("parseSimpleManifestMetadata link_args = %+v, %v", simple, ok)
	}
	if _, ok := parseSimpleManifestMetadata("metadata:\n    link_args:\n        - \"-framework CoreFoundation\"\n"); ok {
		t.Fatal("parseSimpleManifestMetadata should defer quoted link_args to YAML parser")
	}
	if simpleYAML, ok := buildSimpleManifestMetadata(&manifestMetadata{LinkArgs: []string{"-lm", "-Wl,--gc-sections"}, NeedRt: true}); !ok || !strings.Contains(simpleYAML, "-Wl,--gc-sections") {
		t.Fatalf("buildSimpleManifestMetadata = %q, %v", simpleYAML, ok)
	}
	if _, ok := buildSimpleManifestMetadata(&manifestMetadata{LinkArgs: []string{"\"quoted\""}}); ok {
		t.Fatal("buildSimpleManifestMetadata should defer quoted link args to YAML marshaler")
	}

	fullYAML := "env:\n    GOOS: linux\n" + content + "deps:\n    - id: example.com/dep\n      version: v1.0.0\n"
	section, ok := yamlMetadataSection(fullYAML)
	if !ok {
		t.Fatal("yamlMetadataSection should find metadata section")
	}
	if !strings.Contains(section, "metadata:") || strings.Contains(section, "deps:") {
		t.Fatalf("metadata section slice = %q", section)
	}
	section, ok = yamlMetadataSection(content + "deps:\n    - id: example.com/dep\n")
	if !ok || !strings.HasPrefix(section, "metadata:") || strings.Contains(section, "deps:") {
		t.Fatalf("metadata-at-start section slice = %q, %v", section, ok)
	}

	legacy := "[Package]\nLINK_ARGS = -llegacy -lm\nNEED_RT = true\nNEED_PY_INIT = false\n"
	meta, err = parseManifestMetadata(legacy)
	if err != nil {
		t.Fatalf("parseManifestMetadata legacy metadata: %v", err)
	}
	if strings.Join(meta.LinkArgs, " ") != "-llegacy -lm" || !meta.NeedRt || meta.NeedPyInit {
		t.Fatalf("legacy metadata parse = %+v", meta)
	}
}

func TestCollectFingerprintDeterminism(t *testing.T) {
	td := t.TempDir()

	goFile := filepath.Join(td, "main.go")
	if err := os.WriteFile(goFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &context{
		conf: &packages.Config{},
		buildConf: &Config{
			Goos:      "linux",
			Goarch:    "amd64",
			BuildMode: BuildModeExe,
		},
		crossCompile: crosscompile.Export{},
	}

	pkg1 := &aPackage{
		Package: &packages.Package{
			PkgPath: "test/pkg",
			GoFiles: []string{goFile},
		},
	}

	pkg2 := &aPackage{
		Package: &packages.Package{
			PkgPath: "test/pkg",
			GoFiles: []string{goFile},
		},
	}

	if err := ctx.collectFingerprint(pkg1); err != nil {
		t.Fatal(err)
	}
	if err := ctx.collectFingerprint(pkg2); err != nil {
		t.Fatal(err)
	}

	if pkg1.Fingerprint != pkg2.Fingerprint {
		t.Error("same inputs should produce same fingerprint")
	}
}

func TestCollectFingerprintDependencies(t *testing.T) {
	td := t.TempDir()

	depFile := filepath.Join(td, "dep.go")
	if err := os.WriteFile(depFile, []byte("package dep"), 0644); err != nil {
		t.Fatal(err)
	}
	mainFile := filepath.Join(td, "main.go")
	if err := os.WriteFile(mainFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &context{
		conf:         &packages.Config{},
		buildConf:    &Config{Goos: "linux", Goarch: "amd64"},
		crossCompile: crosscompile.Export{},
		pkgs:         map[*packages.Package]Package{},
		pkgByID:      map[string]Package{},
	}

	depPkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/dep",
		PkgPath: "example.com/dep",
		GoFiles: []string{depFile},
	}}
	depWithVersion := &aPackage{Package: &packages.Package{
		ID:      "example.com/depver",
		PkgPath: "example.com/depver",
		GoFiles: []string{depFile},
		Module:  &gopackages.Module{Path: "example.com/depver", Version: "v1.0.0"},
	}}
	ctx.pkgByID[depPkg.ID] = depPkg
	ctx.pkgByID[depWithVersion.ID] = depWithVersion

	mainPkg := &aPackage{Package: &packages.Package{
		ID:      "example.com/main",
		PkgPath: "example.com/main",
		GoFiles: []string{mainFile},
		Imports: map[string]*packages.Package{
			"example.com/dep":    depPkg.Package,
			"example.com/depver": depWithVersion.Package,
		},
	}}

	if err := ctx.collectFingerprint(mainPkg); err != nil {
		t.Fatalf("collectFingerprint: %v", err)
	}

	data, err := decodeManifest(mainPkg.Manifest)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}
	if len(data.Deps) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(data.Deps))
	}
	var seenFingerprint, seenVersion bool
	for _, dep := range data.Deps {
		switch dep.ID {
		case "example.com/depver":
			seenVersion = dep.Version == "v1.0.0" && dep.Fingerprint == ""
		case "example.com/dep":
			seenFingerprint = dep.Fingerprint == depPkg.Fingerprint && dep.Version == ""
		}
	}
	if !seenVersion {
		t.Fatalf("versioned dependency not recorded with version: %+v", data.Deps)
	}
	if !seenFingerprint {
		t.Fatalf("workspace dependency not recorded with fingerprint: %+v", data.Deps)
	}
}

func TestBuildDo_DepFingerprintAndVersion(t *testing.T) {
	root := t.TempDir()
	mainDir := filepath.Join(root, "main")
	depPathDir := filepath.Join(root, "depPath")
	depWorkDir := filepath.Join(root, "depWork")

	must(os.MkdirAll(mainDir, 0o755))
	must(os.MkdirAll(depPathDir, 0o755))
	must(os.MkdirAll(depWorkDir, 0o755))

	writeFile(t, filepath.Join(depPathDir, "go.mod"), "module github.com/matryer/is\n\ngo 1.24\n")
	writeFile(t, filepath.Join(depPathDir, "is.go"), "package is\nfunc OK(v bool) bool { return v }\n")

	writeFile(t, filepath.Join(depWorkDir, "go.mod"), "module github.com/pmezard/go-difflib\n\ngo 1.24\n")
	must(os.MkdirAll(filepath.Join(depWorkDir, "difflib"), 0o755))
	writeFile(t, filepath.Join(depWorkDir, "difflib", "difflib.go"), "package difflib\nconst Name = \"work\"\n")

	mainMod := `module example.com/main

go 1.24

require (
  github.com/davecgh/go-spew v1.1.1
  github.com/matryer/is v1.4.1
  github.com/pmezard/go-difflib v1.0.0
)

replace github.com/davecgh/go-spew v1.1.1 => github.com/davecgh/go-spew v1.1.0
replace github.com/matryer/is v1.4.1 => ../depPath
replace github.com/pmezard/go-difflib v1.0.0 => ../depWork
`
	writeFile(t, filepath.Join(mainDir, "go.mod"), mainMod)
	writeFile(t, filepath.Join(mainDir, "main.go"), "package main\nimport (\"github.com/davecgh/go-spew/spew\"\n\"github.com/matryer/is\"\n\"github.com/pmezard/go-difflib/difflib\"\n)\nvar _ = spew.Sdump(is.OK(true)) + difflib.Name\nfunc main() {}\n")

	oldWD, _ := os.Getwd()
	goWork := "go 1.24\nuse ./main\nuse ./depWork\nuse ./depPath\n"
	writeFile(t, filepath.Join(root, "go.work"), goWork)
	cmd := exec.Command("go", "work", "sync")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go work sync: %v\n%s", err, out)
	}

	must(os.Chdir(mainDir))
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	conf := &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, Mode: ModeBuild, BinPath: filepath.Join(root, "bin")}
	_ = os.MkdirAll(conf.BinPath, 0o755)

	// Let Go discover workspace via parent go.work (no extra env needed)

	pkgs, err := Do(nil, conf)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}

	mainPkg := findPkg(pkgs, "example.com/main")
	if mainPkg == nil {
		t.Fatalf("main package not built")
	}
	data, err := decodeManifest(mainPkg.Manifest)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}

	get := func(prefix string) *depEntry {
		for i := range data.Deps {
			if strings.HasPrefix(data.Deps[i].ID, prefix) {
				return &data.Deps[i]
			}
		}
		return nil
	}

	if dep := get("github.com/davecgh/go-spew"); dep == nil || dep.Version != "v1.1.0" || dep.Fingerprint != "" {
		t.Fatalf("version replace expected version only: %+v", dep)
	}
	if dep := get("github.com/matryer/is"); dep == nil || dep.Version != "" || dep.Fingerprint == "" {
		t.Fatalf("relative replace should fingerprint: %+v", dep)
	}
	if dep := get("github.com/pmezard/go-difflib"); dep == nil || dep.Version != "" || dep.Fingerprint == "" {
		t.Fatalf("workspace dep should fingerprint: %+v", dep)
	}
}

func TestTargetTripleMethod(t *testing.T) {
	tests := []struct {
		name   string
		ctx    *context
		expect string
	}{
		{
			name: "with llvm target",
			ctx: &context{
				buildConf: &Config{
					Goos:   "darwin",
					Goarch: "arm64",
				},
				crossCompile: crosscompile.Export{
					LLVMTarget: "arm64-apple-darwin",
				},
			},
			expect: "arm64-apple-darwin",
		},
		{
			name: "without llvm target",
			ctx: &context{
				buildConf: &Config{
					Goos:   "linux",
					Goarch: "amd64",
				},
				crossCompile: crosscompile.Export{},
			},
			expect: "amd64-linux",
		},
		{
			name: "with abi",
			ctx: &context{
				buildConf: &Config{
					Goos:   "linux",
					Goarch: "arm",
				},
				crossCompile: crosscompile.Export{
					TargetABI: "gnueabihf",
				},
			},
			expect: "arm-linux-gnueabihf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ctx.targetTriple()
			if got != tt.expect {
				t.Errorf("targetTriple() = %q, want %q", got, tt.expect)
			}
			if cached := tt.ctx.targetTriple(); cached != tt.expect {
				t.Errorf("cached targetTriple() = %q, want %q", cached, tt.expect)
			}
			if tt.ctx.targetTripleCached != tt.expect {
				t.Errorf("targetTripleCached = %q, want %q", tt.ctx.targetTripleCached, tt.expect)
			}
		})
	}
}

func TestCollectEnvInputsCachesPerBuildInputs(t *testing.T) {
	t.Setenv(llgoTrace, "1")
	ctx := &context{
		buildConf:    &Config{Goos: "linux", Goarch: "amd64", CompilerHash: "hash"},
		crossCompile: crosscompile.Export{LLVMTarget: "x86_64-test"},
		llvmVersion:  "LLVM test",
	}
	m1 := newManifestBuilder()
	ctx.collectEnvInputs(m1)
	if !ctx.envInputsReady {
		t.Fatal("env inputs were not cached")
	}
	if m1.env.Goos != "linux" || m1.env.Goarch != "amd64" || m1.env.LlvmTriple != "x86_64-test" || m1.env.LlgoCompilerHash != "hash" || m1.env.LlvmVersion != "LLVM test" {
		t.Fatalf("unexpected env inputs: %+v", m1.env)
	}
	if got := m1.env.Vars[llgoTrace]; got != "1" {
		t.Fatalf("env var %s = %q, want 1", llgoTrace, got)
	}

	t.Setenv(llgoTrace, "2")
	m2 := newManifestBuilder()
	ctx.collectEnvInputs(m2)
	if got := m2.env.Vars[llgoTrace]; got != "1" {
		t.Fatalf("cached env var %s = %q, want original per-build value 1", llgoTrace, got)
	}

	ctx2 := &context{buildConf: &Config{}, llvmVersion: "LLVM test"}
	m3 := newManifestBuilder()
	ctx2.collectEnvInputs(m3)
	if got := m3.env.Vars[llgoTrace]; got != "2" {
		t.Fatalf("fresh context env var %s = %q, want 2", llgoTrace, got)
	}
}

func TestCollectCommonInputsCachesBuildTags(t *testing.T) {
	ctx := &context{buildConf: &Config{Tags: "z,a,m", AbiMode: 1}}
	m1 := newManifestBuilder()
	ctx.collectCommonInputs(m1)
	if got := strings.Join(m1.common.BuildTags, ","); got != "a,m,z" {
		t.Fatalf("BuildTags = %q, want sorted tags", got)
	}
	cached := ctx.buildTagsCached
	if len(cached) != 3 {
		t.Fatalf("buildTagsCached = %v", cached)
	}

	m2 := newManifestBuilder()
	ctx.collectCommonInputs(m2)
	if len(m2.common.BuildTags) == 0 || &m2.common.BuildTags[0] != &cached[0] {
		t.Fatalf("collectCommonInputs did not reuse cached build tags")
	}

	ctx.buildConf.Tags = "b,a"
	m3 := newManifestBuilder()
	ctx.collectCommonInputs(m3)
	if got := strings.Join(m3.common.BuildTags, ","); got != "a,b" {
		t.Fatalf("BuildTags after config change = %q, want sorted refreshed tags", got)
	}
}

func TestEnsureCacheManager(t *testing.T) {
	ctx := &context{
		buildConf: &Config{},
	}

	// First call should create manager
	cm1 := ctx.ensureCacheManager()
	if cm1 == nil {
		t.Fatal("ensureCacheManager returned nil")
	}

	// Second call should return same instance
	cm2 := ctx.ensureCacheManager()
	if cm1 != cm2 {
		t.Error("ensureCacheManager should return same instance")
	}
}

func TestTryLoadFromCache_NoFingerprint(t *testing.T) {
	ctx := &context{
		buildConf: &Config{},
	}

	pkg := &aPackage{
		Package: &packages.Package{
			PkgPath: "test/pkg",
		},
		Fingerprint: "", // No fingerprint
	}

	if ctx.tryLoadFromCache(pkg) {
		t.Error("should return false when no fingerprint")
	}
}

func TestTryLoadFromCache_ForceRebuild(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	ctx := &context{
		conf: &packages.Config{},
		buildConf: &Config{
			Goos:         "darwin",
			Goarch:       "arm64",
			ForceRebuild: true, // Force rebuild enabled
		},
		crossCompile: crosscompile.Export{
			LLVMTarget: "arm64-apple-darwin",
		},
	}

	// Create a fake cache entry
	pkg := &aPackage{
		Package: &packages.Package{
			PkgPath: "example.com/cached",
			Name:    "cached",
		},
		Fingerprint: "test123",
		Manifest: func() string {
			m := newManifestBuilder()
			m.env.Goos = "darwin"
			m.pkg.PkgPath = "example.com/cached"
			return m.Build()
		}(),
	}

	// Create a temporary .o file
	objFile, err := os.CreateTemp(td, "test-*.o")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	objFile.WriteString("fake object file")
	objFile.Close()

	pkg.ObjFiles = []string{objFile.Name()}

	// First save to cache
	ctx.buildConf.ForceRebuild = false
	if err := ctx.saveToCache(pkg); err != nil {
		t.Fatalf("saveToCache: %v", err)
	}

	// Verify cache exists
	cm := ctx.ensureCacheManager()
	paths := cm.PackagePaths("arm64-apple-darwin", "example.com/cached", "test123")
	if _, err := os.Stat(paths.Archive); err != nil {
		t.Fatalf("cache should exist: %v", err)
	}

	// Now enable ForceRebuild and try to load
	ctx.buildConf.ForceRebuild = true

	// Clear ObjFiles to verify it's not loaded from cache
	pkg.ObjFiles = nil
	pkg.ArchiveFile = ""
	pkg.CacheHit = false

	if ctx.tryLoadFromCache(pkg) {
		t.Error("should return false when ForceRebuild is enabled, even with valid cache")
	}

	if pkg.CacheHit {
		t.Error("CacheHit should remain false when ForceRebuild is enabled")
	}

	if pkg.ArchiveFile != "" {
		t.Error("ArchiveFile should not be populated when ForceRebuild is enabled")
	}
}

func TestSaveToCache_MainPackage(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	ctx := &context{
		conf: &packages.Config{},
		buildConf: &Config{
			Goos:   "darwin",
			Goarch: "arm64",
		},
		crossCompile: crosscompile.Export{
			LLVMTarget: "arm64-apple-darwin",
		},
	}

	pkg := &aPackage{
		Package: &packages.Package{
			PkgPath: "main",
			Name:    "main", // Main package
		},
		Fingerprint: "abc123",
		Manifest:    "test manifest",
	}

	// Should not error but also should not create cache
	if err := ctx.saveToCache(pkg); err != nil {
		t.Fatalf("saveToCache: %v", err)
	}

	// Check no cache was created
	cm := ctx.ensureCacheManager()
	paths := cm.PackagePaths("arm64-apple-darwin", "main", "abc123")
	if _, err := os.Stat(paths.Manifest); !os.IsNotExist(err) {
		t.Error("main package should not be cached")
	}
}

func TestStartCacheSaveNormalizesMainPackage(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	ctx := &context{
		conf: &packages.Config{},
		buildConf: &Config{
			Goos:   "darwin",
			Goarch: "arm64",
		},
		crossCompile: crosscompile.Export{
			LLVMTarget: "arm64-apple-darwin",
		},
	}

	objFile, err := os.CreateTemp(td, "main-*.o")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := objFile.WriteString("fake object file"); err != nil {
		t.Fatalf("write obj: %v", err)
	}
	if err := objFile.Close(); err != nil {
		t.Fatalf("close obj: %v", err)
	}

	pkg := &aPackage{
		Package: &packages.Package{
			PkgPath: "example.com/main",
			Name:    "main",
		},
		Fingerprint: "abc123",
		Manifest:    "test manifest",
		ObjFiles:    []string{objFile.Name()},
	}

	ctx.startCacheSave(pkg, false)
	if err := ctx.waitCacheSaves(false); err != nil {
		t.Fatalf("waitCacheSaves: %v", err)
	}
	if pkg.ArchiveFile == "" {
		t.Fatal("main package archive was not created")
	}
	if pkg.ObjFiles != nil {
		t.Fatalf("ObjFiles = %v, want nil after archiving", pkg.ObjFiles)
	}
	if _, err := os.Stat(pkg.ArchiveFile); err != nil {
		t.Fatalf("archive should exist: %v", err)
	}

	cm := ctx.ensureCacheManager()
	paths := cm.PackagePaths("arm64-apple-darwin", "example.com/main", "abc123")
	if _, err := os.Stat(paths.Manifest); !os.IsNotExist(err) {
		t.Fatal("main package should not publish a cache manifest")
	}
}

func TestSaveToCache_Success(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	ctx := &context{
		conf: &packages.Config{},
		buildConf: &Config{
			Goos:   "darwin",
			Goarch: "arm64",
		},
		crossCompile: crosscompile.Export{
			LLVMTarget: "arm64-apple-darwin",
		},
	}

	// Create a temporary .o file
	objFile, err := os.CreateTemp(td, "test-*.o")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	objFile.WriteString("fake object file")
	objFile.Close()

	pkg := &aPackage{
		Package: &packages.Package{
			PkgPath: "example.com/lib",
			Name:    "lib",
			GoFiles: []string{objFile.Name()}, // Add GoFiles for manifest generation
		},
		Fingerprint: "def456",
		Manifest: func() string {
			m := newManifestBuilder()
			m.env.Goos = "darwin"
			m.pkg.PkgPath = "example.com/lib"
			return m.Build()
		}(),
		ObjFiles: []string{objFile.Name()},
	}

	if err := ctx.saveToCache(pkg); err != nil {
		t.Fatalf("saveToCache: %v", err)
	}

	// Check cache was created
	cm := ctx.ensureCacheManager()
	paths := cm.PackagePaths("arm64-apple-darwin", "example.com/lib", "def456")

	// Check manifest contains original content and metadata in Package section
	content, err := readManifest(paths.Manifest)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if content != pkg.Manifest {
		t.Errorf("manifest without metadata should be written unchanged\ngot:\n%s\nwant:\n%s", content, pkg.Manifest)
	}
	data, err := decodeManifest(content)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}
	if data.Env.Goos != "darwin" {
		t.Errorf("manifest should contain original env content")
	}
	if data.Metadata != nil {
		t.Errorf("metadata should be empty when no link args/runtime flags")
	}

	// Check archive exists and the package now links from the cache archive.
	if _, err := os.Stat(paths.Archive); err != nil {
		t.Errorf("archive should exist: %v", err)
	}
	if pkg.ArchiveFile != paths.Archive {
		t.Fatalf("ArchiveFile = %q, want cache archive %q", pkg.ArchiveFile, paths.Archive)
	}
	if pkg.ObjFiles != nil {
		t.Fatalf("ObjFiles = %v, want nil after archiving", pkg.ObjFiles)
	}
}

func TestAppendManifestMetadataSimple(t *testing.T) {
	meta := &manifestMetadata{LinkArgs: []string{"-lm", "-Wl,--gc-sections"}, NeedRt: true, NeedPyInit: true}

	withDepsNoTrailingNewline := "env:\n    GOOS: linux\ndeps:\n    - id: example.com/dep"
	got, err := appendManifestMetadata(withDepsNoTrailingNewline, meta)
	if err != nil {
		t.Fatalf("appendManifestMetadata: %v", err)
	}
	metadataIdx := strings.Index(got, "\nmetadata:")
	depsIdx := strings.Index(got, "\ndeps:")
	if metadataIdx < 0 || depsIdx < 0 || metadataIdx > depsIdx {
		t.Fatalf("metadata should be inserted before deps:\n%s", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("missing trailing newline after appending to no-newline manifest: %q", got)
	}
	parsed, err := parseManifestMetadata(got)
	if err != nil {
		t.Fatalf("parse appended metadata: %v", err)
	}
	if strings.Join(parsed.LinkArgs, " ") != "-lm -Wl,--gc-sections" || !parsed.NeedRt || !parsed.NeedPyInit {
		t.Fatalf("metadata parse = %+v", parsed)
	}

	withoutDepsNoTrailingNewline := "env:\n    GOOS: linux"
	got, err = appendManifestMetadata(withoutDepsNoTrailingNewline, meta)
	if err != nil {
		t.Fatalf("appendManifestMetadata no deps: %v", err)
	}
	if !strings.Contains(got, "GOOS: linux\nmetadata:\n") {
		t.Fatalf("metadata should be appended after adding separator newline:\n%s", got)
	}
}

func TestSaveToCache_WithMetadata(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	ctx := &context{
		conf: &packages.Config{},
		buildConf: &Config{
			Goos:   "linux",
			Goarch: "amd64",
		},
		crossCompile: crosscompile.Export{
			LLVMTarget: "x86_64-unknown-linux-gnu",
		},
	}

	objFile, err := os.CreateTemp(td, "test-*.o")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	objFile.WriteString("fake object file")
	objFile.Close()

	m := newManifestBuilder()
	m.env.Goos = "linux"
	m.pkg.PkgPath = "example.com/metadata"
	m.deps = append(m.deps, depEntry{ID: "example.com/dep", Version: "v1.0.0"})
	pkg := &aPackage{
		Package: &packages.Package{
			PkgPath: "example.com/metadata",
			Name:    "metadata",
		},
		Fingerprint: "meta123",
		Manifest:    m.Build(),
		ObjFiles:    []string{objFile.Name()},
		LinkArgs:    []string{"-lm", "-lpthread"},
		NeedRt:      true,
		NeedPyInit:  true,
	}

	if err := ctx.saveToCache(pkg); err != nil {
		t.Fatalf("saveToCache: %v", err)
	}

	paths := ctx.ensureCacheManager().PackagePaths("x86_64-unknown-linux-gnu", "example.com/metadata", "meta123")
	content, err := readManifest(paths.Manifest)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	metadataIdx := strings.Index(content, "\nmetadata:")
	depsIdx := strings.Index(content, "\ndeps:")
	if metadataIdx < 0 {
		t.Fatalf("manifest should contain metadata section:\n%s", content)
	}
	if depsIdx >= 0 && metadataIdx > depsIdx {
		t.Fatalf("metadata section should be written before deps section:\n%s", content)
	}
	meta, err := parseManifestMetadata(content)
	if err != nil {
		t.Fatalf("parseManifestMetadata: %v", err)
	}
	if strings.Join(meta.LinkArgs, " ") != "-lm -lpthread" || !meta.NeedRt || !meta.NeedPyInit {
		t.Fatalf("metadata parse = %+v", meta)
	}
}

func TestGetLLVMVersion(t *testing.T) {
	oldDetect := detectLLVMVersionFunc
	llvmVersionCache = sync.Map{}
	llvmCompilerPathCache = sync.Map{}
	detectCalls := 0
	detectLLVMVersionFunc = func(*context) string {
		detectCalls++
		return "clang version test"
	}
	t.Cleanup(func() {
		llvmVersionCache = sync.Map{}
		llvmCompilerPathCache = sync.Map{}
		detectLLVMVersionFunc = oldDetect
	})

	ctx := &context{
		crossCompile: crosscompile.Export{},
	}

	v1 := ctx.getLLVMVersion()
	v2 := ctx.getLLVMVersion()
	if v1 != v2 {
		t.Error("getLLVMVersion should return cached value")
	}
	if v1 != "clang version test" {
		t.Fatalf("getLLVMVersion = %q, want injected version", v1)
	}
	if detectCalls != 1 {
		t.Fatalf("detectLLVMVersion called %d times, want 1", detectCalls)
	}
}

func TestGetLLVMVersionCachesAcrossContexts(t *testing.T) {
	oldDetect := detectLLVMVersionFunc
	llvmVersionCache = sync.Map{}
	llvmCompilerPathCache = sync.Map{}
	detectCalls := 0
	detectLLVMVersionFunc = func(*context) string {
		detectCalls++
		return "clang version test"
	}
	t.Cleanup(func() {
		llvmVersionCache = sync.Map{}
		llvmCompilerPathCache = sync.Map{}
		detectLLVMVersionFunc = oldDetect
	})

	ctx1 := &context{crossCompile: crosscompile.Export{CC: "test-clang"}}
	ctx2 := &context{crossCompile: crosscompile.Export{CC: "test-clang"}}
	if got := ctx1.getLLVMVersion(); got != "clang version test" {
		t.Fatalf("first LLVM version = %q, want injected version", got)
	}
	if got := ctx2.getLLVMVersion(); got != "clang version test" {
		t.Fatalf("second LLVM version = %q, want cached injected version", got)
	}
	if detectCalls != 1 {
		t.Fatalf("detectLLVMVersion called %d times, want 1", detectCalls)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func findPkg(pkgs []Package, pkgPath string) Package {
	for _, p := range pkgs {
		if p.PkgPath == pkgPath {
			return p
		}
	}
	return nil
}
