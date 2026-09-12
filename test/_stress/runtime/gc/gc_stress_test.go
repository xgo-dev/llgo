//go:build !baremetal && !nogc && !wasm

package gcstress

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func stressCount(t *testing.T, base int) int {
	t.Helper()
	switch profile := os.Getenv("LLGO_STRESS_PROFILE"); profile {
	case "", "default":
		return base
	case "quick":
		return base / 8
	case "heavy":
		return base * 2
	default:
		t.Fatalf("unknown LLGO_STRESS_PROFILE %q", profile)
		return 0
	}
}

func TestGCAllocationProgress(t *testing.T) { testGCProgress(t, false) }

func TestGCMakeFuncProgress(t *testing.T) { testGCProgress(t, true) }

func testGCProgress(t *testing.T, makeFunc bool) {
	t.Helper()
	if os.Getenv("LLGO_STRESS_GC_CHILD") != t.Name() {
		// A collector that starves allocations can also starve testing's alarm
		// and its traceback. Enforce a second deadline outside that runtime.
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, "-test.run=^"+t.Name()+"$", "-test.v", "-test.timeout=75s")
		cmd.Env = append(os.Environ(), "LLGO_STRESS_GC_CHILD="+t.Name())
		output, err := cmd.CombinedOutput()
		t.Logf("%s", output)
		if ctx.Err() != nil {
			t.Fatalf("GC stress helper failed to exit within 90s: %v", ctx.Err())
		}
		if err != nil {
			t.Fatalf("GC stress helper failed: %v", err)
		}
		return
	}

	const workers = 4
	iterations := stressCount(t, 256)
	var collections, allocations, callbacks atomic.Int64
	var badArguments atomic.Int64
	var stop atomic.Bool
	start := make(chan struct{})
	gcDone := make(chan struct{})
	workDone := make(chan struct{})
	var work sync.WaitGroup
	work.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer work.Done()
			<-start
			var live [64]*[4096]byte
			done := make(chan struct{}, iterations)
			for i := 0; i < iterations; i++ {
				p := new([4096]byte)
				p[i%len(p)] = byte(i)
				live[i%len(live)] = p
				if makeFunc {
					if i&1 == 0 {
						f := reflect.MakeFunc(reflect.TypeOf((func(*int))(nil)), func(args []reflect.Value) []reflect.Value {
							if len(args) != 1 || !args[0].IsNil() {
								badArguments.Add(1)
							}
							callbacks.Add(1)
							done <- struct{}{}
							return nil
						}).Interface().(func(*int))
						go f(nil)
					} else {
						f := reflect.MakeFunc(reflect.TypeOf((func())(nil)), func(args []reflect.Value) []reflect.Value {
							if len(args) != 0 {
								badArguments.Add(1)
							}
							callbacks.Add(1)
							done <- struct{}{}
							return nil
						}).Interface().(func())
						go f()
					}
				}
				allocations.Add(1)
			}
			if makeFunc {
				for i := 0; i < iterations; i++ {
					<-done
				}
			}
			runtime.KeepAlive(live)
		}()
	}
	go func() { work.Wait(); stop.Store(true); close(workDone) }()

	// Observe from the collector itself: an allocating watchdog or a timer
	// goroutine can be starved by the same lock. Never sleep or yield here.
	var starved bool
	t.Logf("workers=%d iterations=%d GC_MARKERS=%q makefunc=%v", workers, iterations, os.Getenv("GC_MARKERS"), makeFunc)
	go func() {
		defer close(gcDone)
		runtime.GC()
		collections.Add(1)
		close(start)
		lastProgress := time.Now()
		var previous int64
		for !stop.Load() {
			runtime.GC()
			if collections.Add(1)%64 == 0 {
				progress := allocations.Load() + callbacks.Load()
				if progress != previous {
					previous = progress
					lastProgress = time.Now()
				} else if time.Since(lastProgress) >= 5*time.Second {
					starved = true
					break
				}
			}
		}
	}()
	<-gcDone
	// Stopping pressure after a failed observation is cleanup, never success.
	beforeAlloc, beforeCallbacks, beforeGC := allocations.Load(), callbacks.Load(), collections.Load()
	select {
	case <-workDone:
	case <-time.After(10 * time.Second):
		t.Fatal("allocation/MakeFunc workers did not stop")
	}
	t.Logf("under pressure: allocations=%d callbacks=%d collections=%d", beforeAlloc, beforeCallbacks, beforeGC)
	if starved {
		t.Fatal("allocation/MakeFunc progress stalled for at least 5s during continuous GC")
	}
	if got, want := allocations.Load(), int64(workers*iterations); got != want {
		t.Fatalf("allocations=%d, want %d", got, want)
	}
	if got := callbacks.Load(); makeFunc && got != int64(workers*iterations) {
		t.Fatalf("callbacks=%d, want %d", got, workers*iterations)
	}
	if got := badArguments.Load(); got != 0 {
		t.Fatalf("%d callbacks received incorrect arguments", got)
	}
}
