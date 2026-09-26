//go:build unix

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

func configureRunnerCancellation(cmd *exec.Cmd) func() {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	if input, ok := cmd.Stdin.(*os.File); ok {
		fd := int(input.Fd())
		foreground, err := unix.IoctlGetInt(fd, unix.TIOCGPGRP)
		if err == nil && foreground == syscall.Getpgrp() {
			// A separate background group cannot read the caller's terminal.
			cmd.SysProcAttr.Foreground = true
			cmd.SysProcAttr.Ctty = fd
			return func() {
				ignored := signal.Ignored(syscall.SIGTTOU)
				signal.Ignore(syscall.SIGTTOU)
				if !ignored {
					defer signal.Reset(syscall.SIGTTOU)
				}
				_ = unix.IoctlSetPointerInt(fd, unix.TIOCSPGRP, foreground)
			}
		}
	}
	return func() {}
}
