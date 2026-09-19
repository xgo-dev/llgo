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

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	wasmDir := filepath.Join(dir, "wasm")
	if err := os.Mkdir(nativeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(wasmDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "summary.tsv"), []byte("linux/amd64\tfull\t1.27\t0\t1\tsuccess\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wasmDir, "acceptance.json"), []byte(`{"profile":"J32-GoJS","shard":0,"shards":3,"packages":[{"package":"p","status":"pass"}]}`), 0644); err != nil {
		t.Fatal(err)
	}

	baseArgs := []string{
		"-native-reports", nativeDir,
		"-wasm-reports", wasmDir,
		"-run-url", "https://example.com/run",
		"-mention-on-failure", "fennoai",
		"-native-result", "success",
		"-wasm-result", "success",
		"-browser-result", "success",
		"-sentinels-result", "success",
	}
	var stdout bytes.Buffer
	if err := run(baseArgs, &stdout); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Standard Library test summary") {
		t.Fatalf("run() output = %q", stdout.String())
	}

	output := filepath.Join(dir, "report.md")
	if err := run(append(append([]string{}, baseArgs...), "-output", output), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(output); err != nil || !strings.Contains(string(data), "Standard Library test summary") {
		t.Fatalf("report file: data=%q err=%v", data, err)
	}

	if err := run([]string{"-unknown-flag"}, &bytes.Buffer{}); err == nil {
		t.Fatal("run() accepted an unknown flag")
	}
	if err := run([]string{"-native-reports", filepath.Join(dir, "missing")}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "load native shards") {
		t.Fatalf("missing native reports error = %v", err)
	}
	if err := run([]string{"-wasm-reports", filepath.Join(dir, "missing")}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "load wasm reports") {
		t.Fatalf("missing wasm reports error = %v", err)
	}
	if err := run([]string{"-output", dir}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "write output") {
		t.Fatalf("directory output error = %v", err)
	}
	if err := run(nil, failingWriter{}); err == nil || err.Error() != "write failed" {
		t.Fatalf("stdout error = %v", err)
	}
}

func TestLoadNativeShards(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	data := strings.Join([]string{
		"",
		"too\tshort",
		"linux/amd64\tcompatibility\t1.20-1.26\t0\t1\tsuccess",
		"windows-msvc/arm64\tfull\t1.27\t1\t2\tfailure\textra-field",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(nested, "summary.tsv"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("ignored"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := loadNativeShards(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []nativeShard{
		{Platform: "linux/amd64", Lane: "compatibility", GoVersion: "1.20-1.26", ShardIndex: "0", ShardTotal: "1", Status: "success"},
		{Platform: "windows-msvc/arm64", Lane: "full", GoVersion: "1.27", ShardIndex: "1", ShardTotal: "2", Status: "failure"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loadNativeShards() = %#v, want %#v", got, want)
	}
}

func TestLoadNativeShardsMissingDirectory(t *testing.T) {
	if _, err := loadNativeShards(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("loadNativeShards() succeeded for a missing directory")
	}
}

func TestLoadNativeShardsScannerError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "oversized.tsv"), []byte(strings.Repeat("x", 70*1024)), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadNativeShards(dir); err == nil {
		t.Fatal("loadNativeShards() accepted an oversized TSV record")
	}
}

func TestLoadWasmReports(t *testing.T) {
	dir := t.TempDir()
	reports := map[string]string{
		"gojs.json": `{
			"profile":"J32-GoJS","result":"success","shard":0,"shards":3,
			"packages":[
				{"package":"pass","status":"pass"},
				{"package":"na","status":"not-applicable"},
				{"package":"separate","status":"separate-suite"},
				{"package":"other","status":"other-shard"},
				{"package":"failed","status":"fail","reason":"bad | output"}
			]
		}`,
		"unknown.json": `{
			"profile":"experimental","result":"success","shard":0,"shards":1,
			"packages":[{"package":"z","status":"fail"},{"package":"a","status":"fail"}]
		}`,
		"invalid.json": `{not-json}`,
	}
	for name, data := range reports {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("ignored"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := loadWasmReports(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("loadWasmReports() returned %d profiles, want 5: %#v", len(got), got)
	}
	gojs := got[0]
	if gojs.Profile != "J32-GoJS" || gojs.ShardsFound != 1 || gojs.ShardsTotal != 3 || gojs.Total != 4 || gojs.Passed != 1 || gojs.NotApplicable != 2 || gojs.Failed != 1 {
		t.Fatalf("J32-GoJS summary = %#v", gojs)
	}
	experimental := got[4]
	if experimental.Profile != "experimental" || experimental.ShardsFound != 1 || experimental.ShardsTotal != 1 || experimental.Failed != 2 {
		t.Fatalf("experimental summary = %#v", experimental)
	}
	if gotNames := []string{experimental.FailedPkgs[0].Package, experimental.FailedPkgs[1].Package}; !reflect.DeepEqual(gotNames, []string{"a", "z"}) {
		t.Fatalf("failed packages = %v, want [a z]", gotNames)
	}
}

func TestLoadWasmReportsMissingDirectory(t *testing.T) {
	if _, err := loadWasmReports(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("loadWasmReports() succeeded for a missing directory")
	}
}

func TestGenerateReportSuccess(t *testing.T) {
	native := []nativeShard{
		{
			Platform:   "linux/amd64",
			Lane:       "full",
			GoVersion:  "1.27",
			ShardIndex: "0",
			ShardTotal: "2",
			Status:     "success",
		},
	}
	wasm := []wasmProfileSummary{
		{
			Profile:       "J32-GoJS",
			ShardsFound:   3,
			ShardsTotal:   3,
			Total:         100,
			Passed:        95,
			NotApplicable: 5,
			Failed:        0,
		},
	}
	rep := generateReport(native, wasm, "https://example.com/run", "fennoai", "success", "success", "success", "success")
	if strings.Contains(rep, "@fennoai") {
		t.Fatalf("expected no mention on success, got:\n%s", rep)
	}
	if !strings.Contains(rep, "All standard library tests passed successfully.") {
		t.Fatalf("expected success message, got:\n%s", rep)
	}
	if !strings.Contains(rep, "| J32-GoJS | 3/3 | 100 | 95 | 5 | 0 |") {
		t.Fatalf("missing wasm summary row:\n%s", rep)
	}
}

func TestGenerateReportFailureMentions(t *testing.T) {
	native := []nativeShard{
		{
			Platform:   "linux/amd64",
			Lane:       "full",
			GoVersion:  "1.27",
			ShardIndex: "0",
			ShardTotal: "2",
			Status:     "failure",
		},
	}
	rep := generateReport(native, nil, "", "fennoai", "failure", "success", "success", "success")
	if !strings.Contains(rep, "@fennoai") || !strings.Contains(rep, "PR") {
		t.Fatalf("expected @fennoai with PR triage instruction, got:\n%s", rep)
	}
}

func TestGenerateReportWasmFailures(t *testing.T) {
	wasm := []wasmProfileSummary{
		{
			Profile:     "W32-WASI",
			ShardsFound: 5,
			ShardsTotal: 5,
			Total:       10,
			Passed:      9,
			Failed:      1,
			FailedPkgs: []wasmPackage{
				{
					Package: "test/std/net",
					Status:  "fail",
					Reason:  "network unreachable",
				},
				{
					Package: "test/std/os",
					Status:  "fail",
				},
			},
		},
	}
	rep := generateReport(nil, wasm, "", "fennoai", "success", "failure", "success", "success")
	if !strings.Contains(rep, "@fennoai") || !strings.Contains(rep, "PR") {
		t.Fatalf("expected @fennoai with PR triage instruction, got:\n%s", rep)
	}
	if !strings.Contains(rep, "- `test/std/net`: network unreachable") {
		t.Fatalf("expected failed package in list, got:\n%s", rep)
	}
	if !strings.Contains(rep, "- `test/std/os`: unspecified failure") {
		t.Fatalf("expected default failure reason, got:\n%s", rep)
	}
}

func TestGenerateReportHostFailures(t *testing.T) {
	rep := generateReport(nil, nil, "", "", "success", "success", "failure", "cancelled")
	if !strings.Contains(rep, "❌ failure") || !strings.Contains(rep, "❌ cancelled") {
		t.Fatalf("expected host failure results, got:\n%s", rep)
	}
}

func TestGenerateReportMissingWasmShard(t *testing.T) {
	wasm := []wasmProfileSummary{{Profile: "J32-GoJS", ShardsFound: 2, ShardsTotal: 3}}
	rep := generateReport(nil, wasm, "", "fennoai", "success", "success", "success", "success")
	if !strings.Contains(rep, "@fennoai") {
		t.Fatalf("expected missing shard warning, got:\n%s", rep)
	}
}
