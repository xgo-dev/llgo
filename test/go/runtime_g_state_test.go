package gotest

import (
	"runtime"
	"testing"
)

type runtimeGStateResult struct {
	id        int
	recovered any
	order     string
}

func runtimeGStatePanic(id int, ready chan<- struct{}, start <-chan struct{}, results chan<- runtimeGStateResult) {
	order := ""
	defer func() {
		result := runtimeGStateResult{id: id, recovered: recover(), order: order + "r"}
		results <- result
	}()
	defer func() {
		order += "d"
	}()
	ready <- struct{}{}
	<-start
	order += "p"
	panic(id)
}

func TestRuntimeGStateIsolation(t *testing.T) {
	ready := make(chan struct{}, 2)
	start := make(chan struct{})
	results := make(chan runtimeGStateResult, 2)
	for id := 1; id <= 2; id++ {
		go runtimeGStatePanic(id, ready, start, results)
	}
	<-ready
	<-ready
	close(start)

	seen := [3]bool{}
	for i := 0; i < 2; i++ {
		result := <-results
		if result.id < 1 || result.id > 2 {
			t.Fatalf("unexpected goroutine id %d", result.id)
		}
		if seen[result.id] {
			t.Fatalf("duplicate result from goroutine %d", result.id)
		}
		seen[result.id] = true
		if result.recovered != result.id {
			t.Fatalf("goroutine %d recovered %v", result.id, result.recovered)
		}
		if result.order != "pdr" {
			t.Fatalf("goroutine %d defer order = %q, want %q", result.id, result.order, "pdr")
		}
	}

	goexitDone := make(chan any, 1)
	go func() {
		defer func() {
			goexitDone <- recover()
		}()
		runtime.Goexit()
		goexitDone <- "Goexit returned"
	}()
	if recovered := <-goexitDone; recovered != nil {
		t.Fatalf("recover during Goexit = %v, want nil", recovered)
	}

	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()
		panic("main panic")
	}()
	if recovered != "main panic" {
		t.Fatalf("main goroutine recovered %v, want %q", recovered, "main panic")
	}
}

func TestRuntimeNumGoroutineTracksWorkers(t *testing.T) {
	const workerCount = 8
	ready := make(chan struct{}, workerCount)
	release := make(chan struct{})
	done := make(chan struct{}, workerCount)
	for range workerCount {
		go func() {
			ready <- struct{}{}
			<-release
			done <- struct{}{}
		}()
	}
	for range workerCount {
		<-ready
	}
	if got := runtime.NumGoroutine(); got < workerCount+1 {
		t.Fatalf("NumGoroutine with %d blocked workers = %d, want at least %d", workerCount, got, workerCount+1)
	}

	close(release)
	for range workerCount {
		<-done
	}
}
