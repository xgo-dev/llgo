package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestFullChildCommandProfiles(t *testing.T) {
	for _, name := range []string{"J32-GoJS", "J32-Emscripten", "J64-Emscripten", "W32-WASI", "GoJS-reference", "GoWASI-reference"} {
		p, err := fullProfile(name)
		if err != nil {
			t.Fatal(err)
		}
		cmd := fullPanicCommand(p, "/repo", "/goroot", "/compiled-test")
		if cmd.Program != "timeout" || cmd.Args[1] != "30s" || !slices.Contains(cmd.Args, "/compiled-test") || cmd.Args[len(cmd.Args)-1] != "-llgo.caller-panic-child" {
			t.Fatalf("%s did not reuse and bound the child binary: %+v", name, cmd)
		}
		joined := strings.Join(cmd.Args, " ")
		switch {
		case p.Reference:
			if !strings.Contains(joined, "go_"+p.GOOS+"_wasm_exec") || cmd.Env["GOMAXPROCS"] != "1" {
				t.Fatalf("%s must use the official Go host helper: %+v", name, cmd)
			}
		case p.Target == "wasi":
			if !slices.Contains(cmd.Args, "wasmtime") || !slices.Contains(cmd.Args, "exceptions=y") {
				t.Fatalf("%s missing WASI runner: %+v", name, cmd)
			}
		default:
			if !slices.Contains(cmd.Args, "node") || !strings.Contains(joined, "emscripten") {
				t.Fatalf("%s missing JS runner: %+v", name, cmd)
			}
		}
	}
}

func TestFullCommandsRetainPCLNOnlyForMetadataConsumers(t *testing.T) {
	p, err := fullProfile("J64-Emscripten")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range []string{"test", "test/go", "test/go/callercross", "test/llgoext", "test/std/runtime", "test/std/runtime/debug", "test/std/net/http/pprof", "test/std/log/slog"} {
		if cmd := fullCommand(p, "go", "llgo", "/goroot", pkg); slices.Contains(cmd.Args, "-pclntab=none") {
			t.Errorf("%s lost PCLN coverage: %+v", pkg, cmd)
		}
	}
	for _, pkg := range []string{"test/std/encoding/ascii85", "test/std/crypto/rsa", "test/std/net", "test/syncpool"} {
		if cmd := fullCommand(p, "go", "llgo", "/goroot", pkg); !slices.Contains(cmd.Args, "-pclntab=none") {
			t.Errorf("%s retained unrelated PCLN link work: %+v", pkg, cmd)
		}
	}
}

func TestFullFatalValidatorsRejectFalsePositives(t *testing.T) {
	exitErr := fullHostCheckTestExit(t)
	root := t.TempDir()
	dir := filepath.Join(root, "test", "go")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "caller_runtime_test.go"), []byte("panic() // PANIC_MARK\ncall() // PANIC_CALLER_MARK\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	panicOut := "panic: acceptance-boom\ngoroutine 1 [running]:\ncallerPanicBoom\ncaller_runtime_test.go:1\ncallerPanicCaller\ncaller_runtime_test.go:2\n"
	if err := validateFullPanic(root, []byte(panicOut), exitErr); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []struct {
		out string
		err error
	}{
		{panicOut, nil},
		{"memory access out of bounds", exitErr},
		{strings.ReplaceAll(panicOut, "go:2", "go:20"), exitErr},
	} {
		if validateFullPanic(root, []byte(bad.out), bad.err) == nil {
			t.Fatalf("accepted false panic result %q, %v", bad.out, bad.err)
		}
	}

	if err := validateFullFinalizerInvalid([]byte("fatal error: runtime.SetFinalizer: second argument is int, not a function"), exitErr, "non-function"); err != nil {
		t.Fatal(err)
	}
	if err := validateFullFinalizerInvalid([]byte("fatal error: runtime.SetFinalizer: cannot pass *value to finalizer func(...*value) because dotdotdot"), exitErr, "variadic"); err != nil {
		t.Fatal(err)
	}
	if validateFullFinalizerInvalid([]byte("runtime.SetFinalizer: cannot pass"), nil, "wrong type") == nil {
		t.Fatal("accepted successful invalid-finalizer child")
	}

	if err := validateFullBuiltinPrint([]byte(fullBuiltinPrintWant), nil); err != nil {
		t.Fatal(err)
	}
	if validateFullBuiltinPrint([]byte(fullBuiltinPrintWant+"PASS\n"), nil) == nil {
		t.Fatal("accepted noisy builtin-print output")
	}
	if err := validateFullGoexitLifecycle([]byte("WORKER_RETURNING\nfatal error: no goroutines (main called runtime.Goexit) - deadlock!"), exitErr); err != nil {
		t.Fatal(err)
	}
	if validateFullGoexitLifecycle([]byte("fatal error: no goroutines (main called runtime.Goexit) - deadlock!\nWORKER_RETURNING"), exitErr) == nil {
		t.Fatal("accepted reversed Goexit lifecycle")
	}
}

func TestFullHostCheckExitHelper(t *testing.T) {
	if os.Getenv("LLGO_FULL_HOST_CHECK_EXIT_HELPER") == "1" {
		os.Exit(2)
	}
}

func fullHostCheckTestExit(t *testing.T) error {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(exe, "-test.run=^TestFullHostCheckExitHelper$")
	child.Env = append(os.Environ(), "LLGO_FULL_HOST_CHECK_EXIT_HELPER=1")
	exitErr := child.Run()
	var exit *exec.ExitError
	if !errors.As(exitErr, &exit) || exit.ExitCode() != 2 {
		t.Fatalf("child did not return a real exit status: %v", exitErr)
	}
	return exitErr
}

func TestFullHostChecksReuseThePackageBuild(t *testing.T) {
	exitErr := fullHostCheckTestExit(t)
	root := t.TempDir()
	dir := filepath.Join(root, "test", "go")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := "package gotest\nfunc TestWitness(t *T) {}\nfunc boom() {} // PANIC_MARK\nfunc caller() {} // PANIC_CALLER_MARK\n"
	if err := os.WriteFile(filepath.Join(dir, "caller_runtime_test.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	structured := func(_ string, cmd command) ([]byte, error) {
		if cmd.Args[0] == "env" {
			return []byte("/goroot"), nil
		}
		return json.Marshal(selectedPackage{Dir: dir, TestGoFiles: []string{"caller_runtime_test.go"}})
	}
	builds, children := 0, 0
	var artifact string
	run := func(_ string, cmd command) ([]byte, error) {
		if slices.Contains(cmd.Args, "./test/go") {
			builds++
			i := slices.Index(cmd.Args, "-o")
			if i < 0 {
				t.Fatal("package test did not retain its artifact")
			}
			artifact = cmd.Args[i+1]
			return []byte("--- PASS: TestWitness (0.00s)\nPASS\n"), nil
		}
		children++
		if artifact == "" || !slices.Contains(cmd.Args, artifact) {
			t.Fatal("host check did not reuse the package artifact")
		}
		selector := cmd.Args[len(cmd.Args)-1]
		if selector == "-llgo.caller-panic-child" {
			return []byte("panic: acceptance-boom\ngoroutine 1 [running]:\ncallerPanicBoom\ncaller_runtime_test.go:3\ncallerPanicCaller\ncaller_runtime_test.go:4\n"), exitErr
		}
		out := "fatal error: runtime.SetFinalizer: cannot pass *gotest.value to finalizer func(*int)"
		if strings.Contains(selector, "non-function") {
			out = "fatal error: runtime.SetFinalizer: second argument is int, not a function"
		} else if strings.Contains(selector, "variadic") {
			out += " because dotdotdot"
		}
		return []byte(out), exitErr
	}
	reportPath := filepath.Join(root, "report.json")
	if err := runFullAt(root, "W32-WASI", reportPath, "go", "llgo", 0, 1, structured, run); err != nil {
		t.Fatal(err)
	}
	if builds != 1 || children != 1+len(fullFinalizerInvalidCases) {
		t.Fatalf("builds/children = %d/%d", builds, children)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report struct{ Packages []fullPackage }
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Packages) != 1 || len(report.Packages[0].HostChecks) != 1+len(fullFinalizerInvalidCases) {
		t.Fatalf("missing host checks: %s", data)
	}
	if _, err := os.Stat(filepath.Dir(artifact)); !os.IsNotExist(err) {
		t.Fatalf("temporary artifact directory retained: %v", err)
	}
}

func TestFullHostChecksCoverRootAndLLGoExtensionSuites(t *testing.T) {
	exitErr := fullHostCheckTestExit(t)
	root := t.TempDir()
	packages := []selectedPackage{
		{Dir: filepath.Join(root, "test"), TestGoFiles: []string{"root_test.go"}},
		{Dir: filepath.Join(root, "test", "llgoext"), TestGoFiles: []string{"llgoext_test.go"}},
	}
	for i, pkg := range packages {
		if err := os.MkdirAll(pkg.Dir, 0o755); err != nil {
			t.Fatal(err)
		}
		source := "package acceptance\nfunc TestWitness(t *T) {}\n"
		if err := os.WriteFile(filepath.Join(pkg.Dir, pkg.TestGoFiles[0]), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		packages[i] = pkg
	}
	var listOutput []byte
	for _, pkg := range packages {
		data, err := json.Marshal(pkg)
		if err != nil {
			t.Fatal(err)
		}
		listOutput = append(listOutput, data...)
		listOutput = append(listOutput, '\n')
	}
	structured := func(_ string, cmd command) ([]byte, error) {
		if cmd.Args[0] == "env" {
			return []byte("/goroot"), nil
		}
		return listOutput, nil
	}
	for _, profileName := range []string{"J32-GoJS", "W32-WASI"} {
		t.Run(profileName, func(t *testing.T) {
			artifacts := map[string]string{}
			builds, children := 0, 0
			run := func(_ string, cmd command) ([]byte, error) {
				selector := cmd.Args[len(cmd.Args)-1]
				switch selector {
				case "./test", "./test/llgoext":
					builds++
					i := slices.Index(cmd.Args, "-o")
					if i < 0 {
						t.Fatalf("%s did not retain its artifact: %+v", selector, cmd)
					}
					artifacts[selector] = cmd.Args[i+1]
					return []byte("--- PASS: TestWitness (0.00s)\nPASS\n"), nil
				case "-llgo.builtin-print-child":
					children++
					if !slices.Contains(cmd.Args, artifacts["./test"]) {
						t.Fatalf("builtin-print check did not reuse root artifact: %+v", cmd)
					}
					return []byte(fullBuiltinPrintWant), nil
				case "-llgo.main-goexit-lifecycle-child":
					children++
					if !slices.Contains(cmd.Args, artifacts["./test/llgoext"]) {
						t.Fatalf("Goexit check did not reuse llgoext artifact: %+v", cmd)
					}
					return []byte("WORKER_RETURNING\nfatal error: no goroutines (main called runtime.Goexit) - deadlock!"), exitErr
				default:
					t.Fatalf("unexpected command: %+v", cmd)
					return nil, nil
				}
			}
			reportPath := filepath.Join(root, profileName+".json")
			if err := runFullAt(root, profileName, reportPath, "go", "llgo", 0, 1, structured, run); err != nil {
				t.Fatal(err)
			}
			if builds != 2 || children != 2 {
				t.Fatalf("builds/children = %d/%d", builds, children)
			}
			wantExt := ".wasm"
			if profileName == "J32-GoJS" {
				wantExt = ".mjs"
			}
			for pkg, artifact := range artifacts {
				if filepath.Ext(artifact) != wantExt {
					t.Errorf("%s artifact = %q, want %s", pkg, artifact, wantExt)
				}
			}
			data, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatal(err)
			}
			var report struct{ Packages []fullPackage }
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			if len(report.Packages) != 2 {
				t.Fatalf("packages = %s", data)
			}
			for _, pkg := range report.Packages {
				if pkg.Status != "pass" || len(pkg.HostChecks) != 1 || pkg.HostChecks[0].Status != "pass" {
					t.Fatalf("host check was not recorded: %+v", pkg)
				}
			}
		})
	}
}
