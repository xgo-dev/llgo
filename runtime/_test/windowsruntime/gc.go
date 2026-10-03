//go:build !nogc

package main

import (
	"runtime"
	"time"
)

type windowsGCProbe struct {
	value int
}

//go:noinline
func makeWindowsGCProbe(finalized chan<- int) {
	probe := &windowsGCProbe{value: 42}
	runtime.SetFinalizer(probe, func(value *windowsGCProbe) {
		finalized <- value.value
	})
}

func checkThreadLifecycleGC() {
	const workers = 64
	done := make(chan *windowsGCProbe)
	// LLGo's native backend creates an OS thread for each goroutine. Alternate
	// normal returns and Goexit to exercise both CRT-aware thread exit paths.
	// Receiving done checks the deferred result, not completion of OS teardown.
	for worker := 0; worker < workers; worker++ {
		probe := &windowsGCProbe{value: worker}
		go func(probe *windowsGCProbe, useGoexit bool) {
			defer func() {
				if recover() != nil {
					panic("Windows GC thread lifecycle worker panicked")
				}
				done <- probe
			}()
			if useGoexit {
				runtime.Goexit()
				panic("Windows runtime.Goexit returned")
			}
		}(probe, worker%2 != 0)
		got := <-done
		if got == nil || got.value != worker {
			panic("Windows GC corrupted a short-lived worker root")
		}
		if worker%16 == 15 {
			runtime.GC()
		}
	}
}

func checkConcurrentGC() {
	const workers = 4
	ready := make(chan struct{}, workers)
	release := make(chan struct{})
	done := make(chan int, workers)
	finalized := make(chan int, workers)
	for worker := 0; worker < workers; worker++ {
		go func(id int) {
			probe := &windowsGCProbe{value: 40 + id}
			runtime.SetFinalizer(probe, func(value *windowsGCProbe) {
				finalized <- value.value
			})
			ready <- struct{}{}
			<-release
			value := probe.value
			if value != 40+id {
				panic("Windows GC corrupted a worker stack root")
			}
			runtime.KeepAlive(probe)
			done <- value
		}(worker)
	}
	for worker := 0; worker < workers; worker++ {
		<-ready
	}
	runtime.GC()
	select {
	case <-finalized:
		panic("Windows GC finalized a live worker stack root")
	default:
	}
	close(release)
	want := 40*workers + workers*(workers-1)/2
	got := 0
	for worker := 0; worker < workers; worker++ {
		got += <-done
	}
	if got != want {
		panic("Windows GC worker roots returned corrupt values")
	}
}

func checkGC() {
	checkThreadLifecycleGC()
	checkConcurrentGC()

	finalized := make(chan int, 1)
	created := make(chan struct{})
	go func() {
		makeWindowsGCProbe(finalized)
		close(created)
	}()
	<-created

	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	// Receiving created does not wait for the creator's OS thread to exit.
	// Its conservative stack roots can retain the probe until thread teardown
	// unregisters them, so allow that teardown to run between collections.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		runtime.GC()
		select {
		case value := <-finalized:
			if value != 42 {
				panic("Windows GC finalizer observed a corrupt object")
			}
			var after runtime.MemStats
			runtime.ReadMemStats(&after)
			if after.NumGC <= before.NumGC {
				panic("Windows runtime.GC did not advance MemStats.NumGC")
			}
			return
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
	panic("Windows GC did not run the finalizer")
}
