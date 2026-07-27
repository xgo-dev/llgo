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

func TestWasmPostLinkArgs(t *testing.T) {
	target := &crosscompile.Export{WasmPostLink: crosscompile.WasmPostLink{Asyncify: true}}
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

func TestPostLinkWasmPublishesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "linked.wasm")
	output := filepath.Join(dir, "app.wasm")
	argsFile := filepath.Join(dir, "args")
	if err := os.WriteFile(input, []byte("core module"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := filepath.Join(dir, "wasm-opt")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$ARGS_FILE"
input=
output=
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      output="$2"
      shift 2
      ;;
    -*)
      shift
      ;;
    *)
      input="$1"
      shift
      ;;
  esac
done
cp "$input" "$output"
`
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WASMOPT", tool)
	t.Setenv("ARGS_FILE", argsFile)

	ctx := &context{
		buildConf: &Config{Mode: ModeBuild, LinkOptions: LinkOptions{DWARF: DWARFOmit}},
		crossCompile: crosscompile.Export{
			WasmPostLink: crosscompile.WasmPostLink{Asyncify: true},
		},
	}
	if err := postLinkWasm(ctx, input, output, false); err != nil {
		t.Fatal(err)
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

func TestPostLinkWasmReportsMissingTool(t *testing.T) {
	t.Setenv("WASMOPT", filepath.Join(t.TempDir(), "missing-wasm-opt"))
	ctx := &context{
		buildConf: &Config{},
		crossCompile: crosscompile.Export{
			WasmPostLink: crosscompile.WasmPostLink{Asyncify: true},
		},
	}
	err := postLinkWasm(ctx, "input", filepath.Join(t.TempDir(), "output"), false)
	if err == nil || !strings.Contains(err.Error(), "install Binaryen or set WASMOPT") {
		t.Fatalf("postLinkWasm() error = %v", err)
	}
}
