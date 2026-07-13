package gotest

import (
	"runtime"
	"testing"
)

type goroutineStateResult struct {
	id        int
	recovered any
}

func TestGoroutineRuntimeStateIsolation(t *testing.T) {
	const workers = 16
	start := make(chan struct{})
	results := make(chan goroutineStateResult, workers)
	for id := 0; id < workers; id++ {
		go func(id int) {
			defer func() {
				results <- goroutineStateResult{id: id, recovered: recover()}
			}()
			<-start
			panic(id)
		}(id)
	}
	close(start)

	seen := make([]bool, workers)
	for i := 0; i < workers; i++ {
		result := <-results
		if result.id < 0 || result.id >= workers {
			t.Fatalf("result id = %d, want [0, %d)", result.id, workers)
		}
		if seen[result.id] {
			t.Fatalf("duplicate result for goroutine %d", result.id)
		}
		seen[result.id] = true
		if result.recovered != result.id {
			t.Fatalf("goroutine %d recovered %v, want %d", result.id, result.recovered, result.id)
		}
	}

	exited := make(chan any, 1)
	go func() {
		defer func() {
			exited <- recover()
		}()
		runtime.Goexit()
	}()
	if recovered := <-exited; recovered != nil {
		t.Fatalf("recover during Goexit = %v, want nil", recovered)
	}

	const mainPanic = "main panic"
	if recovered := recoverGoroutineStatePanic(mainPanic); recovered != mainPanic {
		t.Fatalf("main goroutine recovered %v, want %q", recovered, mainPanic)
	}
}

func recoverGoroutineStatePanic(value any) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	panic(value)
}
