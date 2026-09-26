//go:build !llgo && unix

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
	stdcontext "context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunnerPreservesTerminalInput(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is needed to create a controlling terminal")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	const script = `
import errno, os, pty, select, sys, time
pid, fd = pty.fork()
if pid == 0:
    os.execv(sys.argv[1], [sys.argv[1], '-test.run=^TestRunnerTimeoutHelper$', *sys.argv[2:]])
output = b''
restored = False
try:
    os.write(fd, b'child-input\n')
    deadline = time.monotonic() + 10
    while time.monotonic() < deadline:
        if not select.select([fd], [], [], 0.1)[0]:
            continue
        try:
            chunk = os.read(fd, 4096)
        except OSError as err:
            if err.errno == errno.EIO:
                break
            raise
        if not chunk:
            break
        output += chunk
        if b'terminal read: child-input' in output and not restored:
            os.write(fd, b'parent-input\n')
            restored = True
        if b'terminal restored: parent-input' in output:
            break
    assert b'terminal restored: parent-input' in output, output
    # Let the test binary exit normally. Killing it immediately after the
    # marker loses its coverage data and can hide a late process error.
    deadline = time.monotonic() + 3
    while time.monotonic() < deadline:
        completed, status = os.waitpid(pid, os.WNOHANG)
        if completed == pid:
            pid = None
            assert os.waitstatus_to_exitcode(status) == 0, output
            break
        time.sleep(0.02)
    assert pid is None, output
finally:
    os.close(fd)
    if pid is not None:
        try:
            os.kill(pid, 9)
        except ProcessLookupError:
            pass
        os.waitpid(pid, 0)
`
	args := []string{"-c", script, executable}
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.gocoverdir=") {
			// The terminal runner is a child test binary; include its exercised
			// process-group path in go test's coverage data.
			args = append(args, arg)
		}
	}
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, args...)
	cmd.Env = timeoutHelperCommands(t.TempDir(), "terminal-runner").environ
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("terminal runner: %v\n%s", err, output)
	}
}

func TestRunnerBoundsInheritedOutputPipes(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"exit-with-child", "fail-with-child"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			t.Cleanup(func() { killTimeoutHelper(dir) })
			started := time.Now()
			err := runRunnerCommand(timeoutHelperCommands(dir, mode), executable,
				[]string{"-test.run=^TestRunnerTimeoutHelper$"}, runnerDetails{timeout: 30 * time.Second}, io.Discard, io.Discard)
			var failure *runnerFailure
			if !errors.As(err, &failure) {
				t.Fatalf("error = %v, want classified runner failure", err)
			}
			if mode == "exit-with-child" {
				if failure.status != runnerStatusOutputTimeout || !errors.Is(err, exec.ErrWaitDelay) {
					t.Fatalf("error = %v, want inherited output pipe timeout", err)
				}
			} else if failure.status != runnerStatusExit || failure.exitCode != 1 {
				t.Fatalf("error = %v, want preserved exit code 1", err)
			}
			if elapsed := time.Since(started); elapsed > 5*time.Second {
				t.Fatalf("output pipes blocked for %s", elapsed)
			}
			assertTimeoutHelperStopped(t, dir)
		})
	}
}
