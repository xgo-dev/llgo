//go:build !llgo && !windows

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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRunInEmulatorValidation(t *testing.T) {
	commands := commandEnv{dir: t.TempDir(), environ: os.Environ()}
	if err := runInEmulator(commands, "", nil, "", "", &Config{CompileOnly: true}, ModeRun, false); err != nil {
		t.Fatalf("compile-only emulator run failed: %v", err)
	}
	if err := runInEmulator(commands, "", nil, "", "", &Config{Target: "demo"}, ModeRun, false); err == nil {
		t.Fatal("missing emulator succeeded")
	}
	if err := runEmuCmd(commands, nil, "'", nil, false, false); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("malformed emulator command error = %v", err)
	}
	if err := runEmuCmd(commands, nil, "   ", nil, false, false); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty emulator command error = %v", err)
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

func TestRunNativeTestHelper(t *testing.T) {
	args := os.Args
	for i, arg := range args {
		if arg != "--" || i+1 == len(args) {
			continue
		}
		switch args[i+1] {
		case "success":
			fmt.Fprint(os.Stdout, "stdout")
			fmt.Fprint(os.Stdout, os.Getenv("LLGO_RUN_NATIVE_TEST_ENV"))
			fmt.Fprint(os.Stderr, "stderr")
		case "exit":
			os.Exit(3)
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
