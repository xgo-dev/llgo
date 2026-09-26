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
	stdcontext "context"
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
	for _, path := range []string{"native", "wasm-runtime", "emulator"} {
		t.Run(path, func(t *testing.T) {
			dir := t.TempDir()
			commands := timeoutHelperCommands(dir, "tree")
			t.Cleanup(func() { killTimeoutHelper(dir) })
			conf := &Config{RunnerTimeout: 2 * time.Second, RunArgs: []string{"-test.run=^TestRunnerTimeoutHelper$"}}
			started := time.Now()
			var err error
			switch path {
			case "native":
				err = runNative(&context{commands: commands}, executable, "", "probe", conf, ModeRun)
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
			[]string{"-test.run=^TestRunnerTimeoutHelper$"}, runnerDetails{timeout: 5 * time.Second}, os.Stdout, os.Stderr)
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
