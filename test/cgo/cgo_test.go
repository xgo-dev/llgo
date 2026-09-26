//go:build llgo && !wasm
// +build llgo,!wasm

// These tests exercise import "C", whose Go frontend has no wasm target ABI.
// WebAssembly C calls and callbacks are covered by the wasm target fixtures.

package cgo

import (
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"testing"
)

const cFaultTracebackChild = "LLGO_TEST_C_FAULT_TRACEBACK"

func init() {
	if os.Getenv(cFaultTracebackChild) == "1" {
		cFaultViaGo()
	}
}

func TestCgoBasicCall(t *testing.T) {
	if got := Add(20, 22); got != 42 {
		t.Fatalf("c_add mismatch: got %d, want 42", got)
	}
}

func TestC2funcStructs(t *testing.T) {
	sum, err := SumStructs()
	if err != nil {
		t.Fatalf("unexpected errno: %v", err)
	}
	if got := sum; got != 35 {
		t.Fatalf("sum_structs mismatch: got %d, want 35", got)
	}
}

func TestC2funcErrno(t *testing.T) {
	v, err := ErrnoWrap(-1)
	if v != -1 {
		t.Fatalf("c_errno_wrap(-1) value mismatch: got %d, want -1", v)
	}
	if err == nil {
		t.Fatal("c_errno_wrap(-1) expected non-nil errno")
	}
	errno, ok := err.(syscall.Errno)
	if !ok || errno == 0 {
		t.Fatalf("unexpected errno type/value: %T %v", err, err)
	}

	v, err = ErrnoWrap(9)
	if err != nil {
		t.Fatalf("c_errno_wrap(9) unexpected errno: %v", err)
	}
	if got := v; got != 10 {
		t.Fatalf("c_errno_wrap(9) value mismatch: got %d, want 10", got)
	}
}

func TestCgoMallocWrapperSymbols(t *testing.T) {
	if !MallocFree(8) {
		t.Fatal("C.malloc returned nil")
	}
}

//go:noinline
func cFaultViaGo() {
	CauseFault(2)
}

func recoverCFault(captureStack bool) (recovered any, stack string) {
	defer func() {
		recovered = recover()
		if captureStack {
			stack = string(debug.Stack())
		}
	}()
	cFaultViaGo()
	return nil, ""
}

func TestCFaultRecoverable(t *testing.T) {
	for i := 0; i < 3; i++ {
		recovered, stack := recoverCFault(i == 2)
		err, ok := recovered.(error)
		if !ok || err.Error() != "runtime error: invalid memory address or nil pointer dereference" {
			t.Fatalf("fault %d recovered %T %v", i+1, recovered, recovered)
		}
		if i == 2 {
			if !strings.Contains(stack, "cFaultViaGo") {
				t.Fatalf("recovered stack is missing Go fault frame:\n%s", stack)
			}
			if runtime.GOOS == "darwin" && !strings.Contains(stack, "llgo_test_fault") {
				t.Fatalf("recovered stack is missing C fault frame:\n%s", stack)
			}
		}
	}
}

func TestCFaultTraceback(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), cFaultTracebackChild+"=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("unrecovered fault child unexpectedly succeeded:\n%s", output)
	}
	for _, want := range []string{
		"panic: runtime error: invalid memory address or nil pointer dereference",
		"goroutine 1 [running]:",
		"cFaultViaGo",
	} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("fault traceback is missing %q:\n%s", want, output)
		}
	}
	if runtime.GOOS == "darwin" && !strings.Contains(string(output), "llgo_test_fault") {
		t.Fatalf("fault traceback is missing C fault frame:\n%s", output)
	}
}
