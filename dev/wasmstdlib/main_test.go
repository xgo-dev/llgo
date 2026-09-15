package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Stand in for the external Go/LLGo command when testing the driver itself.
// The workflow separately runs real compilers and real wasm host helpers.
func TestMain(m *testing.M) {
	if os.Getenv("LLGO_WASMSTDLIB_TEST_HELPER") == "1" {
		if os.Getenv("GOFLAGS") != "" {
			fmt.Fprintln(os.Stderr, "inherited GOFLAGS would change the acceptance slice")
			os.Exit(10)
		}
		mode := os.Getenv("LLGO_WASMSTDLIB_TEST_MODE")
		switch os.Args[1] {
		case "env":
			if mode == "bad-env" {
				fmt.Println("invalid JSON")
			} else if mode == "env-failure" {
				os.Exit(7)
			} else {
				fmt.Println(`{"GOROOT":"/Go root","GOVERSION":"go1.27.0"}`)
			}
		case "list":
			if mode == "list-failure" {
				os.Exit(7)
			}
			fmt.Fprintln(os.Stderr, "simulated module-download progress")
			if mode == "bad-list" {
				fmt.Println("invalid JSON")
			} else {
				for index, tc := range acceptance {
					if mode == "missing-package" && index == len(acceptance)-1 {
						continue
					}
					fmt.Printf("{\"ImportPath\":%q,\"XTestGoFiles\":[\"suite_test.go\"]}\n", packagePrefix+tc.Package)
				}
			}
		case "test":
			if mode == "test-failure" {
				os.Exit(7)
			}
			pkg := strings.TrimPrefix(os.Args[len(os.Args)-1], "./test/std/")
			for _, tc := range acceptance {
				if tc.Package == pkg {
					fmt.Printf("--- PASS: %s (0.00s)\nPASS\n", tc.Witness)
					os.Exit(0)
				}
			}
			os.Exit(8)
		default:
			os.Exit(9)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func driverFixture(t *testing.T) (root, program string) {
	t.Helper()
	root = t.TempDir()
	for _, tc := range append(append([]testCase{}, acceptance...), testCase{"syscall", "unused"}) {
		dir := filepath.Join(root, "test", "std", tc.Package)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "suite_test.go"), []byte("package p\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	t.Setenv("LLGO_WASMSTDLIB_TEST_HELPER", "1")
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	t.Setenv("GOFLAGS", "-run=TestAs -tags=unrelated")
	program, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return root, program
}

func readReport(t *testing.T, path string) report {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var r report
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestDriverCLI(t *testing.T) {
	var stderr strings.Builder
	if code := runMain([]string{"-unknown"}, &stderr); code != 2 || !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("flag error: code=%d stderr=%q", code, stderr.String())
	}
	stderr.Reset()
	if code := runMain([]string{"-h"}, &stderr); code != 0 || !strings.Contains(stderr.String(), "Usage of wasmstdlib") {
		t.Fatalf("help: code=%d stderr=%q", code, stderr.String())
	}
	stderr.Reset()
	if code := runMain([]string{"-profile=J32-GoJS"}, &stderr); code != 1 || !strings.Contains(stderr.String(), "-report is required") {
		t.Fatalf("run error: code=%d stderr=%q", code, stderr.String())
	}
	stderr.Reset()
	if code := runMain([]string{"-full", "-profile=J32-GoJS"}, &stderr); code != 1 || !strings.Contains(stderr.String(), "requires report") {
		t.Fatalf("full error: code=%d stderr=%q", code, stderr.String())
	}
	root, program := driverFixture(t)
	stderr.Reset()
	reportPath := filepath.Join(root, "cli.json")
	args := []string{"-profile=GoJS-reference", "-report=" + reportPath, "-go=" + program, "-llgo=" + program}
	if code := runMain(args, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("success: code=%d stderr=%q", code, stderr.String())
	}
	if got := readReport(t, reportPath); got.Result != "pass" {
		t.Fatalf("report = %+v", got)
	}
}

func TestDriverMainEntry(t *testing.T) {
	oldArgs, oldExit, oldStderr := os.Args, exitProcess, processStderr
	defer func() {
		os.Args, exitProcess, processStderr = oldArgs, oldExit, oldStderr
	}()
	var stderr strings.Builder
	exitCode := -1
	os.Args = []string{"wasmstdlib", "-h"}
	processStderr = &stderr
	exitProcess = func(code int) { exitCode = code }
	main()
	if exitCode != 0 || !strings.Contains(stderr.String(), "Usage of wasmstdlib") {
		t.Fatalf("main exit = %d, stderr = %q", exitCode, stderr.String())
	}
}

func TestDriverReportAndSummary(t *testing.T) {
	root, program := driverFixture(t)
	for _, name := range []string{"J32-GoJS", "J32-Emscripten", "J64-Emscripten", "W32-WASI", "GoJS-reference", "GoWASI-reference"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, name+".json")
			summary := filepath.Join(root, name+".md")
			t.Setenv("GITHUB_STEP_SUMMARY", summary)
			if err := run(name, path, program, program); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var r report
			if err := json.Unmarshal(data, &r); err != nil || r.Result != "pass" || r.GoVersion != "go1.27.0" || len(r.Packages) != len(acceptance)+1 {
				t.Fatalf("invalid report: %+v, %v", r, err)
			}
			p, _ := selectProfile(name)
			if (r.Implementation == "go-reference") != p.Reference {
				t.Fatalf("misclassified compiler: %+v", r)
			}
			data, err = os.ReadFile(summary)
			if err != nil || !strings.Contains(string(data), fmt.Sprintf("Passed packages: %d; failed: 0; not run: 0; source-excluded (unclassified): 1.", len(acceptance))) {
				t.Fatalf("invalid summary: %s, %v", data, err)
			}
		})
	}
	for _, mode := range []string{"bad-env", "env-failure", "list-failure", "bad-list", "missing-package", "test-failure"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("LLGO_WASMSTDLIB_TEST_MODE", mode)
			path := filepath.Join(root, mode+".json")
			err := run("J32-Emscripten", path, program, program)
			if err == nil {
				t.Fatal("lost external-command failure")
			}
			r := readReport(t, path)
			if r.Result != "fail" || r.Reason != err.Error() || len(r.Packages) != len(acceptance)+1 {
				t.Fatalf("missing failure report: %+v, %v", r, err)
			}
			if mode != "test-failure" {
				for _, e := range r.Packages {
					if e.Status != "not-run" || e.SourceSelected || e.Tests != 0 || !strings.Contains(e.Reason, "source selection not completed") {
						t.Fatalf("preparation failure classified an unvalidated package: %+v", e)
					}
				}
			}
		})
	}
	if err := run("J32-Emscripten", filepath.Join(root, "missing", "report.json"), program, program); err == nil {
		t.Fatal("lost report write failure")
	}
	t.Setenv("GITHUB_STEP_SUMMARY", root) // A directory is not appendable.
	summaryReport := filepath.Join(root, "summary-error.json")
	err := run("J32-Emscripten", summaryReport, program, program)
	if err == nil {
		t.Fatal("lost summary write failure")
	}
	r := readReport(t, summaryReport)
	if r.Result != "fail" || r.Reason != err.Error() {
		t.Fatalf("summary failure left a passing report: %+v, %v", r, err)
	}
	for _, e := range r.Packages {
		if e.SelectedForRun && (e.Status != "pass" || e.Tests != 1) {
			t.Fatalf("summary failure discarded completed tests: %+v", e)
		}
	}
	if err := run("invalid", "unused", program, program); err == nil {
		t.Fatal("accepted unknown profile")
	}
	if err := run("J32-Emscripten", "", program, program); err == nil {
		t.Fatal("accepted missing report path")
	}
}

func TestDriverPreparationFailures(t *testing.T) {
	t.Run("working directory", func(t *testing.T) {
		want := errors.New("working directory unavailable")
		old := getwdForRun
		getwdForRun = func() (string, error) { return "", want }
		defer func() { getwdForRun = old }()
		err := run("J32-Emscripten", filepath.Join(t.TempDir(), "getwd.json"), "unused", "unused")
		if !errors.Is(err, want) {
			t.Fatalf("working-directory error = %v", err)
		}
	})

	t.Run("source inventory", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		err := run("J32-Emscripten", filepath.Join(root, "inventory.json"), "unused", "unused")
		if err == nil || !strings.Contains(err.Error(), "discover source inventory") {
			t.Fatalf("source-inventory error = %v", err)
		}
	})

	t.Run("source inventory report", func(t *testing.T) {
		root, _ := driverFixture(t)
		want := errors.New("source inventory report unavailable")
		old := writeRunFile
		calls := 0
		writeRunFile = func(path string, data []byte, mode os.FileMode) error {
			calls++
			if calls == 1 {
				return os.WriteFile(path, data, mode)
			}
			return want
		}
		defer func() { writeRunFile = old }()
		err := run("J32-Emscripten", filepath.Join(root, "inventory-write.json"), "unused", "unused")
		if !errors.Is(err, want) || calls < 2 {
			t.Fatalf("source-inventory report error = %v, writes = %d", err, calls)
		}
	})
}

func TestDriverPreflightFailureReplacesPreviousSuccess(t *testing.T) {
	root, program := driverFixture(t)
	for _, priorSuccess := range []bool{false, true} {
		t.Run(fmt.Sprint(priorSuccess), func(t *testing.T) {
			path := filepath.Join(root, fmt.Sprintf("rerun-%v.json", priorSuccess))
			if priorSuccess {
				if err := run("GoJS-reference", path, program, program); err != nil {
					t.Fatal(err)
				}
				if r := readReport(t, path); r.Result != "pass" {
					t.Fatalf("initial report did not pass: %+v", r)
				}
			}
			missing := filepath.Join(root, "missing-go")
			err := run("GoJS-reference", path, missing, program)
			var execErr *exec.Error
			if err == nil || (!errors.Is(err, os.ErrNotExist) && !errors.As(err, &execErr)) {
				t.Fatalf("missing executable error = %v", err)
			}
			r := readReport(t, path)
			if r.Result != "fail" || r.Reason != err.Error() || r.Implementation != "go-reference" || len(r.Packages) != len(acceptance)+1 {
				t.Fatalf("preflight failure report = %+v, %v", r, err)
			}
			for _, e := range r.Packages {
				if e.Status != "not-run" || e.SourceSelected || e.Tests != 0 || e.SelectedForRun != (e.Package != "syscall") {
					t.Fatalf("preflight failure retained old test results: %+v", e)
				}
			}
		})
	}
}

func TestProfilesAndCommands(t *testing.T) {
	for _, name := range []string{"J32-GoJS", "J32-Emscripten", "J64-Emscripten", "W32-WASI", "GoJS-reference", "GoWASI-reference"} {
		t.Run(name, func(t *testing.T) {
			p, err := selectProfile(name)
			if err != nil {
				t.Fatal(err)
			}
			c := testCommand(p, "selected-go", "selected-llgo", "/Go root", "errors")
			joined := strings.Join(c.Args, " ")
			for _, arg := range []string{"test -v -count=1 -timeout=60s", "./test/std/errors"} {
				if !strings.Contains(joined, arg) {
					t.Fatalf("missing %q: %v", arg, c)
				}
			}
			if p.Reference {
				if c.Env["GOMAXPROCS"] != "1" {
					t.Fatalf("reference command can request unsupported wasm OS threads: %v", c)
				}
				if c.Program != "selected-go" || c.Env["GOARCH"] != "wasm" || c.Env["GOOS"] != p.GOOS || strings.Contains(joined, "-emulator") {
					t.Fatalf("reference command uses wrong compiler/context: %v", c)
				}
				if !strings.Contains(joined, "go_"+p.GOOS+"_wasm_exec") || !strings.Contains(joined, "-exec=\"") {
					t.Fatalf("official helper not quoted: %v", c)
				}
			} else {
				if c.Program != "selected-llgo" || c.Env["LLGO_BUILD_CACHE"] != "on" {
					t.Fatalf("LLGo profile command = %v", c)
				}
				if p.Target != "" && !strings.Contains(joined, "-target "+p.Target+" -emulator") {
					t.Fatalf("named target command = %v", c)
				}
				if p.Target == "" && (c.Env["GOOS"] != p.GOOS || c.Env["GOARCH"] != "wasm" || strings.Contains(joined, "-target")) {
					t.Fatalf("raw GoJS command = %v", c)
				}
			}
		})
	}
	if _, err := selectProfile("EC32"); err == nil {
		t.Fatal("accepted a retired profile name")
	}
}

func TestProfileSourceContexts(t *testing.T) {
	tests := map[string]struct{ tags, cgo string }{
		"J32-GoJS":         {"llgo,osusergo,llgo.wasm.gc.linear", "0"},
		"J32-Emscripten":   {"llgo,osusergo,llgo.wasm.gc.linear,llgo.wasm.emscripten", "1"},
		"J64-Emscripten":   {"llgo,osusergo,llgo.wasm.gc.linear,llgo.wasm.emscripten,llgo.wasm.emscripten.memory64", "1"},
		"W32-WASI":         {"llgo,osusergo,llgo.wasm.gc.linear,llgo.wasm.wasi", "1"},
		"GoJS-reference":   {"", "0"},
		"GoWASI-reference": {"", "0"},
	}
	for name, expected := range tests {
		p, err := selectProfile(name)
		if err != nil {
			t.Fatal(err)
		}
		tags, cgo := sourceContext(p)
		if tags != expected.tags || cgo != expected.cgo {
			t.Errorf("%s source context = (%q, %q), want (%q, %q)", name, tags, cgo, expected.tags, expected.cgo)
		}
	}
}

func TestDiscoverPreservesBuildTagExcludedDirectories(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"errors/a_test.go", "syscall/native_test.go", "encoding/binary/b_test.go", "testdata/fixture/f_test.go", "_ignored/i_test.go"} {
		path := filepath.Join(root, "test", "std", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("//go:build windows\n\npackage p\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	names, err := discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"encoding/binary", "errors", "syscall"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("discovered %v, want %v", names, want)
	}
	if _, err := discover(filepath.Join(root, "missing")); err == nil {
		t.Fatal("accepted a missing test tree")
	}
}

func TestSourceSelection(t *testing.T) {
	input := `{"ImportPath":"` + packagePrefix + `errors","XTestGoFiles":["errors_test.go"]}
{"ImportPath":"` + packagePrefix + `sort","TestGoFiles":["sort_test.go"]}
{"ImportPath":"` + packagePrefix + `no-tests","GoFiles":["helper.go"]}`
	got, err := sourceSelection([]byte(input))
	if err != nil || !reflect.DeepEqual(got, map[string]bool{"errors": true, "sort": true}) {
		t.Fatalf("selection = %v, %v", got, err)
	}
	for _, bad := range []string{
		`{"ImportPath":"` + packagePrefix + `errors","Error":{"Err":"missing package"}}`,
		`{"ImportPath":"` + packagePrefix + `errors","DepsErrors":[{"Err":"missing dependency"}]}`,
		`{"ImportPath":"outside/module"}`,
		`broken JSON`,
	} {
		if _, err := sourceSelection([]byte(bad)); err == nil {
			t.Errorf("accepted invalid selection %s", bad)
		}
	}
}

func TestInventoryDoesNotPromoteUntestedPackages(t *testing.T) {
	names := []string{"errors", "sort", "syscall"}
	selected := map[string]bool{"errors": true, "sort": true}
	cases := []testCase{{"errors", "TestAs"}}
	entries, err := inventory(names, selected, cases)
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Status != "not-run" || !entries[0].SelectedForRun || entries[1].SelectedForRun || entries[1].Status != "not-run" || entries[2].Status != "source-excluded" {
		t.Fatalf("incorrect classification: %+v", entries)
	}
	for _, bad := range [][]testCase{nil, {{"missing", "Test"}}, {{"syscall", "Test"}}, {{"errors", ""}}, {{"errors", "Test"}, {"errors", "TestAgain"}}} {
		if _, err := inventory(names, selected, bad); err == nil {
			t.Errorf("accepted invalid slice %v", bad)
		}
	}
}

func TestValidateExecution(t *testing.T) {
	pass := "=== RUN   TestAs\n--- PASS: TestAs (0.00s)\nPASS\n"
	for _, text := range []string{pass, strings.ReplaceAll(pass, "\n", "\r\n")} {
		if count, err := validateOutput([]byte(text), "TestAs"); err != nil || count != 1 {
			t.Fatalf("valid output: count %d, error %v", count, err)
		}
	}
	withSkippedSubtest := "=== RUN   TestAs\n=== RUN   TestAs/skipped\n    --- SKIP: TestAs/skipped (0.00s)\n--- PASS: TestAs (0.00s)\nPASS\n"
	if count, err := validateOutput([]byte(withSkippedSubtest), "TestAs"); err != nil || count != 1 {
		t.Fatalf("valid intentional subtest skip: count %d, error %v", count, err)
	}
	for _, text := range []string{"", "PASS\n", "--- PASS: TestOther (0.00s)\nPASS\n", pass + "PASS\n", pass + "--- FAIL: failed (0.00s)\n"} {
		if _, err := validateOutput([]byte(text), "TestAs"); err == nil {
			t.Errorf("accepted incomplete/failed execution: %q", text)
		}
	}
	// fmt.Print's lack of a trailing newline must not undercount a passing
	// test. A nested result must not satisfy the top-level witness instead.
	output := "=== RUN   TestAs\nuser --- output--- PASS: TestAs (0.00s)\n    --- PASS: TestAs/nested (0.00s)\nPASS\n"
	if count, err := validateOutput([]byte(output), "TestAs"); err != nil || count != 1 {
		t.Fatalf("stdout-prefixed result: count %d, error %v", count, err)
	}
	for _, bad := range []string{
		"    --- PASS: TestAs/nested (0.00s)\nPASS\n",
		output + "user output--- SKIP: TestOther (0.00s)\n",
		output + "user output--- FAIL: TestOther (0.00s)\n",
	} {
		if _, err := validateOutput([]byte(bad), "TestAs"); err == nil {
			t.Errorf("accepted nested-only, skipped, or failed execution: %q", bad)
		}
	}
}

func TestRunSliceRecordsFailureAndStops(t *testing.T) {
	cases := []testCase{{"a", "TestA"}, {"b", "TestB"}, {"c", "TestC"}}
	for _, processFailure := range []bool{false, true} {
		entries, err := inventory([]string{"a", "b", "c", "unverified"}, map[string]bool{"a": true, "b": true, "c": true, "unverified": true}, cases)
		if err != nil {
			t.Fatal(err)
		}
		r := &report{Profile: "J64-Emscripten", Packages: entries}
		calls := 0
		var saved report
		err = runSlice(r, cases, func(tc testCase) ([]byte, error) {
			calls++
			if tc.Package == "b" {
				if processFailure {
					return []byte("--- PASS: TestB (0.00s)\nPASS\n"), errors.New("exit status 7")
				}
				return nil, nil // An empty, exit-zero run is not acceptance.
			}
			return []byte("--- PASS: " + tc.Witness + " (0.00s)\nPASS\n"), nil
		}, func(r *report) error {
			data, _ := json.Marshal(r)
			return json.Unmarshal(data, &saved)
		})
		if err == nil || calls != 2 || saved.Result != "fail" || saved.Packages[0].Status != "pass" || saved.Packages[1].Status != "fail" || saved.Packages[1].Tests != 0 || saved.Packages[2].Status != "not-run" || saved.Packages[3].Status != "not-run" {
			t.Fatalf("failure was not preserved: calls=%d saved=%+v err=%v", calls, saved, err)
		}
	}
}

func TestRunSliceSuccessAndSaveErrors(t *testing.T) {
	cases := []testCase{{"errors", "TestAs"}}
	entries, _ := inventory([]string{"errors"}, map[string]bool{"errors": true}, cases)
	r := &report{Packages: entries}
	run := func(testCase) ([]byte, error) { return []byte("--- PASS: TestAs (0.00s)\nPASS\n"), nil }
	if err := runSlice(r, cases, run, func(*report) error { return nil }); err != nil || r.Result != "pass" {
		t.Fatalf("success = %+v, %v", r, err)
	}
	want := errors.New("report is not writable")
	if err := runSlice(r, cases, run, func(*report) error { return want }); !errors.Is(err, want) {
		t.Fatalf("lost report error: %v", err)
	}
	if err := runSlice(&report{}, cases, run, func(*report) error { return nil }); err == nil {
		t.Fatal("accepted missing inventory")
	}
	runErr := errors.New("compiler failed")
	saves := 0
	err := runSlice(r, cases, func(testCase) ([]byte, error) { return nil, runErr }, func(*report) error {
		saves++
		if saves == 1 {
			return nil
		}
		return want
	})
	if !errors.Is(err, runErr) || !errors.Is(err, want) {
		t.Fatalf("lost execution or report write failure: %v", err)
	}
}

func TestCommandEnvironmentDoesNotInheritSourceProfile(t *testing.T) {
	t.Setenv("GOOS", "windows")
	t.Setenv("GOARCH", "386")
	t.Setenv("LLGO_ROOT", "wrong checkout")
	t.Setenv("GOFLAGS", "-run=TestAs")
	cmd := subprocess("/selected/root", command{"llgo", nil, map[string]string{"LLGO_BUILD_CACHE": "off"}})
	joined := strings.Join(cmd.Env, "\n")
	if strings.Contains(joined, "GOOS=") || strings.Contains(joined, "GOARCH=") || strings.Contains(joined, "wrong checkout") || !strings.Contains(joined, "LLGO_ROOT=/selected/root") || cmd.Dir != "/selected/root" {
		t.Fatalf("inherited unrelated source context: %v", cmd.Env)
	}
	if strings.Contains(joined, "-run=TestAs") || !strings.Contains(joined, "GOFLAGS=") {
		t.Fatalf("inherited test filters: %v", cmd.Env)
	}
}

func TestCommandIgnoresPersistentGoFlags(t *testing.T) {
	// Use a real Go command: a fake subprocess cannot detect Go's fallback
	// from an empty GOFLAGS environment variable to its persistent config.
	goCmd, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	config := filepath.Join(root, "goenv")
	if err := os.WriteFile(config, []byte("GOFLAGS=-run=OnlyWitness\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOENV", config)
	t.Setenv("GOFLAGS", "")
	output, err := executeStructured(root, command{goCmd, []string{"env", "-json", "GOFLAGS", "GOENV"}, nil})
	if err != nil {
		t.Fatal(err)
	}
	var env struct{ GOFLAGS, GOENV string }
	if err := json.Unmarshal(output, &env); err != nil || env.GOFLAGS != "" || env.GOENV != "" {
		t.Fatalf("persistent configuration affected test selection: %s, %v", output, err)
	}
}
