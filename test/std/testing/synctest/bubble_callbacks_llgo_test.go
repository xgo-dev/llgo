//go:build llgo && go1.27

package synctest_test

import (
	"sync"
	"testing"
	"testing/synctest"
	"time"
	_ "unsafe"
)

// Pin the callback bridge to the same private Go version as its implementation.
//
//go:linkname acquire internal/synctest.acquire
func acquire() any

//go:linkname release internal/synctest.release
func release(any)

//go:linkname inBubble internal/synctest.inBubble
func inBubble(any, func())

//go:linkname isInBubble internal/synctest.IsInBubble
func isInBubble() bool

//go:linkname bubblesActive github.com/xgo-dev/llgo/runtime/internal/runtime.SynctestActive
func bubblesActive() bool

func TestBubbleFastPathLifetime(t *testing.T) {
	if bubblesActive() {
		t.Fatal("bubble remained active after earlier tests")
	}
	ready := make(chan struct{}, 2)
	done := make(chan struct{})
	stop := [2]chan struct{}{make(chan struct{}), make(chan struct{})}
	for _, ch := range stop {
		go func() {
			synctest.Test(t, func(t *testing.T) {
				var wg sync.WaitGroup
				wg.Go(func() { time.Sleep(time.Hour) })
				wg.Wait()
				ready <- struct{}{}
				<-ch
			})
			done <- struct{}{}
		}()
	}
	<-ready
	<-ready
	if !bubblesActive() {
		t.Error("running bubbles were not counted")
	}
	close(stop[0])
	<-done
	if !bubblesActive() {
		t.Error("first exit cleared the other bubble's count")
	}
	close(stop[1])
	<-done
	if bubblesActive() {
		t.Error("last exit did not restore the semaphore fast path")
	}
}

func TestCallbackMembershipAndLifetime(t *testing.T) {
	requests := make(chan func())
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		for f := range requests {
			f()
			if isInBubble() {
				panic("callback leaked bubble")
			}
		}
	}()
	synctest.Test(t, func(t *testing.T) {
		b := acquire()
		if b == nil {
			t.Fatal("acquire returned nil")
		}
		result := make(chan bool, 1)
		requests <- func() {
			defer release(b)
			inBubble(b, func() {
				defer func() {
					if recover() == nil {
						panic("missing nested panic")
					}
				}()
				result <- isInBubble() && time.Now().UTC().Year() == 2000
				panic("callback cleanup")
			})
		}
		if !<-result {
			t.Fatal("callback lost bubble")
		}
		synctest.Wait()
	})
	close(requests)
	<-workerDone
	if acquire() != nil {
		t.Fatal("outside acquire returned a bubble")
	}
}
