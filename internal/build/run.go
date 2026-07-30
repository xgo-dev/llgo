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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/goplus/llgo/internal/mockable"
	"github.com/goplus/llgo/internal/processenv"
	"github.com/goplus/llgo/internal/shellparse"
)

func runNative(ctx *context, app, pkgDir, pkgName string, conf *Config, mode Mode) error {
	// Skip execution if CompileOnly is true
	if conf.CompileOnly {
		return nil
	}

	switch mode {
	case ModeRun:
		args := make([]string, 0, len(conf.RunArgs)+1)
		if isWasmTarget(conf.Goos) {
			wasmer := os.Expand(WasmRuntimeForConfig(conf), func(key string) string {
				return envConfigValue(conf, key)
			})
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
		cmd := ctx.process.Command(app, args...)
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
	case ModeTest:
		if conf.PrintCommands {
			fmt.Fprintf(os.Stderr, "%s %s\n", app, strings.Join(conf.RunArgs, " "))
		}
		cmd := ctx.process.Command(app, conf.RunArgs...)
		cmd.Dir = pkgDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				fmt.Fprintf(os.Stderr, "%s: exit code %d\n", app, exitErr.ExitCode())
				if !ctx.testFail {
					ctx.testFail = true
				}
			} else {
				fmt.Fprintf(os.Stderr, "failed to run test %s: %v\n", app, err)
				if !ctx.testFail {
					ctx.testFail = true
				}
			}
		}
	case ModeCmpTest:
		cmpTest(ctx.process, pkgDir, pkgName, app, conf.GenExpect, conf.RunArgs)
	}
	return nil
}

func runInEmulator(ctx *context, emulator string, envMap map[string]string, pkgDir, pkgName string, conf *Config, mode Mode, verbose bool) error {
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
		return runEmuCmd(ctx.process, envMap, emulator, conf.RunArgs, verbose, conf.PrintCommands)
	case ModeTest:
		return runEmuCmd(ctx.process, envMap, emulator, conf.RunArgs, verbose, conf.PrintCommands)
	case ModeCmpTest:
		cmpTest(ctx.process, pkgDir, pkgName, envMap["out"], conf.GenExpect, conf.RunArgs)
		return nil
	}
	return nil
}

// runEmuCmd runs the application in emulator by formatting the emulator command template
func runEmuCmd(process processenv.Context, envMap map[string]string, emulatorTemplate string, runArgs []string, verbose bool, printCmds bool) error {
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
		fmt.Fprintf(os.Stderr, "Running in emulator: %s\n", emulatorCmd)
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
		fmt.Fprintf(os.Stderr, "%s %s\n", cmdParts[0], strings.Join(cmdParts[1:], " "))
	}

	// Execute the emulator command
	cmd := process.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	if s := cmd.ProcessState; s != nil {
		mockable.Exit(s.ExitCode())
	}
	return nil
}
