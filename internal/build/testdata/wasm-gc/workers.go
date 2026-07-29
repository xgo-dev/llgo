//go:build llgo.wasm_workers

package main

import (
	"runtime"
	"sync/atomic"
	_ "unsafe"
)

//go:linkname schedulerProcID github.com/goplus/llgo/runtime/internal/runtime.SchedulerProcID
func schedulerProcID() int

func testMultiWorkerGC() {
	testRemoteWorkerGC()
	testConcurrentWorkerAllocation()
}

func testRemoteWorkerGC() {
	const want = uint64(0x76543210)
	live := &payload{value: want}

	for {
		start := make(chan bool)
		ready := make(chan int)
		var (
			done   atomic.Bool
			result atomic.Uint64
		)
		go func() {
			ready <- schedulerProcID()
			if !<-start {
				done.Store(true)
				return
			}
			remoteLive := &payload{value: want}
			for range 8 {
				runtime.GC()
				for i := range 512 {
					garbage = &payload{value: uint64(i)}
				}
				garbage = nil
			}
			result.Store(remoteLive.value)
			done.Store(true)
		}()

		if proc := <-ready; proc == 0 {
			start <- false
			for !done.Load() {
			}
			continue
		}
		start <- true
		for !done.Load() {
			if live.value != want {
				panic("remote GC lost an active worker root")
			}
		}
		if result.Load() != want || live.value != want {
			panic("remote worker GC lost a live root")
		}
		return
	}
}

func testConcurrentWorkerAllocation() {
	const (
		goroutines = 4
		iterations = 4096
		liveCount  = 32
	)
	start := make(chan struct{})
	done := make(chan int, goroutines)
	for id := range goroutines {
		go func() {
			<-start
			var live [liveCount]*payload
			for i := range iterations {
				slot := i % len(live)
				live[slot] = &payload{value: uint64(id*iterations + i + 1)}
				if i%1024 == 0 {
					runtime.GC()
				}
			}
			for _, value := range live {
				if value == nil || value.value == 0 {
					panic("concurrent allocation lost a live object")
				}
			}
			done <- schedulerProcID()
		}()
	}
	close(start)

	workers := make(map[int]bool)
	for range goroutines {
		workers[<-done] = true
	}
	if len(workers) < 2 {
		panic("concurrent GC did not cover multiple workers")
	}
}
