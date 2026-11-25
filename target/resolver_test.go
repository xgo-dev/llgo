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

package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func containsString(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

func repoTargetsDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "targets")
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Skipf("targets directory %s does not exist, skipping real targets test", dir)
	}
	return dir
}

func TestConfigBasics(t *testing.T) {
	config := &Config{
		Name:       "test",
		LLVMTarget: "arm-none-eabi",
		GOOS:       "linux",
		GOARCH:     "arm",
	}
	if config.IsEmpty() {
		t.Error("Config should not be empty when fields are set")
	}
	empty := &Config{}
	if !empty.IsEmpty() {
		t.Error("Empty config should report as empty")
	}
}

func TestRawConfigInheritance(t *testing.T) {
	raw := &RawConfig{
		Inherits: []string{"parent1", "parent2"},
		Config: Config{
			Name: "child",
		},
	}
	if !raw.HasInheritance() {
		t.Error("RawConfig should report having inheritance")
	}
	inherits := raw.GetInherits()
	if len(inherits) != 2 || inherits[0] != "parent1" || inherits[1] != "parent2" {
		t.Errorf("Expected inheritance list [parent1, parent2], got %v", inherits)
	}
	noInherit := &RawConfig{}
	if noInherit.HasInheritance() {
		t.Error("RawConfig with no inherits should not report having inheritance")
	}
}

func TestLoaderLoadRawAndInheritance(t *testing.T) {
	tempDir := t.TempDir()

	parentConfig := `{
		"llvm-target": "thumbv6m-unknown-unknown-eabi",
		"cpu": "cortex-m0plus",
		"goos": "linux",
		"goarch": "arm",
		"cflags": ["-Os"],
		"ldflags": ["--gc-sections"],
		"binary-format": "elf"
	}`
	childConfig := `{
		"inherits": ["parent"],
		"cpu": "cortex-m4",
		"build-tags": ["child"],
		"cflags": ["-O2"],
		"ldflags": ["-g"],
		"binary-format": "uf2"
	}`

	if err := os.WriteFile(filepath.Join(tempDir, "parent.json"), []byte(parentConfig), 0o644); err != nil {
		t.Fatalf("Failed to write parent config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "child.json"), []byte(childConfig), 0o644); err != nil {
		t.Fatalf("Failed to write child config: %v", err)
	}

	loader := NewLoader(tempDir)
	raw, err := loader.LoadRaw("parent")
	if err != nil {
		t.Fatalf("Failed to load raw config: %v", err)
	}
	if raw.Name != "parent" || raw.LLVMTarget != "thumbv6m-unknown-unknown-eabi" || raw.CPU != "cortex-m0plus" {
		t.Fatalf("unexpected raw config: %+v", raw)
	}

	config, err := loader.Load("child")
	if err != nil {
		t.Fatalf("Failed to load child config: %v", err)
	}
	if config.LLVMTarget != "thumbv6m-unknown-unknown-eabi" || config.GOOS != "linux" || config.GOARCH != "arm" {
		t.Errorf("unexpected inherited fields: %+v", config)
	}
	if config.CPU != "cortex-m4" || config.BinaryFormat != "uf2" {
		t.Errorf("unexpected overridden fields: %+v", config)
	}
	if len(config.CFlags) != 2 || config.CFlags[0] != "-Os" || config.CFlags[1] != "-O2" {
		t.Errorf("Expected merged cflags [-Os -O2], got %v", config.CFlags)
	}
	if len(config.LDFlags) != 2 || config.LDFlags[0] != "--gc-sections" || config.LDFlags[1] != "-g" {
		t.Errorf("Expected merged ldflags [--gc-sections -g], got %v", config.LDFlags)
	}
	if len(config.BuildTags) != 1 || config.BuildTags[0] != "child" {
		t.Errorf("Expected build-tags [child], got %v", config.BuildTags)
	}
}

func TestResolver(t *testing.T) {
	tempDir := t.TempDir()
	testConfig := `{
		"llvm-target": "wasm32-unknown-wasi",
		"cpu": "generic",
		"goos": "wasip1",
		"goarch": "wasm"
	}`
	if err := os.WriteFile(filepath.Join(tempDir, "wasi.json"), []byte(testConfig), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	resolver := NewResolver(tempDir)
	if resolver.GetTargetsDirectory() != tempDir {
		t.Fatalf("GetTargetsDirectory() = %q, want %q", resolver.GetTargetsDirectory(), tempDir)
	}
	config, err := resolver.Resolve("wasi")
	if err != nil {
		t.Fatalf("Failed to resolve target: %v", err)
	}
	if config.Name != "wasi" {
		t.Errorf("Expected name 'wasi', got '%s'", config.Name)
	}
	if !resolver.HasTarget("wasi") {
		t.Error("Resolver should report having 'wasi' target")
	}
	if resolver.HasTarget("nonexistent") {
		t.Error("Resolver should not report having 'nonexistent' target")
	}
	targets, err := resolver.ListAvailableTargets()
	if err != nil {
		t.Fatalf("Failed to list available targets: %v", err)
	}
	if len(targets) != 1 || targets[0] != "wasi" {
		t.Errorf("Expected targets [wasi], got %v", targets)
	}
}

func TestWASIMemoryConfig(t *testing.T) {
	dir := t.TempDir()
	parent := `{"goos":"wasip1","goarch":"wasm","wasm-profile":"w32","wasm-provider":"wasi",
		"wasm-initial-memory":16777216,"wasm-max-memory":536870912,"wasm-stack-size":10485760}`
	if err := os.WriteFile(filepath.Join(dir, "parent.json"), []byte(parent), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, fields, wantError string
	}{
		{"inherit", ``, ""},
		{"override", `,"wasm-initial-memory":8388608,"wasm-stack-size":1048576`, ""},
		{"initial alignment", `,"wasm-initial-memory":65537`, "wasm-initial-memory"},
		{"maximum alignment", `,"wasm-max-memory":65537`, "wasm-max-memory"},
		{"initial overflow", `,"wasm-initial-memory":4295032832`, "wasm-initial-memory"},
		{"maximum overflow", `,"wasm-max-memory":4295032832`, "wasm-max-memory"},
		{"stack alignment", `,"wasm-stack-size":17`, "wasm-stack-size"},
		{"stack overflow", `,"wasm-stack-size":4294967296`, "wasm-stack-size"},
		{"inherited conflict deferred to linker", `,"wasm-max-memory":8388608`, ""},
		{"negative", `,"wasm-initial-memory":-65536`, "cannot unmarshal"},
		{"other profile", `,"wasm-profile":"j32","wasm-provider":"emscripten"`, "w32/wasi"},
		{"other OS", `,"goos":"js"`, "w32/wasi"},
		{"other architecture", `,"goarch":"amd64"`, "w32/wasi"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := `{"inherits":["parent"]` + tc.fields + `}`
			if err := os.WriteFile(filepath.Join(dir, "child.json"), []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := NewResolver(dir).Resolve("child")
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			initial, stack, maximum := uint64(16<<20), uint64(10<<20), uint64(512<<20)
			if tc.name == "override" {
				initial, stack = 8<<20, 1<<20
			}
			if tc.name == "inherited conflict deferred to linker" {
				maximum = 8 << 20
			}
			if cfg.WasmInitialMemory != initial || cfg.WasmMaxMemory != maximum || cfg.WasmStackSize != stack {
				t.Fatalf("incorrect inherited memory settings: %+v", cfg)
			}
		})
	}
}

func TestResolverWithRealTargets(t *testing.T) {
	resolver := NewResolver(repoTargetsDir(t))
	targets, err := resolver.ListAvailableTargets()
	if err != nil {
		t.Fatalf("Failed to list real targets: %v", err)
	}
	if len(targets) == 0 {
		t.Fatal("expected non-empty target list")
	}
	for _, targetName := range []string{"wasi", "cortex-m", "rp2040"} {
		if resolver.HasTarget(targetName) {
			if _, err := resolver.Resolve(targetName); err != nil {
				t.Errorf("Failed to resolve known target %s: %v", targetName, err)
			}
		}
	}
}

func TestWebAssemblyProfileTargets(t *testing.T) {
	resolver := NewResolver(repoTargetsDir(t))
	tests := []struct {
		name, llvmTarget, goos, goarch, profile, provider string
		wantTags                                          []string
	}{
		{"emscripten", "wasm32-unknown-emscripten", "js", "wasm", "j32", "emscripten", []string{"llgo.wasm.emscripten"}},
		{"emscripten-memory64", "wasm64-unknown-emscripten", "js", "wasm", "j64", "emscripten", []string{"llgo.wasm.emscripten", "llgo.wasm.emscripten.memory64"}},
		{"wasm", "wasm32-unknown-emscripten", "js", "wasm", "j32", "emscripten", []string{"llgo.wasm.emscripten"}},
		{"wasi", "wasm32-unknown-wasip1", "wasip1", "wasm", "w32", "wasi", []string{"llgo.wasm.wasi"}},
		{"wasip1", "wasm32-unknown-wasip1", "wasip1", "wasm", "w32", "wasi", []string{"llgo.wasm.wasi"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := resolver.Resolve(test.name)
			if err != nil {
				t.Fatal(err)
			}
			if config.LLVMTarget != test.llvmTarget || config.GOOS != test.goos || config.GOARCH != test.goarch ||
				config.WasmProfile != test.profile || config.WasmProvider != test.provider {
				t.Fatalf("profile = LLVM %q, %s/%s, %q/%q; want LLVM %q, %s/%s, %q/%q",
					config.LLVMTarget, config.GOOS, config.GOARCH, config.WasmProfile, config.WasmProvider,
					test.llvmTarget, test.goos, test.goarch, test.profile, test.provider)
			}
			if config.CPU != "generic" {
				t.Errorf("CPU = %q, want generic", config.CPU)
			}
			for _, tag := range test.wantTags {
				if !containsString(config.BuildTags, tag) {
					t.Errorf("build tags %v do not contain %q", config.BuildTags, tag)
				}
			}
			if containsString(config.BuildTags, "tinygo.wasm") {
				t.Errorf("build tags %v retain the obsolete TinyGo compatibility tag", config.BuildTags)
			}
			if test.provider == "emscripten" {
				wantRunner := "emscripten-runner.mjs"
				if test.profile == "j64" {
					wantRunner = "emscripten-memory64-runner.mjs"
				}
				if !strings.Contains(config.Emulator, wantRunner) {
					t.Errorf("emulator %q does not instantiate Emscripten module output", config.Emulator)
				}
			}
			if test.provider == "wasi" {
				// Standard Wasm EH and SIMD must be enabled together with threads.
				if !strings.HasPrefix(config.Emulator, "wasmer ") || !strings.Contains(config.Emulator, "--enable-exceptions") {
					t.Errorf("WASI profile omits its threaded Wasmer runner: %+v", config)
				}
			}
		})
	}
}

func TestResolveAllRealTargets(t *testing.T) {
	resolver := NewResolver(repoTargetsDir(t))
	configs, err := resolver.ResolveAll()
	if err != nil {
		t.Fatalf("Failed to resolve all targets: %v", err)
	}
	for name, config := range configs {
		if config.Name != name {
			t.Errorf("Config name mismatch: key=%s, config.Name=%s", name, config.Name)
		}
		if config.IsEmpty() {
			t.Errorf("Config %s appears to be empty", name)
		}
	}
}
