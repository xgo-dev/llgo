//go:build !wasm && !baremetal && !nogc

package gotest

import (
	"runtime"
	"testing"
	"time"
	"weak"
)

type pooledWeakObj struct {
	value int
	_     [64]byte
}

func TestGoroutinePoolDoesNotPinFinishedStackRoots(t *testing.T) {
	// Ensure an object created on a short-lived goroutine that parks in the idle
	// thread pool is still collectible and not pinned by stale stack/register roots.
	for round := 0; round < 4; round++ {
		res := make(chan weak.Pointer[int], 1)
		done := make(chan struct{})
		go func(v int) {
			obj := &pooledWeakObj{value: v}
			res <- weak.Make(&obj.value)
			close(done)
		}(100 + round)
		wp := <-res
		<-done

		deadline := time.Now().Add(3 * time.Second)
		cleared := false
		for time.Now().Before(deadline) {
			runtime.GC()
			chk := make(chan bool, 1)
			go func() { chk <- wp.Value() == nil }()
			if <-chk {
				cleared = true
				break
			}
			time.Sleep(time.Millisecond)
		}
		if !cleared {
			t.Fatalf("round %d: weak pointer remained live after pooled worker parked", round)
		}
	}
}
