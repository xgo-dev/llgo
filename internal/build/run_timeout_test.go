//go:build !llgo

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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunnerTimeoutKillsDescendants(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"native", "native-test", "wasm-runtime", "emulator"} {
		t.Run(path, func(t *testing.T) {
			dir := t.TempDir()
			commands := timeoutHelperCommands(dir, "tree")
			t.Cleanup(func() { killTimeoutHelper(dir) })
			// The unfixed native-test path has no host watchdog. Bound the
			// fixture itself so a regression fails without hanging this suite.
			fallback := time.AfterFunc(5*time.Second, func() { killTimeoutHelper(dir) })
			t.Cleanup(func() { fallback.Stop() })
			conf := &Config{RunnerTimeout: 2 * time.Second, RunArgs: []string{"-test.run=^TestRunnerTimeoutHelper$"}}
			started := time.Now()
			var err error
			switch path {
			case "native":
				err = runNative(&context{commands: commands}, executable, "", "probe", conf, ModeRun)
			case "native-test":
				err = runNativeTest(commands, testProgram{app: executable, pkgDir: dir, pkgName: "example/native"},
					conf, io.Discard, io.Discard)
			case "wasm-runtime":
				t.Setenv(llgoWasmRuntime, executable+" -test.run=^TestRunnerTimeoutHelper$")
				conf.Goos = "wasip1"
				conf.RunArgs = nil
				err = runNative(&context{commands: commands}, "probe.wasm", "", "probe", conf, ModeRun)
			case "emulator":
				err = runEmuCmdTo(commands, nil, fmt.Sprintf("%q", executable), conf.RunArgs, false, false,
					runnerDetails{phase: "run", timeout: conf.RunnerTimeout}, io.Discard, io.Discard)
			}
			if elapsed := time.Since(started); elapsed > 8*time.Second {
				t.Fatalf("runner took %s to stop", elapsed)
			}
			var failure *runnerFailure
			if !errors.As(err, &failure) || failure.status != runnerStatusTimeout || !errors.Is(err, stdcontext.DeadlineExceeded) {
				t.Fatalf("error = %v, want classified deadline", err)
			}
			assertTimeoutHelperStopped(t, dir)
		})
	}
}

func TestNativeTestHostTimeout(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	program := testProgram{app: executable, pkgDir: dir, pkgName: "example/hanging-test"}
	conf := &Config{
		RunArgs:       []string{"-test.run=^TestRunnerTimeoutHelper$"},
		RunnerTimeout: 2 * time.Second,
	}
	started := time.Now()
	err = runNativeTest(timeoutHelperCommands(dir, "native-hang"), program, conf, &stdout, &stderr)
	var failure *runnerFailure
	if !errors.As(err, &failure) || failure.status != runnerStatusTimeout ||
		failure.phase != "test" || failure.packageName != program.pkgName ||
		!errors.Is(err, stdcontext.DeadlineExceeded) {
		t.Fatalf("native test error = %v, want package-qualified host deadline", err)
	}
	if elapsed := time.Since(started); elapsed > 6*time.Second {
		t.Fatalf("native test took %s to stop", elapsed)
	}
	if !strings.Contains(stdout.String(), "=== RUN   TestHostDeadlineProbe") {
		t.Fatalf("lost output before timeout: %q", stdout.String())
	}
}

func TestNativeTestHostTimeoutBoundsOutputWait(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Cleanup(func() { killTimeoutHelper(dir) })
	fallback := time.AfterFunc(5*time.Second, func() { killTimeoutHelper(dir) })
	t.Cleanup(func() { fallback.Stop() })
	started := time.Now()
	err = runNativeTest(timeoutHelperCommands(dir, "exit-with-child"),
		testProgram{app: executable, pkgDir: dir, pkgName: "example/inherited-output"},
		&Config{RunnerTimeout: 10 * time.Second, RunArgs: []string{"-test.run=^TestRunnerTimeoutHelper$"}},
		io.Discard, io.Discard)
	var failure *runnerFailure
	if !errors.As(err, &failure) || failure.status != runnerStatusOutputTimeout || !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("native output wait = %v, want bounded inherited-pipe failure", err)
	}
	if elapsed := time.Since(started); elapsed > 4*time.Second {
		t.Fatalf("native output wait took %s to stop", elapsed)
	}
	// Windows taskkill cannot recover descendants after their parent exits.
	// The fixture cleanup owns that case; this assertion covers bounded host
	// pipe waiting, while the live-parent tree test checks process cleanup.
}

func TestNativeTestHostTimeoutControls(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// Native tests historically receive EOF on stdin, unlike llgo run and
	// external runners. A shared command implementation must preserve that.
	input, err := os.CreateTemp(t.TempDir(), "parent-stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	if _, err := input.WriteString("parent-only input"); err != nil {
		t.Fatal(err)
	}
	if _, err := input.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	originalStdin := os.Stdin
	os.Stdin = input
	t.Cleanup(func() { os.Stdin = originalStdin })
	for _, timeout := range []time.Duration{0, 2 * time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			dir := t.TempDir()
			commands := timeoutHelperCommands(dir, "native-control")
			commands.dir = t.TempDir()
			program := testProgram{app: executable, pkgDir: dir, pkgName: "example/control"}
			conf := &Config{RunnerTimeout: timeout, RunArgs: []string{"-test.run=^TestRunnerTimeoutHelper$"}}
			var stdout, stderr bytes.Buffer
			if err := runNativeTest(commands, program, conf, &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			var result struct{ Dir, Env, Input string }
			if err := json.NewDecoder(&stdout).Decode(&result); err != nil {
				t.Fatal(err)
			}
			actualDir, err := filepath.EvalSymlinks(result.Dir)
			if err != nil {
				t.Fatal(err)
			}
			wantDir, err := filepath.EvalSymlinks(dir)
			if err != nil {
				t.Fatal(err)
			}
			if actualDir != wantDir || result.Env != dir || result.Input != "" {
				t.Fatalf("native invocation = %+v, want directory/environment %q and EOF stdin", result, dir)
			}
			if stderr.String() != "native stderr" {
				t.Fatalf("native stderr = %q", stderr.String())
			}
		})
	}
}

func TestNativeTestHostTimeoutPreservesExit(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	err = runNativeTest(timeoutHelperCommands(dir, "native-exit"),
		testProgram{app: executable, pkgDir: dir, pkgName: "example/exit"},
		&Config{RunnerTimeout: 2 * time.Second, RunArgs: []string{"-test.run=^TestRunnerTimeoutHelper$"}}, &stdout, &stderr)
	var exitErr *exec.ExitError
	var failure *runnerFailure
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 23 || errors.As(err, &failure) {
		t.Fatalf("native exit = %v, want original exit code 23", err)
	}
	if stdout.String() != "native stdout" || !strings.Contains(stderr.String(), "exit code 23") {
		t.Fatalf("lost exit output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestNativeRunWithoutTimeout(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	commands := timeoutHelperCommands(t.TempDir(), "success")
	conf := &Config{RunArgs: []string{"-test.run=^TestRunnerTimeoutHelper$"}}
	if err := runNative(&context{commands: commands}, executable, "", "probe", conf, ModeRun); err != nil {
		t.Fatal(err)
	}
}

func timeoutHelperCommands(dir, mode string) commandEnv {
	return commandEnv{dir: dir, environ: withEnv(os.Environ(), "LLGO_TIMEOUT_HELPER="+mode, "LLGO_TIMEOUT_DIR="+dir)}
}

func killTimeoutHelper(dir string) {
	raw, _ := os.ReadFile(filepath.Join(dir, "child.pid"))
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err == nil {
		if process, err := os.FindProcess(pid); err == nil {
			_ = process.Kill()
		}
	}
}

func assertTimeoutHelperStopped(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "heartbeat")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("descendant did not start: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	after, err := os.Stat(path)
	if err != nil || after.Size() != before.Size() {
		t.Fatalf("descendant kept writing after timeout: %v", err)
	}
}

func TestRunnerTimeoutHelper(t *testing.T) {
	mode := os.Getenv("LLGO_TIMEOUT_HELPER")
	if mode == "" || mode == "success" {
		return
	}
	dir := os.Getenv("LLGO_TIMEOUT_DIR")
	if mode == "native-hang" {
		fmt.Fprintln(os.Stdout, "=== RUN   TestHostDeadlineProbe")
		time.Sleep(4 * time.Second)
		return
	}
	if mode == "native-control" {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(struct{ Dir, Env, Input string }{cwd, dir, string(input)}); err != nil {
			t.Fatal(err)
		}
		fmt.Fprint(os.Stderr, "native stderr")
		return
	}
	if mode == "native-exit" {
		fmt.Fprint(os.Stdout, "native stdout")
		os.Exit(23)
	}
	if mode == "terminal-input" {
		var word string
		if _, err := fmt.Fscanln(os.Stdin, &word); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintln(os.Stdout, "terminal read:", word)
		return
	}
	if mode == "terminal-runner" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		err = runRunnerCommand(timeoutHelperCommands(dir, "terminal-input"), executable,
			[]string{"-test.run=^TestRunnerTimeoutHelper$"}, runnerDetails{phase: "run", timeout: 5 * time.Second}, os.Stdout, os.Stderr)
		if err != nil {
			t.Fatal(err)
		}
		var word string
		if _, err := fmt.Fscanln(os.Stdin, &word); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintln(os.Stdout, "terminal restored:", word)
		return
	}
	if mode == "terminal-test-runner" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintln(os.Stdout, "terminal parent ready")
		err = runRunnerCommand(timeoutHelperCommands(dir, "terminal-hold"), executable,
			[]string{"-test.run=^TestRunnerTimeoutHelper$"}, runnerDetails{phase: "test", timeout: 5 * time.Second}, os.Stdout, os.Stderr)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintln(os.Stdout, "terminal test runner done")
		return
	}
	if mode == "terminal-hold" {
		fmt.Fprintln(os.Stdout, "terminal test child started")
		time.Sleep(500 * time.Millisecond)
		return
	}
	if mode == "child" {
		if err := os.WriteFile(filepath.Join(dir, "child.pid"), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(filepath.Join(dir, "heartbeat"))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		for {
			if _, err := file.WriteString("."); err != nil {
				t.Fatal(err)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestRunnerTimeoutHelper$")
	cmd.Env = withEnv(os.Environ(), "LLGO_TIMEOUT_HELPER=child")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if mode == "exit-with-child" {
		return
	}
	if mode == "fail-with-child" {
		t.Fatal("runner failed while its child is still running")
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}
