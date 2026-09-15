//go:build llgo && !baremetal && !nogc

package gotest

import (
	"runtime"
	"sync"
	"testing"
	"time"
	_ "unsafe"
)

type concurrentFinalizerValue struct {
	value int
	pad   [64]byte
}

//go:linkname getBDWGCFinalizeOnDemand C.GC_get_finalize_on_demand
func getBDWGCFinalizeOnDemand() int32

//go:linkname setBDWGCFinalizeOnDemand C.GC_set_finalize_on_demand
func setBDWGCFinalizeOnDemand(enabled int32)

func TestRuntimeSetFinalizerCancelAfterBoxedThenTyped(t *testing.T) {
	finalized := make(chan struct{}, 2)
	finalizerCancelDone = finalized
	func() {
		x := new(int)
		runtime.SetFinalizer(x, func(*int) {
			finalized <- struct{}{}
		})
		runtime.SetFinalizer(x, finalizerCancelSentinel)
		runtime.SetFinalizer(x, nil)
	}()
	assertCanceledFinalizer(t, finalized)
}

func TestRuntimeSetFinalizerCancelPreservesCleanup(t *testing.T) {
	cleaned := make(chan struct{}, 1)
	registerFinalizerForTest(func() {
		x := new(int)
		runtime.AddCleanup(x, func(struct{}) {
			cleaned <- struct{}{}
		}, struct{}{})
		runtime.SetFinalizer(x, nil)
	})
	waitForFinalizerSignal(t, cleaned)
}

func TestRuntimeSetFinalizerCancelAfterBoxedKeepsCleanup(t *testing.T) {
	cleaned := make(chan struct{}, 1)
	finalized := make(chan struct{}, 1)
	registerFinalizerForTest(func() {
		x := new(int)
		runtime.AddCleanup(x, func(struct{}) {
			cleaned <- struct{}{}
		}, struct{}{})
		runtime.SetFinalizer(x, func(*int) {
			finalized <- struct{}{}
		})
		runtime.SetFinalizer(x, nil)
	})
	waitForFinalizerSignal(t, cleaned)
	select {
	case <-finalized:
		t.Fatal("canceled finalizer ran")
	default:
	}
}

func TestRuntimeSetFinalizerCancelAfterBoxedThenCleanup(t *testing.T) {
	cleaned := make(chan struct{}, 1)
	finalized := make(chan struct{}, 1)
	registerFinalizerForTest(func() {
		x := new(int)
		runtime.SetFinalizer(x, func(*int) {
			finalized <- struct{}{}
		})
		runtime.AddCleanup(x, func(struct{}) {
			cleaned <- struct{}{}
		}, struct{}{})
		runtime.SetFinalizer(x, nil)
	})
	waitForFinalizerSignal(t, cleaned)
	select {
	case <-finalized:
		t.Fatal("canceled finalizer ran")
	default:
	}
}

func waitForFinalizerSignal(t *testing.T, done <-chan struct{}) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		runGCWithTimeout(t)
		select {
		case <-done:
			return
		case <-deadline:
			t.Fatal("cleanup did not run")
		default:
		}
	}
}

func TestRuntimeAddCleanupStop(t *testing.T) {
	old := getBDWGCFinalizeOnDemand()
	setBDWGCFinalizeOnDemand(1)
	t.Cleanup(func() {
		setBDWGCFinalizeOnDemand(old)
	})

	const n = 32
	stopped := make(chan int32, n)
	active := make(chan int32, n)
	activeHandles := make(chan runtime.Cleanup, n)
	created := make(chan struct{})
	go func() {
		for i := range int32(n) {
			stoppedObject := new([64]byte)
			cleanup := runtime.AddCleanup(stoppedObject, func(value int32) {
				stopped <- value
			}, i)
			cleanup.Stop()
			cleanup.Stop()
			runtime.KeepAlive(stoppedObject)

			activeObject := new([64]byte)
			activeHandles <- runtime.AddCleanup(activeObject, func(value int32) {
				active <- value
			}, i)
		}
		close(created)
	}()
	<-created

	deadline := time.After(3 * time.Second)
	for len(active) <= n/2 {
		runtime.Gosched()
		runGCWithTimeout(t)
		select {
		case <-deadline:
			t.Fatalf("only %d/%d active cleanups ran", len(active), n)
		default:
		}
	}
	for range n {
		(<-activeHandles).Stop()
	}
	for range 3 {
		runGCWithTimeout(t)
	}
	if got := len(stopped); got != 0 {
		t.Fatalf("%d stopped cleanups ran", got)
	}
}

func TestRuntimeGCDrainsBDWGCFinalizersOnDemand(t *testing.T) {
	// BDWGC normally may invoke ready finalizers during a later allocation.
	// On-demand mode makes runtime.GC's explicit drain observable without
	// relying on allocation timing. This setting is process-global, so this
	// test and the neighboring finalizer tests must remain sequential.
	old := getBDWGCFinalizeOnDemand()
	setBDWGCFinalizeOnDemand(1)
	t.Cleanup(func() {
		setBDWGCFinalizeOnDemand(old)
	})

	const n = 32
	finalized := make(chan int32, n)
	created := make(chan struct{})
	go func() {
		makeFinalizerTinyObjects(n, finalized)
		close(created)
	}()
	<-created

	// InvokeFinalizers runs synchronously once BDWGC has queued an object.
	// The retries only let the producer goroutine exit so that conservative
	// stack and register roots no longer keep the objects reachable.
	for range 8 {
		runtime.Gosched()
		runGCWithTimeout(t)
		if len(finalized) > n/2 {
			return
		}
	}
	if got := len(finalized); got <= n/2 {
		t.Fatalf("runtime.GC ran only %d/%d on-demand finalizers", got, n)
	}
}

func TestRuntimeConcurrentGCFinalizers(t *testing.T) {
	// Keep finalization inside the concurrent runtime.GC calls below instead
	// of letting an unrelated allocation drain BDWGC's ready queue first.
	old := getBDWGCFinalizeOnDemand()
	setBDWGCFinalizeOnDemand(1)
	t.Cleanup(func() {
		setBDWGCFinalizeOnDemand(old)
	})

	const (
		// Keep the routine regression small. The opt-in high-load variant lives
		// in test/_stress/runtime/finalizer.
		objects = 32
		workers = 4
	)
	finalized := make(chan int, objects)
	registered := make(chan struct{})
	go func() {
		for i := range objects {
			p := &concurrentFinalizerValue{value: i}
			runtime.SetFinalizer(p, func(p *concurrentFinalizerValue) {
				finalized <- p.value
			})
		}
		close(registered)
	}()
	<-registered

	seen := make(map[int]bool)
	deadline := time.After(3 * time.Second)
	for len(seen) <= objects/2 {
		var wg sync.WaitGroup
		wg.Add(workers)
		for range workers {
			go func() {
				defer wg.Done()
				runtime.GC()
			}()
		}
		wg.Wait()

		for {
			select {
			case value := <-finalized:
				if value < 0 || value >= objects {
					t.Fatalf("finalizer got %d, want [0,%d)", value, objects)
				}
				if seen[value] {
					t.Fatalf("finalizer got duplicate value %d", value)
				}
				seen[value] = true
			default:
				goto drained
			}
		}
	drained:
		if len(seen) > objects/2 {
			return
		}
		runtime.Gosched()
		select {
		case <-deadline:
			t.Fatalf("only %d/%d concurrent finalizers ran", len(seen), objects)
		default:
		}
	}
}
