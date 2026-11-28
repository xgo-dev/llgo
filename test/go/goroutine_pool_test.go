package gotest

import (
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func countStackGoroutines(stacks string) int {
	n := 0
	for _, line := range strings.Split(stacks, "\n") {
		if strings.HasPrefix(line, "goroutine ") {
			n++
		}
	}
	return n
}

func TestGoroutinePoolStackAndNumGoroutine(t *testing.T) {
	base := runtime.NumGoroutine()
	buf := make([]byte, 64<<10)
	baseHeaders := countStackGoroutines(string(buf[:runtime.Stack(buf, true)]))

	const waves = 10
	const perWave = 12
	for w := 0; w < waves; w++ {
		var wg sync.WaitGroup
		wg.Add(perWave)
		for i := 0; i < perWave; i++ {
			mode := i % 3
			go func() {
				defer wg.Done()
				switch mode {
				case 0:
					// Normal return (eligible for thread pool reuse).
				case 1:
					// Recovered panic followed by normal return.
					defer func() { _ = recover() }()
					panic("pool-recover")
				case 2:
					// Explicit runtime.Goexit exits via mexit cleanly.
					runtime.Goexit()
				}
			}()
		}
		wg.Wait()
	}

	// Wait briefly for finished workers to release their goroutine count and
	// detach their traceback nodes before parking in the idle pool.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= base {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if got := runtime.NumGoroutine(); got > base {
		t.Fatalf("NumGoroutine() = %d after pooled workers finished, want <= %d", got, base)
	}

	n := runtime.Stack(buf, true)
	stacks := string(buf[:n])
	headers := countStackGoroutines(stacks)
	if headers > baseHeaders {
		t.Fatalf("runtime.Stack reported %d goroutines (baseline %d); idle pool threads leaked into traceback:\n%s", headers, baseHeaders, stacks)
	}
}
