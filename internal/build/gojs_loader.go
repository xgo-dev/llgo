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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/env"
)

const (
	goJSLoaderScriptName = "wasm_exec.js"
	goJSWasmMarkerPrefix = "// llgo-wasm: "
	// goJSLoaderDriverSentinel is any path with emcc's GoJS driver extension.
	// needsGoJSLoader uses conf.OutFile as the real gate for explicit JS outputs.
	goJSLoaderDriverSentinel = "app.mjs"
)

// Keep run/test glue as an ES module for emscripten-runner.mjs. Only a
// standalone GoJS build uses the synchronous factory consumed by wasm_exec.js.
func goJSLoaderLinkFlags(conf *Config, target *crosscompile.Export) []string {
	if !needsGoJSLoader(conf, target, goJSLoaderDriverSentinel) {
		return nil
	}
	return []string{
		"-sMODULARIZE=1",
		"-sEXPORT_ES6=0",
		"-sEXPORT_NAME=LLGoModule",
		"-sINVOKE_RUN=0",
		"-sEXPORTED_RUNTIME_METHODS=cwrap,allocateUTF8,stringToUTF8,UTF8ToString,FS,setValue,getValue,ENV,callMain",
	}
}

func needsGoJSLoader(conf *Config, target *crosscompile.Export, output string) bool {
	if target == nil {
		target = inferredGoJSLoaderTarget(conf)
	}
	return needsRawGoJSLoader(conf, output) && target != nil &&
		target.WasmProvider == crosscompile.WasmProviderGoJS
}

func inferredGoJSLoaderTarget(conf *Config) *crosscompile.Export {
	if conf == nil || conf.Target != "" || conf.Goos != "js" || conf.Goarch != "wasm" {
		return nil
	}
	return &crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}
}

func needsRawGoJSLoader(conf *Config, output string) bool {
	if conf == nil {
		return false
	}
	switch filepath.Ext(conf.OutFile) {
	case ".mjs", ".js", ".html":
		return false
	}
	return conf.Target == "" && conf.Mode == ModeBuild &&
		conf.BuildMode == BuildModeExe && conf.Goos == "js" && conf.Goarch == "wasm" &&
		filepath.Ext(output) == ".mjs"
}

func goJSLoaderSidecar(output string) string {
	return filepath.Join(filepath.Dir(output), goJSLoaderScriptName)
}

func goJSUniqueLoaderSidecar(output string) string {
	base := strings.TrimSuffix(filepath.Base(output), filepath.Ext(output))
	return filepath.Join(filepath.Dir(output), base+"_wasm_exec.js")
}

func publishGoJSLoader(ctx *context, driverOutput string, verbose bool) error {
	return publishGoJSLoaderFrom(ctx, driverOutput, verbose, env.LLGoROOT())
}

func publishGoJSLoaderFrom(ctx *context, driverOutput string, verbose bool, root string) error {
	if ctx == nil || !needsGoJSLoader(ctx.buildConf, &ctx.crossCompile, driverOutput) {
		return nil
	}
	if root == "" {
		return fmt.Errorf("copy %s: LLGO_ROOT is not set", goJSLoaderScriptName)
	}
	shared := goJSLoaderSidecar(driverOutput)
	unique := goJSUniqueLoaderSidecar(driverOutput)
	src := filepath.Join(root, "targets", goJSLoaderScriptName)
	protected := []string{driverOutput, debugWasmModulePath(ctx.buildConf, driverOutput)}
	if explicitOutputFile(ctx.buildConf, false) {
		protected = append(protected, ctx.buildConf.OutFile)
	}
	for _, output := range protected {
		for _, sidecar := range []string{shared, unique} {
			if err := checkGoJSCollision(output, sidecar); err != nil {
				return err
			}
		}
	}
	if err := checkGoJSCollision(shared, unique); err != nil {
		return err
	}
	if err := checkGoJSCollision(src, shared); err != nil {
		return err
	}
	if err := checkGoJSCollision(src, unique); err != nil {
		return err
	}
	// Validate the loader before publishing. Keep the driver's output intact
	// so a failed publication never removes the linked artifact.
	if info, err := os.Stat(src); err != nil {
		return fmt.Errorf("copy %s: %w", goJSLoaderScriptName, err)
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("copy %s: source is not a regular file", goJSLoaderScriptName)
	}
	dest, err := chooseGoJSLoaderDest(ctx, driverOutput, unique, shared)
	if err != nil {
		return err
	}
	factory, err := os.ReadFile(driverOutput)
	if err != nil {
		return fmt.Errorf("copy %s: %w", filepath.Base(dest), err)
	}
	loader, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("copy %s: %w", goJSLoaderScriptName, err)
	}
	combined := combineGoJSExec(goJSWasmModuleName(ctx, driverOutput), factory, loader)
	if err := writeFileAtomic(dest, combined); err != nil {
		return fmt.Errorf("copy %s: %w", filepath.Base(dest), err)
	}
	if err := os.Remove(driverOutput); err != nil {
		return fmt.Errorf("remove intermediate GoJS glue %s: %w", driverOutput, err)
	}
	if ctx.shouldPrintCommands(verbose) {
		fmt.Fprintf(os.Stderr, "copy %s + %s -> %s\n", driverOutput, src, dest)
	}
	return nil
}

func goJSWasmModuleName(ctx *context, driverOutput string) string {
	if ctx != nil && ctx.buildConf != nil {
		if out := ctx.buildConf.OutFile; strings.HasSuffix(out, ".wasm") {
			return filepath.Base(out)
		}
	}
	return strings.TrimSuffix(filepath.Base(driverOutput), filepath.Ext(driverOutput)) + ".wasm"
}

func combineGoJSExec(wasmName string, factory, loader []byte) []byte {
	var buf bytes.Buffer
	if len(factory) > 0 {
		buf.Write(factory)
		if factory[len(factory)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}
	buf.Write(loader)
	if len(loader) > 0 && loader[len(loader)-1] != '\n' {
		buf.WriteByte('\n')
	}
	return withGoJSWasmMarker(wasmName, buf.Bytes())
}

func withGoJSWasmMarker(wasmName string, data []byte) []byte {
	return append([]byte(goJSWasmMarkerPrefix+wasmName+"\n"), data...)
}

func goJSWasmMarker(data []byte) string {
	if !bytes.HasPrefix(data, []byte(goJSWasmMarkerPrefix)) {
		return ""
	}
	rest := data[len(goJSWasmMarkerPrefix):]
	if i := bytes.IndexByte(rest, '\n'); i >= 0 {
		rest = rest[:i]
	}
	return string(bytes.TrimSpace(rest))
}

func writeFileAtomic(dst string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), filepath.Base(dst)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dst)
}

func chooseGoJSLoaderDest(ctx *context, driverOutput, unique, shared string) (string, error) {
	existing, err := os.ReadFile(shared)
	if os.IsNotExist(err) {
		return shared, nil
	}
	if err != nil {
		return "", fmt.Errorf("copy %s: %w", goJSLoaderScriptName, err)
	}
	owner := goJSWasmMarker(existing)
	if owner == "" || owner == goJSWasmModuleName(ctx, driverOutput) {
		return shared, nil
	}
	return unique, nil
}

func checkGoJSCollision(output, sidecar string) error {
	same, err := sameFilePath(output, sidecar)
	if err != nil {
		return fmt.Errorf("publish GoJS sidecars: %w", err)
	}
	if same {
		return fmt.Errorf("publish GoJS sidecars: output %s collides with sidecar %s", output, sidecar)
	}
	return nil
}
