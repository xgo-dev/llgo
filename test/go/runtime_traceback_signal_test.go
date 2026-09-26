//go:build linux || darwin

package gotest

import (
	"bytes"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestRuntimeStackAllCPUProfile(t *testing.T) {
	var profile bytes.Buffer
	if err := pprof.StartCPUProfile(&profile); err != nil {
		t.Fatal(err)
	}
	defer pprof.StopCPUProfile()
	ready, done := make(chan struct{}), make(chan struct{})
	var stop int32
	go func() {
		defer close(done)
		tracebackBusy(ready, &stop)
	}()
	<-ready
	defer func() { atomic.StoreInt32(&stop, 1); <-done }()
	buf := make([]byte, 128<<10)
	deadline := time.Now().Add(200 * time.Millisecond)
	sawWorker := false
	for time.Now().Before(deadline) {
		n := runtime.Stack(buf, true)
		if n == 0 {
			t.Fatal("empty traceback while CPU profiling")
		}
		sawWorker = sawWorker || strings.Contains(string(buf[:n]), ".tracebackBusy(")
	}
	if !sawWorker {
		t.Fatal("CPU profiling prevented other-thread stack capture")
	}
}

func TestRuntimeStackAllSignal(t *testing.T) {
	ready, stop := make(chan struct{}), make(chan struct{})
	defer close(stop)
	go tracebackBlocked(ready, stop)
	<-ready
	ch := make(chan os.Signal, 1)
	defer signal.Stop(ch)
	defer signal.Reset(syscall.SIGURG)
	buf := make([]byte, 128<<10)
	for i := 0; i < 3; i++ {
		signal.Notify(ch, syscall.SIGURG)
		if err := syscall.Kill(os.Getpid(), syscall.SIGURG); err != nil {
			t.Fatal(err)
		}
		select {
		case <-ch:
		case <-time.After(5 * time.Second):
			t.Fatal("SIGURG notification was lost")
		}
		for _, action := range []func(){func() {}, func() { signal.Ignore(syscall.SIGURG) }, func() { signal.Reset(syscall.SIGURG) }} {
			action()
			out := string(buf[:runtime.Stack(buf, true)])
			if !strings.Contains(out, ".tracebackBlocked(") {
				t.Fatalf("SIGURG disposition prevented traceback:\n%s", out)
			}
		}
	}
}
