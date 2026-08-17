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

	"github.com/xgo-dev/llgo/internal/buildenv"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/meta"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
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

func TestDisablePackageCache(t *testing.T) {
	ctx := &context{}
	ctx.disablePackageCache(map[string]bool{
		"example.com/a": true,
		"example.com/b": false,
	})

	for _, id := range []string{"example.com/a", "example.com/b"} {
		if !ctx.packageCacheDisabled(id) {
			t.Fatalf("package cache for %q remains enabled", id)
		}
	}
	if ctx.packageCacheDisabled("example.com/c") {
		t.Fatal("unlisted package cache was disabled")
	}
}

func TestDisabledPackageCacheSkipsLoadAndSave(t *testing.T) {
	t.Setenv(llgoBuildCache, "1")
	pkg := &aPackage{Package: &packages.Package{ID: "example.com/disabled", PkgPath: "example.com/disabled"}}
	ctx := &context{buildConf: &Config{}, cacheDisabled: map[string]none{pkg.ID: {}}}
	if ctx.tryLoadFromCache(pkg) {
		t.Fatal("tryLoadFromCache loaded a disabled package")
	}
	if err := ctx.saveToCache(pkg); err != nil {
		t.Fatalf("saveToCache disabled package: %v", err)
	}

	ctx.cacheDisabled = nil
	ctx.buildConf.BuildMode = BuildModeCArchive
	if ctx.tryLoadFromCache(pkg) {
		t.Fatal("tryLoadFromCache loaded a C archive package")
	}
}

func TestCollectFingerprintDisablesCycles(t *testing.T) {
	pkg := &aPackage{Package: &packages.Package{ID: "example.com/cycle", PkgPath: "example.com/cycle"}}
	ctx := &context{fingerprinting: map[string]bool{pkg.ID: true}}
	if err := ctx.collectFingerprint(pkg); err != nil {
		t.Fatal(err)
	}
	if !ctx.packageCacheDisabled(pkg.ID) {
		t.Fatal("fingerprint cycle did not disable package cache")
	}
}

func TestCollectFingerprintIncludesEmitDWARF(t *testing.T) {
	td := t.TempDir()
	goFile := filepath.Join(td, "main.go")
	if err := os.WriteFile(goFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	newPkg := func() *aPackage {
		return &aPackage{Package: &packages.Package{
			PkgPath: "test/pkg",
			GoFiles: []string{goFile},
		}}
	}
	newContext := func(opts LinkOptions, debugInfo ...crosscompile.DebugInfoPolicy) *context {
		var policy crosscompile.DebugInfoPolicy
		if len(debugInfo) != 0 {
			policy = debugInfo[0]
		}
		return &context{
			conf:         &packages.Config{},
			buildConf:    &Config{Goos: "linux", Goarch: "amd64", BuildMode: BuildModeExe, LinkOptions: opts},
			crossCompile: crosscompile.Export{DebugInfo: policy},
		}
	}

	withDWARF := newPkg()
	if err := newContext(LinkOptions{}).collectFingerprint(withDWARF); err != nil {
		t.Fatal(err)
	}
	withoutDWARF := newPkg()
	if err := newContext(LinkOptions{DWARF: DWARFOmit}).collectFingerprint(withoutDWARF); err != nil {
		t.Fatal(err)
	}
	if withDWARF.Fingerprint == withoutDWARF.Fingerprint {
		t.Fatal("-w did not change the package fingerprint")
	}
	data, err := decodeManifest(withDWARF.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	if data.Common == nil || !data.Common.EmitDWARF {
		t.Fatalf("manifest does not contain EMIT_DWARF=true:\n%s", withDWARF.Manifest)
	}
	withoutData, err := decodeManifest(withoutDWARF.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	if withoutData.Common != nil && withoutData.Common.EmitDWARF {
		t.Fatalf("-w manifest unexpectedly contains EMIT_DWARF=true:\n%s", withoutDWARF.Manifest)
	}

	targetWithoutDWARF := newPkg()
	if err := newContext(LinkOptions{}, crosscompile.DebugInfoPolicy{AlwaysOmit: true}).collectFingerprint(targetWithoutDWARF); err != nil {
		t.Fatal(err)
	}
	if withDWARF.Fingerprint == targetWithoutDWARF.Fingerprint {
		t.Fatal("target baseline DWARF omission did not change the package fingerprint")
	}
	targetData, err := decodeManifest(targetWithoutDWARF.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	if targetData.Common != nil && targetData.Common.EmitDWARF {
		t.Fatalf("always-omit target manifest unexpectedly contains EMIT_DWARF=true:\n%s", targetWithoutDWARF.Manifest)
	}
}

func TestCollectFingerprintIncludesPCLNMode(t *testing.T) {
	t.Setenv(llgoFuncInfo, "")
	t.Setenv(llgoFuncInfoSites, "")
	td := t.TempDir()
	goFile := filepath.Join(td, "main.go")
	if err := os.WriteFile(goFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	newPkg := func() *aPackage {
		return &aPackage{Package: &packages.Package{
			PkgPath: "test/pkg",
			GoFiles: []string{goFile},
		}}
	}
	newContext := func(mode PCLNMode) *context {
		return &context{
			conf:      &packages.Config{},
			buildConf: &Config{Goos: "linux", Goarch: "amd64", BuildMode: BuildModeExe, PCLNMode: mode},
		}
	}

	tests := []struct {
		mode PCLNMode
		want string
	}{
		{mode: PCLNEmbedded, want: "embedded"},
		{mode: PCLNExternal, want: "external"},
		{mode: PCLNNone, want: "none"},
	}
	fingerprints := make(map[string]bool, len(tests))
	var embeddedFingerprint string
	for _, tt := range tests {
		pkg := newPkg()
		if err := newContext(tt.mode).collectFingerprint(pkg); err != nil {
			t.Fatal(err)
		}
		data, err := decodeManifest(pkg.Manifest)
		if err != nil {
			t.Fatal(err)
		}
		if data.Common == nil || data.Common.PCLNMode != tt.want {
			t.Fatalf("PCLN mode manifest = %+v, want %q", data.Common, tt.want)
		}
		if !strings.Contains(pkg.Manifest, "PCLN_MODE: "+tt.want) {
			t.Fatalf("manifest does not contain PCLN_MODE=%s:\n%s", tt.want, pkg.Manifest)
		}
		fingerprints[pkg.Fingerprint] = true
		if tt.mode == PCLNEmbedded {
			embeddedFingerprint = pkg.Fingerprint
		}
	}
	if len(fingerprints) != len(tests) {
		t.Fatalf("PCLN modes produced %d distinct fingerprints, want %d", len(fingerprints), len(tests))
	}

	defaultMode := newPkg()
	if err := newContext(PCLNMode(0)).collectFingerprint(defaultMode); err != nil {
		t.Fatal(err)
	}
	if defaultMode.Fingerprint != embeddedFingerprint {
		t.Fatal("zero-value PCLN mode and embedded mode produced different fingerprints")
	}
}

func TestCollectFingerprintCanonicalizesPCLNEnvironment(t *testing.T) {
	td := t.TempDir()
	goFile := filepath.Join(td, "main.go")
	if err := os.WriteFile(goFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	collect := func(conf Config, funcInfo, sites string) (*aPackage, manifestData) {
		t.Helper()
		t.Setenv(llgoFuncInfo, funcInfo)
		t.Setenv(llgoFuncInfoSites, sites)
		ctx := &context{
			conf:      &packages.Config{},
			buildConf: &conf,
		}
		pkg := &aPackage{Package: &packages.Package{
			PkgPath: "test/pkg",
			GoFiles: []string{goFile},
		}}
		if err := ctx.collectFingerprint(pkg); err != nil {
			t.Fatal(err)
		}
		data, err := decodeManifest(pkg.Manifest)
		if err != nil {
			t.Fatal(err)
		}
		return pkg, data
	}
	base := Config{Goos: "linux", Goarch: "amd64", BuildMode: BuildModeExe}

	legacyNone, legacyNoneData := collect(base, "0", "1")
	explicitNoneConf := base
	explicitNoneConf.PCLNMode = PCLNNone
	explicitNoneConf.PCLNModeSet = true
	explicitNone, explicitNoneData := collect(explicitNoneConf, "1", "0")
	if legacyNone.Fingerprint != explicitNone.Fingerprint {
		t.Fatal("legacy-disabled and typed none produced different fingerprints")
	}
	for _, data := range []manifestData{legacyNoneData, explicitNoneData} {
		if data.Common == nil || data.Common.PCLNMode != "none" {
			t.Fatalf("effective PCLN mode = %+v, want none", data.Common)
		}
		if _, ok := data.Env.Vars[llgoFuncInfo]; ok {
			t.Fatalf("manifest redundantly contains %s: %+v", llgoFuncInfo, data.Env.Vars)
		}
		if _, ok := data.Env.Vars[llgoFuncInfoSites]; ok {
			t.Fatalf("none manifest contains irrelevant %s: %+v", llgoFuncInfoSites, data.Env.Vars)
		}
	}

	explicitEmbeddedConf := base
	explicitEmbeddedConf.PCLNModeSet = true
	embeddedDefault, embeddedDefaultData := collect(explicitEmbeddedConf, "0", "")
	embeddedOne, embeddedOneData := collect(explicitEmbeddedConf, "1", "1")
	embeddedTrue, _ := collect(explicitEmbeddedConf, "", "true")
	legacyEmbedded, _ := collect(base, "on", "on")
	if embeddedDefault.Fingerprint != embeddedOne.Fingerprint ||
		embeddedDefault.Fingerprint != embeddedTrue.Fingerprint ||
		embeddedDefault.Fingerprint != legacyEmbedded.Fingerprint {
		t.Fatal("equivalent enabled PCLN inputs produced different fingerprints")
	}
	for _, data := range []manifestData{embeddedDefaultData, embeddedOneData} {
		if data.Common == nil || data.Common.PCLNMode != "embedded" {
			t.Fatalf("explicit embedded mode = %+v, want embedded", data.Common)
		}
		if got := data.Env.Vars[llgoFuncInfoSites]; got != "true" {
			t.Fatalf("%s = %q, want canonical true", llgoFuncInfoSites, got)
		}
		if _, ok := data.Env.Vars[llgoFuncInfo]; ok {
			t.Fatalf("manifest redundantly contains %s: %+v", llgoFuncInfo, data.Env.Vars)
		}
	}

	embeddedOff, embeddedOffData := collect(explicitEmbeddedConf, "", "0")
	if embeddedOff.Fingerprint == embeddedDefault.Fingerprint {
		t.Fatal("disabling LLGO_FUNCINFO_SITES did not change the fingerprint")
	}
	if got := embeddedOffData.Env.Vars[llgoFuncInfoSites]; got != "false" {
		t.Fatalf("%s = %q, want canonical false", llgoFuncInfoSites, got)
	}

	explicitExternalConf := base
	explicitExternalConf.PCLNMode = PCLNExternal
	explicitExternalConf.PCLNModeSet = true
	externalLegacyOff, externalLegacyOffData := collect(explicitExternalConf, "0", "1")
	externalLegacyOn, _ := collect(explicitExternalConf, "1", "true")
	if externalLegacyOff.Fingerprint != externalLegacyOn.Fingerprint {
		t.Fatal("LLGO_FUNCINFO changed an explicit external fingerprint")
	}
	if externalLegacyOffData.Common == nil || externalLegacyOffData.Common.PCLNMode != "external" {
		t.Fatalf("explicit external mode = %+v, want external", externalLegacyOffData.Common)
	}
	if _, ok := externalLegacyOffData.Env.Vars[llgoFuncInfo]; ok {
		t.Fatalf("external manifest redundantly contains %s: %+v", llgoFuncInfo, externalLegacyOffData.Env.Vars)
	}
}

func TestCollectFingerprintLocalContextMode(t *testing.T) {
	td := t.TempDir()
	goFile := filepath.Join(td, "state.go")
	if err := os.WriteFile(goFile, []byte("package state"), 0644); err != nil {
		t.Fatal(err)
	}
	newPackage := func() *aPackage {
		return &aPackage{Package: &packages.Package{
			ID:      "example.com/state",
			PkgPath: "example.com/state",
			GoFiles: []string{goFile},
		}}
	}
	newContext := func(prog llssa.Program) *context {
		return &context{
			conf:         &packages.Config{},
			prog:         prog,
			buildConf:    &Config{Goos: "linux", Goarch: "amd64"},
			crossCompile: crosscompile.Export{LLVMTarget: "x86_64-unknown-linux"},
		}
	}
	fingerprint := func(prog llssa.Program) (*aPackage, manifestData) {
		pkg := newPackage()
		if err := newContext(prog).collectFingerprint(pkg); err != nil {
			t.Fatal(err)
		}
		data, err := decodeManifest(pkg.Manifest)
		if err != nil {
			t.Fatal(err)
		}
		return pkg, data
	}

	plain, plainManifest := fingerprint(llssa.NewProgram(nil))
	nativeProg := llssa.NewProgram(nil)
	nativeProg.SetLocalityInfo("example.com/state.value", llssa.LocalityInfo{Locality: llssa.ThreadLocal})
	nativeProg.SetLocalStorage("example.com/state.value", llssa.LocalStorageNativeTLS)
	native, nativeManifest := fingerprint(nativeProg)
	contextProg := llssa.NewProgram(nil)
	contextProg.SetLocalityInfo("example.com/state.value", llssa.LocalityInfo{Locality: llssa.GoroutineLocal})
	contextProg.SetLocalStorage("example.com/state.value", llssa.LocalStoragePackage)
	withContext, contextManifest := fingerprint(contextProg)
	initializedProg := llssa.NewProgram(nil)
	initializedProg.SetLocalityInfo("example.com/state.value", llssa.LocalityInfo{
		Locality:       llssa.ThreadLocal,
		HasInitializer: true,
		InitFunc:       "example.com/state.__llgo_local_init_0",
		InitOrder:      1,
	})
	initializedProg.SetLocalStorage("example.com/state.value", llssa.LocalStorageNativeTLS)
	initialized, initializedManifest := fingerprint(initializedProg)

	if plain.Fingerprint != native.Fingerprint {
		t.Fatal("native TLS changed the package cache fingerprint")
	}
	if withContext.Fingerprint == plain.Fingerprint {
		t.Fatal("local-context and plain builds shared a package cache fingerprint")
	}
	if initialized.Fingerprint == plain.Fingerprint {
		t.Fatal("initialized native TLS and plain builds shared a package cache fingerprint")
	}
	if plainManifest.Common.LocalContext || nativeManifest.Common.LocalContext {
		t.Fatal("plain or native-TLS manifest enabled the local context")
	}
	if !contextManifest.Common.LocalContext {
		t.Fatal("context-backed locality was not recorded in the manifest")
	}
	if !initializedManifest.Common.LocalContext {
		t.Fatal("native TLS initializer failure storage was not recorded in the manifest")
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

func TestCollectFingerprintIncludesLTOPlugin(t *testing.T) {
	td := t.TempDir()
	goFile := filepath.Join(td, "main.go")
	if err := os.WriteFile(goFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	collect := func(plugin lto.PassPlugin) *aPackage {
		pkg := &aPackage{Package: &packages.Package{
			PkgPath: "test/pkg",
			GoFiles: []string{goFile},
		}}
		ctx := &context{
			conf: &packages.Config{},
			buildConf: &Config{
				Goos:      "linux",
				Goarch:    "amd64",
				LTO:       lto.Full,
				LTOPlugin: plugin,
			},
			crossCompile: crosscompile.Export{LLVMTarget: "x86_64-unknown-linux"},
		}
		if err := ctx.collectFingerprint(pkg); err != nil {
			t.Fatal(err)
		}
		return pkg
	}

	withoutPlugin := collect(lto.PassPlugin{})
	withPlugin := collect(lto.PassPlugin{Path: "/tmp/LLGOLTOPlugin.so"})
	withOtherPluginPath := collect(lto.PassPlugin{Path: "/opt/LLGOLTOPlugin.so"})

	if withoutPlugin.Fingerprint == withPlugin.Fingerprint {
		t.Fatal("plugin and non-plugin builds should not share a cache fingerprint")
	}
	if withPlugin.Fingerprint != withOtherPluginPath.Fingerprint {
		t.Fatal("plugin path should not affect package cache fingerprint")
	}

	data, err := decodeManifest(withPlugin.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	if data.Common == nil || !data.Common.EnableLTOPlugin {
		t.Fatalf("manifest should contain ENABLE_LTO_PLUGIN=true:\n%s", withPlugin.Manifest)
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
		Meta: func() *meta.PackageMeta { pm, _ := meta.NewBuilder().Build(); return pm }(),
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
			Goos:               "darwin",
			Goarch:             "arm64",
			CollectPackageMeta: true,
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
			Goos:               "darwin",
			Goarch:             "arm64",
			CollectPackageMeta: true,
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

	ctx := &context{
		conf: &packages.Config{},
		buildConf: &Config{
			Goos:               "darwin",
			Goarch:             "arm64",
			CollectPackageMeta: true,
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
		Meta:     func() *meta.PackageMeta { pm, _ := meta.NewBuilder().Build(); return pm }(),
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
	if data.Metadata != nil {
		t.Errorf("metadata should be empty when no link args/runtime flags")
	}

	// Check archive exists
	if _, err := os.Stat(paths.Archive); err != nil {
		t.Errorf("archive should exist: %v", err)
	}

	pm, err := meta.Open(paths.Meta)
	if err != nil {
		t.Errorf("meta should exist: %v", err)
	} else {
		defer pm.Close()
	}
}

func TestTryLoadFromCache_LoadsPackageMeta(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	ctx := &context{
		conf: &packages.Config{},
		buildConf: &Config{
			Goos:               "darwin",
			Goarch:             "arm64",
			CollectPackageMeta: true,
		},
		crossCompile: crosscompile.Export{
			LLVMTarget: "arm64-apple-darwin",
		},
	}

	objFile, err := os.CreateTemp(td, "test-*.o")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	objFile.WriteString("fake object file")
	objFile.Close()

	builder := meta.NewBuilder()
	main := builder.Sym("pkg.main")
	helper := builder.Sym("pkg.helper")
	builder.AddOrdinaryEdge(main, helper)

	pkg := &aPackage{
		Package: &packages.Package{
			PkgPath: "example.com/loadmeta",
			Name:    "loadmeta",
		},
		Fingerprint: "loadmeta123",
		Manifest: func() string {
			m := newManifestBuilder()
			m.env.Goos = "darwin"
			m.pkg.PkgPath = "example.com/loadmeta"
			return m.Build()
		}(),
		ObjFiles: []string{objFile.Name()},
	}
	pkg.Meta, _ = builder.Build()

	if err := ctx.saveToCache(pkg); err != nil {
		t.Fatalf("saveToCache: %v", err)
	}

	pkg.ObjFiles = nil
	pkg.ArchiveFile = ""
	pkg.CacheHit = false
	pkg.Meta = nil

	if !ctx.tryLoadFromCache(pkg) {
		t.Fatal("tryLoadFromCache = false, want true")
	}
	if pkg.Meta == nil {
		t.Fatal("Meta was not loaded from cache")
	}
	summary, err := meta.NewGlobalSummary([]*meta.PackageMeta{pkg.Meta})
	if err != nil {
		t.Fatalf("NewGlobalSummary: %v", err)
	}
	mainSym, ok := summary.LookupSymbol("pkg.main")
	if !ok {
		t.Fatal("pkg.main not found in cached metadata")
	}
	edges := summary.OrdinaryEdges(mainSym)
	if len(edges) != 1 || summary.SymbolName(edges[0]) != "pkg.helper" {
		t.Fatalf("cached metadata edge mismatch: %#v", edges)
	}
}

func TestTryLoadFromCacheRejectsBadMeta(t *testing.T) {
	td := t.TempDir()
	oldFunc := cacheRootFunc
	cacheRootFunc = func() string { return td }
	defer func() { cacheRootFunc = oldFunc }()

	ctx := &context{
		conf: &packages.Config{},
		buildConf: &Config{
			Goos:               "darwin",
			Goarch:             "arm64",
			CollectPackageMeta: true,
		},
		crossCompile: crosscompile.Export{
			LLVMTarget: "arm64-apple-darwin",
		},
	}

	pkg := &aPackage{
		Package: &packages.Package{
			PkgPath: "example.com/badmeta",
			Name:    "badmeta",
		},
		Fingerprint: "badmeta123",
	}
	cm := ctx.ensureCacheManager()
	paths := cm.PackagePaths("arm64-apple-darwin", "example.com/badmeta", "badmeta123")
	if err := cm.EnsureDir(paths); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Archive, []byte("archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newManifestBuilder()
	m.env.Goos = "darwin"
	m.pkg.PkgPath = "example.com/badmeta"
	if err := writeManifest(paths.Manifest, m.Build()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Meta, []byte("bad meta"), 0o644); err != nil {
		t.Fatal(err)
	}

	if ctx.tryLoadFromCache(pkg) {
		t.Fatal("tryLoadFromCache accepted invalid meta")
	}
}

func TestTryLoadFromCacheIgnoresMetaWhenPackageMetaDisabled(t *testing.T) {
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
			PkgPath: "example.com/nometa",
			Name:    "nometa",
		},
		Fingerprint: "nometa123",
	}
	cm := ctx.ensureCacheManager()
	paths := cm.PackagePaths("arm64-apple-darwin", "example.com/nometa", "nometa123")
	if err := cm.EnsureDir(paths); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Archive, []byte("archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newManifestBuilder()
	m.env.Goos = "darwin"
	m.pkg.PkgPath = "example.com/nometa"
	if err := writeManifest(paths.Manifest, m.Build()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Meta, []byte("bad meta"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !ctx.tryLoadFromCache(pkg) {
		t.Fatal("tryLoadFromCache rejected cache while deadcode drop is disabled")
	}
	if pkg.Meta != nil {
		t.Fatal("Meta should not be loaded while deadcode drop is disabled")
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
