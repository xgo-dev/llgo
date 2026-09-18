//go:build wasm

package exec_test

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

// Go wasm has no process creation or anonymous pipes. Test those exact error
// contracts, and retain command construction, environment and validation tests.
// Host process/pipe success cases remain in exec_test.go on native targets.
func TestWasmCommandConstruction(t *testing.T) {
	cmd := exec.Command("/program", "first", "second")
	if cmd.Path != "/program" || !reflect.DeepEqual(cmd.Args, []string{"/program", "first", "second"}) || cmd.String() != "/program first second" {
		t.Fatalf("command = %+v, string=%q", cmd, cmd.String())
	}
	cmd.Env = []string{"LLGO_EXEC_FIRST=old", "LLGO_EXEC_SECOND=value", "LLGO_EXEC_FIRST=new"}
	if got := cmd.Environ(); !reflect.DeepEqual(got, []string{"LLGO_EXEC_SECOND=value", "LLGO_EXEC_FIRST=new"}) {
		t.Fatalf("Environ = %q", got)
	}
	if cmd := exec.CommandContext(context.Background(), "/program"); cmd.Cancel == nil {
		t.Fatal("CommandContext did not install cancellation")
	}
}

func TestWasmLookPathAndError(t *testing.T) {
	for _, name := range []string{"program", "/program"} {
		path, err := exec.LookPath(name)
		var ee *exec.Error
		if path != "" || !errors.Is(err, exec.ErrNotFound) || !errors.As(err, &ee) || ee.Name != name {
			t.Fatalf("LookPath(%q) = %q, %v", name, path, err)
		}
		if ee.Unwrap() != exec.ErrNotFound || ee.Error() == "" {
			t.Fatalf("exec.Error contract: %+v", ee)
		}
	}
	if exec.ErrDot == nil || exec.ErrWaitDelay == nil {
		t.Fatal("missing exported error sentinels")
	}
}

func TestWasmProcessCreationUnsupported(t *testing.T) {
	for _, method := range []string{"Run", "Start", "Output", "CombinedOutput"} {
		t.Run(method, func(t *testing.T) {
			cmd := exec.Command("/program")
			var err error
			switch method {
			case "Run", "Start":
				// Avoid incidental /dev/null or pipe creation; reach StartProcess.
				cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
				if method == "Run" {
					err = cmd.Run()
				} else {
					err = cmd.Start()
				}
			case "Output":
				_, err = cmd.Output()
			case "CombinedOutput":
				_, err = cmd.CombinedOutput()
			}
			if !errors.Is(err, syscall.ENOSYS) {
				t.Fatalf("%s = %v, want ENOSYS", method, err)
			}
			if cmd.Process != nil || cmd.ProcessState != nil {
				t.Fatal("failed start published a process")
			}
			if err := cmd.Wait(); err == nil || !strings.Contains(err.Error(), "not started") {
				t.Fatalf("Wait after failed start = %v", err)
			}
		})
	}
}

func TestWasmCommandPipesUnsupported(t *testing.T) {
	for _, method := range []string{"stdin", "stdout", "stderr"} {
		t.Run(method, func(t *testing.T) {
			cmd := exec.Command("/program")
			var pipe io.Closer
			var err error
			switch method {
			case "stdin":
				pipe, err = cmd.StdinPipe()
			case "stdout":
				pipe, err = cmd.StdoutPipe()
			case "stderr":
				pipe, err = cmd.StderrPipe()
			}
			if pipe != nil || !errors.Is(err, syscall.ENOSYS) {
				t.Fatalf("%s pipe = %v, %v", method, pipe, err)
			}
		})
	}
}

func TestWasmCommandContextAndValidation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := exec.CommandContext(ctx, "/program")
	if err := cmd.Run(); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Run = %v", err)
	}
	cmd = exec.Command("/program")
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if _, err := cmd.Output(); err == nil || !strings.Contains(err.Error(), "Stdout already set") {
		t.Fatalf("Output with Stdout = %v", err)
	}
	if _, err := cmd.CombinedOutput(); err == nil || !strings.Contains(err.Error(), "Stdout already set") {
		t.Fatalf("CombinedOutput with Stdout = %v", err)
	}
	if _, err := cmd.StdinPipe(); err == nil || !strings.Contains(err.Error(), "Stdin already set") {
		t.Fatalf("StdinPipe with Stdin = %v", err)
	}
	if _, err := cmd.StdoutPipe(); err == nil || !strings.Contains(err.Error(), "Stdout already set") {
		t.Fatalf("StdoutPipe with Stdout = %v", err)
	}
	if _, err := cmd.StderrPipe(); err == nil || !strings.Contains(err.Error(), "Stderr already set") {
		t.Fatalf("StderrPipe with Stderr = %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("CommandContext(nil) did not panic")
		}
	}()
	exec.CommandContext(nil, "/program")
}
