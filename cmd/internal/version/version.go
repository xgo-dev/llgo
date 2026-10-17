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
 * See the License for the specific language governing permissions and limitations under the License.
 */

// Package version implements the "llgo version" command.
package version

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/cmd/internal/flags"
	"github.com/xgo-dev/llgo/cmd/internal/gocommand"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/mockable"
)

// llgo version
var Cmd = &base.Command{
	UsageLine: "llgo version [-m] [-v] [-json] [file ...]",
	Short:     "Print LLGo version or inspect Go executable build information",
}

func init() {
	Cmd.Run = runCmd
}

func runCmd(cmd *base.Command, args []string) {
	fs := flag.NewFlagSet("llgo version", flag.ContinueOnError)
	// Parse only to distinguish LLGo's identity query from file inspection.
	// Go owns inspection diagnostics, including invalid flags and help.
	fs.SetOutput(io.Discard)
	flags.AddDirectoryFlag(fs)
	modules := fs.Bool("m", false, "Print embedded Go module information")
	fs.Bool("v", false, "Report unrecognized files during Go executable inspection")
	json := fs.Bool("json", false, "Print Go executable build information as JSON")
	flags.FinalizeNativeFlags("version", fs)
	remaining, err := flags.ChangeDirectory(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
		return
	}
	if err := fs.Parse(remaining); err != nil {
		gocommand.Main("version", remaining)
		return
	}
	if fs.NArg() != 0 || *modules || *json {
		gocommand.Main("version", remaining)
		return
	}
	fmt.Printf("llgo %s %s/%s\n", env.Version(), runtime.GOOS, runtime.GOARCH)
}
