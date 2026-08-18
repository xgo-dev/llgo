//go:build go1.25

package sync_test

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestWaitGroupGo(t *testing.T) {
	var group sync.WaitGroup
	var count atomic.Int32
	group.Go(func() {
		count.Add(1)
		group.Go(func() {
			count.Add(1)
		})
	})
	group.Go(func() {
		count.Add(1)
	})
	group.Wait()
	if got := count.Load(); got != 3 {
		t.Fatalf("completed tasks = %d, want 3", got)
	}
}
