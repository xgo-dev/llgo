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
	"fmt"
	"os"

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
	UsageLine: "llgo run [-target platform] [build flags] package [arguments...]",
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
)

func init() {
	Cmd.Run = runCmd
	CmpTestCmd.Run = runCmpTest
	runGoBuildFlags = flags.CaptureGoBuildFlags(Cmd)
	flags.AddCommonFlags(&Cmd.Flag)
	flags.AddBuildFlags(&Cmd.Flag)
	flags.AddEmulatorFlags(&Cmd.Flag)
	flags.AddEmbeddedFlags(&Cmd.Flag) // for -target support

	cmpTestGoBuildFlags = flags.CaptureGoBuildFlags(CmpTestCmd)
	flags.AddCommonFlags(&CmpTestCmd.Flag)
	flags.AddBuildFlags(&CmpTestCmd.Flag)
	flags.AddEmulatorFlags(&CmpTestCmd.Flag)
	flags.AddEmbeddedFlags(&CmpTestCmd.Flag) // for -target support
	flags.AddCmpTestFlags(&CmpTestCmd.Flag)
}

func runCmd(cmd *base.Command, args []string) {
	runCmdEx(cmd, args, build.ModeRun, runGoBuildFlags) // support target
}

func runCmpTest(cmd *base.Command, args []string) {
	runCmdEx(cmd, args, build.ModeCmpTest, cmpTestGoBuildFlags) // no target support
}

func runCmdEx(cmd *base.Command, args []string, mode build.Mode, goBuildFlags *base.PassArgs) {

	if err := cmd.Flag.Parse(args); err != nil {
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
	check(err)
	conf.RunArgs = runArgs
	_, err = build.Do(args, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
}

func parseRunArgs(args []string) ([]string, []string, error) {
	if len(args) == 0 {
		return nil, nil, errNoProj
	}

	return args[:1], args[1:], nil
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
