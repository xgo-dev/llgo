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

	"github.com/goplus/llgo/internal/crosscompile"
)

func needsWasmPostLink(conf *Config, target *crosscompile.Export) bool {
	return conf != nil && conf.BuildMode == BuildModeExe &&
		target != nil &&
		(target.WasmPostLink.Asyncify || target.WasmPostLink.TranslateToExnref)
}

func wasmPostLinkArgs(target *crosscompile.Export, input, output string, debug bool) []string {
	if target == nil ||
		(!target.WasmPostLink.Asyncify && !target.WasmPostLink.TranslateToExnref) {
		return nil
	}
	var args []string
	if target.WasmPostLink.Asyncify {
		args = append(args, "--asyncify")
	}
	// LLVM 19 lowers Wasm SjLj through the legacy EH encoding. When Asyncify
	// is enabled, translate only after instrumentation so the final module
	// uses the standardized exnref-based EH instructions.
	if target.WasmPostLink.TranslateToExnref {
		args = append(args, "--translate-to-exnref")
	}
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
	wasmOpt := os.Getenv("WASMOPT")
	if wasmOpt == "" {
		wasmOpt = "wasm-opt"
	}
	resolved, err := exec.LookPath(wasmOpt)
	if err != nil {
		return fmt.Errorf("WebAssembly post-link requires wasm-opt; install Binaryen or set WASMOPT: %w", err)
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
	)
	if ctx.shouldPrintCommands(verbose) {
		fmt.Fprintln(os.Stderr, resolved, args)
	}
	cmd := exec.Command(resolved, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wasm-opt post-link failed: %w", err)
	}
	if err := os.Rename(tmpName, output); err != nil {
		return err
	}
	return nil
}
