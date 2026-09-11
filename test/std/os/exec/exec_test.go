//go:build !wasm

package exec_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const execHelperEnv = "LLGO_TEST_EXEC_HELPER"

func helperCommand(mode string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=^TestExecHelperProcess$")
	cmd.Env = append(os.Environ(), execHelperEnv+"="+mode)
	return cmd
}

func normalizeHelperStderr(output string) string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		// A coverage-enabled test binary with no non-test source emits this
		// diagnostic when the helper intentionally exits before test teardown.
		// It is unrelated to the os/exec stderr behavior under test.
		if line != "program not built with -cover" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func TestExecHelperProcess(t *testing.T) {
	mode := os.Getenv(execHelperEnv)
	if mode == "" {
		return
	}

	var err error
	switch mode {
	case "noop":
	case "echo-hello":
		_, err = fmt.Fprintln(os.Stdout, "hello")
	case "echo-test":
		_, err = fmt.Fprintln(os.Stdout, "test")
	case "echo-stdout":
		_, err = fmt.Fprintln(os.Stdout, "stdout test")
	case "stderr-test":
		_, err = fmt.Fprintln(os.Stderr, "stderr test")
	case "pipe-output":
		_, err = fmt.Fprintln(os.Stdout, "pipe output")
	case "pipe-error":
		_, err = fmt.Fprintln(os.Stderr, "pipe error")
	case "combined":
		if _, err = fmt.Fprintln(os.Stdout, "combined stdout"); err == nil {
			_, err = fmt.Fprintln(os.Stderr, "combined stderr")
		}
	case "cat":
		_, err = io.Copy(os.Stdout, os.Stdin)
	case "env":
		_, err = fmt.Fprintln(os.Stdout, os.Getenv("TEST_VAR"))
	case "pwd":
		var directory string
		directory, err = os.Getwd()
		if err == nil {
			_, err = fmt.Fprintln(os.Stdout, directory)
		}
	case "sleep":
		time.Sleep(100 * time.Millisecond)
	case "exit-1":
		os.Exit(1)
	case "exit-42":
		os.Exit(42)
	default:
		err = fmt.Errorf("unknown helper mode %q", mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func TestCommand(t *testing.T) {
	cmd := helperCommand("noop")
	if cmd == nil {
		t.Fatal("Command returned nil")
	}

	if cmd.Path == "" {
		t.Error("Command Path is empty")
	}
}

func TestCommandContext(t *testing.T) {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestExecHelperProcess$")
	if cmd == nil {
		t.Fatal("CommandContext returned nil")
	}
}

func TestCmdRun(t *testing.T) {
	cmd := helperCommand("noop")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
}

func TestCmdOutput(t *testing.T) {
	cmd := helperCommand("echo-hello")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output error: %v", err)
	}

	result := strings.TrimSpace(string(output))
	if result != "hello" {
		t.Errorf("Output = %q, want %q", result, "hello")
	}
}

func TestCmdCombinedOutput(t *testing.T) {
	cmd := helperCommand("combined")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CombinedOutput error: %v", err)
	}

	result := string(output)
	if !strings.Contains(result, "combined stdout") || !strings.Contains(result, "combined stderr") {
		t.Errorf("CombinedOutput = %q, want stdout and stderr", result)
	}
}

func TestCmdStdin(t *testing.T) {
	cmd := helperCommand("cat")
	cmd.Stdin = strings.NewReader("test input")

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output error: %v", err)
	}

	result := string(output)
	if result != "test input" {
		t.Errorf("Output = %q, want %q", result, "test input")
	}
}

func TestCmdStdout(t *testing.T) {
	var buf bytes.Buffer
	cmd := helperCommand("echo-stdout")
	cmd.Stdout = &buf

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "stdout test" {
		t.Errorf("Stdout = %q, want %q", output, "stdout test")
	}
}

func TestCmdStderr(t *testing.T) {
	var buf bytes.Buffer
	cmd := helperCommand("stderr-test")
	cmd.Stderr = &buf

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	output := normalizeHelperStderr(buf.String())
	if output != "stderr test" {
		t.Errorf("Stderr = %q, want %q", output, "stderr test")
	}
}

func TestCmdStart(t *testing.T) {
	cmd := helperCommand("sleep")
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	if cmd.Process == nil {
		t.Error("Process is nil after Start")
	}

	err = cmd.Wait()
	if err != nil {
		t.Errorf("Wait error: %v", err)
	}
}

func TestCmdWait(t *testing.T) {
	cmd := helperCommand("noop")
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	err = cmd.Wait()
	if err != nil {
		t.Errorf("Wait error: %v", err)
	}

	if cmd.ProcessState == nil {
		t.Error("ProcessState is nil after Wait")
	}
}

func TestCmdStdinPipe(t *testing.T) {
	cmd := helperCommand("cat")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe error: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}

	io.WriteString(stdin, "pipe test")
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		t.Errorf("Wait error: %v", err)
	}
}

func TestCmdStdoutPipe(t *testing.T) {
	cmd := helperCommand("pipe-output")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe error: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}

	data, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}

	output := strings.TrimSpace(string(data))
	if output != "pipe output" {
		t.Errorf("Output = %q, want %q", output, "pipe output")
	}

	if err := cmd.Wait(); err != nil {
		t.Errorf("Wait error: %v", err)
	}
}

func TestCmdStderrPipe(t *testing.T) {
	cmd := helperCommand("pipe-error")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("StderrPipe error: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}

	data, err := io.ReadAll(stderr)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}

	output := normalizeHelperStderr(string(data))
	if output != "pipe error" {
		t.Errorf("Output = %q, want %q", output, "pipe error")
	}

	if err := cmd.Wait(); err != nil {
		t.Errorf("Wait error: %v", err)
	}
}

func TestCmdEnv(t *testing.T) {
	cmd := helperCommand("env")
	cmd.Env = append(cmd.Env, "TEST_VAR=test_value")

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output error: %v", err)
	}

	result := strings.TrimSpace(string(output))
	if result != "test_value" {
		t.Errorf("Output = %q, want %q", result, "test_value")
	}
}

func TestCmdDir(t *testing.T) {
	tmpDir := t.TempDir()
	cmd := helperCommand("pwd")
	cmd.Dir = tmpDir

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output error: %v", err)
	}

	result := filepath.Clean(strings.TrimSpace(string(output)))
	wantInfo, wantErr := os.Stat(tmpDir)
	gotInfo, gotErr := os.Stat(result)
	if wantErr != nil || gotErr != nil || !os.SameFile(wantInfo, gotInfo) {
		t.Errorf("working directory = %q, want %q (stat errors: %v, %v)", result, tmpDir, gotErr, wantErr)
	}
}

func TestCmdString(t *testing.T) {
	cmd := exec.Command("echo", "test")
	str := cmd.String()
	if str == "" {
		t.Error("String() returned empty")
	}
}

func TestLookPath(t *testing.T) {
	name := "echo"
	if runtime.GOOS == "windows" {
		name = "cmd"
	}
	path, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf("LookPath error: %v", err)
	}

	if path == "" {
		t.Error("LookPath returned empty path")
	}
}

func TestError(t *testing.T) {
	err := &exec.Error{
		Name: "test",
		Err:  os.ErrNotExist,
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error.Error() returned empty string")
	}
}

func TestExitError(t *testing.T) {
	cmd := helperCommand("exit-1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("Expected error for exit code 1")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("Error is not ExitError: %T", err)
	}

	if exitErr.ExitCode() != 1 {
		t.Errorf("ExitCode = %d, want 1", exitErr.ExitCode())
	}
}

func TestErrNotFound(t *testing.T) {
	if exec.ErrNotFound == nil {
		t.Error("ErrNotFound should not be nil")
	}
}

func TestErrDot(t *testing.T) {
	if exec.ErrDot == nil {
		t.Error("ErrDot should not be nil")
	}
}

func TestErrWaitDelay(t *testing.T) {
	if exec.ErrWaitDelay == nil {
		t.Error("ErrWaitDelay should not be nil")
	}
}

func TestCmdEnviron(t *testing.T) {
	cmd := helperCommand("noop")
	cmd.Env = []string{"VAR1=value1", "VAR2=value2"}

	environ := cmd.Environ()
	if len(environ) == 0 {
		t.Error("Environ returned empty slice")
	}

	found := false
	for _, env := range environ {
		if strings.HasPrefix(env, "VAR1=") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Environ doesn't contain VAR1")
	}
}

func TestErrorUnwrap(t *testing.T) {
	baseErr := os.ErrNotExist
	err := &exec.Error{
		Name: "test",
		Err:  baseErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("Unwrap = %v, want %v", unwrapped, baseErr)
	}
}

func TestExitErrorError(t *testing.T) {
	cmd := helperCommand("exit-42")
	err := cmd.Run()
	if err == nil {
		t.Fatal("Expected error for exit code 42")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("Error is not ExitError: %T", err)
	}

	errStr := exitErr.Error()
	if errStr == "" {
		t.Error("ExitError.Error() returned empty string")
	}
}
