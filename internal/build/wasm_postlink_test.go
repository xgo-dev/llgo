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
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/crosscompile"
)

func wasmPostLinkTestContext() *context {
	return &context{
		buildConf: &Config{LinkOptions: LinkOptions{DWARF: DWARFOmit}},
		crossCompile: crosscompile.Export{
			WasmPostLink: crosscompile.WasmPostLink{
				Asyncify:          true,
				TranslateToExnref: true,
			},
		},
	}
}

func writeWasmOptTestTool(t *testing.T, dir, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	tool := filepath.Join(dir, "wasm-opt")
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return tool
}

func TestWasmPostLinkArgs(t *testing.T) {
	target := &crosscompile.Export{WasmPostLink: crosscompile.WasmPostLink{
		Asyncify:          true,
		TranslateToExnref: true,
	}}
	if got, want := wasmPostLinkArgs(target, "in.wasm", "out.wasm", false),
		[]string{"--asyncify", "--translate-to-exnref", "in.wasm", "-o", "out.wasm"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wasmPostLinkArgs() = %v, want %v", got, want)
	}
	if got, want := wasmPostLinkArgs(target, "in.wasm", "out.wasm", true),
		[]string{"--asyncify", "--translate-to-exnref", "-g", "in.wasm", "-o", "out.wasm"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wasmPostLinkArgs(debug) = %v, want %v", got, want)
	}
	if got := wasmPostLinkArgs(&crosscompile.Export{}, "in", "out", false); got != nil {
		t.Fatalf("wasmPostLinkArgs(disabled) = %v, want nil", got)
	}
	target.WasmPostLink.Asyncify = false
	if got, want := wasmPostLinkArgs(target, "in.wasm", "out.wasm", false),
		[]string{"--translate-to-exnref", "in.wasm", "-o", "out.wasm"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wasmPostLinkArgs(translate only) = %v, want %v", got, want)
	}
}

func TestNeedsWasmPostLink(t *testing.T) {
	target := &crosscompile.Export{WasmPostLink: crosscompile.WasmPostLink{Asyncify: true}}
	tests := []struct {
		name string
		conf *Config
		want bool
	}{
		{name: "executable", conf: &Config{BuildMode: BuildModeExe}, want: true},
		{name: "archive", conf: &Config{BuildMode: BuildModeCArchive}},
		{name: "shared", conf: &Config{BuildMode: BuildModeCShared}},
		{name: "nil config"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := needsWasmPostLink(test.conf, target); got != test.want {
				t.Fatalf("needsWasmPostLink() = %v, want %v", got, test.want)
			}
		})
	}
	if needsWasmPostLink(&Config{BuildMode: BuildModeExe}, nil) {
		t.Fatal("needsWasmPostLink() enabled for a nil target")
	}
	target.WasmPostLink.Asyncify = false
	target.WasmPostLink.TranslateToExnref = true
	if !needsWasmPostLink(&Config{BuildMode: BuildModeExe}, target) {
		t.Fatal("needsWasmPostLink() disabled for exnref translation")
	}
}

func TestPrepareWasmLinkOutput(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "app.wasm")
	target := &crosscompile.Export{WasmPostLink: crosscompile.WasmPostLink{Asyncify: true}}

	input, err := prepareWasmLinkOutput(&Config{BuildMode: BuildModeExe}, target, output)
	if err != nil {
		t.Fatal(err)
	}
	if input == output || filepath.Dir(input) != dir {
		t.Fatalf("temporary link output = %q, want a distinct file in %q", input, dir)
	}
	if _, err := os.Stat(input); err != nil {
		t.Fatalf("temporary link output was not created: %v", err)
	}
	cleanupWasmLinkOutput(input, output)
	if _, err := os.Stat(input); !os.IsNotExist(err) {
		t.Fatalf("temporary link output remains after cleanup: %v", err)
	}

	if err := os.WriteFile(output, []byte("final"), 0o644); err != nil {
		t.Fatal(err)
	}
	input, err = prepareWasmLinkOutput(&Config{BuildMode: BuildModeCArchive}, target, output)
	if err != nil || input != output {
		t.Fatalf("disabled post-link output = %q, %v; want %q, nil", input, err, output)
	}
	cleanupWasmLinkOutput(input, output)
	if data, err := os.ReadFile(output); err != nil || string(data) != "final" {
		t.Fatalf("cleanup removed final output: %q, %v", data, err)
	}
	if err := publishWasmLinkOutput(nil, output, output, false); err != nil {
		t.Fatalf("disabled publish failed: %v", err)
	}

	missingOutput := filepath.Join(dir, "missing", "app.wasm")
	if _, err := prepareWasmLinkOutput(&Config{BuildMode: BuildModeExe}, target, missingOutput); err == nil {
		t.Fatal("prepareWasmLinkOutput succeeded with a missing output directory")
	}
}

func TestPostLinkWasmPublishesOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "linked.wasm")
	output := filepath.Join(dir, "app.wasm")
	argsFile := filepath.Join(dir, "args")
	if err := os.WriteFile(input, []byte("core module"), 0o644); err != nil {
		t.Fatal(err)
	}

	script := `#!/bin/sh
printf '%s\n' "$@" > "$ARGS_FILE"
cp "$3" "$5"
`
	tool := writeWasmOptTestTool(t, dir, script)
	t.Setenv("WASMOPT", "")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ARGS_FILE", argsFile)

	ctx := wasmPostLinkTestContext()
	stderr, err := os.CreateTemp(dir, "stderr")
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = stderr
	t.Cleanup(func() { os.Stderr = oldStderr })

	if err := publishWasmLinkOutput(ctx, input, output, true); err != nil {
		t.Fatal(err)
	}
	if err := stderr.Close(); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(stderr.Name()); err != nil ||
		!strings.Contains(string(got), tool) ||
		!strings.Contains(string(got), "--asyncify") {
		t.Fatalf("verbose command = %q, %v", got, err)
	}
	if data, err := os.ReadFile(output); err != nil || string(data) != "core module" {
		t.Fatalf("published output = %q, %v", data, err)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(args); !strings.Contains(got, "--asyncify\n--translate-to-exnref\n") ||
		!strings.Contains(got, input+"\n-o\n") {
		t.Fatalf("wasm-opt args = %q", got)
	}
}

func TestPostLinkWasmReportsToolFailure(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "linked.wasm")
	output := filepath.Join(dir, "app.wasm")
	if err := os.WriteFile(input, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	tool := writeWasmOptTestTool(t, dir, "#!/bin/sh\nexit 7\n")
	t.Setenv("WASMOPT", tool)

	ctx := wasmPostLinkTestContext()
	err := postLinkWasm(ctx, input, output, false)
	if err == nil || !strings.Contains(err.Error(), "wasm-opt post-link failed") {
		t.Fatalf("postLinkWasm() error = %v", err)
	}
	if data, err := os.ReadFile(output); err != nil || string(data) != "old" {
		t.Fatalf("failed post-link changed final output: %q, %v", data, err)
	}
}

func TestPostLinkWasmReportsPublishFailure(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "linked.wasm")
	output := filepath.Join(dir, "existing-directory")
	if err := os.WriteFile(input, []byte("core module"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(output, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ncp \"$3\" \"$5\"\n"
	tool := writeWasmOptTestTool(t, dir, script)
	t.Setenv("WASMOPT", tool)

	ctx := wasmPostLinkTestContext()
	err := postLinkWasm(ctx, input, output, false)
	if err == nil {
		t.Fatal("postLinkWasm succeeded when the final output was a directory")
	}
	if strings.Contains(err.Error(), "wasm-opt post-link failed") {
		t.Fatalf("postLinkWasm failed before publishing output: %v", err)
	}
}

func TestPostLinkWasmReportsMissingTool(t *testing.T) {
	t.Setenv("WASMOPT", filepath.Join(t.TempDir(), "missing-wasm-opt"))
	ctx := wasmPostLinkTestContext()
	err := postLinkWasm(ctx, "input", filepath.Join(t.TempDir(), "output"), false)
	if err == nil || !strings.Contains(err.Error(), "install Binaryen or set WASMOPT") {
		t.Fatalf("postLinkWasm() error = %v", err)
	}
}

func TestPostLinkWasmReportsInvalidOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	tool := writeWasmOptTestTool(t, dir, "")
	t.Setenv("WASMOPT", tool)
	ctx := wasmPostLinkTestContext()
	err := postLinkWasm(ctx, "input", filepath.Join(dir, "missing", "output"), false)
	if err == nil {
		t.Fatal("postLinkWasm succeeded with a missing output directory")
	}
}
