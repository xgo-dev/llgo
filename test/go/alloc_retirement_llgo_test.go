//go:build llgo && !wasm && !baremetal && !nogc

package gotest

import (
	"runtime"
	"testing"
	"time"
	_ "unsafe"
)

//go:linkname retiredRootCacheCounts github.com/xgo-dev/llgo/runtime/internal/runtime.retiredRootCacheCountsForTesting
func retiredRootCacheCounts() (retired, undrained uint32)

func TestSmallGoroutineStartArgBurstRetirementUnderGC(t *testing.T) {
	// Exercise waves larger than the native worker pool idle cap (max 32) so
	// excess recyclable workers retire through finishRetiredM and flush any
	// cached small root startup blocks instead of leaking them.
	retiredBefore, undrainedBefore := retiredRootCacheCounts()
	const waves = 6
	const burst = 96
	for wave := 0; wave < waves; wave++ {
		ready := make(chan struct{}, burst)
		release := make(chan struct{})
		done := make(chan int, burst)
		for i := 0; i < burst; i++ {
			payload := new(int)
			*payload = wave*1000 + i
			go func(p *int, want int) {
				ready <- struct{}{}
				<-release
				if *p != want {
					done <- -1
					return
				}
				done <- *p
			}(payload, wave*1000+i)
		}
		for i := 0; i < burst; i++ {
			<-ready
		}
		close(release)
		sum := 0
		wantSum := 0
		for i := 0; i < burst; i++ {
			v := <-done
			if v < 0 {
				t.Fatalf("wave %d: startup arg pointer corrupted", wave)
			}
			sum += v
			wantSum += wave*1000 + i
		}
		if sum != wantSum {
			t.Fatalf("wave %d: sum = %d, want %d", wave, sum, wantSum)
		}
		if wave%2 == 0 {
			runtime.GC()
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		retired, undrained := retiredRootCacheCounts()
		if undrained != undrainedBefore {
			t.Fatalf("%d retiring workers still held cached roots before context destruction", undrained-undrainedBefore)
		}
		if retired != retiredBefore {
			return
		}
		if !time.Now().Before(deadline) {
			t.Fatal("no worker retired with cached roots")
		}
		time.Sleep(time.Millisecond)
	}
}
