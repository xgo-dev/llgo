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

package gotest

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
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
		compareToolCompileResult(t, dir, args, true, "")
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
				compareToolCompileResult(t, dir, []string{tt.flag, "-o=compat.o", "valid.go"}, true, "")
			})
		}
	})

	t.Run("language version", func(t *testing.T) {
		dir := t.TempDir()
		writeToolCompileSource(t, dir, "generic.go", `package compat

func Identity[T any](v T) T { return v }
`)
		compareToolCompileResult(t, dir, []string{
			"-lang=go1.17", "-o=generic.o", "generic.go",
		}, false, "requires go1.18 or later")
	})

	t.Run("complete package", func(t *testing.T) {
		dir := t.TempDir()
		writeToolCompileSource(t, dir, "nobody.go", `package compat

func External()
`)
		compareToolCompileResult(t, dir, []string{
			"-lang=go1.22", "-complete", "-o=nobody.o", "nobody.go",
		}, false, "missing function body")
		compareToolCompileResult(t, dir, []string{
			"-lang=go1.22", "-o=nobody.o", "nobody.go",
		}, true, "")
	})

	t.Run("diagnostic columns", func(t *testing.T) {
		dir := t.TempDir()
		writeToolCompileSource(t, dir, "invalid.go", `package compat

var _ = missing
`)
		goOutput, llgoOutput := compareToolCompileResult(t, dir, []string{
			"-C", "-e", "-o=invalid.o", "invalid.go",
		}, false, "undefined: missing")
		column := regexp.MustCompile(`invalid\.go:3:[0-9]+:`)
		for name, output := range map[string]string{"go": goOutput, "llgo": llgoOutput} {
			if column.MatchString(output) {
				t.Fatalf("%s -C diagnostic contains a column: %s", name, output)
			}
			if !strings.Contains(output, "invalid.go:3:") {
				t.Fatalf("%s -C diagnostic has no line position: %s", name, output)
			}
		}
	})
}

func TestToolCompileFrontendDiagnosticNormalization(t *testing.T) {
	t.Run("absolute import", func(t *testing.T) {
		dir := t.TempDir()
		writeToolCompileSource(t, dir, "invalid.go", "package compat\nimport _ \"/foo\"\n")
		writeToolCompileSource(t, dir, "importcfg", "")
		compareToolCompileResult(t, dir, []string{
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
		compareToolCompileResult(t, dir, []string{
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
		compareToolCompileResult(t, dir, []string{
			"-C", "-e", "-lang=go1.15", "-importcfg=importcfg", "-o=invalid.o", "invalid.go",
		}, false, "go:embed requires go1.16 or later")
	})

}

func compareToolCompileResult(t *testing.T, dir string, args []string, wantSuccess bool, wantText string) (goOutput, llgoOutput string) {
	t.Helper()

	goArgs := append([]string{"tool", "compile"}, args...)
	goCmd := exec.Command("go", goArgs...)
	goCmd.Dir = dir
	goCmd.Env = os.Environ()
	goBytes, goErr := goCmd.CombinedOutput()

	llgoOutput, llgoErr := runLLGoInModule(t, dir, append([]string{"tool", "compile"}, args...)...)
	goOutput = string(goBytes)
	if (goErr == nil) != wantSuccess {
		t.Fatalf("go tool compile success = %v, want %v; output:\n%s", goErr == nil, wantSuccess, goOutput)
	}
	if (llgoErr == nil) != wantSuccess {
		t.Fatalf("llgo tool compile success = %v, want %v; output:\n%s", llgoErr == nil, wantSuccess, llgoOutput)
	}
	if wantText != "" {
		if !strings.Contains(goOutput, wantText) {
			t.Fatalf("go tool compile output does not contain %q:\n%s", wantText, goOutput)
		}
		if !strings.Contains(llgoOutput, wantText) {
			t.Fatalf("llgo tool compile output does not contain %q:\n%s", wantText, llgoOutput)
		}
	}
	return goOutput, llgoOutput
}

func writeToolCompileSource(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeToolCompileStdlibImportCfg(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("go", "list", "-export", "-f", "{{if .Export}}packagefile {{.ImportPath}}={{.Export}}{{end}}", "std")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOENV=off", "GOFLAGS=")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list stdlib export files: %v\n%s", err, output)
	}
	writeToolCompileSource(t, dir, "importcfg", string(output))
}
