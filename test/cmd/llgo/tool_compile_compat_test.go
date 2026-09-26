//go:build !wasm

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

package llgocmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

var (
	stdlibImportCfgOnce   sync.Once
	stdlibImportCfgOutput []byte
	stdlibImportCfgErr    error
)

func TestToolCompileGoFlagCompatibility(t *testing.T) {
	t.Run("frontend flags", func(t *testing.T) {
		dir := t.TempDir()
		writeToolCompileSource(t, dir, "valid.go", `package compat

type Box[T any] struct{ Value T }

func Identity[T any](v T) T { return v }
`)
		writeToolCompileSource(t, dir, "importcfg", "")
		args := []string{
			"-c=1", "-C", "-e", "-N", "-l=4", "-lang=go1.22", "-complete",
			"-d=panic,ssa/check/on", "-p=compat", "-importcfg=importcfg",
			"-D=.", "-I=.", "-o=compat.o", "valid.go",
		}
		checkToolCompileResult(t, dir, args, true, "")
	})

	t.Run("SSA check seed", func(t *testing.T) {
		tests := []struct {
			name string
			flag string
		}{
			{name: "default", flag: "-d=ssa/check/seed"},
			{name: "explicit", flag: "-d=ssa/check/seed=1"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := t.TempDir()
				writeToolCompileSource(t, dir, "valid.go", "package compat\n\nfunc F() {}\n")
				checkToolCompileResult(t, dir, []string{tt.flag, "-o=compat.o", "valid.go"}, true, "")
			})
		}
	})

	t.Run("language version", func(t *testing.T) {
		dir := t.TempDir()
		writeToolCompileSource(t, dir, "generic.go", `package compat

func Identity[T any](v T) T { return v }
`)
		checkToolCompileResult(t, dir, []string{
			"-lang=go1.17", "-o=generic.o", "generic.go",
		}, false, "requires go1.18 or later")
	})

	t.Run("complete package", func(t *testing.T) {
		dir := t.TempDir()
		writeToolCompileSource(t, dir, "nobody.go", `package compat

func External()
`)
		checkToolCompileResult(t, dir, []string{
			"-lang=go1.22", "-complete", "-o=nobody.o", "nobody.go",
		}, false, "missing function body")
		checkToolCompileResult(t, dir, []string{
			"-lang=go1.22", "-o=nobody.o", "nobody.go",
		}, true, "")
	})

	t.Run("diagnostic columns", func(t *testing.T) {
		dir := t.TempDir()
		writeToolCompileSource(t, dir, "invalid.go", `package compat

var _ = missing
`)
		output := checkToolCompileResult(t, dir, []string{
			"-C", "-e", "-o=invalid.o", "invalid.go",
		}, false, "undefined: missing")
		column := regexp.MustCompile(`invalid\.go:3:[0-9]+:`)
		if column.MatchString(output) {
			t.Fatalf("%s -C diagnostic contains a column: %s", toolCompilerName, output)
		}
		if !strings.Contains(output, "invalid.go:3:") {
			t.Fatalf("%s -C diagnostic has no line position: %s", toolCompilerName, output)
		}
	})
}

func TestToolCompileFrontendDiagnosticNormalization(t *testing.T) {
	t.Run("absolute import", func(t *testing.T) {
		dir := t.TempDir()
		writeToolCompileSource(t, dir, "invalid.go", "package compat\nimport _ \"/foo\"\n")
		writeToolCompileSource(t, dir, "importcfg", "")
		checkToolCompileResult(t, dir, []string{
			"-C", "-e", "-importcfg=importcfg", "-o=invalid.o", "invalid.go",
		}, false, "import path cannot be absolute path")
	})

	t.Run("embed local var", func(t *testing.T) {
		dir := t.TempDir()
		writeToolCompileStdlibImportCfg(t, dir)
		writeToolCompileSource(t, dir, "x.txt", "x")
		writeToolCompileSource(t, dir, "invalid.go", `package compat
import _ "embed"
func f() {
	//go:embed x.txt // ERROR
	var x string
	_ = x
}`)
		checkToolCompileResult(t, dir, []string{
			"-C", "-e", "-importcfg=importcfg", "-o=invalid.o", "invalid.go",
		}, false, "go:embed cannot apply to var inside func")
	})

	t.Run("embed language version", func(t *testing.T) {
		dir := t.TempDir()
		writeToolCompileStdlibImportCfg(t, dir)
		writeToolCompileSource(t, dir, "x.txt", "x")
		writeToolCompileSource(t, dir, "invalid.go", `package compat
import _ "embed"
//go:embed x.txt // ERROR
var x string`)
		checkToolCompileResult(t, dir, []string{
			"-C", "-e", "-lang=go1.15", "-importcfg=importcfg", "-o=invalid.o", "invalid.go",
		}, false, "go:embed requires go1.16 or later")
	})

}

func checkToolCompileResult(t *testing.T, dir string, args []string, wantSuccess bool, wantText string) string {
	t.Helper()
	output, err := runToolCompile(t, dir, args...)
	if (err == nil) != wantSuccess {
		t.Fatalf("%s tool compile success = %v, want %v; error=%T %v; output:\n%s", toolCompilerName, err == nil, wantSuccess, err, err, output)
	}
	if wantText != "" {
		if !strings.Contains(output, wantText) {
			t.Fatalf("%s tool compile output does not contain %q; error=%T %v:\n%s", toolCompilerName, wantText, err, err, output)
		}
	}
	return output
}

func writeToolCompileSource(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeToolCompileStdlibImportCfg(t *testing.T, dir string) {
	t.Helper()
	stdlibImportCfgOnce.Do(func() {
		cmd := exec.Command("go", "list", "-export", "-f", "{{if .Export}}packagefile {{.ImportPath}}={{.Export}}{{end}}", "std")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOENV=off", "GOFLAGS=")
		stdlibImportCfgOutput, stdlibImportCfgErr = cmd.CombinedOutput()
	})
	if stdlibImportCfgErr != nil {
		t.Fatalf("list stdlib export files: %v\n%s", stdlibImportCfgErr, stdlibImportCfgOutput)
	}
	writeToolCompileSource(t, dir, "importcfg", string(stdlibImportCfgOutput))
}
