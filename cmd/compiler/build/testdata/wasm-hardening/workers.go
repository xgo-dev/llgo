//go:build llgo.wasm.workers

package main

import (
	"runtime"
	"strings"
	_ "unsafe"
)

//go:linkname schedulerProcID github.com/xgo-dev/llgo/runtime/internal/runtime.SchedulerProcID
func schedulerProcID() int

const workerPayloadValue = 0xabcdef01

func testWorkerCallers() {
	done := make(chan struct{})
	go func() {
		var pcs [8]uintptr
		n := runtime.Callers(0, pcs[:])
		if n == 0 {
			panic("worker runtime.Callers returned no frames")
		}
		frames := runtime.CallersFrames(pcs[:n])
		frame, _ := frames.Next()
		if !strings.Contains(frame.Function, "runtime.Callers") {
			panic("worker runtime.Callers returned the wrong first frame")
		}
		close(done)
	}()
	<-done
}

func testNestedWorkerSpawn() {
	done := make(chan struct{})
	go func() {
		inner := make(chan struct{})
		go func() {
			close(inner)
		}()
		<-inner
		close(done)
	}()
	<-done
}

func testWorkerCBoundary() {
	const attempts = 16
	for range attempts {
		proc := make(chan int)
		start := make(chan bool)
		done := make(chan uint64)
		go func() {
			proc <- schedulerProcID()
			if !<-start {
				done <- 0
				return
			}
			live := &payload{value: workerPayloadValue}
			value := holdPayloadInC(live)
			runtime.Gosched()
			done <- value
		}()
		if <-proc == 0 {
			start <- false
			<-done
			continue
		}
		start <- true
		for cHoldEntered() == 0 {
			runtime.Gosched()
		}
		runtime.GC()
		if value := <-done; value != workerPayloadValue {
			panic("C boundary lost a live Go pointer")
		}
		return
	}
	panic("C boundary test did not run on a secondary worker")
}
