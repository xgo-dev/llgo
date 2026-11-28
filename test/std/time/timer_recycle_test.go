package time_test

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestTimerStateFreelistRecycling(t *testing.T) {
	// Churn well past timerStateFreeCap (64) with interleaved Stop, Reset,
	// AfterFunc, and firing timers to ensure recycled timerState entries never
	// retain stale callbacks or trigger unexpected fires.
	var staleFires atomic.Int32
	for i := 0; i < 200; i++ {
		tm := time.AfterFunc(200*time.Millisecond, func() {
			staleFires.Add(1)
		})
		if !tm.Stop() {
			t.Fatalf("iter %d: expected Stop() == true", i)
		}
	}

	var goodFires atomic.Int32
	done := make(chan struct{}, 100)
	for i := 0; i < 100; i++ {
		tm := time.AfterFunc(1*time.Millisecond, func() {
			goodFires.Add(1)
			done <- struct{}{}
		})
		tm.Reset(2 * time.Millisecond)
	}
	for i := 0; i < 100; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for recycled timer %d", i)
		}
	}
	if got := goodFires.Load(); got != 100 {
		t.Fatalf("goodFires = %d, want 100", got)
	}
	if got := staleFires.Load(); got != 0 {
		t.Fatalf("staleFires = %d, want 0 (stopped timer fired after recycle)", got)
	}
}
