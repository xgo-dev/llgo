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
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/xgo-dev/llgo/internal/mockable"
	"github.com/xgo-dev/llgo/internal/shellparse"
)

type testProgram struct {
	app              string
	pkgDir           string
	pkgName          string
	temporaryOutputs *OutFmtDetails
	runner           string
	runnerEnv        map[string]string
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

func runNativeTest(commands commandEnv, program testProgram, conf *Config, stdout, stderr io.Writer) error {
	defer removeOutFmts(program.temporaryOutputs)
	if program.runner != "" {
		// Like native go test, execute each package in its source directory.
		// WASI runners forward PWD explicitly into the guest environment.
		if program.pkgDir != "" {
			commands.dir = program.pkgDir
			commands.environ = withEnv(commands.environ, "PWD="+program.pkgDir)
		}
		return runEmuCmdTo(commands, program.runnerEnv, program.runner, conf.RunArgs, false, conf.PrintCommands, stdout, stderr)
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
		fmt.Fprintf(stderr, "%s: exit code %d\n", program.app, exitErr.ExitCode())
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
		cmd := exec.Command(app, args...)
		ctx.commands.configure(cmd)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return err
		}
		if s := cmd.ProcessState; s != nil {
			mockable.Exit(s.ExitCode())
		}
	case ModeCmpTest:
		cmpTest(ctx.commands, pkgDir, pkgName, app, conf.GenExpect, conf.RunArgs)
	}
	return nil
}

func runInEmulator(commands commandEnv, emulator string, envMap map[string]string, pkgDir, pkgName string, conf *Config, mode Mode, verbose bool) error {
	// Skip execution if CompileOnly is true
	if conf.CompileOnly {
		return nil
	}

	if emulator == "" {
		return fmt.Errorf("target %s does not have emulator configured", conf.Target)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Using emulator: %s\n", emulator)
	}

	switch mode {
	case ModeRun:
		return runEmuCmd(commands, envMap, emulator, conf.RunArgs, verbose, conf.PrintCommands)
	case ModeTest:
		return runEmuCmd(commands, envMap, emulator, conf.RunArgs, verbose, conf.PrintCommands)
	case ModeCmpTest:
		cmpTest(commands, pkgDir, pkgName, envMap["out"], conf.GenExpect, conf.RunArgs)
		return nil
	}
	return nil
}

// runEmuCmd runs the application in emulator by formatting the emulator command template
func runEmuCmd(commands commandEnv, envMap map[string]string, emulatorTemplate string, runArgs []string, verbose bool, printCmds bool) error {
	return runEmuCmdTo(commands, envMap, emulatorTemplate, runArgs, verbose, printCmds, os.Stdout, os.Stderr)
}

func runEmuCmdTo(commands commandEnv, envMap map[string]string, emulatorTemplate string, runArgs []string, verbose bool, printCmds bool, stdout, stderr io.Writer) error {
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
		return fmt.Errorf("failed to parse emulator command: %w", err)
	}
	if len(cmdParts) == 0 {
		return fmt.Errorf("empty emulator command")
	}

	// Add run arguments to the end
	cmdParts = append(cmdParts, runArgs...)
	if printCmds {
		fmt.Fprintf(stderr, "%s %s\n", cmdParts[0], strings.Join(cmdParts[1:], " "))
	}

	// Execute the emulator command
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	commands.configure(cmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	// A nil Run error is already exit status zero. Returning normally keeps the
	// caller's cleanup and trace defers reachable.
	return nil
}
