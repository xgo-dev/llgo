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
	"testing"

	"github.com/goplus/llgo/internal/buildenv"
	"github.com/goplus/llgo/internal/crosscompile"
	"github.com/goplus/llgo/internal/lto"
	"github.com/goplus/llgo/internal/packages"
	llssa "github.com/goplus/llgo/ssa"
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

func TestDevLTOGlobalDCECollectFingerprint(t *testing.T) {
	td := t.TempDir()

	goFile := filepath.Join(td, "main.go")
	if err := os.WriteFile(goFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	pkg := func() *aPackage {
		return &aPackage{Package: &packages.Package{
			PkgPath: "test/pkg",
			GoFiles: []string{goFile},
		}}
	}
	ctx := func(conf *Config) *context {
		return &context{
			conf:      &packages.Config{},
			buildConf: conf,
			crossCompile: crosscompile.Export{
				LLVMTarget: "x86_64-unknown-linux",
			},
		}
	}

	withDCE := pkg()
	if err := ctx(&Config{Goos: "linux", Goarch: "amd64", LTO: lto.Full}).collectFingerprint(withDCE); err != nil {
		t.Fatal(err)
	}
	data, err := decodeManifest(withDCE.Manifest)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}
	if buildenv.Dev && (data.Common == nil || !data.Common.GoGlobalDCE) {
		t.Fatalf("manifest should contain GO_GLOBAL_DCE=true:\n%s", withDCE.Manifest)
	}
	if !buildenv.Dev && data.Common != nil && data.Common.GoGlobalDCE {
		t.Fatalf("non-dev builds should not contain GO_GLOBAL_DCE=true:\n%s", withDCE.Manifest)
	}

	withoutDCE := pkg()
	if err := ctx(&Config{
		Goos:               "linux",
		Goarch:             "amd64",
		LTO:                lto.Full,
		DisableGoGlobalDCE: true,
	}).collectFingerprint(withoutDCE); err != nil {
		t.Fatal(err)
	}
	if buildenv.Dev && withDCE.Fingerprint == withoutDCE.Fingerprint {
		t.Fatal("globaldce enabled and disabled builds should not share a cache fingerprint")
	}
	if !buildenv.Dev && withDCE.Fingerprint != withoutDCE.Fingerprint {
		t.Fatal("non-dev globaldce settings should not affect cache fingerprint")
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
		})
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

func TestDeclarationMetadataRoundTrip(t *testing.T) {
	prog := llssa.NewProgram(nil)
	prog.SetLinkname("example.com/p.F", "C.f")
	prog.SetTypeBackground("example.com/p.T", llssa.InC)
	prog.SetVarLocality("example.com/p.value", llssa.GoroutineLocal, true)
	prog.SetLocalInitFunc("example.com/p.value", "example.com/p.__llgo_local_init_0")
	prog.SetLocalEnsureFunc("example.com/p.value", "example.com/p.__llgo_local_init_0$ensure")

	encoded := declarationMetadataFor(prog.PackageDecls("example.com/p"))
	if len(encoded) != 3 {
		t.Fatalf("encoded declarations = %d, want 3", len(encoded))
	}
	for i := 1; i < len(encoded); i++ {
		if encoded[i-1].Name >= encoded[i].Name {
			t.Fatalf("declarations are not sorted: %+v", encoded)
		}
	}
	if got := declarationMetadataFor(nil); got != nil {
		t.Fatalf("declarationMetadataFor(nil) = %+v", got)
	}
	for _, declaration := range encoded {
		if declaration.Name == "example.com/p.value" && declaration.Locality != "gls" {
			t.Fatalf("encoded locality = %q, want gls", declaration.Locality)
		}
		if declaration.Name == "example.com/p.T" && declaration.Background != "c" {
			t.Fatalf("encoded background = %q, want c", declaration.Background)
		}
	}
	restored := llssa.NewProgram(nil)
	if !mergeCachedDeclarations(restored, "example.com/p", encoded) {
		t.Fatal("mergeCachedDeclarations rejected matching metadata")
	}
	if got, ok := restored.DeclInfo("example.com/p.value"); !ok || got.Locality != llssa.GoroutineLocal || got.InitFunc == "" || got.EnsureFunc == "" {
		t.Fatalf("restored locality = %+v, %v", got, ok)
	}
	restored.SetVarLocality("example.com/p.value", llssa.ThreadLocal, true)
	if mergeCachedDeclarations(restored, "example.com/p", encoded) {
		t.Fatal("mergeCachedDeclarations accepted conflicting source metadata")
	}
}

func TestDeclarationMetadataValidation(t *testing.T) {
	valid := declarationMetadata{Name: "example.com/p.value", Background: "go", Locality: "tls"}
	tests := []struct {
		name  string
		decls []declarationMetadata
	}{
		{name: "wrong package", decls: []declarationMetadata{{Name: "example.com/q.value"}}},
		{name: "duplicate", decls: []declarationMetadata{valid, valid}},
		{name: "background", decls: []declarationMetadata{{Name: valid.Name, Background: "invalid"}}},
		{name: "locality", decls: []declarationMetadata{{Name: valid.Name, Locality: "invalid"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prog := llssa.NewProgram(nil)
			if mergeCachedDeclarations(prog, "example.com/p", test.decls) {
				t.Fatalf("mergeCachedDeclarations accepted %+v", test.decls)
			}
		})
	}

	backgrounds := []llssa.Background{0, llssa.InGo, llssa.InC, llssa.InPython}
	for _, background := range backgrounds {
		name := declarationBackgroundName(background)
		if got, ok := parseDeclarationBackground(name); !ok || got != background {
			t.Fatalf("background %v round trip = %v, %v", background, got, ok)
		}
	}
	if name := declarationBackgroundName(llssa.Background(99)); name != "invalid:99" {
		t.Fatalf("invalid background name = %q", name)
	}
	if _, ok := parseDeclarationBackground("invalid:99"); ok {
		t.Fatal("invalid background parsed successfully")
	}

	localities := []llssa.Locality{llssa.LocalityNone, llssa.ThreadLocal, llssa.GoroutineLocal}
	for _, locality := range localities {
		name := declarationLocalityName(locality)
		if got, ok := parseDeclarationLocality(name); !ok || got != locality {
			t.Fatalf("locality %v round trip = %v, %v", locality, got, ok)
		}
	}
	if name := declarationLocalityName(llssa.Locality(99)); name != "invalid:99" {
		t.Fatalf("invalid locality name = %q", name)
	}
	if _, ok := parseDeclarationLocality("invalid:99"); ok {
		t.Fatal("invalid locality parsed successfully")
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

func TestTryLoadFromCache_MainPackage(t *testing.T) {
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
			Name:    "main",
		},
		Fingerprint: "abc123",
	}

	cm := ctx.ensureCacheManager()
	paths := cm.PackagePaths("arm64-apple-darwin", "main", "abc123")
	if err := cm.EnsureDir(paths); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Archive, []byte("stale main archive"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Manifest, []byte("metadata: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if ctx.tryLoadFromCache(pkg) {
		t.Fatal("main package should not be loaded from cache")
	}
	if pkg.CacheHit {
		t.Fatal("main package cache hit should remain false")
	}
	if pkg.ArchiveFile != "" {
		t.Fatalf("main package ArchiveFile = %q, want empty", pkg.ArchiveFile)
	}
}

func TestSaveToCache_Success(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	prog := llssa.NewProgram(nil)
	prog.SetVarLocality("example.com/lib.value", llssa.ThreadLocal, true)
	prog.SetLocalInitFunc("example.com/lib.value", "example.com/lib.__llgo_local_init_0")
	prog.SetLocalEnsureFunc("example.com/lib.value", "example.com/lib.__llgo_local_init_0$ensure")
	ctx := &context{
		conf: &packages.Config{},
		prog: prog,
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
	data, err := decodeManifest(content)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}
	if data.Env.Goos != "darwin" {
		t.Errorf("manifest should contain original env content")
	}
	if data.Metadata == nil || len(data.Metadata.Declarations) != 1 {
		t.Fatalf("cached declaration metadata = %+v", data.Metadata)
	}

	// Check archive exists
	if _, err := os.Stat(paths.Archive); err != nil {
		t.Errorf("archive should exist: %v", err)
	}

	newCachedPackage := func() *aPackage {
		return &aPackage{
			Package:     &packages.Package{PkgPath: "example.com/lib", Name: "lib"},
			Fingerprint: "def456",
		}
	}
	nilProgCtx := &context{
		conf:         &packages.Config{},
		buildConf:    ctx.buildConf,
		crossCompile: ctx.crossCompile,
		cacheManager: cm,
	}
	if nilProgCtx.tryLoadFromCache(newCachedPackage()) {
		t.Fatal("cache metadata loaded without a compiler program")
	}

	restored := llssa.NewProgram(nil)
	loadCtx := &context{
		conf:         &packages.Config{},
		prog:         restored,
		buildConf:    ctx.buildConf,
		crossCompile: ctx.crossCompile,
		cacheManager: cm,
	}
	loaded := newCachedPackage()
	if !loadCtx.tryLoadFromCache(loaded) || !loaded.CacheHit {
		t.Fatal("cache entry with declaration metadata was not loaded")
	}
	if got, ok := restored.DeclInfo("example.com/lib.value"); !ok || got.Locality != llssa.ThreadLocal || got.InitFunc == "" || got.EnsureFunc == "" {
		t.Fatalf("restored declaration metadata = %+v, %v", got, ok)
	}
}

func TestGetLLVMVersion(t *testing.T) {
	ctx := &context{
		crossCompile: crosscompile.Export{},
	}

	// First call should detect version
	v1 := ctx.getLLVMVersion()
	// May be empty if clang is not installed, but should not panic

	// Second call should return cached version
	v2 := ctx.getLLVMVersion()
	if v1 != v2 {
		t.Error("getLLVMVersion should return cached value")
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
