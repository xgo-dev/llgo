//go:build !wasm && !baremetal && !nogc

package gotest

import (
	"runtime"
	"testing"
)

func TestRuntimeConcurrentGCRequests(t *testing.T) {
	// Overlap a bounded set of requests while retaining a private object per
	// goroutine. NumGC checks global progress, not ownership of a GC cycle.
	const workers = 8
	start, done := make(chan struct{}), make(chan bool, workers)
	for id := range workers {
		go func() {
			live := new([64]int)
			live[0] = id + 1
			<-start
			for range 4 {
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				runtime.GC()
				runtime.ReadMemStats(&after)
				if after.NumGC-before.NumGC == 0 || live[0] != id+1 {
					done <- false
					return
				}
			}
			runtime.KeepAlive(live)
			done <- true
		}()
	}
	close(start)
	for range workers {
		if !<-done {
			t.Fatal("overlapping GC requests made no global progress or lost a live object")
		}
	}
}
