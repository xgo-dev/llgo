/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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
	"runtime"
	"strings"
	"time"

	"github.com/xgo-dev/llgo/internal/shellparse"
)

type testProgram struct {
	coverage         bool
	app              string
	pkgDir           string
	pkgName          string
	temporaryOutputs *OutFmtDetails
	runner           string
	runnerEnv        map[string]string
	profile          string
}

type testRunResult struct {
	failed  bool
	skipped int
}

type testProgramResult struct {
	program testProgram
	output  []byte
	err     error
}

const (
	runnerStatusInvalidCommand = "invalid-command"
	runnerStatusNotConfigured  = "not-configured"
	runnerStatusUnavailable    = "unavailable"
	runnerStatusExit           = "exit"
	runnerStatusStart          = "start-error"
	runnerStatusTimeout        = "timeout"
	runnerStatusOutputTimeout  = "output-timeout"
)

// runnerDetails identifies the command boundary that owns a host process.
// Keep this independent from the target command template: aliases such as
// wasm and wasip1 can share a runner while retaining their selected target and
// physical ABI in diagnostics.
type runnerDetails struct {
	phase       string
	target      string
	profile     string
	artifact    string
	packageName string
	timeout     time.Duration
}

// runnerFailure preserves the host-runner outcome while adding enough build
// context to reproduce it. In particular, a missing runner and a program that
// deliberately exits non-zero must not collapse into the same bare exec error.
type runnerFailure struct {
	runnerDetails
	runner   string
	status   string
	exitCode int
	err      error
}

func (e *runnerFailure) Error() string {
	var message strings.Builder
	message.WriteString("runner failed")
	if e.phase != "" {
		fmt.Fprintf(&message, ": phase=%s", e.phase)
	}
	if e.target != "" {
		fmt.Fprintf(&message, " target=%s", e.target)
	}
	if e.profile != "" {
		fmt.Fprintf(&message, " profile=%s", e.profile)
	}
	if e.artifact != "" {
		fmt.Fprintf(&message, " artifact=%q", e.artifact)
	}
	if e.runner != "" {
		fmt.Fprintf(&message, " runner=%q", e.runner)
	}
	if e.packageName != "" {
		fmt.Fprintf(&message, " package=%q", e.packageName)
	}
	if e.status != "" {
		fmt.Fprintf(&message, " status=%s", e.status)
	}
	if e.status == runnerStatusTimeout && e.timeout > 0 {
		fmt.Fprintf(&message, " timeout=%s", e.timeout)
	}
	if e.exitCode >= 0 {
		fmt.Fprintf(&message, " exit_code=%d", e.exitCode)
	}
	if e.err != nil {
		fmt.Fprintf(&message, ": %v", e.err)
	}
	return message.String()
}

func (e *runnerFailure) Unwrap() error {
	return e.err
}

func newRunnerFailure(details runnerDetails, runner, status string, exitCode int, err error) error {
	return &runnerFailure{
		runnerDetails: details,
		runner:        runner,
		status:        status,
		exitCode:      exitCode,
		err:           err,
	}
}

func runnerPhase(mode Mode) string {
	switch mode {
	case ModeRun:
		return "run"
	case ModeTest:
		return "test"
	case ModeCmpTest:
		return "cmptest"
	default:
		return "execute"
	}
}

func runNativeTest(commands commandEnv, program testProgram, conf *Config, stdout, stderr io.Writer) error {
	if conf.coverage != nil {
		return runCoveredTest(commands, program, conf, stdout, stderr)
	}
	defer removeOutFmts(program.temporaryOutputs)
	if program.runner != "" {
		// Like native go test, execute each package in its source directory.
		// WASI runners forward PWD explicitly into the guest environment.
		if program.pkgDir != "" {
			commands.dir = program.pkgDir
			commands.environ = withEnv(commands.environ, "PWD="+program.pkgDir)
		}
		return runEmuCmdTo(commands, program.runnerEnv, program.runner, conf.RunArgs, false, conf.PrintCommands,
			runnerDetails{phase: "test", target: conf.Target, profile: program.profile, artifact: program.app, packageName: program.pkgName, timeout: conf.RunnerTimeout},
			stdout, stderr)
	}
	if conf.PrintCommands {
		fmt.Fprintf(stderr, "%s %s\n", program.app, strings.Join(conf.RunArgs, " "))
	}
	cmd := exec.Command(program.app, conf.RunArgs...)
	commands.configure(cmd)
	cmd.Dir = program.pkgDir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if !program.coverage {
			if exitErr.Exited() {
				fmt.Fprintf(stderr, "%s: exit code %d\n", program.app, exitErr.ExitCode())
			} else {
				// Signal termination has no exit code; retain the OS reason.
				fmt.Fprintf(stderr, "%s: %v\n", program.app, exitErr)
			}
		}
	} else {
		fmt.Fprintf(stderr, "failed to run test %s: %v\n", program.app, err)
	}
	return err
}

func runNativeTestPrograms(commands commandEnv, programs []testProgram, conf *Config, stdout, stderr io.Writer) testRunResult {
	defer func() {
		// Fail-fast can leave programs unstarted; reclaim their implicit outputs
		// as well as the ones runNativeTest already removed.
		for _, program := range programs {
			removeOutFmts(program.temporaryOutputs)
		}
	}()
	// "go test -c" links test binaries but never executes them. Keep this
	// check at the batch execution boundary so it also applies when several
	// test roots were linked from one shared build graph.
	if conf.CompileOnly {
		return testRunResult{}
	}
	parallelism := conf.BuildParallelism
	if conf.TestRunSequential {
		parallelism = 1
	}
	return runTestPrograms(programs, parallelism, conf.TestFailFast, conf.TestJSON, stdout, stderr,
		func(program testProgram, output io.Writer) error {
			return runNativeTest(commands, program, conf, output, output)
		})
}

func reportTestProgramResult(stdout, stderr io.Writer, result testProgramResult, json bool) {
	if len(result.output) != 0 {
		_, _ = stdout.Write(result.output)
		if result.output[len(result.output)-1] != '\n' {
			fmt.Fprintln(stdout)
		}
	}
	if result.program.coverage {
		return // Coverage reporting includes the Go-compatible package record.
	}
	if result.err != nil {
		fmt.Fprintf(stderr, "FAIL\t%s\n", result.program.pkgName)
	} else if !json {
		fmt.Fprintf(stdout, "ok  \t%s\n", result.program.pkgName)
	}
}

func runTestPrograms(
	programs []testProgram,
	parallelism int,
	failFast bool,
	json bool,
	stdout, stderr io.Writer,
	run func(testProgram, io.Writer) error,
) testRunResult {
	if len(programs) == 0 {
		return testRunResult{}
	}
	if parallelism == 0 {
		parallelism = runtime.GOMAXPROCS(0)
	}
	if parallelism < 1 {
		parallelism = 1
	}
	if parallelism > len(programs) {
		parallelism = len(programs)
	}

	results := make(chan testProgramResult, parallelism)
	start := func(program testProgram) {
		go func() {
			var output bytes.Buffer
			err := run(program, &output)
			results <- testProgramResult{program: program, output: output.Bytes(), err: err}
		}()
	}

	next := 0
	running := 0
	for next < len(programs) && running < parallelism {
		start(programs[next])
		next++
		running++
	}

	var result testRunResult
	for running != 0 {
		completed := <-results
		reportTestProgramResult(stdout, stderr, completed, json)
		if completed.err != nil {
			result.failed = true
		}
		running--
		if next < len(programs) && !(failFast && result.failed) {
			start(programs[next])
			next++
			running++
		}
	}
	result.skipped = len(programs) - next
	return result
}

func runNative(ctx *context, app, pkgDir, pkgName string, conf *Config, mode Mode) error {
	// Skip execution if CompileOnly is true
	if conf.CompileOnly {
		return nil
	}

	switch mode {
	case ModeRun:
		details := runnerDetails{phase: "run", target: conf.Target, artifact: app, packageName: pkgName, timeout: conf.RunnerTimeout}
		args := make([]string, 0, len(conf.RunArgs)+1)
		if isWasmTarget(conf.Goos) {
			wasmer := os.ExpandEnv(WasmRuntime())
			wasmerArgs := strings.Split(wasmer, " ")
			wasmerCmd := wasmerArgs[0]
			wasmerArgs = wasmerArgs[1:]
			switch wasmer {
			case "wasmtime":
				args = append(args, "--wasm", "multi-memory=true", app)
				args = append(args, conf.RunArgs...)
			case "iwasm":
				args = append(args, "--stack-size=819200000", "--heap-size=800000000", app)
				args = append(args, conf.RunArgs...)
			default:
				args = append(args, wasmerArgs...)
				args = append(args, app)
				args = append(args, conf.RunArgs...)
			}
			app = wasmerCmd
		} else {
			args = conf.RunArgs
		}
		if conf.PrintCommands {
			fmt.Fprintf(os.Stderr, "%s %s\n", app, strings.Join(args, " "))
		}
		return runRunnerCommand(ctx.commands, app, args, details, os.Stdout, os.Stderr)
	case ModeCmpTest:
		cmpTest(ctx.commands, pkgDir, pkgName, app, conf.GenExpect, conf.RunArgs)
	}
	return nil
}

func runInEmulator(commands commandEnv, emulator, profile string, envMap map[string]string, pkgDir, pkgName string, conf *Config, mode Mode, verbose bool) error {
	// Skip execution if CompileOnly is true
	if conf.CompileOnly {
		return nil
	}
	details := runnerDetails{
		phase:       runnerPhase(mode),
		target:      conf.Target,
		profile:     profile,
		artifact:    envMap["out"],
		packageName: pkgName,
		timeout:     conf.RunnerTimeout,
	}

	if emulator == "" {
		err := fmt.Errorf("target %s does not have an emulator configured", conf.Target)
		return newRunnerFailure(details, "", runnerStatusNotConfigured, -1, err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Using emulator: %s\n", emulator)
	}

	switch mode {
	case ModeRun, ModeTest:
		return runEmuCmd(commands, envMap, emulator, conf.RunArgs, verbose, conf.PrintCommands, details)
	case ModeCmpTest:
		cmpTest(commands, pkgDir, pkgName, envMap["out"], conf.GenExpect, conf.RunArgs)
		return nil
	}
	return nil
}

// runEmuCmd runs the application in emulator by formatting the emulator command template
func runEmuCmd(commands commandEnv, envMap map[string]string, emulatorTemplate string, runArgs []string, verbose bool, printCmds bool, details runnerDetails) error {
	return runEmuCmdTo(commands, envMap, emulatorTemplate, runArgs, verbose, printCmds, details, os.Stdout, os.Stderr)
}

func runEmuCmdTo(commands commandEnv, envMap map[string]string, emulatorTemplate string, runArgs []string, verbose bool, printCmds bool, details runnerDetails, stdout, stderr io.Writer) error {
	// Expand the emulator command template
	emulatorCmd := emulatorTemplate
	for placeholder, path := range envMap {
		var target string
		if placeholder == "" {
			target = "{}"
		} else {
			target = "{" + placeholder + "}"
		}
		emulatorCmd = strings.ReplaceAll(emulatorCmd, target, path)
	}

	if verbose {
		fmt.Fprintf(stderr, "Running in emulator: %s\n", emulatorCmd)
	}

	// Parse command and arguments safely handling quoted strings
	cmdParts, err := shellparse.Parse(emulatorCmd)
	if err != nil {
		return newRunnerFailure(details, "", runnerStatusInvalidCommand, -1,
			fmt.Errorf("failed to parse emulator command: %w", err))
	}
	if len(cmdParts) == 0 {
		return newRunnerFailure(details, "", runnerStatusInvalidCommand, -1,
			errors.New("empty emulator command"))
	}

	// Add run arguments to the end
	cmdParts = append(cmdParts, runArgs...)
	if printCmds {
		fmt.Fprintf(stderr, "%s %s\n", cmdParts[0], strings.Join(cmdParts[1:], " "))
	}

	return runRunnerCommand(commands, cmdParts[0], cmdParts[1:], details, stdout, stderr)
}

func runRunnerCommand(commands commandEnv, name string, args []string, details runnerDetails, stdout, stderr io.Writer) error {
	// The test binary owns its Go-level timeout;
	// this outer deadline also covers a host runner that stops forwarding exit
	// or otherwise hangs after the guest should have terminated.
	var runContext stdcontext.Context = stdcontext.Background()
	cancel := func() {}
	if details.timeout > 0 {
		runContext, cancel = stdcontext.WithTimeout(runContext, details.timeout)
	}
	defer cancel()
	cmd := exec.CommandContext(runContext, name, args...)
	commands.configure(cmd)
	cmd.Stdin = os.Stdin
	if details.timeout > 0 {
		restore := configureRunnerCancellation(cmd)
		defer restore()
		cmd.WaitDelay = time.Second
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil && details.timeout > 0 && cmd.Process != nil && cmd.Cancel != nil {
		// A failed runner can leave descendants holding output pipes. Wait
		// preserves a nonzero exit status over ErrWaitDelay, so clean up after
		// either result while keeping the original failure for diagnostics.
		_ = cmd.Cancel()
	}
	if err != nil {
		status := runnerStatusStart
		exitCode := -1
		var exitErr *exec.ExitError
		switch {
		case errors.Is(runContext.Err(), stdcontext.DeadlineExceeded):
			status = runnerStatusTimeout
			err = fmt.Errorf("runner exceeded %s: %w", details.timeout, stdcontext.DeadlineExceeded)
		case errors.Is(err, exec.ErrWaitDelay):
			status = runnerStatusOutputTimeout
		case errors.As(err, &exitErr):
			status = runnerStatusExit
			exitCode = exitErr.ExitCode()
		case errors.Is(err, exec.ErrNotFound), errors.Is(err, os.ErrNotExist):
			status = runnerStatusUnavailable
		}
		return newRunnerFailure(details, name, status, exitCode, err)
	}
	// Returning normally keeps cleanup of implicit modules and target sidecars reachable.
	return nil
}
