/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAndCheck(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src", "simd", "archsimd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, data string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "VERSION"), "go1.27.0\n")
	source := filepath.Join(dir, "api.go")
	write(source, "package archsimd\nfunc Load(x *[4]float32) [4]float32\n")
	full, summary := filepath.Join(root, "full.json"), filepath.Join(root, "summary.json")
	var output bytes.Buffer
	for _, args := range [][]string{
		{"-goroot", root, "-out", full},
		{"-goroot", root, "-summary", "-out", summary},
		{"-goroot", root, "-check", full},
		{"-goroot", root, "-check-summary", summary},
	} {
		if err := run(args, &output); err != nil {
			t.Fatal(err)
		}
	}
	write(source, "package archsimd\nfunc Load(x *[8]float32) [8]float32\n")
	output.Reset()
	if err := run([]string{"-goroot", root, "-check", full}, &output); err == nil {
		t.Fatal("signature drift did not fail check mode")
	}
	if !strings.Contains(output.String(), `"kind": "signature"`) || !strings.Contains(output.String(), "simd/archsimd.Load") {
		t.Fatalf("missing actionable API diff: %s", &output)
	}
	if err := run([]string{"-goroot", root, "-check-summary", summary}, &output); err == nil {
		t.Fatal("compact baseline missed drift")
	}
	// Check mode must not overwrite the evidence being compared.
	before, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-goroot", root, "-check", full, "-out", full}, &output); err == nil {
		t.Fatal("conflicting check/write flags accepted")
	}
	after, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("check mode overwrote baseline")
	}
}
