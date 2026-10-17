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

// Package run implements the "llgo run" command.
package run

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/cmd/internal/flags"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/mockable"
)

var (
	errNoProj = errors.New("llgo: no go files listed")
)

// llgo run
var Cmd = &base.Command{
	UsageLine: "llgo run [-target platform] [build flags] package|files [arguments...]",
	Short:     "Compile and run Go program",
}

// llgo cmptest
var CmpTestCmd = &base.Command{
	UsageLine: "llgo cmptest [-gen] [build flags] package [arguments...]",
	Short:     "Compile and run with llgo, compare result (stdout/stderr/exitcode) with go or llgo.expect; generate llgo.expect file if -gen is specified",
}

var (
	runGoBuildFlags     *base.PassArgs
	cmpTestGoBuildFlags *base.PassArgs
	runTimeout          time.Duration
	runExec             *[]string
)

func init() {
	Cmd.Run = runCmd
	CmpTestCmd.Run = runCmpTest
	runGoBuildFlags = flags.CaptureGoBuildFlags(Cmd)
	flags.AddCommonFlags(&Cmd.Flag)
	flags.AddCompilerVerboseFlag(&Cmd.Flag)
	flags.AddBuildFlags(&Cmd.Flag)
	flags.AddEmulatorFlags(&Cmd.Flag)
	flags.AddEmbeddedFlags(&Cmd.Flag) // for -target support
	Cmd.Flag.DurationVar(&runTimeout, "timeout", 0, "Timeout for the executed program (0 disables)")
	runExec = flags.AddExecFlag(&Cmd.Flag)

	cmpTestGoBuildFlags = flags.CaptureGoBuildFlags(CmpTestCmd)
	flags.AddCommonFlags(&CmpTestCmd.Flag)
	flags.AddBuildFlags(&CmpTestCmd.Flag)
	flags.AddEmulatorFlags(&CmpTestCmd.Flag)
	flags.AddEmbeddedFlags(&CmpTestCmd.Flag) // for -target support
	flags.AddCmpTestFlags(&CmpTestCmd.Flag)
	flags.FinalizeNativeFlags("run", &Cmd.Flag)
	flags.FinalizeNativeFlags("run", &CmpTestCmd.Flag)
}

func runCmd(cmd *base.Command, args []string) {
	runCmdEx(cmd, args, build.ModeRun, runGoBuildFlags) // support target
}

func runCmpTest(cmd *base.Command, args []string) {
	runCmdEx(cmd, args, build.ModeCmpTest, cmpTestGoBuildFlags) // no target support
}

func runCmdEx(cmd *base.Command, args []string, mode build.Mode, goBuildFlags *base.PassArgs) {

	if err := flags.Parse(&cmd.Flag, args); err != nil {
		if err != flag.ErrHelp {
			mockable.Exit(1)
		}
		return
	}

	conf := build.NewDefaultConf(mode)
	if err := flags.UpdateConfig(conf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
	if err := flags.ApplyGoBuildFlags(conf, goBuildFlags.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}

	args = cmd.Flag.Args()
	args, runArgs, err := parseRunArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
		return
	}
	conf.RunArgs = runArgs
	if mode == build.ModeRun {
		conf.RunnerTimeout = runTimeout
		conf.Exec = *runExec
	}
	_, err = build.Do(args, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if code, ok := build.RunnerExitCode(err); ok {
			mockable.Exit(code)
			return
		}
		mockable.Exit(1)
	}
}

func parseRunArgs(args []string) ([]string, []string, error) {
	if len(args) == 0 {
		return nil, nil, errNoProj
	}

	if strings.HasSuffix(args[0], ".go") {
		n := 0
		for n < len(args) && strings.HasSuffix(args[n], ".go") {
			if strings.HasSuffix(args[n], "_test.go") {
				return nil, nil, fmt.Errorf("llgo: cannot run *_test.go files (%s)", args[n])
			}
			n++
		}
		return args[:n], args[n:], nil
	}
	if strings.Contains(args[0], "@") {
		return nil, nil, fmt.Errorf("llgo: versioned run operands are unsupported; use a module with pinned dependencies: %s", args[0])
	}
	return args[:1], args[1:], nil
}
