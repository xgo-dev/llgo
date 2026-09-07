package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestFullAuditContinuesAndPreservesShardAccounting(t *testing.T) {
	root := t.TempDir()
	var inventory []byte
	for _, name := range []string{"a", "b", "c", "d"} {
		dir := filepath.Join(root, "test", name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte("package test\nimport \"testing\"\nfunc TestWorks(t *testing.T) {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(selectedPackage{Dir: dir, TestGoFiles: []string{"a_test.go"}})
		if err != nil {
			t.Fatal(err)
		}
		inventory = append(inventory, data...)
	}
	structured := func(_ string, c command) ([]byte, error) {
		if c.Args[0] == "env" {
			return []byte("/goroot"), nil
		}
		return inventory, nil
	}
	var visited []string
	run := func(_ string, c command) ([]byte, error) {
		pkg := c.Args[len(c.Args)-1]
		visited = append(visited, pkg)
		if pkg == "./test/a" {
			return []byte("compile failed"), errors.New("compile failed")
		}
		return []byte("=== RUN   TestWorks\n--- PASS: TestWorks (0.00s)\nPASS\n"), nil
	}
	report := filepath.Join(root, "report.json")
	if err := runFullAt(root, "J32-Emscripten", report, "go", "llgo", 0, 2, structured, run); err == nil {
		t.Fatal("failure was hidden")
	}
	if !reflect.DeepEqual(visited, []string{"./test/a", "./test/c"}) {
		t.Fatalf("execution stopped or crossed shards: %v", visited)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Result   string
		Packages []fullPackage
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Result != "fail" || len(got.Packages) != 4 {
		t.Fatalf("bad report: %s", data)
	}
	for i, status := range []string{"fail", "other-shard", "pass", "other-shard"} {
		if got.Packages[i].Status != status {
			t.Fatalf("bad accounting: %s", data)
		}
	}
	if log, err := os.ReadFile(filepath.Join(report+".logs", "test_a.log")); err != nil || string(log) != "compile failed" {
		t.Fatalf("lost failure log: %q %v", log, err)
	}
}

func TestFullDiscoveryIncludesRootAndExcludedSource(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"test/main_test.go", "test/_stress/runtime/cpuprof/a_test.go", "test/windows/a_test.go", "test/std/io/a_test.go", "test/goroot/runner_test.go", "test/testdata/hidden_test.go"} {
		name := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("//go:build windows\n\npackage test\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := discoverFull(root)
	want := []string{"test", "test/_stress/runtime/cpuprof", "test/goroot", "test/std/io", "test/windows"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, %v; want %v", got, err, want)
	}
}

func TestFullAuditClassifiesHostDriverSuiteWithoutExecutingIt(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "test", "cmd", "llgo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "driver_test.go"), []byte("package llgo_test\nimport \"testing\"\nfunc TestDriver(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	inventory, err := json.Marshal(selectedPackage{Dir: dir, TestGoFiles: []string{"driver_test.go"}})
	if err != nil {
		t.Fatal(err)
	}
	structured := func(_ string, c command) ([]byte, error) {
		if c.Args[0] == "env" {
			return []byte("/goroot"), nil
		}
		return inventory, nil
	}
	run := func(string, command) ([]byte, error) {
		t.Fatal("host driver suite executed as a wasm target")
		return nil, nil
	}
	reportPath := filepath.Join(root, "report.json")
	if err := runFullAt(root, "J32-Emscripten", reportPath, "go", "llgo", 0, 1, structured, run); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report struct{ Packages []fullPackage }
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Packages) != 1 || report.Packages[0].Status != "separate-suite" {
		t.Fatalf("host suite accounting: %s", data)
	}
}

func TestFullAuditAcceptsReviewedSourceExclusions(t *testing.T) {
	root := t.TempDir()
	packages := []string{
		"test/_stress/runtime/cpuprof",
		"test/_stress/runtime/finalizer",
		"test/_stress/runtime/signal",
		"test/cgo",
		"test/std/plugin",
		"test/std/runtime/cgo",
		"test/std/syscall",
		"test/windows",
	}
	for _, pkg := range packages {
		name := filepath.Join(root, pkg, "excluded_test.go")
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("//go:build windows\n\npackage excluded\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	structured := func(_ string, c command) ([]byte, error) {
		if c.Args[0] == "env" {
			return []byte("/goroot"), nil
		}
		return nil, nil
	}
	run := func(string, command) ([]byte, error) {
		t.Fatal("source-excluded package was executed")
		return nil, nil
	}
	reportPath := filepath.Join(root, "report.json")
	if err := runFullAt(root, "J32-GoJS", reportPath, "go", "llgo", 0, 1, structured, run); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report struct{ Packages []fullPackage }
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Packages) != len(packages) {
		t.Fatalf("reviewed exclusion accounting: %s", data)
	}
	for _, pkg := range report.Packages {
		if pkg.Status != "not-applicable" || pkg.Reason == "" {
			t.Fatalf("reviewed exclusion accounting: %s", data)
		}
	}
}

func TestFullProfileCommandsKeepLLGoAndReferenceDistinct(t *testing.T) {
	for _, name := range []string{"J32-GoJS", "J32-Emscripten", "J64-Emscripten", "W32-WASI", "GoJS-reference", "GoWASI-reference"} {
		p, err := fullProfile(name)
		if err != nil {
			t.Fatal(err)
		}
		cmd := fullCommand(p, "official-go", "llgo", "/goroot", "test")
		if cmd.Program != "timeout" || cmd.Args[0] != "--kill-after=10s" || cmd.Args[1] != "5m" {
			t.Fatalf("unbounded command: %+v", cmd)
		}
		want := "llgo"
		if p.Reference {
			want = "official-go"
		}
		if cmd.Args[2] != want || cmd.Args[len(cmd.Args)-1] != "./test" || !slices.Contains(cmd.Args, "-count=1") {
			t.Fatalf("%s: %+v", name, cmd)
		}
		if p.Target != "" && !slices.Contains(cmd.Args, p.Target) {
			t.Fatalf("lost target: %+v", cmd)
		}
		if p.Target == "" && (cmd.Env["GOOS"] != p.GOOS || cmd.Env["GOARCH"] != "wasm") {
			t.Fatalf("lost raw profile: %+v", cmd)
		}
	}
}

func TestFullSourceContextMatchesCompilerProfiles(t *testing.T) {
	tests := []struct {
		name, wantCGO string
		wantTags      []string
	}{
		{"J32-GoJS", "0", []string{"llgo", "osusergo", "llgo.wasm.gc.linear"}},
		{"J32-Emscripten", "1", []string{"llgo", "osusergo", "llgo.wasm.gc.linear", "llgo.wasm.emscripten"}},
		{"J64-Emscripten", "1", []string{"llgo", "osusergo", "llgo.wasm.gc.linear", "llgo.wasm.emscripten", "llgo.wasm.emscripten.memory64"}},
		{"W32-WASI", "1", []string{"llgo", "osusergo", "llgo.wasm.gc.linear", "llgo.wasm.wasi"}},
		{"GoJS-reference", "0", nil},
		{"GoWASI-reference", "0", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := fullProfile(tt.name)
			if err != nil {
				t.Fatal(err)
			}
			tags, cgo := fullSourceContext(p)
			if cgo != tt.wantCGO {
				t.Fatalf("CGO_ENABLED=%q, want %q", cgo, tt.wantCGO)
			}
			for _, want := range tt.wantTags {
				if !slices.Contains(strings.Split(tags, ","), want) {
					t.Fatalf("tags %q do not contain %q", tags, want)
				}
			}
			if len(tt.wantTags) == 0 && tags != "" {
				t.Fatalf("reference tags = %q", tags)
			}
		})
	}
}

func TestFullSourceExclusionsAreProfileSpecific(t *testing.T) {
	for _, name := range []string{"J32-GoJS", "J32-Emscripten", "J64-Emscripten", "W32-WASI", "GoJS-reference", "GoWASI-reference"} {
		p, err := fullProfile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, pkg := range []string{"test/std/plugin", "test/std/syscall", "test/windows"} {
			if reason, ok := fullSourceExclusion(p, pkg); !ok || reason == "" {
				t.Fatalf("%s did not classify %s", name, pkg)
			}
		}
		for _, pkg := range []string{"test/cgo", "test/std/runtime/cgo"} {
			if reason, ok := fullSourceExclusion(p, pkg); !ok || reason == "" {
				t.Fatalf("%s did not classify %s", name, pkg)
			}
		}
		if _, ok := fullSourceExclusion(p, "test/std/fmt"); ok {
			t.Fatalf("%s classified an applicable package", name)
		}
	}
}

func TestFullLongTimeoutIsTargeted(t *testing.T) {
	for _, pkg := range []string{"test/std/crypto/dsa", "test/std/crypto/rsa", "test/std/go/types", "test/std/os", "test/std/runtime/pprof", "test/_stress/runtime/example"} {
		if got := fullTestTimeout(pkg); got != "3m" {
			t.Fatalf("%s timeout = %q", pkg, got)
		}
	}
	if got := fullTestTimeout("test/std/crypto/aes"); got != "60s" {
		t.Fatalf("default timeout = %q", got)
	}
}

func TestFullWitnessUsesOnlySelectedSources(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte("package test\nimport \"testing\"\nfunc TestMain(m *testing.M) {}\nfunc TestWorks(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	witness, err := testWitness(selectedPackage{Dir: dir, TestGoFiles: []string{"a_test.go"}})
	if err != nil || witness != "TestWorks" {
		t.Fatalf("%q %v", witness, err)
	}
	if _, err := testWitness(selectedPackage{Dir: dir}); err == nil {
		t.Fatal("accepted unselected tests")
	}
}
