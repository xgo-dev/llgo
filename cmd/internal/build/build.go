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

// Package build implements the "llgo build" command.
package build

import (
	"flag"
	"fmt"
	"os"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/cmd/internal/flags"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/mockable"
	"github.com/xgo-dev/llgo/xtool/env/llvm"
)

// llgo build
var Cmd = &base.Command{
	UsageLine: "llgo build [-o output] [-target platform] [build flags] [packages]",
	Short:     "Compile packages and dependencies",
}

var goBuildFlags *base.PassArgs

func init() {
	Cmd.Run = runCmd
	goBuildFlags = flags.CaptureGoBuildFlags(Cmd)

	flags.AddCommonFlags(&Cmd.Flag)
	flags.AddCompilerVerboseFlag(&Cmd.Flag)
	flags.AddBuildFlags(&Cmd.Flag)
	flags.AddCoverageFlags(&Cmd.Flag)
	flags.AddBuildTraceFlag(&Cmd.Flag)
	flags.AddBuildModeFlags(&Cmd.Flag)
	flags.AddTargetFlag(&Cmd.Flag)
	flags.AddSizeFlags(&Cmd.Flag)
	flags.AddOutputFlags(&Cmd.Flag)
	flags.FinalizeNativeFlags("build", &Cmd.Flag)
}

func runCmd(cmd *base.Command, args []string) {

	if err := flags.Parse(&cmd.Flag, args); err != nil {
		if err != flag.ErrHelp {
			mockable.Exit(1)
		}
		return
	}

	conf := build.NewDefaultConf(build.ModeBuild)
	if err := flags.UpdateBuildConfig(conf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
	conf.BuildTrace = flags.BuildTrace
	if flags.Cover || flags.CoverMode != "" || flags.CoverPkg != "" {
		conf.Coverage = &build.CoverageConfig{
			Mode:     flags.CoverMode,
			Packages: flags.CoverPkg,
		}
	}
	if err := flags.ApplyGoBuildFlags(conf, goBuildFlags.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}

	llvm.SetupPath()
	args = cmd.Flag.Args()

	_, err := build.Do(args, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
}
