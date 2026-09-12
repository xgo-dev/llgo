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
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/internal/env"
)

const wasmFSScriptName = "wasm_fs.js"

func emscriptenBrowserHostExt(output string) string {
	switch filepath.Ext(output) {
	case ".html", ".js", ".mjs":
		return filepath.Ext(output)
	default:
		return ""
	}
}

func needsEmscriptenBrowserHost(conf *Config, output string) bool {
	if conf == nil || conf.BuildMode != BuildModeExe || emscriptenBrowserHostExt(output) == "" {
		return false
	}
	if conf.Goos == "js" {
		return true
	}
	switch conf.Target {
	case "emscripten", "emscripten-memory64", "wasm":
		return true
	}
	return false
}

func publishEmscriptenBrowserHost(ctx *context, output string, verbose bool) error {
	return publishEmscriptenBrowserHostFrom(ctx, output, verbose, env.LLGoROOT())
}

func staleEmscriptenGluePath(driverOut string) string {
	ext := filepath.Ext(driverOut)
	base := strings.TrimSuffix(driverOut, ext)
	switch ext {
	case ".html", ".js":
		return base + ".mjs"
	case ".mjs":
		return base + ".js"
	default:
		return ""
	}
}

func removeStaleEmscriptenGlue(conf *Config, driverOut string) error {
	if !isEmscriptenJSTarget(conf) {
		return nil
	}
	stale := staleEmscriptenGluePath(driverOut)
	if stale == "" {
		return nil
	}
	if err := os.Remove(stale); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale Emscripten glue %s: %w", stale, err)
	}
	return nil
}

func publishEmscriptenBrowserHostFrom(ctx *context, output string, verbose bool, root string) error {
	if ctx == nil || !needsEmscriptenBrowserHost(ctx.buildConf, output) {
		return nil
	}
	if root == "" {
		return fmt.Errorf("copy %s: LLGO_ROOT is not set", wasmFSScriptName)
	}
	src := filepath.Join(root, "targets", wasmFSScriptName)
	if err := installEmscriptenBrowserHost(src, output); err != nil {
		return err
	}
	if ctx.shouldPrintCommands(verbose) {
		fmt.Fprintf(os.Stderr, "copy %s -> %s\n", src, filepath.Join(filepath.Dir(output), wasmFSScriptName))
	}
	return nil
}

func emscriptenBrowserHostPath(output string) string {
	return filepath.Join(filepath.Dir(output), wasmFSScriptName)
}

func emscriptenBrowserHostCollision(output, dst string) (bool, error) {
	same, err := sameFilePath(output, dst)
	if err != nil {
		return false, err
	}
	return same, nil
}

func installEmscriptenBrowserHost(src, output string) error {
	dst := emscriptenBrowserHostPath(output)
	same, err := emscriptenBrowserHostCollision(output, dst)
	if err != nil {
		return fmt.Errorf("copy %s: %w", wasmFSScriptName, err)
	}
	if same {
		return fmt.Errorf("copy %s: output %s collides with the browser host sidecar", wasmFSScriptName, output)
	}
	if err := copyFileAtomic(src, dst); err != nil {
		return fmt.Errorf("copy %s: %w", wasmFSScriptName, err)
	}
	if filepath.Ext(output) != ".html" {
		return nil
	}
	if err := injectLLGoFSScript(output); err != nil {
		return fmt.Errorf("insert %s into %s: %w", wasmFSScriptName, output, err)
	}
	return nil
}

// injectLLGoFSScript inserts the wasm_fs.js tag immediately before the first
// ES module script in toolchain-generated Emscripten HTML. It does not try to
// parse comments or string literals; emcc's shell always emits a module loader.
func injectLLGoFSScript(htmlPath string) error {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		return err
	}
	html := string(data)
	tag := `<script src="./` + wasmFSScriptName + `"></script>`
	if strings.Contains(html, tag) {
		return nil
	}
	insertAt := -1
	for _, marker := range []string{
		`<script type=module>`,
		`<script type="module">`,
		`<script type='module'>`,
		`<script type=module `,
		`<script type="module" `,
		`<script type='module' `,
	} {
		if i := strings.Index(html, marker); i >= 0 && (insertAt < 0 || i < insertAt) {
			insertAt = i
		}
	}
	if insertAt < 0 {
		return fmt.Errorf("no ES module script tag")
	}
	out := html[:insertAt] + tag + html[insertAt:]
	return os.WriteFile(htmlPath, []byte(out), 0o644)
}
