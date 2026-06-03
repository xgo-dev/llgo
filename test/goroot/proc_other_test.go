//go:build !unix

package goroot

import "os/exec"

func configureProcessGroup(cmd *exec.Cmd) {}

func killProcessTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
