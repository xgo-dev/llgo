//go:build !llgo

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
	"runtime"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func needsWasmPostLink(conf *Config, target *crosscompile.Export) bool {
	return conf != nil && conf.BuildMode == BuildModeExe &&
		target != nil && target.WasmPostLink.Asyncify
}

func wasmPostLinkArgs(target *crosscompile.Export, input, output string, debug bool, level optlevel.Level) []string {
	if target == nil || !target.WasmPostLink.Asyncify {
		return nil
	}
	// LLVM 19 lowers Wasm SjLj through the legacy EH encoding. Asyncify
	// understands that form; translate it only after instrumentation so the
	// final module uses the standardized exnref-based EH instructions.
	args := []string{"--asyncify", "--translate-to-exnref"}
	if level.IsValid() && level != optlevel.O0 {
		// Asyncify expands control flow and locals after LLVM has finished its
		// optimization pipeline. Run the matching Binaryen pipeline afterwards;
		// without it, ordinary test binaries can exceed Wasmtime's per-function
		// decoding limit even when the requested LLGo level is optimized.
		args = append(args, level.Flag())
	}
	if debug {
		args = append(args, "-g")
	}
	return append(args, input, "-o", output)
}

func wasmPreAsyncifyArgs(target *crosscompile.Export, input, output string, debug bool, level optlevel.Level) []string {
	if target == nil || !target.WasmPostLink.Asyncify || !level.IsValid() || level == optlevel.O0 {
		return nil
	}
	// Clang normally runs this optimization after wasm-ld. Keep it explicit so
	// LLGo can disable clang's implicit wasm-opt pass without changing the
	// established optimization order or binary size.
	args := []string{level.Flag()}
	if debug {
		args = append(args, "-g")
	}
	return append(args, input, "-o", output)
}

func prepareWasmLinkOutput(conf *Config, target *crosscompile.Export, output string) (string, error) {
	if !needsWasmPostLink(conf, target) {
		return output, nil
	}
	return createClosedTemp(
		filepath.Dir(output),
		"."+filepath.Base(output)+".linked-*",
	)
}

func cleanupWasmLinkOutput(input, output string) {
	if input != output {
		os.Remove(input)
	}
}

func publishWasmLinkOutput(ctx *context, input, output string, verbose bool) error {
	if input == output {
		return nil
	}
	return postLinkWasm(ctx, input, output, verbose)
}

func createClosedTemp(dir, pattern string) (string, error) {
	tmp, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

func postLinkWasm(ctx *context, input, output string, verbose bool) error {
	resolved, err := resolveWasmOpt()
	if err != nil {
		return fmt.Errorf("WebAssembly Asyncify requires wasm-opt; install Binaryen or set WASMOPT or EM_BINARYEN_ROOT: %w", err)
	}

	preAsyncifyArgs := wasmPreAsyncifyArgs(
		&ctx.crossCompile,
		input,
		input,
		shouldEmitDebugInfo(ctx.buildConf, &ctx.crossCompile),
		ctx.buildConf.OptLevel,
	)
	if preAsyncifyArgs != nil {
		if err := runWasmOpt(resolved, preAsyncifyArgs, verbose, ctx); err != nil {
			return fmt.Errorf("wasm-opt pre-Asyncify optimization failed using %s: %w", wasmOptIdentity(resolved), err)
		}
	}

	tmpName, err := createClosedTemp(
		filepath.Dir(output),
		"."+filepath.Base(output)+".wasm-opt-*",
	)
	if err != nil {
		return err
	}
	defer os.Remove(tmpName)

	args := wasmPostLinkArgs(
		&ctx.crossCompile,
		input,
		tmpName,
		shouldEmitDebugInfo(ctx.buildConf, &ctx.crossCompile),
		ctx.buildConf.OptLevel,
	)
	if err := runWasmOpt(resolved, args, verbose, ctx); err != nil {
		return fmt.Errorf("wasm-opt Asyncify failed using %s: %w", wasmOptIdentity(resolved), err)
	}
	if err := os.Rename(tmpName, output); err != nil {
		return err
	}
	return nil
}

func resolveWasmOpt() (string, error) {
	wasmOpt := os.Getenv("WASMOPT")
	if wasmOpt == "" {
		if root := os.Getenv("EM_BINARYEN_ROOT"); root != "" {
			name := "wasm-opt"
			if runtime.GOOS == "windows" {
				name += ".exe"
			}
			wasmOpt = filepath.Join(root, "bin", name)
		} else {
			wasmOpt = "wasm-opt"
		}
	}
	return exec.LookPath(wasmOpt)
}

func wasmOptIdentity(path string) string {
	output, err := exec.Command(path, "--version").CombinedOutput()
	version := strings.TrimSpace(string(output))
	if err != nil || version == "" {
		return path + " (version unavailable)"
	}
	return path + " (" + version + ")"
}

func runWasmOpt(resolved string, args []string, verbose bool, ctx *context) error {
	if ctx.shouldPrintCommands(verbose) {
		fmt.Fprintln(os.Stderr, resolved, args)
	}
	cmd := exec.Command(resolved, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
