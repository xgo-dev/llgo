//go:build !windows

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

package build

import (
	"bytes"
	stdcontext "context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

func TestRunInEmulatorValidation(t *testing.T) {
	commands := commandEnv{dir: t.TempDir(), environ: os.Environ()}
	if err := runInEmulator(commands, "", "", nil, "", "", &Config{CompileOnly: true}, ModeRun, false); err != nil {
		t.Fatalf("compile-only emulator run failed: %v", err)
	}
	if err := runInEmulator(commands, "", "", nil, "", "", &Config{Target: "demo"}, ModeRun, false); err == nil {
		t.Fatal("missing emulator succeeded")
	} else {
		var runnerErr *runnerFailure
		if !errors.As(err, &runnerErr) || runnerErr.status != runnerStatusNotConfigured || runnerErr.target != "demo" {
			t.Fatalf("missing emulator error = %#v, want classified runner failure", err)
		}
	}
	details := runnerDetails{phase: "run", target: "demo", artifact: "firmware.elf", packageName: "example/main"}
	if err := runEmuCmd(commands, nil, "'", nil, false, false, details); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("malformed emulator command error = %v", err)
	}
	if err := runEmuCmd(commands, nil, "   ", nil, false, false, details); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty emulator command error = %v", err)
	}
}

func TestWASIHostDirectories(t *testing.T) {
	work, temp, cwd := wasiHostDirectories(`C:\project with spaces`, `D:\Temp`, true)
	if work != `C:\project with spaces:/work` || temp != `D:\Temp:/tmp` || cwd != "/work" {
		t.Fatalf("Windows WASI directories = %q, %q, %q", work, temp, cwd)
	}
	work, temp, cwd = wasiHostDirectories("/project with spaces", "/host-temp", false)
	if work != "/project with spaces" || temp != "/tmp" || cwd != "/project with spaces" {
		t.Fatalf("Unix WASI directories = %q, %q, %q", work, temp, cwd)
	}
}

func TestWASIRunnerIsolatesEngineLogging(t *testing.T) {
	dir := t.TempDir()
	// Guest stderr may itself look like an engine log. It must survive intact;
	// the runner controls the engine's logger instead of filtering the stream.
	guestLog := `{"level":"WARN","target":"wasmer","fields":{"message":"guest stderr"}}`
	script := "#!/bin/sh\n" +
		"if [ \"$RUST_LOG\" != off ]; then echo 'engine diagnostic' >&2; fi\n" +
		"echo 'guest stdout'\n" +
		"echo '" + guestLog + "' >&2\nexit 7\n"
	if err := os.WriteFile(filepath.Join(dir, "wasmer"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	commands := commandEnv{dir: dir, environ: withEnv(os.Environ(), "RUST_LOG=warn")}
	for _, phase := range []string{"test", "run"} {
		t.Run(phase, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := runEmuCmdTo(commands, map[string]string{"": "program.wasm"},
				crosscompile.WASIThreadedEmulator, nil, false, false,
				runnerDetails{phase: phase}, &stdout, &stderr)
			var failure *runnerFailure
			if !errors.As(err, &failure) || failure.exitCode != 7 || failure.status != runnerStatusExit {
				t.Fatalf("runner failure = %v, want exit 7", err)
			}
			wantErr := guestLog + "\n"
			if phase == "run" {
				wantErr = "engine diagnostic\n" + wantErr
			}
			if stdout.String() != "guest stdout\n" || stderr.String() != wantErr {
				t.Fatalf("stdout = %q, stderr = %q; want guest output and stderr %q", stdout.String(), stderr.String(), wantErr)
			}
		})
	}
	if commands.lookup("RUST_LOG") != "warn" {
		t.Fatal("test runner changed the caller's logging environment")
	}
}

func TestWASIThreadedEmulatorHostContract(t *testing.T) {
	dir := t.TempDir()
	runner := filepath.Join(dir, "wasmer")
	argsFile := filepath.Join(dir, "args")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\n", argsFile)
	if err := os.WriteFile(runner, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	commands := commandEnv{dir: dir, environ: append(os.Environ(), "PATH="+dir, "LLGO_STRESS_PROFILE=quick", "LLGO_PRIVATE_SENTINEL=not-forwarded")}
	artifact := filepath.Join(dir, "program.wasm")
	err := runEmuCmd(commands, map[string]string{"": artifact}, crosscompile.WASIThreadedEmulator,
		[]string{"-test.v"}, false, false, runnerDetails{phase: "test"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSpace(string(data)), "\n")
	if want := "--volume=" + dir; !slices.Contains(args, want) {
		t.Fatalf("Wasmer arguments %q omit working directory %q", args, want)
	}
	for _, want := range []string{"--env=PWD=" + dir, "--env=PATH=" + dir} {
		if !slices.Contains(args, want) {
			t.Fatalf("Wasmer arguments omit %q: %q", want, args)
		}
	}
	if slices.Contains(args, "--env=LLGO_PRIVATE_SENTINEL=not-forwarded") {
		t.Fatal("Wasmer forwarded an unrelated host environment variable")
	}
	if got, want := args[len(args)-4:], []string{"--env=LLGO_STRESS_PROFILE=quick", artifact, "--", "-test.v"}; !slices.Equal(got, want) {
		t.Fatalf("runner tail = %q, want %q", got, want)
	}
	commands.dir = ""
	if err := runEmuCmd(commands, map[string]string{"": artifact}, crosscompile.WASIThreadedEmulator,
		nil, false, false, runnerDetails{phase: "test"}); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	args = strings.Split(strings.TrimSpace(string(data)), "\n")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if want := "--volume=" + cwd; !slices.Contains(args, want) {
		t.Fatalf("Wasmer arguments %q omit default working directory %q", args, want)
	}
}

func TestNativeWasmerRunUsesThreadedHostContract(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	if err := os.WriteFile(filepath.Join(dir, "wasmer"), []byte(fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\n", argsFile)), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(llgoWasmRuntime, "wasmer")
	commands := commandEnv{dir: dir, environ: os.Environ()}
	conf := &Config{Goos: "wasip1", Goarch: "wasm", RunArgs: []string{"hello world"}}
	artifact := filepath.Join(dir, "app.wasm")
	if err := runNative(&context{commands: commands}, artifact, dir, "main", conf, ModeRun); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, want := range []string{"run", "--cranelift", "--enable-exceptions", "--enable-simd", "--stack-size=1048576", "--volume=" + dir, "--env=PWD=" + dir} {
		if !slices.Contains(args, want) {
			t.Fatalf("Wasmer arguments omit %q: %q", want, args)
		}
	}
	if !slices.Equal(args[len(args)-3:], []string{artifact, "--", "hello world"}) {
		t.Fatalf("artifact/arguments = %q", args)
	}
}

func TestWASIThreadedEmulatorReportsInvalidWorkingDirectory(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux getcwd reports a removed working directory")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	err := runEmuCmd(commandEnv{environ: os.Environ()}, map[string]string{"": "program.wasm"},
		crosscompile.WASIThreadedEmulator, nil, false, false, runnerDetails{phase: "test"})
	if err == nil || !strings.Contains(err.Error(), "resolve Wasmer working directory") {
		t.Fatalf("runner error = %v, want working-directory failure", err)
	}
}

func TestRunInEmulatorFailureDiagnostics(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	commands := commandEnv{dir: t.TempDir(), environ: os.Environ()}
	artifact := filepath.Join(t.TempDir(), "program.mjs")
	template := fmt.Sprintf("%q -test.run=^TestRunNativeTestHelper$ -- exit %q", executable, "{}")
	conf := &Config{Target: "emscripten", RunArgs: []string{"ignored-program-argument"}}
	err = runInEmulator(commands, template, "emscripten", map[string]string{"": artifact, "out": artifact}, "", "example/main", conf, ModeRun, false)
	if err == nil {
		t.Fatal("runner with non-zero exit status unexpectedly succeeded")
	}

	var runnerErr *runnerFailure
	if !errors.As(err, &runnerErr) {
		t.Fatalf("error type = %T, want *runnerFailure: %v", err, err)
	}
	if runnerErr.phase != "run" || runnerErr.target != "emscripten" || runnerErr.profile != "emscripten" ||
		runnerErr.artifact != artifact || runnerErr.packageName != "example/main" || runnerErr.status != runnerStatusExit || runnerErr.exitCode != 3 {
		t.Fatalf("runner failure = %+v", runnerErr)
	}
	if code, ok := RunnerExitCode(err); !ok || code != 3 {
		t.Fatalf("RunnerExitCode(%v) = (%d, %v), want (3, true)", err, code, ok)
	}
	for _, want := range []string{
		"phase=run",
		"target=emscripten",
		"profile=emscripten",
		fmt.Sprintf("artifact=%q", artifact),
		fmt.Sprintf("runner=%q", executable),
		`package="example/main"`,
		"status=exit",
		"exit_code=3",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("runner error %q does not contain %q", err, want)
		}
	}
}

func TestRunnerExitCodeOnlyForGuestExit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		phase  string
		status string
		code   int
		want   bool
	}{
		{"guest exit", "run", runnerStatusExit, 7, true},
		{"test exit", "test", runnerStatusExit, 7, false},
		{"timeout", "run", runnerStatusTimeout, -1, false},
		{"start failure", "run", runnerStatusStart, -1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := newRunnerFailure(runnerDetails{phase: tc.phase}, "runner", tc.status, tc.code, errors.New("runner failed"))
			code, ok := RunnerExitCode(fmt.Errorf("wrapped: %w", err))
			if ok != tc.want || (ok && code != tc.code) {
				t.Fatalf("RunnerExitCode = (%d, %v), want (%d, %v)", code, ok, tc.code, tc.want)
			}
		})
	}
}

func TestRunInEmulatorUnavailableRunner(t *testing.T) {
	commands := commandEnv{dir: t.TempDir(), environ: os.Environ()}
	missing := filepath.Join(t.TempDir(), "missing-runner")
	artifact := filepath.Join(t.TempDir(), "program.wasm")
	conf := &Config{Target: "wasi"}
	err := runInEmulator(commands, fmt.Sprintf("%q %q", missing, "{}"), "wasi-preview1",
		map[string]string{"": artifact, "out": artifact}, "", "example/test", conf, ModeTest, false)
	if err == nil {
		t.Fatal("missing runner unexpectedly succeeded")
	}

	var runnerErr *runnerFailure
	if !errors.As(err, &runnerErr) {
		t.Fatalf("error type = %T, want *runnerFailure: %v", err, err)
	}
	if runnerErr.status != runnerStatusUnavailable || runnerErr.runner != missing || runnerErr.exitCode != -1 {
		t.Fatalf("runner failure = %+v", runnerErr)
	}
	for _, want := range []string{
		"phase=test",
		"target=wasi",
		"profile=wasi-preview1",
		fmt.Sprintf("artifact=%q", artifact),
		fmt.Sprintf("runner=%q", missing),
		`package="example/test"`,
		"status=unavailable",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("runner error %q does not contain %q", err, want)
		}
	}
}

func TestRunInEmulatorTimeout(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	commands := commandEnv{dir: t.TempDir(), environ: os.Environ()}
	artifact := filepath.Join(t.TempDir(), "program.mjs")
	template := fmt.Sprintf("%q -test.run=^TestRunNativeTestHelper$ -- hang %q", executable, "{}")
	conf := &Config{Target: "emscripten", RunnerTimeout: 50 * time.Millisecond}

	started := time.Now()
	err = runInEmulator(commands, template, "emscripten", map[string]string{"": artifact, "out": artifact}, "", "example/main", conf, ModeRun, false)
	if err == nil {
		t.Fatal("hanging runner unexpectedly succeeded")
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("hanging runner took %s to stop", elapsed)
	}

	var runnerErr *runnerFailure
	if !errors.As(err, &runnerErr) {
		t.Fatalf("error type = %T, want *runnerFailure: %v", err, err)
	}
	if runnerErr.status != runnerStatusTimeout || runnerErr.exitCode != -1 || runnerErr.timeout != 50*time.Millisecond {
		t.Fatalf("runner failure = %+v", runnerErr)
	}
	if !errors.Is(err, stdcontext.DeadlineExceeded) {
		t.Fatalf("runner error does not wrap context deadline: %v", err)
	}
	for _, want := range []string{
		"phase=run",
		"target=emscripten",
		"profile=emscripten",
		"status=timeout",
		"timeout=50ms",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("runner error %q does not contain %q", err, want)
		}
	}
}

func testPrograms(names ...string) []testProgram {
	programs := make([]testProgram, len(names))
	for i, name := range names {
		programs[i] = testProgram{app: name + ".test", pkgName: name}
	}
	return programs
}

func TestRunNativeTest(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	commands := commandEnv{
		dir:     t.TempDir(),
		environ: withEnv(os.Environ(), "LLGO_RUN_NATIVE_TEST_ENV=isolated"),
	}
	args := []string{"-test.run=^TestRunNativeTestHelper$", "--"}

	t.Run("success", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		conf := &Config{PrintCommands: true, RunArgs: append(args, "success")}
		program := testProgram{app: executable, pkgDir: t.TempDir(), pkgName: "success"}
		if err := runNativeTest(commands, program, conf, &stdout, &stderr); err != nil {
			t.Fatalf("runNativeTest: %v", err)
		}
		if got := stdout.String(); !strings.Contains(got, "stdout") {
			t.Fatalf("stdout = %q, want helper output", got)
		}
		if got := stdout.String(); !strings.Contains(got, "isolated") {
			t.Fatalf("stdout = %q, want invocation environment", got)
		}
		if got := stderr.String(); !strings.Contains(got, executable+" ") || !strings.HasSuffix(got, "stderr") {
			t.Fatalf("stderr = %q, want command followed by helper stderr", got)
		}
	})

	t.Run("Go-compatible wasm runner", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		program := testProgram{
			app:       "program.wasm",
			pkgDir:    t.TempDir(),
			pkgName:   "wasm",
			runner:    fmt.Sprintf("%q -test.run=^TestRunNativeTestHelper$ -- success %q", executable, "{}"),
			runnerEnv: map[string]string{"": "program.wasm"},
		}
		if err := runNativeTest(commands, program, &Config{PrintCommands: true}, &stdout, &stderr); err != nil {
			t.Fatalf("runNativeTest with wasm runner: %v", err)
		}
		if !strings.Contains(stdout.String(), "PASS") || !strings.Contains(stderr.String(), executable) || !strings.Contains(stderr.String(), "program.wasm") {
			t.Fatalf("runner output not captured: stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
	})

	t.Run("Go-compatible wasm runner timeout", func(t *testing.T) {
		program := testProgram{
			app:       "program.wasm",
			pkgDir:    t.TempDir(),
			pkgName:   "wasm",
			runner:    fmt.Sprintf("%q -test.run=^TestRunNativeTestHelper$ -- hang %q", executable, "{}"),
			runnerEnv: map[string]string{"": "program.wasm"},
			profile:   "j32",
		}
		err := runNativeTest(commands, program, &Config{RunnerTimeout: 50 * time.Millisecond}, io.Discard, io.Discard)
		var runnerErr *runnerFailure
		if !errors.As(err, &runnerErr) || runnerErr.status != runnerStatusTimeout || runnerErr.profile != "j32" ||
			!errors.Is(err, stdcontext.DeadlineExceeded) {
			t.Fatalf("Go-compatible wasm runner timeout = %v, want j32 timeout", err)
		}
	})

	t.Run("exit error", func(t *testing.T) {
		var stderr bytes.Buffer
		conf := &Config{RunArgs: append(args, "exit")}
		program := testProgram{app: executable, pkgDir: t.TempDir(), pkgName: "exit"}
		if err := runNativeTest(commands, program, conf, io.Discard, &stderr); err == nil {
			t.Fatal("runNativeTest unexpectedly succeeded")
		}
		if got := stderr.String(); !strings.Contains(got, "exit code 3") {
			t.Fatalf("stderr = %q, want exit code", got)
		}
	})

	t.Run("signal exit", func(t *testing.T) {
		var stderr bytes.Buffer
		conf := &Config{RunArgs: append(args, "signal")}
		program := testProgram{app: executable, pkgDir: t.TempDir(), pkgName: "signal"}
		err := runNativeTest(commands, program, conf, io.Discard, &stderr)
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("runNativeTest error = %v, want signal termination", err)
		}
		status, ok := exitErr.Sys().(syscall.WaitStatus)
		if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
			t.Fatalf("process status = %v, want SIGKILL", exitErr.Sys())
		}
		if got := stderr.String(); !strings.Contains(got, "killed") || strings.Contains(got, "exit code") {
			t.Fatalf("stderr = %q, want signal reason without an exit code", got)
		}
	})

	t.Run("start error", func(t *testing.T) {
		var stderr bytes.Buffer
		program := testProgram{app: filepath.Join(t.TempDir(), "missing"), pkgName: "missing"}
		if err := runNativeTest(commands, program, &Config{}, io.Discard, &stderr); err == nil {
			t.Fatal("runNativeTest unexpectedly succeeded")
		}
		if got := stderr.String(); !strings.Contains(got, "failed to run test") {
			t.Fatalf("stderr = %q, want start error", got)
		}
	})
}

func TestGoCompatibleWasmRunner(t *testing.T) {
	t.Setenv("LLGO_WASI_THREADS", "")
	js := goCompatibleWasmRunner(&Config{Goos: "js", Goarch: "wasm"})
	if !strings.Contains(js, "emscripten-runner.mjs") || !strings.Contains(js, "--browser-only") || !strings.Contains(js, "{}") {
		t.Fatalf("js runner = %q", js)
	}
	if got := goCompatibleWasmRunner(&Config{Goos: "wasip1", Goarch: "wasm"}); got != crosscompile.WASIThreadedEmulator {
		t.Fatalf("WASI runner = %q", got)
	}
	if got := goCompatibleWasmRunner(&Config{Target: "wasi", Goos: "wasip1", Goarch: "wasm"}); got != "" {
		t.Fatalf("named target acquired raw runner %q", got)
	}
	if got := goCompatibleWasmRunner(&Config{Goos: "plan9", Goarch: "wasm"}); got != "" {
		t.Fatalf("unsupported wasm host acquired raw runner %q", got)
	}
}

func TestGoCompatibleWASIThreadRunner(t *testing.T) {
	t.Setenv("LLGO_WASI_THREADS", "1")
	if got := goCompatibleWasmRunner(&Config{Goos: "wasip1", Goarch: "wasm"}); got != crosscompile.WASIThreadedEmulator {
		t.Fatalf("WASI thread runner = %q, want %q", got, crosscompile.WASIThreadedEmulator)
	}
}

func TestWasmTestRunnerUsesPackageDirectory(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	program := testProgram{
		pkgDir: dir,
		runner: fmt.Sprintf("%q -test.run=^TestRunNativeTestHelper$ -- cwd", executable),
	}
	commands := commandEnv{environ: withEnv(os.Environ(), "PWD=stale-working-directory")}
	var stdout, stderr bytes.Buffer
	if err := runNativeTest(commands, program, &Config{}, &stdout, &stderr); err != nil {
		t.Fatalf("runner: %v; stderr=%s", err, stderr.String())
	}
	// TempDir can contain symlinked ancestors (for example /var on macOS).
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), resolved+"\n"+dir+"\n") {
		t.Fatalf("runner did not use package directory and PWD: %q", stdout.String())
	}
}

func TestRunNativeTestHelper(t *testing.T) {
	args := os.Args
	for i, arg := range args {
		if arg != "--" || i+1 == len(args) {
			continue
		}
		switch args[i+1] {
		case "cwd":
			dir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			dir, err = filepath.EvalSymlinks(dir)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintln(os.Stdout, dir)
			fmt.Fprintln(os.Stdout, os.Getenv("PWD"))
		case "success":
			fmt.Fprint(os.Stdout, "stdout")
			fmt.Fprint(os.Stdout, os.Getenv("LLGO_RUN_NATIVE_TEST_ENV"))
			fmt.Fprint(os.Stderr, "stderr")
		case "exit":
			os.Exit(3)
		case "signal":
			if err := syscall.Kill(os.Getpid(), syscall.SIGKILL); err != nil {
				t.Fatalf("kill helper: %v", err)
			}
			panic("SIGKILL returned without terminating the helper")
		case "hang":
			time.Sleep(time.Hour)
		}
		return
	}
}

func TestRunNativeTestProgramsSequential(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	programs := []testProgram{
		{app: executable, pkgDir: t.TempDir(), pkgName: "a"},
		{app: executable, pkgDir: t.TempDir(), pkgName: "b"},
	}
	conf := &Config{
		BuildParallelism:  2,
		TestRunSequential: true,
		RunArgs:           []string{"-test.run=^TestRunNativeTestHelper$", "--", "success"},
	}
	var stdout, stderr bytes.Buffer
	commands := commandEnv{dir: t.TempDir(), environ: os.Environ()}
	result := runNativeTestPrograms(commands, programs, conf, &stdout, &stderr)
	if result.failed || result.skipped != 0 {
		t.Fatalf("runNativeTestPrograms result = %+v", result)
	}
	for _, name := range []string{"a", "b"} {
		if !strings.Contains(stdout.String(), "ok  \t"+name+"\n") {
			t.Errorf("stdout does not contain result for %s: %q", name, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunNativeTestProgramsReportsRunnerTimeout(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	program := testProgram{
		app:       "program.wasm",
		pkgDir:    t.TempDir(),
		pkgName:   "example/wasm",
		runner:    fmt.Sprintf("%q -test.run=^TestRunNativeTestHelper$ -- hang %q", executable, "{}"),
		runnerEnv: map[string]string{"": "program.wasm"},
		profile:   "j32",
	}
	var stdout, stderr bytes.Buffer
	commands := commandEnv{dir: t.TempDir(), environ: os.Environ()}
	result := runNativeTestPrograms(commands, []testProgram{program},
		&Config{Target: "emscripten", RunnerTimeout: 50 * time.Millisecond}, &stdout, &stderr)
	if !result.failed || result.skipped != 0 {
		t.Fatalf("runNativeTestPrograms result = %+v", result)
	}
	for _, want := range []string{"phase=test", "profile=j32", "status=timeout", "FAIL\texample/wasm"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr %q does not contain %q", stderr.String(), want)
		}
	}
}

func TestRunNativeTestProgramsCompileOnly(t *testing.T) {
	var stdout, stderr bytes.Buffer
	result := runNativeTestPrograms(commandEnv{}, []testProgram{{
		app:     filepath.Join(t.TempDir(), "must-not-run"),
		pkgName: "compile-only",
	}}, &Config{CompileOnly: true}, &stdout, &stderr)
	if result != (testRunResult{}) {
		t.Fatalf("runNativeTestPrograms result = %+v, want no execution", result)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("compile-only test produced output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunTestProgramsLimitAndFailure(t *testing.T) {
	started := make(chan struct{}, 4)
	release := make(chan struct{})
	done := make(chan testRunResult)
	var stdout, stderr bytes.Buffer
	var active atomic.Int32
	var maximum atomic.Int32

	go func() {
		done <- runTestPrograms(testPrograms("a", "b", "c", "d"), 2, false, false, &stdout, &stderr,
			func(program testProgram, output io.Writer) error {
				now := active.Add(1)
				for {
					old := maximum.Load()
					if now <= old || maximum.CompareAndSwap(old, now) {
						break
					}
				}
				started <- struct{}{}
				<-release
				active.Add(-1)
				fmt.Fprintln(output, program.pkgName)
				if program.pkgName == "d" {
					return errors.New("failed")
				}
				return nil
			})
	}()

	<-started
	<-started
	select {
	case <-started:
		t.Fatal("more than two test programs started concurrently")
	default:
	}
	close(release)

	result := <-done
	if !result.failed {
		t.Fatal("runTestPrograms reported success after a test program failed")
	}
	if result.skipped != 0 {
		t.Fatalf("runTestPrograms skipped %d programs, want 0", result.skipped)
	}
	if got := maximum.Load(); got != 2 {
		t.Fatalf("maximum concurrency = %d, want 2", got)
	}
	for _, name := range []string{"a", "b", "c", "d"} {
		if !strings.Contains(stdout.String(), name+"\n") {
			t.Errorf("stdout does not contain output for %s: %q", name, stdout.String())
		}
	}
	if got, want := stderr.String(), "FAIL\td\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestRunTestProgramsParallelismBoundsAndOutput(t *testing.T) {
	if got := runTestPrograms(nil, 1, false, false, io.Discard, io.Discard, nil); got != (testRunResult{}) {
		t.Fatalf("empty run result = %+v", got)
	}

	for name, parallelism := range map[string]int{
		"default":  0,
		"negative": -1,
		"clamped":  2,
	} {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			result := runTestPrograms(testPrograms("pkg"), parallelism, false, false, &stdout, &stderr,
				func(_ testProgram, output io.Writer) error {
					fmt.Fprint(output, "output")
					return nil
				})
			if result.failed || result.skipped != 0 {
				t.Fatalf("runTestPrograms result = %+v", result)
			}
			if got, want := stdout.String(), "output\nok  \tpkg\n"; got != want {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestRunTestProgramsFailFast(t *testing.T) {
	var runs atomic.Int32
	result := runTestPrograms(testPrograms("a", "b", "c"), 1, true, false, io.Discard, io.Discard,
		func(testProgram, io.Writer) error {
			runs.Add(1)
			return errors.New("failed")
		})
	if !result.failed {
		t.Fatal("runTestPrograms reported success")
	}
	if result.skipped != 2 {
		t.Fatalf("skipped %d test programs, want 2", result.skipped)
	}
	if got := runs.Load(); got != 1 {
		t.Fatalf("ran %d test programs after the first failure, want 1", got)
	}
}

func TestRunTestProgramsJSONOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	result := runTestPrograms(testPrograms("json"), 1, false, true, &stdout, &stderr,
		func(testProgram, io.Writer) error {
			return nil
		})
	if result.failed || result.skipped != 0 {
		t.Fatalf("runTestPrograms result = %+v", result)
	}
	if stdout.Len() != 0 {
		t.Fatalf("JSON success output includes a plain-text package result: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("JSON success stderr = %q", stderr.String())
	}
}
