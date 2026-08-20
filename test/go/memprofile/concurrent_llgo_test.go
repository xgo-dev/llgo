//go:build llgo

package memprofile

import (
	"runtime"
	"strings"
	"sync"
	"testing"
)

var concurrentSink [][]*int

func TestConcurrentSamplingAndSnapshots(t *testing.T) {
	oldRate := runtime.MemProfileRate
	runtime.MemProfileRate = 1
	defer func() {
		runtime.MemProfileRate = oldRate
	}()

	const (
		workers     = 16
		allocations = 256
	)
	concurrentSink = make([][]*int, workers)
	for i := range concurrentSink {
		concurrentSink[i] = make([]*int, allocations)
	}
	before := memProfileObjectsForFunction(t, "concurrentProfileAlloc")

	start := make(chan struct{})
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		<-start
		for i := 0; i < 256; i++ {
			runtime.MemProfile(nil, false)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range concurrentSink {
		go func(dst []*int) {
			defer wg.Done()
			<-start
			concurrentProfileAlloc(dst)
		}(concurrentSink[i])
	}
	close(start)
	wg.Wait()
	<-readerDone

	after := memProfileObjectsForFunction(t, "concurrentProfileAlloc")
	if got, want := after-before, int64(workers*allocations); got != want {
		t.Fatalf("concurrent sampled allocations = %d, want %d", got, want)
	}
}

func concurrentProfileAlloc(dst []*int) {
	for i := range dst {
		p := new(int)
		*p = i
		dst[i] = p
	}
}

func memProfileObjectsForFunction(t *testing.T, suffix string) int64 {
	t.Helper()
	var total int64
	for _, record := range readMemProfile(t) {
		frames := runtime.CallersFrames(record.Stack())
		for {
			frame, more := frames.Next()
			if strings.HasSuffix(frame.Function, "."+suffix) {
				total += record.AllocObjects
				break
			}
			if !more {
				break
			}
		}
	}
	return total
}
