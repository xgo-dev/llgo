//go:build !go1.26
// +build !go1.26

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
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const printLegacyExponentProbe = `package main

func printLegacyProbeFloat(v float64) {
	println(v)
}

func printLegacyProbeComplex(c complex128) {
	println(c)
}

func main() {
	println(5.0, 8.0)
	printLegacyProbeFloat(100.1)
	println(1 + 2i)
	printLegacyProbeComplex(1 + 2i)
	c := complex(4.0, 6.0)
	println(c)
}
`

func TestBuiltinPrintLegacyExponentWidth(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(printLegacyExponentProbe), 0644); err != nil {
		t.Fatal(err)
	}

	root := findLLGoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/llgo", "run", mainFile)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "LLGO_ROOT="+root)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("builtin print probe failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	got := filterPrintProbeOutput(stderr.String())
	want := "" +
		"+5.000000e+000 +8.000000e+000\n" +
		"+1.001000e+002\n" +
		"(+1.000000e+000+2.000000e+000i)\n" +
		"(+1.000000e+000+2.000000e+000i)\n" +
		"(+4.000000e+000+6.000000e+000i)\n"
	if got != want {
		t.Fatalf("builtin print output mismatch:\n got %q\nwant %q", got, want)
	}
}

func filterPrintProbeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	var out strings.Builder
	for _, line := range strings.SplitAfter(s, "\n") {
		trimmed := strings.TrimLeft(strings.TrimRight(line, "\n"), " \t")
		switch {
		case strings.HasPrefix(trimmed, "ld64.lld: warning: "):
			continue
		case strings.HasPrefix(trimmed, "ld.lld: warning: "):
			continue
		case strings.HasPrefix(trimmed, "ld: warning: "):
			continue
		case strings.HasPrefix(trimmed, "# "):
			continue
		}
		out.WriteString(line)
	}
	return out.String()
}
