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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

func TestWasmFinalOptions(t *testing.T) {
	good := Config{BuildMode: BuildModeExe, Mode: ModeBuild, WasmOpt: WasmOptSize, DebugArtifactMode: DebugArtifactNone}
	for _, tc := range []struct {
		name   string
		change func(*Config)
		bad    bool
	}{
		{"size", func(*Config) {}, false},
		{"strip only", func(c *Config) { c.WasmOpt = WasmOptOff; c.WasmStrip = WasmStripAll }, false},
		{"external debug", func(c *Config) { c.DebugArtifactMode = DebugArtifactExternal; c.WasmStrip = WasmStripDebug }, false},
		{"invalid opt", func(c *Config) { c.WasmOpt = "fast" }, true},
		{"invalid strip", func(c *Config) { c.WasmStrip = "invalid" }, true},
		{"library", func(c *Config) { c.BuildMode = BuildModeCArchive }, true},
		{"generation", func(c *Config) { c.Mode = ModeGen }, true},
		{"embedded debug", func(c *Config) { c.DebugArtifactMode = DebugArtifactEmbedded; c.WasmStrip = WasmStripDebug }, true},
		{"firmware", func(c *Config) { c.OutFmts.Bin = true }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := good
			tc.change(&c)
			target := crosscompile.Export{WasmProfile: crosscompile.WasmProfileW32}
			if err := validateWasmFinalOptions(&c, &target); (err != nil) != tc.bad {
				t.Fatalf("validate=%v", err)
			}
		})
	}
	if err := validateWasmFinalOptions(&good, &crosscompile.Export{}); err == nil {
		t.Fatal("native accepted")
	}
	if err := validateWasmFinalOptions(&Config{}, &crosscompile.Export{}); err != nil {
		t.Fatal(err)
	}
	if needsWasmFinalization(nil) {
		t.Fatal("nil enabled")
	}
	target := crosscompile.Export{WasmProfile: crosscompile.WasmProfileJ32, WasmProvider: crosscompile.WasmProviderEmscripten, LDFLAGS: []string{"-sEXPORT_ALL=1"}}
	for range 2 {
		if err := validateWasmFinalOptions(&good, &target); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(target.LDFLAGS, []string{"-sEXPORT_ALL=1"}) {
		t.Fatal("validation mutated linker flags")
	}
	if !slices.Equal(wasmFinalLinkFlags(&good, &target), []string{"-sEXPORT_ALL=0"}) {
		t.Fatal("size link omitted export policy")
	}
	if len(wasmFinalLinkFlags(&Config{}, &target)) != 0 || len(wasmFinalLinkFlags(&good, &crosscompile.Export{})) != 0 {
		t.Fatal("changed other link modes")
	}
	for _, profile := range []crosscompile.WasmProfile{crosscompile.WasmProfileW32, crosscompile.WasmProfileJ32, crosscompile.WasmProfileJ64} {
		target.WasmProfile = profile
		args := wasmFinalArgs(&good, &target, "in", "out")
		if slices.Contains(args, "--all-features") || !slices.Contains(args, "-Oz") || !slices.Contains(args, "-g") {
			t.Fatalf("args=%v", args)
		}
		if slices.Contains(args, "--enable-memory64") != (profile == crosscompile.WasmProfileJ64) {
			t.Fatalf("memory64=%v", args)
		}
	}
}

func TestDoRejectsWasmFinalOptionsOnNativeTarget(t *testing.T) {
	for _, conf := range []Config{{WasmOpt: WasmOptSize}, {WasmStrip: WasmStripAll}} {
		// Reject the target before loading or compiling any input package.
		_, err := Do([]string{filepath.Join(t.TempDir(), "missing")}, &conf)
		if err == nil || !strings.Contains(err.Error(), "require a hosted WebAssembly executable") {
			t.Fatalf("native final pipeline: %v", err)
		}
	}
}

func TestWasmManifestDeterminism(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("app.wasm", []byte("\x00asm\x01\x00\x00\x00"))
	write("app.mjs", []byte("glue"))
	write("app.debug.wasm", []byte("DWARF"))
	ctx := &context{buildConf: &Config{PCLNMode: PCLNEmbedded, debugPointerSize: 4}, llvmVersion: "LLVM test", crossCompile: crosscompile.Export{WasmProfile: crosscompile.WasmProfileJ32, CC: filepath.Join(dir, "emcc")}}
	name := "app.wasm-manifest.json"
	run := func() ([]byte, wasmManifest) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, "app.wasm"))
		if err != nil {
			t.Fatal(err)
		}
		if err := writeWasmManifest(ctx, dir, "app.wasm", data, name, wasmManifestTool{Path: filepath.Join(dir, "wasm-opt"), Version: "132"}, []string{"-g"}, []string{"--sysroot=" + dir}, 100); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		var m wasmManifest
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		return raw, m
	}
	first, m := run()
	if m.Tool.Path != "wasm-opt" || m.LinkDriver != "emcc" || strings.Contains(string(first), dir) {
		t.Fatalf("manifest contains local tool paths: %s", first)
	}
	if len(m.Artifacts) != 3 || m.PointerBits != 32 || m.Optimization != WasmOptOff || m.Strip != WasmStripNone {
		t.Fatalf("manifest=%+v", m)
	}
	if m.Payload.Bytes != 12 || m.Debug.Bytes != 5 || m.DriverVersion != "LLVM test" || len(m.AllowedFeatures) == 0 || m.Features == nil {
		t.Fatalf("manifest totals/tools/features=%+v", m)
	}
	for _, a := range m.Artifacts {
		h, err := fileSHA256(filepath.Join(dir, a.Path))
		if err != nil || h != a.SHA256 || a.GzipBytes <= 0 {
			t.Fatalf("artifact=%+v, %v", a, err)
		}
		if a.Path == "app.debug.wasm" && a.Role != "debug" {
			t.Fatal("debug counted as deployment")
		}
	}
	second, _ := run()
	if string(first) != string(second) {
		t.Fatal("manifest is nondeterministic")
	}
	if err := os.Remove(filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
	write("app.mjs", []byte("changed"))
	_, changed := run()
	if m.BuildID == changed.BuildID {
		t.Fatal("glue change did not change identity")
	}
}

func TestPublishWasmArtifactsRollback(t *testing.T) {
	for _, failure := range []string{"", "app.wasm", "app.wasm-manifest.json"} {
		t.Run(failure, func(t *testing.T) {
			stage, dst := t.TempDir(), t.TempDir()
			names := []string{"app.mjs", "app.wasm", "app.wasm-manifest.json"}
			for _, name := range names {
				for _, dir := range []string{stage, dst} {
					text := "new "
					if dir == dst {
						text = "old "
					}
					if err := os.WriteFile(filepath.Join(dir, name), []byte(text+name), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			var published []string
			rename := func(from, to string) error {
				if filepath.Dir(from) == stage {
					published = append(published, filepath.Base(from))
					if filepath.Base(from) == failure {
						return errors.New("injected publish failure")
					}
				}
				return os.Rename(from, to)
			}
			err := publishWasmArtifacts(stage, dst, names[2], nil, rename)
			if (err != nil) != (failure != "") {
				t.Fatalf("publish=%v", err)
			}
			for _, name := range names {
				got, err := os.ReadFile(filepath.Join(dst, name))
				prefix := "new "
				if failure != "" {
					prefix = "old "
				}
				if err != nil || string(got) != prefix+name {
					t.Fatalf("%s=%q,%v", name, got, err)
				}
			}
			if failure == "" && published[len(published)-1] != names[2] {
				t.Fatalf("manifest not last: %v", published)
			}
		})
	}
}

func TestWasmFinalFeaturesRejectUnsupported(t *testing.T) {
	module := func(feature string) []byte {
		payload := append([]byte{15}, []byte("target_features")...)
		payload = append(payload, 1, '+', byte(len(feature)))
		payload = append(payload, feature...)
		return append([]byte("\x00asm\x01\x00\x00\x00"), append([]byte{0, byte(len(payload))}, payload...)...)
	}
	target := crosscompile.Export{WasmProfile: crosscompile.WasmProfileW32}
	for _, name := range []string{"atomics", "nontrapping-fptoint", "simd128", "reference-types"} {
		if err := validateWasmFinalFeatures(module(name), &target); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"multimemory", "memory64", "gc", "unknown"} {
		if err := validateWasmFinalFeatures(module(name), &target); err == nil || !strings.Contains(err.Error(), name) {
			t.Fatalf("%s=%v", name, err)
		}
	}
	target.WasmProvider = crosscompile.WasmProviderWASI
	if err := validateWasmFinalFeatures(module("multimemory"), &target); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(wasmFinalArgs(&Config{WasmOpt: WasmOptSize}, &target, "in", "out"), "--enable-multimemory") {
		t.Fatal("multimemory optimizer flag missing")
	}
	if err := validateWasmFinalFeatures([]byte("bad"), &target); err == nil {
		t.Fatal("invalid module accepted")
	}
}

func TestFinalizeWasmArtifacts(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "optimizer failure"}[fail], func(t *testing.T) {
			dir := t.TempDir()
			tool := writeWasmOptTestTool(t, t.TempDir())
			t.Setenv("WASMOPT", tool)
			t.Setenv("LLGO_TEST_FINAL_WASM", "1")
			t.Setenv("LLGO_TEST_LINKER_HELPER", "write-wasm")
			mode := "copy"
			if fail {
				mode = "fail"
			}
			t.Setenv("LLGO_TEST_WASM_OPT_HELPER", mode)
			out := &OutFmtDetails{Out: filepath.Join(dir, "app.wasm")}
			manifest := filepath.Join(dir, "app.wasm-manifest.json")
			stale := dwarfSidecarPath(out.Out)
			for _, path := range []string{out.Out, manifest, stale} {
				if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			ctx := &context{buildConf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm", Target: "wasi", WasmOpt: WasmOptSize, WasmStrip: WasmStripAll, DebugArtifactMode: DebugArtifactNone, PCLNMode: PCLNNone, debugPointerSize: 4}, llvmVersion: "test", crossCompile: crosscompile.Export{CC: os.Args[0], WasmProfile: crosscompile.WasmProfileW32, WasmProvider: crosscompile.WasmProviderWASI}}
			err := finalizeWasmArtifacts(ctx, &mainLinkPlan{outputPath: out.Out}, out, false)
			if (err != nil) != fail {
				t.Fatalf("finalize=%v", err)
			}
			if fail {
				for _, path := range []string{out.Out, manifest, stale} {
					raw, err := os.ReadFile(path)
					if err != nil || string(raw) != "old" {
						t.Fatalf("failed pipeline changed %s", path)
					}
				}
			} else {
				if out.WasmManifest != manifest {
					t.Fatalf("manifest=%q", out.WasmManifest)
				}
				if _, err := os.Stat(stale); !os.IsNotExist(err) {
					t.Fatal("stale DWARF remains")
				}
				raw, err := os.ReadFile(out.Out)
				if err != nil || string(raw) != "\x00asm\x01\x00\x00\x00" {
					t.Fatalf("final=%q,%v", raw, err)
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), ".llgo-") {
					t.Fatalf("temporary artifact remains: %s", e.Name())
				}
			}
		})
	}
}

func TestWasmStripOnlyPreservesEncoding(t *testing.T) {
	t.Setenv("WASMOPT", filepath.Join(t.TempDir(), "missing-wasm-opt"))
	t.Setenv("LLGO_TEST_FINAL_WASM", "1")
	t.Setenv("LLGO_TEST_LINKER_HELPER", "write-padded-wasm")
	t.Setenv("LLGO_TEST_WASM_OPT_HELPER", "rewrite-wasm")
	out := &OutFmtDetails{Out: filepath.Join(t.TempDir(), "app.wasm")}
	ctx := &context{buildConf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm", Target: "wasi", WasmOpt: WasmOptOff, WasmStrip: WasmStripAll, DebugArtifactMode: DebugArtifactNone, PCLNMode: PCLNNone, debugPointerSize: 4}, llvmVersion: "test", crossCompile: crosscompile.Export{CC: os.Args[0], WasmProfile: crosscompile.WasmProfileW32, WasmProvider: crosscompile.WasmProviderWASI}}
	if err := finalizeWasmArtifacts(ctx, &mainLinkPlan{outputPath: out.Out}, out, false); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out.Out)
	if err != nil || string(raw) != "\x00asm\x01\x00\x00\x00\x0a\x81\x80\x80\x80\x00\x00" {
		t.Fatalf("strip-only changed code section encoding: %x, %v", raw, err)
	}
}

func TestPortableSizeAndWasmStripPolicy(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		level                    optlevel.Level
		native, strip, keepDWARF bool
		wantOpt                  WasmOptMode
		wantStrip                WasmStripMode
	}{
		{name: "implicit target default"},
		{name: "Os retains compiler pipeline", level: optlevel.Os},
		{name: "explicit Oz", level: optlevel.Oz, wantOpt: WasmOptSize},
		{name: "O0", level: optlevel.O0},
		{name: "O2", level: optlevel.O2},
		{name: "native Os", level: optlevel.Os, native: true},
		{name: "native Oz and strip", level: optlevel.Oz, native: true, strip: true},
		{name: "Go s", strip: true, wantStrip: WasmStripDebug},
		{name: "Go s w=0", strip: true, keepDWARF: true, wantStrip: WasmStripSymbols},
		{name: "Oz and Go s", level: optlevel.Oz, strip: true, wantOpt: WasmOptSize, wantStrip: WasmStripDebug},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := &Config{Target: "wasi", Goos: "wasip1", Goarch: "wasm", OptLevel: tc.level}
			target := crosscompile.Export{WasmProfile: crosscompile.WasmProfileW32}
			if tc.native {
				input.Target, input.Goos, input.Goarch = "", "linux", "amd64"
				target.WasmProfile = crosscompile.WasmProfileNone
			}
			input.LinkOptions.OmitSymbolTable = tc.strip
			input.DebugArtifactMode = DebugArtifactNone
			if tc.keepDWARF {
				input.LinkOptions.DWARF = DWARFPreserve
				input.DebugArtifactMode = DebugArtifactEmbedded
			}
			conf, err := resolveBuildConfig(input)
			if err != nil {
				t.Fatal(err)
			}
			// Multi-package fallbacks resolve cloned configurations again.
			conf, err = resolveBuildConfig(conf)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateWasmFinalOptions(conf, &target); err != nil {
				t.Fatal(err)
			}
			if conf.WasmOpt != tc.wantOpt || conf.WasmStrip != tc.wantStrip {
				t.Fatalf("opt=%s strip=%s; want opt=%s strip=%s", conf.WasmOpt, conf.WasmStrip, tc.wantOpt, tc.wantStrip)
			}
			if tc.wantOpt == WasmOptSize && !slices.Contains(wasmFinalArgs(conf, &target, "in", "out"), tc.level.Flag()) {
				t.Fatal("final optimizer did not use the requested level")
			}
			if input.WasmOpt != "" || input.WasmStrip != "" {
				t.Fatal("mutated caller configuration")
			}
		})
	}
	conf := &Config{BuildMode: BuildModeExe, LinkOptions: LinkOptions{OmitSymbolTable: true}, WasmStrip: WasmStripNone}
	if err := validateWasmFinalOptions(conf, &crosscompile.Export{WasmProfile: crosscompile.WasmProfileW32}); err == nil {
		t.Fatal("explicit strip=none silently overrode -s")
	}
}

func TestRemoveStaleWasmManifest(t *testing.T) {
	out := &OutFmtDetails{Out: filepath.Join(t.TempDir(), "app.wasm")}
	path := wasmManifestPath(out.Out)
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleWasmManifest(&Config{Goarch: "arm64"}, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("native build removed unrelated manifest")
	}
	for i := 0; i < 2; i++ {
		if err := removeStaleWasmManifest(&Config{Goarch: "wasm"}, out); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("stale manifest remains")
		}
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleWasmManifest(&Config{Goarch: "wasm"}, out); err == nil {
		t.Fatal("removed directory occupying manifest path")
	}
}

func TestWasmFinalDebugPasses(t *testing.T) {
	for _, debug := range []DebugArtifactMode{DebugArtifactNone, DebugArtifactEmbedded, DebugArtifactExternal} {
		args := wasmFinalArgs(&Config{WasmOpt: WasmOptSize, DebugArtifactMode: debug}, &crosscompile.Export{}, "in", "out")
		for _, pass := range []string{"precompute", "precompute-propagate", "local-cse"} {
			if slices.Contains(args, "--skip-pass="+pass) != (debug != DebugArtifactNone) {
				t.Fatalf("debug=%v: %v", debug, args)
			}
		}
	}
}

func TestWasmFinalFailurePreservesArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name, linker, optimizer string
		configure               func(*context)
	}{
		{name: "missing tool"},
		{name: "version query", optimizer: "fail-version"},
		{name: "wrong version", optimizer: "wrong-version"},
		{name: "link failure", linker: "fail"},
		{name: "missing module", linker: "missing-output"},
		{name: "invalid module", linker: "invalid-wasm"},
		{name: "missing optimized module", optimizer: "missing-output"},
		{name: "strip-only invalid input", linker: "invalid-wasm", configure: func(ctx *context) { ctx.buildConf.WasmOpt = WasmOptOff }},
		{name: "invalid optimized metadata", optimizer: "invalid-wasm"},
		{name: "invalid debug output", optimizer: "invalid-wasm", configure: func(ctx *context) { ctx.buildConf.DebugArtifactMode = DebugArtifactExternal }},
		{name: "invalid manifest input", optimizer: "invalid-wasm", configure: func(ctx *context) { ctx.buildConf.WasmStrip = WasmStripNone }},
		{name: "unsupported PCLN", configure: func(ctx *context) { ctx.buildConf.PCLNMode = PCLNExternal }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			out := &OutFmtDetails{Out: filepath.Join(dir, "app.wasm")}
			paths := []string{out.Out, wasmManifestPath(out.Out), dwarfSidecarPath(out.Out)}
			for _, path := range paths {
				if err := os.WriteFile(path, []byte("previous artifact"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("WASMOPT", writeWasmOptTestTool(t, t.TempDir()))
			if tc.name == "missing tool" {
				t.Setenv("WASMOPT", filepath.Join(t.TempDir(), "missing"))
			}
			t.Setenv("LLGO_TEST_FINAL_WASM", "1")
			linker, optimizer := tc.linker, tc.optimizer
			if linker == "" {
				linker = "write-wasm"
			}
			if optimizer == "" {
				optimizer = "copy"
			}
			t.Setenv("LLGO_TEST_LINKER_HELPER", linker)
			t.Setenv("LLGO_TEST_WASM_OPT_HELPER", optimizer)
			ctx := &context{buildConf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm", Target: "wasi", WasmOpt: WasmOptSize, WasmStrip: WasmStripAll, DebugArtifactMode: DebugArtifactNone, PCLNMode: PCLNNone, debugPointerSize: 4}, llvmVersion: "test", crossCompile: crosscompile.Export{CC: os.Args[0], WasmProfile: crosscompile.WasmProfileW32, WasmProvider: crosscompile.WasmProviderWASI}}
			if tc.configure != nil {
				tc.configure(ctx)
			}
			if err := finalizeWasmArtifacts(ctx, &mainLinkPlan{outputPath: out.Out}, out, false); err == nil {
				t.Fatal("accepted failed pipeline")
			}
			for _, path := range paths {
				if data, err := os.ReadFile(path); err != nil || string(data) != "previous artifact" {
					t.Fatalf("changed %s: %q, %v", path, data, err)
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != len(paths) {
				t.Fatalf("temporary artifacts remain: %v, %v", entries, err)
			}
		})
	}
}

func TestPublishWasmArtifactsRefusesNonFiles(t *testing.T) {
	for _, obstruction := range []string{"directory", "symlink"} {
		t.Run(obstruction, func(t *testing.T) {
			stage, dst := t.TempDir(), t.TempDir()
			for _, dir := range []string{stage, dst} {
				if err := os.WriteFile(filepath.Join(dir, "app.mjs"), []byte(dir), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(stage, "app.wasm"), []byte("new"), 0600); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dst, "app.wasm")
			if obstruction == "directory" {
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Symlink(filepath.Join(dst, "app.mjs"), path); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			if err := publishWasmArtifacts(stage, dst, "app.wasm-manifest.json", nil, os.Rename); err == nil {
				t.Fatal("replaced non-file")
			}
			if data, err := os.ReadFile(filepath.Join(dst, "app.mjs")); err != nil || string(data) != dst {
				t.Fatalf("rollback lost original: %q, %v", data, err)
			}
		})
	}
}

func TestPublishWasmArtifactsRetainsFailedRestore(t *testing.T) {
	stage, dst := t.TempDir(), t.TempDir()
	for _, dir := range []string{stage, dst} {
		if err := os.WriteFile(filepath.Join(dir, "app.wasm"), []byte(dir), 0600); err != nil {
			t.Fatal(err)
		}
	}
	rename := func(from, to string) error {
		if filepath.Dir(from) != dst {
			return errors.New("disk unavailable")
		}
		return os.Rename(from, to)
	}
	err := publishWasmArtifacts(stage, dst, "app.wasm-manifest.json", nil, rename)
	if err == nil || !strings.Contains(err.Error(), "backup retained") {
		t.Fatalf("missing recovery location: %v", err)
	}
	backups, err := filepath.Glob(filepath.Join(dst, ".llgo-wasm-backup-*", "app.wasm"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("lost backup: %v, %v", backups, err)
	}
	if data, err := os.ReadFile(backups[0]); err != nil || string(data) != dst {
		t.Fatalf("lost original data: %q, %v", data, err)
	}
}

func TestWasmManifestRejectsInvalidArtifacts(t *testing.T) {
	ctx := &context{buildConf: &Config{}, llvmVersion: "test"}
	missing := filepath.Join(t.TempDir(), "missing")
	if err := writeWasmManifest(ctx, missing, "app.wasm", []byte("\x00asm\x01\x00\x00\x00"), "manifest", wasmManifestTool{}, nil, nil, 0); !os.IsNotExist(err) {
		t.Fatalf("missing artifact directory: %v", err)
	}
	for _, kind := range []string{"missing", "invalid module", "directory", "write failure"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			module := filepath.Join(dir, "app.wasm")
			if kind != "missing" {
				data := []byte("\x00asm\x01\x00\x00\x00")
				if kind == "invalid module" {
					data = []byte("invalid")
				}
				if err := os.WriteFile(module, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "directory" || kind == "write failure" {
				name := "unexpected"
				if kind == "write failure" {
					name = "app.wasm-manifest.json"
				}
				if err := os.Mkdir(filepath.Join(dir, name), 0700); err != nil {
					t.Fatal(err)
				}
			}
			data, _ := os.ReadFile(module)
			if err := writeWasmManifest(ctx, dir, "app.wasm", data, "app.wasm-manifest.json", wasmManifestTool{}, nil, nil, 0); err == nil {
				t.Fatal("accepted invalid artifacts")
			}
		})
	}
	for _, path := range []string{filepath.Join(t.TempDir(), "missing"), t.TempDir()} {
		if _, err := fileSHA256(path); err == nil {
			t.Fatal("hashed non-file")
		}
		if _, err := wasmArtifact(path); err == nil {
			t.Fatal("measured non-file")
		}
	}
	path := filepath.Join(t.TempDir(), "app.pclntab")
	if err := os.WriteFile(path, []byte("symbols"), 0600); err != nil {
		t.Fatal(err)
	}
	if a, err := wasmArtifact(path); err != nil || a.Role != "runtime-symbols" {
		t.Fatalf("runtime symbols misclassified: %+v, %v", a, err)
	}
}

func TestPublishWasmArtifactsMissingPaths(t *testing.T) {
	if err := publishWasmArtifacts(filepath.Join(t.TempDir(), "missing"), t.TempDir(), "manifest", nil, os.Rename); err == nil {
		t.Fatal("accepted missing stage")
	}
	if err := publishWasmArtifacts(t.TempDir(), filepath.Join(t.TempDir(), "missing"), "manifest", nil, os.Rename); err == nil {
		t.Fatal("accepted missing destination")
	}
	stage, dst := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(dst, "app.wasm"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "app.wasm"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	err := publishWasmArtifacts(stage, dst, "manifest", nil, func(string, string) error { return errors.New("backup unavailable") })
	if err == nil {
		t.Fatal("accepted backup failure")
	}
	if data, err := os.ReadFile(filepath.Join(dst, "app.wasm")); err != nil || string(data) != "old" {
		t.Fatalf("backup failure changed original: %q, %v", data, err)
	}
}

func TestFinalizeWasmPackagedOutputs(t *testing.T) {
	for _, name := range []string{"external DWARF", "Emscripten", "publication failure", "missing destination"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			out := &OutFmtDetails{Out: filepath.Join(dir, "app.wasm")}
			ctx := &context{buildConf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm", Target: "wasi", WasmOpt: WasmOptSize, WasmStrip: WasmStripDebug, DebugArtifactMode: DebugArtifactNone, PCLNMode: PCLNNone, debugPointerSize: 4}, llvmVersion: "test", crossCompile: crosscompile.Export{CC: os.Args[0], WasmProfile: crosscompile.WasmProfileW32, WasmProvider: crosscompile.WasmProviderWASI}}
			t.Setenv("WASMOPT", writeWasmOptTestTool(t, t.TempDir()))
			t.Setenv("LLGO_TEST_FINAL_WASM", "1")
			t.Setenv("LLGO_TEST_WASM_OPT_HELPER", "copy")
			t.Setenv("LLGO_TEST_LINKER_HELPER", "write-wasm")
			var stale string
			switch name {
			case "external DWARF":
				out.DWARF = dwarfSidecarPath(out.Out)
				ctx.buildConf.DebugArtifactMode = DebugArtifactExternal
				t.Setenv("LLGO_TEST_LINKER_HELPER", "write-debug-wasm")
			case "Emscripten":
				out.Out = filepath.Join(dir, "app.mjs")
				ctx.buildConf.Goos, ctx.buildConf.Target = "js", "emscripten"
				ctx.crossCompile.WasmProfile, ctx.crossCompile.WasmProvider = crosscompile.WasmProfileJ32, crosscompile.WasmProviderEmscripten
				t.Setenv("LLGO_TEST_LINKER_HELPER", "write-emscripten")
				stale = filepath.Join(dir, "app.js")
				if err := os.WriteFile(stale, []byte("old glue"), 0600); err != nil {
					t.Fatal(err)
				}
			case "publication failure":
				if err := os.Mkdir(out.Out, 0700); err != nil {
					t.Fatal(err)
				}
			case "missing destination":
				out.Out = filepath.Join(dir, "missing", "app.wasm")
			}
			err := finalizeWasmArtifacts(ctx, &mainLinkPlan{outputPath: out.Out}, out, false)
			if name == "publication failure" {
				if err == nil {
					t.Fatal("accepted unavailable output")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			artifacts, err := CollectArtifacts(ctx.buildConf, out)
			if err != nil {
				t.Fatal(err)
			}
			want := 3
			if name == "missing destination" {
				want = 2
			}
			if name == "Emscripten" {
				want = 4
			}
			if len(artifacts) != want {
				t.Fatalf("incomplete published set: %+v", artifacts)
			}
			if stale != "" {
				if _, err := os.Stat(stale); !os.IsNotExist(err) {
					t.Fatal("stale glue remains")
				}
			}
		})
	}
}

func TestWasmManifestUnreadableArtifactPreservesManifest(t *testing.T) {
	dir := t.TempDir()
	module, glue, manifest := filepath.Join(dir, "app.wasm"), filepath.Join(dir, "app.mjs"), filepath.Join(dir, "manifest")
	for path, data := range map[string]string{module: "\x00asm\x01\x00\x00\x00", glue: "glue", manifest: "previous manifest"} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(glue, 0); err != nil {
		t.Skipf("permissions unavailable: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(glue, 0600) })
	if f, err := os.Open(glue); err == nil {
		f.Close()
		t.Skip("filesystem grants reads despite mode 000")
	}
	ctx := &context{buildConf: &Config{}, llvmVersion: "test"}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeWasmManifest(ctx, dir, "app.wasm", data, "manifest", wasmManifestTool{}, nil, nil, 0); err == nil {
		t.Fatal("accepted unreadable artifact")
	}
	if data, err := os.ReadFile(manifest); err != nil || string(data) != "previous manifest" {
		t.Fatalf("replaced manifest on failure: %q, %v", data, err)
	}
}

func TestWasmManifestArguments(t *testing.T) {
	absolute := filepath.Join(t.TempDir(), "library")
	for _, tc := range []struct{ input, want string }{
		{absolute, "<path>/library"},
		{"--sysroot=" + absolute, "--sysroot=<path>/library"},
		{"-Wl,-rpath," + absolute, "-Wl,-rpath,<path>/library"},
		{"-L" + absolute, "-L<path>/library"},
		{"-I" + absolute, "-I<path>/library"},
		{"@" + absolute, "@<path>/library"},
		{"-Wl,--export=main", "-Wl,--export=main"},
		{"relative/path", "relative/path"},
	} {
		if got := wasmManifestArg(tc.input); got != tc.want {
			t.Fatalf("%q => %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestWasmManifestPrimaryDebugSuffix(t *testing.T) {
	dir := t.TempDir()
	data := []byte("\x00asm\x01\x00\x00\x00")
	if err := os.WriteFile(filepath.Join(dir, "app.debug.wasm"), data, 0600); err != nil {
		t.Fatal(err)
	}
	ctx := &context{buildConf: &Config{}, llvmVersion: "test"}
	if err := writeWasmManifest(ctx, dir, "app.debug.wasm", data, "manifest", wasmManifestTool{}, nil, nil, 0); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "manifest"))
	if err != nil {
		t.Fatal(err)
	}
	var m wasmManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.Payload.Bytes != int64(len(data)) || m.Debug.Bytes != 0 {
		t.Fatalf("primary module misclassified: %+v", m)
	}
}

func TestWasmManifestEffectiveLinkArguments(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CCFLAGS", "-DENV_CC=1")
	t.Setenv("LDFLAGS", "-Wl,--export=env_ld")
	t.Setenv("WASMOPT", filepath.Join(dir, "must-not-run"))
	t.Setenv("LLGO_TEST_LINKER_HELPER", "write-wasm")
	argsFile := filepath.Join(t.TempDir(), "link-args")
	t.Setenv("LINK_ARGS_FILE", argsFile)
	ctx := &context{buildConf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm", Target: "wasi", WasmStrip: WasmStripAll, DebugArtifactMode: DebugArtifactNone, PCLNMode: PCLNNone}, llvmVersion: "test", crossCompile: crosscompile.Export{Linker: os.Args[0], LinkerArgs: []string{"--driver-prefix"}, LDFLAGS: []string{"--configured-flag"}, WasmProfile: crosscompile.WasmProfileW32, WasmProvider: crosscompile.WasmProviderWASI}}
	out := &OutFmtDetails{Out: filepath.Join(dir, "app.wasm")}
	plan := &mainLinkPlan{outputPath: out.Out, linkArgs: []string{"-Wl,--export=user_export"}, linkInputs: []string{filepath.Join(t.TempDir(), "input.o")}}
	if err := finalizeWasmArtifacts(ctx, plan, out, false); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out.WasmManifest)
	if err != nil {
		t.Fatal(err)
	}
	var m wasmManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	actual := strings.Split(strings.TrimSpace(string(raw)), "\n")
	for i, arg := range actual {
		actual[i] = wasmManifestArg(arg)
	}
	if !slices.Equal(m.LinkFlags, actual) {
		t.Fatalf("manifest flags=%q; actual driver=%q", m.LinkFlags, actual)
	}
	for _, flag := range []string{"--driver-prefix", "-DENV_CC=1", "-Wl,--export=env_ld", "--configured-flag", "-Wl,--export=user_export", defaultWASIHeapFlag, "<path>/input.o"} {
		if !slices.Contains(m.LinkFlags, flag) {
			t.Errorf("missing %q in %q", flag, m.LinkFlags)
		}
	}
}

func TestFinalizeWasmSourceMaps(t *testing.T) {
	for _, strip := range []WasmStripMode{WasmStripNone, WasmStripSymbols, WasmStripDebug} {
		t.Run(string(strip), func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("WASMOPT", writeWasmOptTestTool(t, t.TempDir()))
			t.Setenv("LLGO_TEST_FINAL_WASM", "1")
			t.Setenv("LLGO_TEST_WASM_OPT_HELPER", "copy")
			t.Setenv("LLGO_TEST_LINKER_HELPER", "write-emscripten-map")
			argsFile := filepath.Join(t.TempDir(), "optimizer-args")
			t.Setenv("ARGS_FILE", argsFile)
			ctx := &context{buildConf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", Target: "emscripten", WasmOpt: WasmOptSize, WasmStrip: strip, DebugArtifactMode: DebugArtifactNone, PCLNMode: PCLNNone}, llvmVersion: "test", crossCompile: crosscompile.Export{CC: os.Args[0], WasmProfile: crosscompile.WasmProfileJ32, WasmProvider: crosscompile.WasmProviderEmscripten}}
			out := &OutFmtDetails{Out: filepath.Join(dir, "app.mjs")}
			module := filepath.Join(dir, "app.wasm")
			if err := os.WriteFile(module+".map", []byte("old map"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := finalizeWasmArtifacts(ctx, &mainLinkPlan{outputPath: out.Out}, out, false); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(module)
			if err != nil {
				t.Fatal(err)
			}
			url, err := wasmdebug.SourceMappingURL(raw)
			if err != nil {
				t.Fatal(err)
			}
			mapping, mapErr := os.ReadFile(module + ".map")
			if strip == WasmStripDebug {
				if url != "" || !os.IsNotExist(mapErr) {
					t.Fatalf("stripped map remains: %q, %v", url, mapErr)
				}
			} else if url != "https://debug.example/app.wasm.map" || mapErr != nil || !strings.Contains(string(mapping), `"sources":["input.go"]`) {
				t.Fatalf("map=%q URL=%q, %v", mapping, url, mapErr)
			}
			args, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatal(err)
			}
			for _, flag := range []string{"--input-source-map=", "--output-source-map=", "--output-source-map-url=https://debug.example/app.wasm.map"} {
				if !strings.Contains(string(args), flag) {
					t.Errorf("missing %s: %s", flag, args)
				}
			}
			manifest, err := os.ReadFile(out.WasmManifest)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(manifest), dir) {
				t.Fatal("manifest leaks staging paths")
			}
		})
	}
}

func TestFinalizeWasmSourceMapFailurePreservesOutputs(t *testing.T) {
	for _, mode := range []string{"missing-output-map", "invalid-output-map"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("WASMOPT", writeWasmOptTestTool(t, t.TempDir()))
			t.Setenv("LLGO_TEST_FINAL_WASM", "1")
			t.Setenv("LLGO_TEST_WASM_OPT_HELPER", mode)
			t.Setenv("LLGO_TEST_LINKER_HELPER", "write-emscripten-map")
			ctx := &context{buildConf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", Target: "emscripten", WasmOpt: WasmOptSize, DebugArtifactMode: DebugArtifactNone, PCLNMode: PCLNNone}, llvmVersion: "test", crossCompile: crosscompile.Export{CC: os.Args[0], WasmProfile: crosscompile.WasmProfileJ32, WasmProvider: crosscompile.WasmProviderEmscripten}}
			out := &OutFmtDetails{Out: filepath.Join(dir, "app.mjs")}
			names := []string{"app.mjs", "app.wasm", "app.wasm.map", "app.wasm-manifest.json"}
			for _, name := range names {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("old "+name), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := finalizeWasmArtifacts(ctx, &mainLinkPlan{outputPath: out.Out}, out, false); err == nil {
				t.Fatal("accepted missing or invalid optimized map")
			}
			for _, name := range names {
				raw, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil || string(raw) != "old "+name {
					t.Fatalf("failed pipeline changed %s: %q, %v", name, raw, err)
				}
			}
		})
	}
}

func TestWasmSymbolStrippingPreservesExternalSourceMaps(t *testing.T) {
	conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, DebugArtifactMode: DebugArtifactExternal, LinkOptions: LinkOptions{OmitSymbolTable: true, DWARF: DWARFPreserve}}
	if err := validateWasmFinalOptions(conf, &crosscompile.Export{WasmProfile: crosscompile.WasmProfileJ32, WasmProvider: crosscompile.WasmProviderEmscripten}); err != nil {
		t.Fatal(err)
	}
	if conf.WasmStrip != WasmStripSymbols {
		t.Fatalf("external -s -w=0 strip policy=%s", conf.WasmStrip)
	}
}
