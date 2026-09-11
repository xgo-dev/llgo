package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestFullFailureOutputIsImmediateAndLiteral(t *testing.T) {
	for _, tt := range []struct {
		name string
		out  string
		want string
	}{
		{name: "empty"},
		{name: "unterminated", out: "compile failed", want: "\n--- J32-Emscripten test/std/crypto/dsa failure output ---\n| compile failed\n--- end failure output ---\n"},
		{name: "multiline", out: "=== RUN TestGenerateParameters\npanic: test timed out\n", want: "\n--- J32-Emscripten test/std/crypto/dsa failure output ---\n| === RUN TestGenerateParameters\n| panic: test timed out\n--- end failure output ---\n"},
		{name: "workflow commands", out: "::error::literal diagnostic\n\n::endgroup::\n", want: "\n--- J32-Emscripten test/std/crypto/dsa failure output ---\n| ::error::literal diagnostic\n| \n| ::endgroup::\n--- end failure output ---\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			writeFullFailureOutput(&output, "J32-Emscripten", "test/std/crypto/dsa", []byte(tt.out))
			if got := output.String(); got != tt.want {
				t.Fatalf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

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
	for _, path := range []string{"test/main_test.go", "test/_stress/runtime/cpuprof/a_test.go", "test/_stress/runtime/timer/a_test.go", "test/windows/a_test.go", "test/std/io/a_test.go", "test/goroot/runner_test.go", "test/_manualtest/fail/a_test.go", "test/testdata/hidden_test.go"} {
		name := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("//go:build windows\n\npackage test\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := discoverFull(root)
	want := []string{"test", "test/_stress/runtime/cpuprof", "test/_stress/runtime/timer", "test/goroot", "test/std/io", "test/windows"}
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
	var inventory []byte
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
		data, err := json.Marshal(selectedPackage{Dir: filepath.Dir(name), TestGoFiles: []string{"excluded_test.go"}})
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

func TestFullAuditDoesNotHideSourceSelectionErrors(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "test", "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken_test.go"), []byte("package broken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := selectedPackage{Dir: dir}
	pkg.Error = &struct{ Err string }{Err: "synthetic go list failure"}
	inventory, err := json.Marshal(pkg)
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
		t.Fatal("package with a source-selection error was executed")
		return nil, nil
	}
	reportPath := filepath.Join(root, "report.json")
	if err := runFullAt(root, "J32-GoJS", reportPath, "go", "llgo", 0, 1, structured, run); err == nil {
		t.Fatal("source-selection error was hidden")
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report struct{ Packages []fullPackage }
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Packages) != 1 || report.Packages[0].Status != "fail" || !strings.Contains(report.Packages[0].Reason, "synthetic go list failure") {
		t.Fatalf("source-selection accounting: %s", data)
	}
}

func TestFullAuditRejectsInvalidPreparation(t *testing.T) {
	unused := func(string, command) ([]byte, error) {
		t.Fatal("external command ran after invalid preparation")
		return nil, nil
	}
	if err := runFullAt(t.TempDir(), "J32-GoJS", "", "go", "llgo", 0, 1, unused, unused); err == nil {
		t.Fatal("accepted an empty report path")
	}
	if err := runFullAt(t.TempDir(), "J32-GoJS", "report.json", "go", "llgo", 1, 1, unused, unused); err == nil {
		t.Fatal("accepted an out-of-range shard")
	}
	if err := runFullAt(t.TempDir(), "unknown", "report.json", "go", "llgo", 0, 1, unused, unused); err == nil {
		t.Fatal("accepted an unknown profile")
	}
	if err := runFullAt(t.TempDir(), "J32-GoJS", "report.json", "go", "llgo", 0, 1, unused, unused); err == nil {
		t.Fatal("accepted a root without test sources")
	}
	t.Chdir(t.TempDir())
	if err := runFull("J32-GoJS", "", "go", "llgo", 0, 1); err == nil {
		t.Fatal("runFull lost argument validation")
	}
	getwdErr := errors.New("synthetic getwd failure")
	if err := runFullFrom(func() (string, error) { return "", getwdErr }, "J32-GoJS", "report.json", "go", "llgo", 0, 1); !errors.Is(err, getwdErr) {
		t.Fatal("runFull hid a missing working directory")
	}
}

func TestFullAuditReportsPreparationCommandFailures(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "test", "a")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte("package a\nimport \"testing\"\nfunc TestA(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	selected, err := json.Marshal(selectedPackage{Dir: dir, TestGoFiles: []string{"a_test.go"}})
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("synthetic preparation failure")
	for _, tt := range []struct {
		name       string
		structured func(string, command) ([]byte, error)
	}{
		{name: "go env", structured: func(string, command) ([]byte, error) { return nil, sentinel }},
		{name: "go list", structured: func(_ string, c command) ([]byte, error) {
			if c.Args[0] == "env" {
				return []byte("/goroot"), nil
			}
			return nil, sentinel
		}},
		{name: "malformed list", structured: func(_ string, c command) ([]byte, error) {
			if c.Args[0] == "env" {
				return []byte("/goroot"), nil
			}
			return []byte("{"), nil
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reportPath := filepath.Join(t.TempDir(), "report.json")
			err := runFullAt(root, "J32-GoJS", reportPath, "go", "llgo", 0, 1, tt.structured, func(string, command) ([]byte, error) {
				t.Fatal("package command ran after preparation failure")
				return nil, nil
			})
			if err == nil {
				t.Fatal("preparation failure was hidden")
			}
			if tt.name != "malformed list" && !errors.Is(err, sentinel) {
				t.Fatalf("error = %v, want %v", err, sentinel)
			}
		})
	}
	if err := runFullAt(root, "J32-GoJS", filepath.Join(root, "missing", "report.json"), "go", "llgo", 0, 1, unusedStructured(selected), unusedStructured(selected)); err == nil {
		t.Fatal("initial report write failure was hidden")
	}
}

func unusedStructured(selected []byte) func(string, command) ([]byte, error) {
	return func(_ string, c command) ([]byte, error) {
		if c.Args[0] == "env" {
			return []byte("/goroot"), nil
		}
		return selected, nil
	}
}

func fullAuditFixture(t *testing.T, pkg string) (root string, selected []byte) {
	t.Helper()
	root = t.TempDir()
	dir := filepath.Join(root, filepath.FromSlash(pkg))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture_test.go"), []byte("package fixture\nimport \"testing\"\nfunc TestFixture(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(selectedPackage{Dir: dir, TestGoFiles: []string{"fixture_test.go"}})
	if err != nil {
		t.Fatal(err)
	}
	return root, data
}

func TestFullAuditReportsArtifactWriteFailures(t *testing.T) {
	validRun := func(string, command) ([]byte, error) {
		return []byte("=== RUN   TestFixture\n--- PASS: TestFixture (0.00s)\nPASS\n"), nil
	}

	t.Run("log directory", func(t *testing.T) {
		root, selected := fullAuditFixture(t, "test/a")
		reportPath := filepath.Join(root, "report.json")
		if err := os.WriteFile(reportPath+".logs", []byte("blocks directory creation"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := runFullAt(root, "J32-GoJS", reportPath, "go", "llgo", 0, 1, unusedStructured(selected), validRun); err == nil {
			t.Fatal("accepted an unwritable log directory")
		}
	})

	t.Run("package log", func(t *testing.T) {
		root, selected := fullAuditFixture(t, "test/a")
		reportPath := filepath.Join(root, "report.json")
		if err := os.MkdirAll(filepath.Join(reportPath+".logs", "test_a.log"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := runFullAt(root, "J32-GoJS", reportPath, "go", "llgo", 0, 1, unusedStructured(selected), validRun); err == nil {
			t.Fatal("accepted an unwritable package log")
		}
	})

	t.Run("incremental report", func(t *testing.T) {
		root, selected := fullAuditFixture(t, "test/a")
		reportPath := filepath.Join(root, "report.json")
		run := func(string, command) ([]byte, error) {
			if err := os.Remove(reportPath); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(reportPath, 0o755); err != nil {
				t.Fatal(err)
			}
			return validRun("", command{})
		}
		if err := runFullAt(root, "J32-GoJS", reportPath, "go", "llgo", 0, 1, unusedStructured(selected), run); err == nil {
			t.Fatal("accepted an unwritable incremental report")
		}
	})

	t.Run("final report", func(t *testing.T) {
		root, selected := fullAuditFixture(t, "test/a")
		reportPath := filepath.Join(root, "report.json")
		structured := func(_ string, c command) ([]byte, error) {
			if c.Args[0] == "env" {
				return []byte("/goroot"), nil
			}
			if err := os.Remove(reportPath); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(reportPath, 0o755); err != nil {
				t.Fatal(err)
			}
			return selected, nil
		}
		if err := runFullAt(root, "J32-GoJS", reportPath, "go", "llgo", 1, 2, structured, validRun); err == nil {
			t.Fatal("accepted an unwritable final report")
		}
	})

	t.Run("host artifact", func(t *testing.T) {
		root, selected := fullAuditFixture(t, "test/go")
		reportPath := filepath.Join(root, "report.json")
		blocker := filepath.Join(root, "not-a-temp-directory")
		if err := os.WriteFile(blocker, []byte("block"), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
			t.Setenv(name, blocker)
		}
		if err := runFullAt(root, "J32-GoJS", reportPath, "go", "llgo", 0, 1, unusedStructured(selected), validRun); err == nil {
			t.Fatal("accepted failure to create a reusable host artifact")
		}
	})
}

func TestFullAuditClassifiesUnknownSelectionAndWitnessFailures(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"test/goroot/runner_test.go": "package goroot\n",
		"test/missing/a_test.go":     "package missing\nimport \"testing\"\nfunc TestMissing(t *testing.T) {}\n",
		"test/empty/a_test.go":       "package empty\n",
		"test/bad/a_test.go":         "package bad\nfunc TestBad(\n",
	}
	for name, contents := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var selected []byte
	for _, pkg := range []selectedPackage{
		{Dir: filepath.Join(root, "test", "empty")},
		{Dir: filepath.Join(root, "test", "bad"), TestGoFiles: []string{"a_test.go"}},
	} {
		data, err := json.Marshal(pkg)
		if err != nil {
			t.Fatal(err)
		}
		selected = append(selected, data...)
	}
	reportPath := filepath.Join(root, "report.json")
	err := runFullAt(root, "J32-GoJS", reportPath, "go", "llgo", 0, 1, unusedStructured(selected), func(string, command) ([]byte, error) {
		t.Fatal("unvalidated package was executed")
		return nil, nil
	})
	if err == nil {
		t.Fatal("unresolved inventory was accepted")
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report struct{ Packages []fullPackage }
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"test/bad":     "unresolved",
		"test/empty":   "source-excluded",
		"test/goroot":  "separate-suite",
		"test/missing": "source-excluded",
	}
	for _, pkg := range report.Packages {
		if pkg.Status != want[pkg.Package] || pkg.Reason == "" {
			t.Fatalf("classification for %s = %+v", pkg.Package, pkg)
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

func TestFullStressCommandsUseQuickProfile(t *testing.T) {
	p, err := fullProfile("J32-Emscripten")
	if err != nil {
		t.Fatal(err)
	}
	cmd := fullCommand(p, "go", "llgo", "/goroot", "test/_stress/runtime/timer")
	if got := cmd.Env["LLGO_STRESS_PROFILE"]; got != "quick" {
		t.Fatalf("stress profile = %q", got)
	}
	if !slices.Contains(cmd.Args, "-timeout=3m") {
		t.Fatalf("stress test deadline is not targeted: %+v", cmd)
	}
	if got := cmd.Args[len(cmd.Args)-1]; got != "./test/_stress/runtime/timer/timer_stress_test.go" {
		t.Fatalf("stress package argument = %q", got)
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
		for _, pkg := range []string{"test/_stress/runtime/cpuprof", "test/_stress/runtime/finalizer", "test/_stress/runtime/signal", "test/std/plugin", "test/std/syscall", "test/windows"} {
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
		for _, pkg := range []string{"test/llgoext", "test/llgoext/localitymulti"} {
			_, excluded := fullSourceExclusion(p, pkg)
			if excluded != p.Reference {
				t.Fatalf("%s extension classification for %s = %v", name, pkg, excluded)
			}
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
	if got := fullTestTimeout("test/_stress/runtime/timer"); got != "3m" {
		t.Fatalf("stress timeout = %q", got)
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
