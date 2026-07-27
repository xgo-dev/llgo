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
		target != nil && target.WasmPostLink.Asyncify
}

func wasmPostLinkArgs(target *crosscompile.Export, input, output string, debug bool) []string {
	if target == nil || !target.WasmPostLink.Asyncify {
		return nil
	}
	// LLVM 19 lowers Wasm SjLj through the legacy EH encoding. Asyncify
	// understands that form; translate it only after instrumentation so the
	// final module uses the standardized exnref-based EH instructions.
	args := []string{"--asyncify", "--translate-to-exnref"}
	if debug {
		args = append(args, "-g")
	}
	return append(args, input, "-o", output)
}

func postLinkWasm(ctx *context, input, output string, verbose bool) error {
	wasmOpt := os.Getenv("WASMOPT")
	if wasmOpt == "" {
		wasmOpt = "wasm-opt"
	}
	resolved, err := exec.LookPath(wasmOpt)
	if err != nil {
		return fmt.Errorf("WebAssembly Asyncify requires wasm-opt; install Binaryen or set WASMOPT: %w", err)
	}

	outDir := filepath.Dir(output)
	tmp, err := os.CreateTemp(outDir, "."+filepath.Base(output)+".wasm-opt-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
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
		return fmt.Errorf("wasm-opt Asyncify failed: %w", err)
	}
	if err := os.Rename(tmpName, output); err != nil {
		return err
	}
	return nil
}
