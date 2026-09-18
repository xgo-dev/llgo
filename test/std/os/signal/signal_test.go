//go:build !windows && !plan9 && !wasm

package signal_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"
)

const atomicStopHelperEnv = "LLGO_TEST_ATOMIC_SIGNAL_STOP"

func TestNotify(t *testing.T) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGWINCH)
	defer signal.Stop(c)

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess error: %v", err)
	}

	err = proc.Signal(syscall.SIGWINCH)
	if err != nil {
		t.Fatalf("Signal error: %v", err)
	}

	select {
	case sig := <-c:
		if sig != syscall.SIGWINCH {
			t.Errorf("Received signal %v, want SIGWINCH", sig)
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for signal")
	}
}

func TestNotifyMultipleSignals(t *testing.T) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGWINCH, syscall.SIGCHLD)
	defer signal.Stop(c)

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess error: %v", err)
	}

	err = proc.Signal(syscall.SIGWINCH)
	if err != nil {
		t.Fatalf("Signal SIGWINCH error: %v", err)
	}

	timeout := time.After(time.Second)
	for {
		select {
		case sig := <-c:
			// SIGCHLD may arrive first, so wait for the signal sent above.
			if sig == syscall.SIGWINCH {
				return
			}
		case <-timeout:
			t.Fatal("Timeout waiting for SIGWINCH")
		}
	}
}

func TestStop(t *testing.T) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGWINCH)
	signal.Stop(c)

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess error: %v", err)
	}

	err = proc.Signal(syscall.SIGWINCH)
	if err != nil {
		t.Fatalf("Signal error: %v", err)
	}

	select {
	case sig := <-c:
		t.Errorf("Received signal %v after Stop", sig)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestAtomicStop(t *testing.T) {
	if os.Getenv(atomicStopHelperEnv) == "1" {
		runAtomicStopHelper()
		return
	}

	// Keep SIGINT from being inherited as ignored by the helper. A caught
	// signal is reset to its default disposition by exec, so every helper
	// must either receive SIGINT through os/signal or terminate from it.
	parentSignals := make(chan os.Signal, 1)
	signal.Notify(parentSignals, syscall.SIGINT)
	defer signal.Stop(parentSignals)

	for i := 0; i < 10; i++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestAtomicStop$")
		cmd.Env = append(os.Environ(), atomicStopHelperEnv+"=1")
		out, err := cmd.CombinedOutput()
		if bytes.Contains(out, []byte("lost signal")) {
			t.Fatalf("iteration %d dropped a signal during Stop:\n%s", i, out)
		}
		if err == nil {
			continue
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("iteration %d: %v", i, err)
		}
		status, ok := exitErr.Sys().(syscall.WaitStatus)
		if !ok || !status.Signaled() || status.Signal() != syscall.SIGINT {
			t.Fatalf("iteration %d exited unexpectedly: %v\n%s", i, err, out)
		}
	}
}

func runAtomicStopHelper() {
	if signal.Ignored(syscall.SIGINT) {
		fmt.Println("SIGINT is ignored")
		os.Exit(2)
	}

	pid := syscall.Getpid()
	lost := false
	for i := 0; i < 10; i++ {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT)

		var stopped sync.WaitGroup
		stopped.Add(1)
		go func() {
			defer stopped.Done()
			signal.Stop(c)
		}()

		if err := syscall.Kill(pid, syscall.SIGINT); err != nil {
			fmt.Printf("kill: %v\n", err)
			os.Exit(2)
		}
		select {
		case <-c:
		case <-time.After(2 * time.Second):
			fmt.Printf("lost signal on try %d\n", i)
			lost = true
		}
		stopped.Wait()
	}
	if lost {
		os.Exit(3)
	}
	os.Exit(0)
}

func TestReset(t *testing.T) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGWINCH)
	signal.Reset(syscall.SIGWINCH)

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess error: %v", err)
	}

	err = proc.Signal(syscall.SIGWINCH)
	if err != nil {
		t.Fatalf("Signal error: %v", err)
	}

	select {
	case sig := <-c:
		t.Errorf("Received signal %v after Reset", sig)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestResetAll(t *testing.T) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGWINCH)
	signal.Reset()

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess error: %v", err)
	}

	err = proc.Signal(syscall.SIGWINCH)
	if err != nil {
		t.Fatalf("Signal error: %v", err)
	}

	select {
	case sig := <-c:
		t.Errorf("Received signal %v after Reset()", sig)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIgnore(t *testing.T) {
	signal.Ignore(syscall.SIGWINCH)
	defer signal.Reset(syscall.SIGWINCH)

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess error: %v", err)
	}

	err = proc.Signal(syscall.SIGWINCH)
	if err != nil {
		t.Fatalf("Signal error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}

func TestIgnored(t *testing.T) {
	wasIgnored := signal.Ignored(syscall.SIGWINCH)

	signal.Ignore(syscall.SIGWINCH)
	defer signal.Reset(syscall.SIGWINCH)

	if !signal.Ignored(syscall.SIGWINCH) {
		t.Error("Expected SIGWINCH to be ignored after Ignore()")
	}

	signal.Reset(syscall.SIGWINCH)

	afterReset := signal.Ignored(syscall.SIGWINCH)
	if afterReset != wasIgnored {
		t.Logf("Signal ignored state changed after Reset: was=%v, after=%v", wasIgnored, afterReset)
	}
}

func TestNotifyContext(t *testing.T) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGWINCH)
	defer stop()

	select {
	case <-ctx.Done():
		t.Error("Context should not be done before signal")
	case <-time.After(100 * time.Millisecond):
	}

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess error: %v", err)
	}

	err = proc.Signal(syscall.SIGWINCH)
	if err != nil {
		t.Fatalf("Signal error: %v", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for context cancellation")
	}
}

func TestNotifyContextStop(t *testing.T) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGWINCH)

	stop()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("Context should be cancelled after stop()")
	}
}

func TestMultipleChannels(t *testing.T) {
	c1 := make(chan os.Signal, 1)
	c2 := make(chan os.Signal, 1)

	signal.Notify(c1, syscall.SIGWINCH)
	signal.Notify(c2, syscall.SIGWINCH)
	defer signal.Stop(c1)
	defer signal.Stop(c2)

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess error: %v", err)
	}

	err = proc.Signal(syscall.SIGWINCH)
	if err != nil {
		t.Fatalf("Signal error: %v", err)
	}

	receivedC1 := false
	receivedC2 := false

	timeout := time.After(time.Second)
	for !receivedC1 || !receivedC2 {
		select {
		case <-c1:
			receivedC1 = true
		case <-c2:
			receivedC2 = true
		case <-timeout:
			t.Fatal("Timeout waiting for signals on both channels")
		}
	}
}
