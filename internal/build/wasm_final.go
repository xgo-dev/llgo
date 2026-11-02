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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

// WasmOptMode controls the optional pass after the toolchain's required
// lowering (including Emscripten Asyncify). It never changes runtime features.
type WasmOptMode string

const (
	WasmOptOff  WasmOptMode = "off"
	WasmOptSize WasmOptMode = "size"
)

// WasmStripMode is independent of PCLN and code optimization. All removes
// debug/name and producer metadata, preserving custom sections used at runtime.
type WasmStripMode string

const (
	WasmStripNone  WasmStripMode = "none"
	WasmStripDebug WasmStripMode = "debug"
	WasmStripAll   WasmStripMode = "all"
	// Symbols is the internal -s -w=0 policy: retain DWARF and source maps.
	WasmStripSymbols WasmStripMode = "symbols"
)

func needsWasmFinalization(conf *Config) bool {
	return conf != nil && (conf.WasmOpt == WasmOptSize || conf.WasmStrip == WasmStripDebug || conf.WasmStrip == WasmStripAll || conf.WasmStrip == WasmStripSymbols)
}

func validateWasmFinalOptions(conf *Config, target *crosscompile.Export) error {
	switch conf.WasmOpt {
	case "", WasmOptOff, WasmOptSize:
	default:
		return fmt.Errorf("invalid wasm-opt mode %q", conf.WasmOpt)
	}
	switch conf.WasmStrip {
	case "", WasmStripNone, WasmStripDebug, WasmStripAll, WasmStripSymbols:
	default:
		return fmt.Errorf("invalid wasm-strip mode %q", conf.WasmStrip)
	}
	hostedExe := conf.BuildMode == BuildModeExe && conf.Mode != ModeGen &&
		target.WasmProfile != crosscompile.WasmProfileNone && conf.OutFmts == (OutFmts{})
	if hostedExe {
		// The aggressive cross-platform size level also optimizes the final
		// Wasm executable. -Os and inferred target defaults retain their
		// established pipeline and tool dependencies.
		if conf.WasmOpt == "" && !conf.optLevelDefaulted && conf.OptLevel == optlevel.Oz {
			conf.WasmOpt = WasmOptSize
		}
		if conf.LinkOptions.OmitSymbolTable {
			switch conf.WasmStrip {
			case WasmStripNone:
				return fmt.Errorf("-wasm-strip=none conflicts with -ldflags=-s")
			case "":
				conf.WasmStrip = WasmStripDebug
				if conf.DebugArtifactMode == DebugArtifactEmbedded || conf.DebugArtifactMode == DebugArtifactExternal {
					conf.WasmStrip = WasmStripSymbols
				}
			}
		}
	}
	if !needsWasmFinalization(conf) {
		return nil
	}
	if conf.BuildMode != BuildModeExe || conf.Mode == ModeGen || target.WasmProfile == crosscompile.WasmProfileNone {
		return fmt.Errorf("final optimization and wasm-strip require a hosted WebAssembly executable")
	}
	if (conf.WasmStrip == WasmStripDebug || conf.WasmStrip == WasmStripAll) && conf.DebugArtifactMode == DebugArtifactEmbedded {
		return fmt.Errorf("wasm-strip conflicts with embedded DWARF; use -debug-artifact=external or none")
	}
	if conf.OutFmts != (OutFmts{}) {
		return fmt.Errorf("final optimization and wasm-strip do not support firmware output conversion")
	}
	return nil
}

func wasmFinalLinkFlags(conf *Config, target *crosscompile.Export) []string {
	if conf.WasmOpt == WasmOptSize && target.WasmProvider == crosscompile.WasmProviderEmscripten {
		// Keep explicit runtime methods and user/native exports. Size builds
		// expose that ABI instead of every incidental JS implementation symbol.
		return []string{"-sEXPORT_ALL=0"}
	}
	return nil
}

func validateWasmFinalFeatures(module []byte, target *crosscompile.Export) error {
	features, err := wasmdebug.TargetFeatures(module)
	if err != nil {
		return err
	}
	allowed := wasmFinalFeatures(target)
	for _, feature := range features {
		name := feature
		switch name {
		case "atomics":
			name = "threads"
		case "nontrapping-fptoint":
			name = "nontrapping-float-to-int"
		case "simd128":
			name = "simd"
		}
		if !slices.Contains(allowed, name) {
			return fmt.Errorf("Wasm feature %q is outside the %s target allowlist", feature, target.WasmProfile)
		}
	}
	return nil
}

// These are the capabilities supported by the hosted runtime/toolchain ABI.
// Never use --all-features: it permits instructions outside the target contract.
func wasmFinalFeatures(target *crosscompile.Export) []string {
	features := []string{"threads", "mutable-globals", "nontrapping-float-to-int", "simd", "bulk-memory", "sign-ext", "exception-handling", "reference-types", "multivalue", "bulk-memory-opt", "call-indirect-overlong", "extended-const"}
	if target.WasmProvider == crosscompile.WasmProviderWASI {
		features = append(features, "multimemory")
	}
	if target.WasmProfile == crosscompile.WasmProfileJ64 {
		features = append(features, "memory64")
	}
	return features
}

func wasmFinalArgs(conf *Config, target *crosscompile.Export, input, output string) []string {
	args := []string{"--mvp-features"}
	for _, f := range wasmFinalFeatures(target) {
		args = append(args, "--enable-"+f)
	}
	if conf.WasmOpt == WasmOptSize {
		args = append(args, optlevel.Oz.Flag())
		if conf.DebugArtifactMode == DebugArtifactEmbedded || conf.DebugArtifactMode == DebugArtifactExternal {
			// Binaryen 132 loses surviving Go call-site line mappings in these
			// passes. Keep the rest of -Oz, including its DWARF compatibility
			// checks, until those passes preserve the debugger contract too.
			args = append(args, "--skip-pass=precompute", "--skip-pass=precompute-propagate", "--skip-pass=local-cse")
		}
	}
	// Keep names and DWARF until debugger packaging and explicit stripping.
	return append(args, "-g", input, "-o", output)
}

func finalWasmTool() (wasmManifestTool, error) {
	path, err := resolveWasmOpt()
	if err != nil {
		return wasmManifestTool{}, fmt.Errorf("final Wasm pipeline requires LLGo Binaryen llgo-v132.3; set WASMOPT or EM_BINARYEN_ROOT: %w", err)
	}
	version, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		return wasmManifestTool{}, fmt.Errorf("query wasm-opt version: %w", err)
	}
	// LLGo's patched release keeps Binaryen's upstream version string. Record
	// the executable digest as well; version alone cannot identify a fork.
	if strings.TrimSpace(string(version)) != "wasm-opt version 132" {
		return wasmManifestTool{}, fmt.Errorf("final Wasm pipeline requires Binaryen 132 (LLGo llgo-v132.3), found %q", strings.TrimSpace(string(version)))
	}
	digest, err := fileSHA256(path)
	return wasmManifestTool{Path: path, Version: strings.TrimSpace(string(version)), SHA256: digest}, err
}

// Keep the complete output set private until optimization, validation,
// debugger packaging and hashing succeed. Basenames stay unchanged so emcc's
// JS/worker URLs remain valid after publication. The manifest is published last.
func finalizeWasmArtifacts(ctx *context, plan *mainLinkPlan, out *OutFmtDetails, verbose bool) error {
	// Go-compatible stripping uses the built-in parser and needs no Binaryen.
	var tool wasmManifestTool
	if ctx.buildConf.WasmOpt == WasmOptSize {
		var err error
		tool, err = finalWasmTool()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(out.Out), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	dir, err := os.MkdirTemp(filepath.Dir(out.Out), ".llgo-wasm-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	staged := *out
	staged.Out = filepath.Join(dir, filepath.Base(out.Out))
	if out.DWARF != "" {
		staged.DWARF = filepath.Join(dir, filepath.Base(out.DWARF))
	}
	localPlan := *plan
	localPlan.outputPath = staged.Out
	if err := executeMainLink(ctx, &localPlan, verbose); err != nil {
		return err
	}
	module := debugWasmModulePath(ctx.buildConf, staged.Out)
	before, err := os.Stat(module)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(module)
	if err != nil {
		return err
	}
	if err := validateWasmFinalFeatures(data, &ctx.crossCompile); err != nil {
		return err
	}
	var args []string
	if ctx.buildConf.WasmOpt == WasmOptSize {
		tmp := module + ".optimized"
		args = wasmFinalArgs(ctx.buildConf, &ctx.crossCompile, module, tmp)
		args, err = wasmFinalSourceMapArgs(data, module, tmp, args)
		if err != nil {
			return err
		}
		if err := runWasmOpt(tool.Path, args, verbose, ctx); err != nil {
			return fmt.Errorf("final Wasm optimization failed: %w", err)
		}
		if _, err := os.Stat(module + ".map"); err == nil {
			if err := replaceFile(tmp+".map", module+".map"); err != nil {
				return fmt.Errorf("publish optimized source map: %w", err)
			}
		}
		if err := replaceFile(tmp, module); err != nil {
			return err
		}
		args = args[:len(args)-3]
		for i, arg := range args {
			args[i] = wasmManifestArg(arg)
		}
	}
	if err := finalizeRuntimePCLN(ctx, &staged, verbose); err != nil {
		return err
	}
	if err := finalizeDebugArtifact(ctx.buildConf, &staged, verbose); err != nil {
		return err
	}
	data, err = os.ReadFile(module)
	if err != nil {
		return err
	}
	if ctx.buildConf.WasmStrip != "" && ctx.buildConf.WasmStrip != WasmStripNone {
		if ctx.buildConf.WasmStrip == WasmStripSymbols {
			data, err = wasmdebug.StripNames(data)
		} else {
			data, err = wasmdebug.StripMetadata(data, ctx.buildConf.WasmStrip == WasmStripAll)
		}
		if err != nil {
			return err
		}
		if ctx.buildConf.WasmStrip != WasmStripSymbols {
			if err := os.Remove(module + ".map"); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		if err := os.WriteFile(module, data, 0o644); err != nil {
			return err
		}
	}
	if err := relocateWasmSourceMap(module+".map", filepath.Dir(out.Out)); err != nil {
		return err
	}
	manifestName := filepath.Base(wasmManifestPath(module))
	linkFlags := ctx.effectiveLinkArgs
	if err := writeWasmManifest(ctx, dir, filepath.Base(module), data, manifestName, tool, args, linkFlags, before.Size()); err != nil {
		return err
	}
	var remove []string
	if _, err := os.Stat(module + ".map"); os.IsNotExist(err) {
		remove = append(remove, filepath.Base(module)+".map")
	} else if err != nil {
		return err
	}
	if ctx.buildConf.PCLNMode != PCLNExternal {
		remove = append(remove, filepath.Base(pclnSidecarPath(staged.Out)))
	}
	if ctx.buildConf.DebugArtifactMode != DebugArtifactExternal {
		remove = append(remove, filepath.Base(dwarfSidecarPath(module)))
	}
	driverOut := emscriptenDriverOutput(ctx.buildConf, staged.Out)
	if needsEmscriptenBrowserHost(ctx.buildConf, driverOut) {
		if stale := staleEmscriptenGluePath(driverOut); stale != "" {
			remove = append(remove, filepath.Base(stale))
		}
	}
	if err := publishWasmArtifacts(dir, filepath.Dir(out.Out), manifestName, remove, os.Rename); err != nil {
		return err
	}
	out.WasmManifest = filepath.Join(filepath.Dir(out.Out), manifestName)
	return nil
}

func wasmManifestPath(module string) string {
	return strings.TrimSuffix(module, ".wasm") + ".wasm-manifest.json"
}

func removeStaleWasmManifest(conf *Config, out *OutFmtDetails) error {
	if conf.Goarch != "wasm" || out.Out == "" {
		return nil
	}
	path := wasmManifestPath(debugWasmModulePath(conf, out.Out))
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to remove non-file Wasm manifest %s", path)
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	out.WasmManifest = ""
	return nil
}

// Source maps describe code offsets and must be regenerated along with the
// optimized module. Keep the driver's URL, including any source-map base URL.
func wasmFinalSourceMapArgs(data []byte, module, output string, args []string) ([]string, error) {
	url, err := wasmdebug.SourceMappingURL(data)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(module + ".map")
	if os.IsNotExist(err) && url == "" {
		return args, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read linked Wasm source map: %w", err)
	}
	if url == "" {
		url = filepath.Base(module) + ".map"
	}
	mapArgs := []string{"--input-source-map=" + module + ".map", "--output-source-map=" + output + ".map", "--output-source-map-url=" + url}
	return slices.Concat(args[:len(args)-3], mapArgs, args[len(args)-3:]), nil
}
