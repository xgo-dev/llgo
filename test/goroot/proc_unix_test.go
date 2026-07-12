//go:build unix

package goroot

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
)

func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func resourceMonitoringSupported() bool { return true }

func processGroupRSS(processGroupID int) (uint64, error) {
	output, err := exec.Command("ps", "-axo", "pgid=,rss=").Output()
	if err != nil {
		return 0, err
	}
	var totalKiB uint64
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		fields := bytes.Fields(scanner.Bytes())
		if len(fields) != 2 {
			continue
		}
		pgid, err := strconv.Atoi(string(fields[0]))
		if err != nil || pgid != processGroupID {
			continue
		}
		rssKiB, err := strconv.ParseUint(string(fields[1]), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse RSS from ps output %q: %w", scanner.Text(), err)
		}
		totalKiB += rssKiB
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return totalKiB << 10, nil
}
