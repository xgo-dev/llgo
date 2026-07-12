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
	"fmt"
	"os"

	"github.com/goplus/llgo/cmd/internal/base"
	"github.com/goplus/llgo/cmd/internal/flags"
	"github.com/goplus/llgo/internal/build"
	"github.com/goplus/llgo/internal/mockable"
)

// llgo build
var Cmd = &base.Command{
	UsageLine: "llgo build [-o output] [-target platform] [build flags] [packages]",
	Short:     "Compile packages and dependencies",
}

var goBuildFlags *base.PassArgs

func init() {
	Cmd.Run = runCmd
	goBuildFlags = base.PassBuildFlags(Cmd)

	flags.AddCommonFlags(&Cmd.Flag)
	flags.AddBuildFlags(&Cmd.Flag)
	flags.AddBuildModeFlags(&Cmd.Flag)
	flags.AddEmulatorFlags(&Cmd.Flag)
	flags.AddEmbeddedFlags(&Cmd.Flag)
	flags.AddOutputFlags(&Cmd.Flag)
}

func runCmd(cmd *base.Command, args []string) {

	if err := cmd.Flag.Parse(args); err != nil {
		return
	}

	conf := build.NewDefaultConf(build.ModeBuild)
	if err := flags.UpdateBuildConfig(conf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
	conf.GoBuildFlags = append(conf.GoBuildFlags, goBuildFlags.Args...)

	args = cmd.Flag.Args()

	_, err := build.Do(args, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
}
