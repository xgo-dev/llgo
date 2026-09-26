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
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func wasmPostLinkTestContext() *context {
	return &context{
		buildConf: &Config{
			OptLevel:    optlevel.O2,
			LinkOptions: LinkOptions{DWARF: DWARFOmit},
		},
		crossCompile: crosscompile.Export{
			WasmPostLink: crosscompile.WasmPostLink{Asyncify: true},
		},
	}
}

func writeWasmOptTestTool(t *testing.T, dir string) string {
	return writeBuildTestTool(t, dir, "wasm-opt")
}

func writeBuildTestTool(t *testing.T, dir, name string) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(dir, name)
	if filepath.Ext(executable) == ".exe" {
		tool += ".exe"
	}
	if err := os.Link(executable, tool); err == nil {
		return tool
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tool, data, 0o755); err != nil {
		t.Fatal(err)
	}
	return tool
}

func TestWasmPostLinkArgs(t *testing.T) {
	target := &crosscompile.Export{WasmPostLink: crosscompile.WasmPostLink{Asyncify: true}}
	if got, want := wasmPostLinkArgs(target, "in.wasm", "out.wasm", false, optlevel.O2),
		[]string{"--asyncify", "--translate-to-exnref", "-O2", "in.wasm", "-o", "out.wasm"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wasmPostLinkArgs() = %v, want %v", got, want)
	}
	if got, want := wasmPostLinkArgs(target, "in.wasm", "out.wasm", true, optlevel.Oz),
		[]string{"--asyncify", "--translate-to-exnref", "-Oz", "-g", "in.wasm", "-o", "out.wasm"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wasmPostLinkArgs(debug) = %v, want %v", got, want)
	}
	if got, want := wasmPostLinkArgs(target, "in.wasm", "out.wasm", false, optlevel.O0),
		[]string{"--asyncify", "--translate-to-exnref", "in.wasm", "-o", "out.wasm"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wasmPostLinkArgs(O0) = %v, want %v", got, want)
	}
	if got := wasmPostLinkArgs(&crosscompile.Export{}, "in", "out", false, optlevel.O2); got != nil {
		t.Fatalf("wasmPostLinkArgs(disabled) = %v, want nil", got)
	}
}

func TestWasmPreAsyncifyArgs(t *testing.T) {
	target := &crosscompile.Export{WasmPostLink: crosscompile.WasmPostLink{Asyncify: true}}
	if got, want := wasmPreAsyncifyArgs(target, "in.wasm", "out.wasm", false, optlevel.Oz),
		[]string{"-Oz", "in.wasm", "-o", "out.wasm"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wasmPreAsyncifyArgs() = %v, want %v", got, want)
	}
	if got, want := wasmPreAsyncifyArgs(target, "in.wasm", "out.wasm", true, optlevel.O2),
		[]string{"-O2", "-g", "in.wasm", "-o", "out.wasm"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wasmPreAsyncifyArgs(debug) = %v, want %v", got, want)
	}
	if got := wasmPreAsyncifyArgs(target, "in", "out", false, optlevel.O0); got != nil {
		t.Fatalf("wasmPreAsyncifyArgs(O0) = %v, want nil", got)
	}
	if got := wasmPreAsyncifyArgs(target, "in", "out", false, optlevel.Unset); got != nil {
		t.Fatalf("wasmPreAsyncifyArgs(unset) = %v, want nil", got)
	}
	if got := wasmPreAsyncifyArgs(nil, "in", "out", false, optlevel.O2); got != nil {
		t.Fatalf("wasmPreAsyncifyArgs(nil) = %v, want nil", got)
	}
	if got := wasmPreAsyncifyArgs(&crosscompile.Export{}, "in", "out", false, optlevel.O2); got != nil {
		t.Fatalf("wasmPreAsyncifyArgs(disabled) = %v, want nil", got)
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

	tool := writeWasmOptTestTool(t, dir)
	t.Setenv("WASMOPT", "")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ARGS_FILE", argsFile)
	t.Setenv("LLGO_TEST_WASM_OPT_HELPER", "copy")

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
	if got := string(args); !strings.Contains(got, "---\n-O2\n"+input+"\n-o\n") ||
		!strings.Contains(got, "---\n--asyncify\n--translate-to-exnref\n-O2\n"+input+"\n-o\n") {
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
	tool := writeWasmOptTestTool(t, dir)
	t.Setenv("WASMOPT", tool)
	t.Setenv("LLGO_TEST_WASM_OPT_HELPER", "fail")

	ctx := wasmPostLinkTestContext()
	ctx.buildConf.OptLevel = optlevel.O0
	err := postLinkWasm(ctx, input, output, false)
	if err == nil || !strings.Contains(err.Error(), "wasm-opt Asyncify failed") {
		t.Fatalf("postLinkWasm() error = %v", err)
	}
	if !strings.Contains(err.Error(), tool) || !strings.Contains(err.Error(), "wasm-opt version 132 (test helper)") {
		t.Fatalf("postLinkWasm() error omits selected Binaryen tool: %v", err)
	}
	if data, err := os.ReadFile(output); err != nil || string(data) != "old" {
		t.Fatalf("failed post-link changed final output: %q, %v", data, err)
	}
}

func TestPostLinkWasmReportsPreAsyncifyFailure(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "linked.wasm")
	output := filepath.Join(dir, "app.wasm")
	if err := os.WriteFile(input, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	tool := writeWasmOptTestTool(t, dir)
	t.Setenv("WASMOPT", tool)
	t.Setenv("LLGO_TEST_WASM_OPT_HELPER", "fail")

	err := postLinkWasm(wasmPostLinkTestContext(), input, output, false)
	if err == nil || !strings.Contains(err.Error(), "wasm-opt pre-Asyncify optimization failed") {
		t.Fatalf("postLinkWasm() error = %v", err)
	}
	if !strings.Contains(err.Error(), tool) || !strings.Contains(err.Error(), "wasm-opt version 132 (test helper)") {
		t.Fatalf("postLinkWasm() error omits selected Binaryen tool: %v", err)
	}
	if data, err := os.ReadFile(output); err != nil || string(data) != "old" {
		t.Fatalf("failed pre-optimization changed final output: %q, %v", data, err)
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
	tool := writeWasmOptTestTool(t, dir)
	t.Setenv("WASMOPT", tool)
	t.Setenv("LLGO_TEST_WASM_OPT_HELPER", "copy")

	ctx := wasmPostLinkTestContext()
	err := postLinkWasm(ctx, input, output, false)
	if err == nil {
		t.Fatal("postLinkWasm succeeded when the final output was a directory")
	}
	if strings.Contains(err.Error(), "wasm-opt Asyncify failed") {
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

func TestResolveWasmOptUsesEmscriptenBinaryenRoot(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	want := writeWasmOptTestTool(t, bin)
	t.Setenv("EM_BINARYEN_ROOT", root)
	t.Setenv("WASMOPT", "")
	got, err := resolveWasmOpt()
	if err != nil || got != want {
		t.Fatalf("resolveWasmOpt() = %q, %v; want %q", got, err, want)
	}

	override := writeWasmOptTestTool(t, t.TempDir())
	t.Setenv("WASMOPT", override)
	got, err = resolveWasmOpt()
	if err != nil || got != override {
		t.Fatalf("resolveWasmOpt() with WASMOPT = %q, %v; want %q", got, err, override)
	}

	t.Setenv("EM_BINARYEN_ROOT", "")
	t.Setenv("WASMOPT", "")
	t.Setenv("PATH", bin)
	got, err = resolveWasmOpt()
	if err != nil || got != want {
		t.Fatalf("resolveWasmOpt() from PATH = %q, %v; want %q", got, err, want)
	}
}

func TestWasmOptIdentityWhenVersionUnavailable(t *testing.T) {
	tool := writeWasmOptTestTool(t, t.TempDir())
	t.Setenv("LLGO_TEST_WASM_OPT_HELPER", "fail-version")
	if got, want := wasmOptIdentity(tool), tool+" (version unavailable)"; got != want {
		t.Fatalf("wasmOptIdentity() = %q; want %q", got, want)
	}
}

func TestPostLinkWasmReportsInvalidOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	tool := writeWasmOptTestTool(t, dir)
	t.Setenv("WASMOPT", tool)
	t.Setenv("LLGO_TEST_WASM_OPT_HELPER", "copy")
	ctx := wasmPostLinkTestContext()
	err := postLinkWasm(ctx, "input", filepath.Join(dir, "missing", "output"), false)
	if err == nil {
		t.Fatal("postLinkWasm succeeded with a missing output directory")
	}
}
