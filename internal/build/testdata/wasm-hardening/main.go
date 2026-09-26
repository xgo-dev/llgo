package main

import (
	"os"
	"runtime"
	"strconv"
	"time"
)

type payload struct {
	value uint64
}

type indirectRunner interface {
	run(chan<- struct{}, <-chan struct{}) *payload
}

type indirectState struct {
	value *payload
}

func (state *indirectState) run(ready chan<- struct{}, resume <-chan struct{}) (result *payload) {
	defer func() {
		if recovered := recover(); recovered != "indirect panic" {
			panic("unexpected indirect recover value")
		}
		result = state.value
	}()
	ready <- struct{}{}
	<-resume
	panic("indirect panic")
}

func main() {
	testProcessArgs()
	testWorkerCallers()
	testNestedWorkerSpawn()
	testIndirectSuspension()
	testSchedulerHandoffs()
	testBlockedGRoots(blockedGCount())
	testWorkerCBoundary()
	testCanceledExitTimer()
	println("wasm hardening ok")
}

func testProcessArgs() {
	expected := os.Getenv("LLGO_WASM_EXPECT_ARG")
	if expected == "" {
		return
	}
	if len(os.Args) < 2 || os.Args[len(os.Args)-1] != expected {
		panic("process arguments were not preserved")
	}
}

func testIndirectSuspension() {
	state := &indirectState{value: &payload{value: 0x12345678}}
	var runner indirectRunner = state
	call := runner.run
	ready := make(chan struct{})
	resume := make(chan struct{})
	done := make(chan *payload)
	go func() {
		done <- call(ready, resume)
	}()
	<-ready
	runtime.GC()
	close(resume)
	if got := <-done; got == nil || got.value != 0x12345678 {
		panic("indirect suspended root was not retained")
	}
}

func testSchedulerHandoffs() {
	const count = 10_000
	ping := make(chan int)
	pong := make(chan int)
	done := make(chan struct{})
	go func() {
		for i := range count {
			if value := <-ping; value != i {
				panic("scheduler ping lost ordering")
			}
			pong <- i
		}
		close(done)
	}()
	for i := range count {
		ping <- i
		if value := <-pong; value != i {
			panic("scheduler pong lost ordering")
		}
	}
	<-done
}

func blockedGCount() int {
	const defaultCount = 100
	value := os.Getenv("LLGO_WASM_BLOCKED_G")
	if value == "" {
		return defaultCount
	}
	count, err := strconv.Atoi(value)
	if err != nil || count < 1 || count > 10_000 {
		panic("invalid LLGO_WASM_BLOCKED_G")
	}
	return count
}

func testBlockedGRoots(count int) {
	ready := make(chan struct{}, count)
	release := make(chan struct{})
	done := make(chan uint64, count)
	for i := range count {
		go func() {
			live := &payload{value: uint64(i + 1)}
			ready <- struct{}{}
			<-release
			done <- live.value
		}()
	}
	for range count {
		<-ready
	}
	runtime.GC()
	close(release)
	var sum uint64
	for range count {
		sum += <-done
	}
	want := uint64(count) * uint64(count+1) / 2
	if sum != want {
		panic("blocked goroutine roots were not retained")
	}
}

func testCanceledExitTimer() {
	timer := time.AfterFunc(time.Hour, func() {
		panic("canceled exit timer fired")
	})
	if !timer.Stop() {
		panic("failed to cancel exit timer")
	}
}
