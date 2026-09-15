//go:build llgo && !baremetal && !nogc && !wasm

package gotest

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// Hosted LLGo uses an OS thread per goroutine. Cleanup batches must reuse
// one worker rather than starting a goroutine for each callback.
func TestRuntimeCleanupSerialWorker(t *testing.T) {
	const n = 32
	var active, overlap atomic.Int32
	done := make(chan struct{}, n)
	created := make(chan struct{})
	go func() {
		for range n {
			p := new([1024]byte)
			runtime.AddCleanup(p, func(struct{}) {
				if active.Add(1) != 1 {
					overlap.Store(1)
				}
				time.Sleep(5 * time.Millisecond)
				active.Add(-1)
				done <- struct{}{}
			}, struct{}{})
			runtime.KeepAlive(p)
		}
		close(created)
	}()
	<-created
	deadline := time.Now().Add(5 * time.Second)
	for len(done) <= n/2 && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(time.Millisecond)
	}
	if len(done) <= n/2 {
		t.Fatalf("only %d/%d cleanups ran", len(done), n)
	}
	if overlap.Load() != 0 {
		t.Fatal("cleanup callbacks ran concurrently instead of reusing one worker")
	}
}
