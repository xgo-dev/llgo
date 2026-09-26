//go:build !wasm && !baremetal

package gotest

import (
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

//go:noinline
func tracebackBlocked(ready chan<- struct{}, stop <-chan struct{}) {
	close(ready)
	<-stop
}

//go:noinline
func tracebackBusy(ready chan<- struct{}, stop *int32) {
	close(ready)
	for atomic.LoadInt32(stop) == 0 {
	}
}

func TestRuntimeStackAll(t *testing.T) {
	ready, stop := make(chan struct{}), make(chan struct{})
	defer close(stop)
	go func() { tracebackDepth(250, func() { tracebackBlocked(ready, stop) }) }()
	<-ready
	busyReady := make(chan struct{})
	var busyStop int32
	defer atomic.StoreInt32(&busyStop, 1)
	go tracebackBusy(busyReady, &busyStop)
	<-busyReady
	buf := make([]byte, 256<<10)
	single := string(buf[:runtime.Stack(buf, false)])
	if strings.Contains(single, ".tracebackBlocked(") || strings.Contains(single, ".tracebackBusy(") {
		t.Fatalf("Stack(false) included another goroutine:\n%s", single)
	}
	deadline := time.Now().Add(5 * time.Second)
	var all string
	for {
		all = string(buf[:runtime.Stack(buf, true)])
		blockedState := false
		for _, stack := range strings.Split(all, "\ngoroutine ") {
			if strings.Contains(stack, ".tracebackBlocked(") &&
				strings.Contains(strings.SplitN(stack, "\n", 2)[0], "[chan receive]") {
				blockedState = true
				break
			}
		}
		if blockedState && strings.Contains(all, ".tracebackBusy(") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Stack(true) missed a blocked or running goroutine:\n%s", all)
		}
		runtime.Gosched()
	}
	for _, want := range []string{" frames elided...", "created by ", " in goroutine ", ".TestRuntimeStackAll"} {
		if !strings.Contains(all, want) {
			t.Errorf("Stack(true) missing %q:\n%s", want, all)
		}
	}
}

func TestRuntimeStackAllConcurrent(t *testing.T) {
	// Exercise simultaneous capture, goroutine registration and thread exit.
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			buf := make([]byte, 128<<10)
			for j := 0; j < 10; j++ {
				done := make(chan struct{})
				go func() { close(done) }()
				if n := runtime.Stack(buf, true); n == 0 || !strings.Contains(string(buf[:n]), "goroutine ") {
					t.Error("empty all-goroutine stack")
				}
				<-done
			}
		}()
	}
	workers.Wait()
}
