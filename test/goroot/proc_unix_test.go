//go:build unix

package goroot

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"
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

func TestRunProgramWaitDelayCleansDescendant(t *testing.T) {
	guardTestTimeout(t)
	disableSystemMemoryLimits(t)

	oldWaitDelay := runProgramWaitDelay
	runProgramWaitDelay = 100 * time.Millisecond
	t.Cleanup(func() { runProgramWaitDelay = oldWaitDelay })

	for _, exitCode := range []int{0, 7} {
		t.Run(fmt.Sprintf("exit-%d", exitCode), func(t *testing.T) {
			pidFile := filepath.Join(t.TempDir(), "descendant.pid")
			script := `sleep 60 & echo $! > "$1"; exit "$2"`
			_, _, gotExitCode, elapsed, err := runProgram(
				t.TempDir(),
				"/bin/sh",
				os.Environ(),
				5*time.Second,
				"-c", script, "sh", pidFile, strconv.Itoa(exitCode),
			)
			if exitCode == 0 {
				if !errors.Is(err, exec.ErrWaitDelay) {
					t.Fatalf("runProgram error = %v, want exec.ErrWaitDelay", err)
				}
			} else {
				if err != nil {
					t.Fatalf("runProgram error = %v, want nil ExitError wrapper", err)
				}
				if gotExitCode != exitCode {
					t.Fatalf("exit code = %d, want %d", gotExitCode, exitCode)
				}
			}
			if elapsed >= 5*time.Second {
				t.Fatalf("runProgram took %s, want bounded WaitDelay return", elapsed)
			}

			pidBytes, readErr := os.ReadFile(pidFile)
			if readErr != nil {
				t.Fatal(readErr)
			}
			pid, parseErr := strconv.Atoi(string(bytes.TrimSpace(pidBytes)))
			if parseErr != nil {
				t.Fatalf("parse descendant PID %q: %v", pidBytes, parseErr)
			}
			t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

			deadline := time.Now().Add(2 * time.Second)
			for {
				exited, probeErr := processHasExited(pid)
				if probeErr != nil {
					t.Fatalf("probe descendant %d: %v", pid, probeErr)
				}
				if exited {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("descendant %d is still running after runProgram returned", pid)
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}

func processHasExited(pid int) (bool, error) {
	err := syscall.Kill(pid, 0)
	if errors.Is(err, syscall.ESRCH) {
		return true, nil
	}
	if err != nil || runtime.GOOS != "linux" {
		return false, err
	}
	// A container's PID 1 need not reap orphaned descendants. kill(pid, 0)
	// still succeeds for a zombie, although SIGKILL has already stopped it.
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if os.IsNotExist(err) || errors.Is(err, syscall.ESRCH) {
		return true, nil // Reaped between the signal probe and the proc read.
	}
	if err != nil {
		return false, err
	}
	return procStatHasExited(stat)
}

func procStatHasExited(stat []byte) (bool, error) {
	// comm may contain spaces and parentheses; state follows the last ')'.
	end := bytes.LastIndexByte(stat, ')')
	if end < 0 || end+3 >= len(stat) || stat[end+1] != ' ' || stat[end+3] != ' ' {
		return false, fmt.Errorf("invalid process stat %q", stat)
	}
	return stat[end+2] == 'Z' || stat[end+2] == 'X', nil
}

func TestProcStatHasExited(t *testing.T) {
	for _, tt := range []struct {
		stat    string
		exited  bool
		invalid bool
	}{
		{"42 (sleep) S 1 42 0", false, false},
		{"42 (sleep) R 1 42 0", false, false},
		{"42 (sleep) Z 1 42 0", true, false},
		{"42 (a name ) with parentheses) Z 1 42 0", true, false},
		{"42 (sleep) X 1 42 0", true, false},
		{"", false, true},
		{"42 (sleep)", false, true},
		{"42 (sleep) Z", false, true},
	} {
		t.Run(tt.stat, func(t *testing.T) {
			got, err := procStatHasExited([]byte(tt.stat))
			if got != tt.exited || (err != nil) != tt.invalid {
				t.Fatalf("process stat = %v, %v; want exited=%v, invalid=%v", got, err, tt.exited, tt.invalid)
			}
		})
	}
}

func TestProcessHasExited(t *testing.T) {
	guardTestTimeout(t)
	if exited, err := processHasExited(os.Getpid()); err != nil || exited {
		t.Fatalf("live process: exited=%v, err=%v", exited, err)
	}
	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Wait() })
	if runtime.GOOS == "linux" {
		// Leave our child unreaped so this test does not depend on PID 1's
		// behavior, even on a host with a fully functioning init process.
		deadline := time.Now().Add(2 * time.Second)
		for {
			exited, err := processHasExited(cmd.Process.Pid)
			if err != nil {
				t.Fatal(err)
			}
			if exited {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("terminated but unreaped child was reported as running")
			}
			time.Sleep(time.Millisecond)
		}
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if exited, err := processHasExited(cmd.Process.Pid); err != nil || !exited {
		t.Fatalf("reaped child: exited=%v, err=%v", exited, err)
	}
}
