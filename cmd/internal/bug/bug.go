/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package bug prepares reports for the LLGo issue tracker.
package bug

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"runtime"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/cmd/internal/flags"
	"github.com/xgo-dev/llgo/cmd/internal/gocommand"
	"github.com/xgo-dev/llgo/cmd/compiler/env"
	"github.com/xgo-dev/llvm"
)

var Cmd = &base.Command{
	UsageLine: "llgo bug [-n]",
	Short:     "Start an LLGo bug report (-n prints the report)",
}

func init() { Cmd.Run = runCmd }

// report deliberately collects an allowlist of version and platform fields.
// Paths, environment variables and module dependencies are never collected.
func report() string {
	return fmt.Sprintf("### What did you do?\n\n\n### What did you expect to see?\n\n\n### What did you see instead?\n\n\n### Toolchain\n\n```text\nllgo %s\nLLVM %s\nbuilt with %s\n%s/%s\n```\n", env.Version(), llvm.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func runCmd(_ *base.Command, args []string) {
	gocommand.Exit("bug", run(args, os.Stdout, os.Stderr, openBrowser))
}

func run(args []string, stdout, stderr io.Writer, open func(string) error) error {
	fs := flag.NewFlagSet("llgo bug", flag.ContinueOnError)
	fs.SetOutput(stderr)
	flags.AddDirectoryFlag(fs)
	remaining, err := flags.ChangeDirectory(args)
	if err != nil {
		return err
	}
	printOnly := fs.Bool("n", false, "Print the report without opening a browser")
	if err := fs.Parse(remaining); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	body := report()
	if *printOnly {
		_, err := io.WriteString(stdout, body)
		return err
	}
	link := "https://github.com/xgo-dev/llgo/issues/new?" + url.Values{"body": {body}}.Encode()
	if err := open(link); err != nil {
		fmt.Fprint(stdout, body)
		return fmt.Errorf("open browser: %w (report printed above)", err)
	}
	return nil
}

func openBrowser(link string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", link)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", link)
	default:
		cmd = exec.Command("xdg-open", link)
	}
	return cmd.Run()
}
