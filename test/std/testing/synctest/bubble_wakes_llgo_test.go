//go:build llgo && go1.27

package synctest_test

import (
	"sync"
	"testing"
	"testing/synctest"
	_ "unsafe"
)

//go:linkname acquireWaitGroup sync.runtime_SemacquireWaitGroup
func acquireWaitGroup(*uint32, bool)

//go:linkname releaseSemaphore sync.runtime_Semrelease
func releaseSemaphore(*uint32, bool, int)

// Run invalid wakes on a pre-existing outside G, then recover and verify that
// the same object can still be signaled by its bubble. Outside channels keep
// the caller active while waiting for the result.
func checkRejectedWake(t *testing.T, prepare func() (wake func(), done <-chan struct{})) {
	t.Helper()
	requests := make(chan func())
	results := make(chan any)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		for f := range requests {
			func() {
				defer func() { results <- recover() }()
				f()
			}()
		}
	}()
	synctest.Test(t, func(t *testing.T) {
		wake, done := prepare()
		synctest.Wait()
		requests <- wake
		if p := <-results; p == nil {
			t.Fatal("outside wake did not panic")
		}
		wake()
		<-done
	})
	close(requests)
	<-workerDone
}

func TestCondWakeAfterRecoveredPanic(t *testing.T) {
	for _, broadcast := range []bool{false, true} {
		name := "Signal"
		if broadcast {
			name = "Broadcast"
		}
		t.Run(name, func(t *testing.T) {
			checkRejectedWake(t, func() (func(), <-chan struct{}) {
				cond := sync.NewCond(new(sync.Mutex))
				done := make(chan struct{})
				go func() {
					cond.L.Lock()
					cond.Wait()
					cond.L.Unlock()
					close(done)
				}()
				wake := cond.Signal
				if broadcast {
					wake = cond.Broadcast
				}
				return wake, done
			})
		})
	}
}

func TestSemaphoreWakeAfterRecoveredPanic(t *testing.T) {
	checkRejectedWake(t, func() (func(), <-chan struct{}) {
		sema := new(uint32)
		done := make(chan struct{})
		go func() { acquireWaitGroup(sema, true); close(done) }()
		return func() { releaseSemaphore(sema, false, 0) }, done
	})
}
