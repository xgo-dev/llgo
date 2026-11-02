package runtime_test

import (
	"runtime"
	"sync"
	"testing"
)

func TestGOMAXPROCSSetting(t *testing.T) {
	if runtime.GOARCH == "wasm" && runtime.Compiler != "llgo" {
		t.Skip("the Go wasm runtime restricts GOMAXPROCS to one")
	}
	original := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(original)
	cpus := runtime.NumCPU()
	previous := original
	for _, n := range []int{1, 2, 4, 1, original} {
		if got := runtime.GOMAXPROCS(n); got != previous {
			t.Fatalf("GOMAXPROCS(%d) returned %d, want previous setting %d", n, got, previous)
		}
		for _, query := range []int{0, -1, -100} {
			if got := runtime.GOMAXPROCS(query); got != n {
				t.Fatalf("GOMAXPROCS(%d) = %d, want %d", query, got, n)
			}
		}
		if got := runtime.NumCPU(); got != cpus {
			t.Fatalf("GOMAXPROCS changed NumCPU from %d to %d", cpus, got)
		}
		previous = n
	}
}

func TestGOMAXPROCSConcurrentUpdates(t *testing.T) {
	if runtime.GOARCH == "wasm" && runtime.Compiler != "llgo" {
		t.Skip("the Go wasm runtime restricts GOMAXPROCS to one")
	}
	const workers = 16
	original := runtime.GOMAXPROCS(workers + 1)
	defer runtime.GOMAXPROCS(original)
	var previous [workers]int
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range previous {
		wg.Go(func() {
			<-start
			previous[i] = runtime.GOMAXPROCS(i + 1)
		})
	}
	close(start)
	wg.Wait()
	// Go permits concurrent setters to return the same previous setting.
	// Every observed setting must still come from a caller, without a torn or
	// transient zero value, and normal updates must work after they finish.
	for _, n := range append(previous[:], runtime.GOMAXPROCS(0)) {
		if n < 1 || n > workers+1 {
			t.Fatalf("invalid concurrent setting: previous=%v, final=%d", previous, runtime.GOMAXPROCS(0))
		}
	}
	runtime.GOMAXPROCS(1)
	if got := runtime.GOMAXPROCS(0); got != 1 {
		t.Fatalf("setting after concurrent updates = %d, want 1", got)
	}
}
