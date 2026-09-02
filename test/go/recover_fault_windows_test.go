//go:build windows

package gotest

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime/debug"
	"runtime/trace"
	"testing"
	"unsafe"
)

const (
	nonNilFaultChildEnv      = "LLGO_TEST_NON_NIL_FAULT"
	recoverableFaultChildEnv = "LLGO_TEST_RECOVERABLE_FAULT"
	traceFaultChildEnv       = "LLGO_TEST_TRACE_FAULT"
)

type traceFaultValue struct {
	a [16]int
}

//go:noinline
func copyTraceFaultValue(x, y *traceFaultValue) {
	*x = *y
}

func enterRecoverableFaultTest(t *testing.T) bool {
	return enterWindowsFaultTest(
		t, recoverableFaultChildEnv,
		"^TestRecoverAfterFaultPreservesNamedResult$",
	)
}

func enterWindowsFaultTest(t *testing.T, childEnv, testPattern string) bool {
	t.Helper()
	if os.Getenv(childEnv) == "1" {
		return true
	}

	// Keep a possible host Go runtime defect from corrupting the parent test
	// process. The process-level 0xc0000005 observed in CI may be an instance of
	// golang/go#81238, where Windows exception recovery can write below a
	// goroutine stack and corrupt the adjacent heap on some hosts.
	// The child still has to pass the complete fault/panic/recover assertions;
	// if and when the upstream runtime issue is resolved, revisit this isolation
	// and the stack-headroom preparation below.
	cmd := exec.Command(os.Args[0], "-test.run="+testPattern, "-test.v")
	cmd.Env = append(os.Environ(), childEnv+"=1", "GOTRACEBACK=system")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Windows fault test child failed: %v\n%s", err, output)
	}
	return false
}

// Regression test matching GOROOT/test/fixedbugs/issue73748b.go.
func TestRecoverFaultWhileTracing(t *testing.T) {
	if !enterWindowsFaultTest(t, traceFaultChildEnv, "^TestRecoverFaultWhileTracing$") {
		return
	}

	oldGCPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGCPercent)
	if err := trace.Start(io.Discard); err != nil {
		t.Fatal(err)
	}
	defer trace.Stop()
	ensureWindowsExceptionStackHeadroom()

	var recovered bool
	func() {
		defer func() {
			recovered = recover() != nil
			trace.Log(context.Background(), "a", "b")
		}()
		copyTraceFaultValue(nil, nil)
	}()
	if !recovered {
		t.Fatal("nil fault did not panic")
	}
}

func checkRecoveredFaultAddress(t *testing.T, err error, address *byte) {
	t.Helper()
	addressError, ok := err.(interface{ Addr() uintptr })
	if !ok {
		t.Fatalf("recovered fault %T does not report its address", err)
	}
	if got, want := addressError.Addr(), uintptr(unsafe.Pointer(address)); got != want {
		t.Fatalf("recovered fault address %#x, want %#x", got, want)
	}
}

func TestNonNilFaultRequiresPanicOnFault(t *testing.T) {
	if os.Getenv(nonNilFaultChildEnv) == "1" {
		func() {
			defer func() {
				_ = recover()
			}()
			page, _ := protectedMemory(t, 1, 0, 1)
			if page[0] != 0 {
				t.Fatal("unexpected protected-page value")
			}
		}()
		// A runtime that incorrectly turns the access violation into a Go
		// panic reaches here after recover and lets the child exit cleanly.
		// The parent requires Windows to terminate the process instead.
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestNonNilFaultRequiresPanicOnFault$")
	cmd.Env = append(os.Environ(), nonNilFaultChildEnv+"=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("non-nil fault exited successfully:\n%s", output)
	}
	if _, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("start non-nil fault child: %v", err)
	}
}
