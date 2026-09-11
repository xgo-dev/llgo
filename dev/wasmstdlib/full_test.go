package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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
	if err := runFullAt(root, "EC32", report, "go", "llgo", 0, 2, structured, run); err == nil {
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
	for _, path := range []string{"test/main_test.go", "test/windows/a_test.go", "test/std/io/a_test.go", "test/goroot/runner_test.go", "test/testdata/hidden_test.go"} {
		name := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("//go:build windows\n\npackage test\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := discoverFull(root)
	want := []string{"test", "test/goroot", "test/std/io", "test/windows"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, %v; want %v", got, err, want)
	}
}

func TestFullProfileCommandsKeepLLGoAndReferenceDistinct(t *testing.T) {
	for _, name := range []string{"EC32", "EC64", "WC32", "GJS", "GWASI", "GJS-reference", "GWASI-reference"} {
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
