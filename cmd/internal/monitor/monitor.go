/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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

package monitor

import (
	"fmt"
	"os"

	"github.com/goplus/llgo/cmd/internal/base"
	"github.com/goplus/llgo/cmd/internal/flags"
	"github.com/goplus/llgo/internal/crosscompile"
	"github.com/goplus/llgo/internal/monitor"
	"github.com/goplus/llgo/internal/optlevel"
)

// Cmd represents the monitor command.
var Cmd = &base.Command{
	UsageLine: "llgo monitor [flags] [executable]",
	Short:     "Monitor serial output from device",
}

func init() {
	flags.AddCommonFlags(&Cmd.Flag)
	flags.AddOptLevelFlags(&Cmd.Flag)
	flags.AddEmbeddedFlags(&Cmd.Flag)
	Cmd.Run = runMonitor
}

func runMonitor(cmd *base.Command, args []string) {
	cmd.Flag.Parse(args)
	args = cmd.Flag.Args()

	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "llgo monitor: too many arguments\n")
		os.Exit(1)
	}

	var executable string
	if len(args) == 1 {
		executable = args[0]
	}

	var serialPort []string
	if flags.Target != "" {
		level := flags.OptLevel
		if !level.IsValid() {
			level = optlevel.Oz
		}
		conf, err := crosscompile.UseTarget(flags.Target, level)
		if err != nil {
			fmt.Fprintf(os.Stderr, "llgo monitor: %v\n", err)
			os.Exit(1)
		}
		serialPort = conf.Device.SerialPort
	}

	config := monitor.MonitorConfig{
		Port:       flags.Port,
		Target:     flags.Target,
		BaudRate:   flags.BaudRate,
		Executable: executable,
		SerialPort: serialPort,
	}

	if err := monitor.Monitor(config, flags.Verbose); err != nil {
		fmt.Fprintf(os.Stderr, "llgo monitor: %v\n", err)
		os.Exit(1)
	}
}
