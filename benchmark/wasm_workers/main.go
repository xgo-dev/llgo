package main

import (
	"sync"
	"time"
)

const (
	lifecycleIterations = 10_000
	channelIterations   = 100_000
	cpuIterations       = 20_000_000
)

var cpuResult uint64

func main() {
	report("WasmGoroutineLifecycle", lifecycleIterations, benchmarkLifecycle())
	report("WasmChannelRoundTrip", channelIterations, benchmarkChannel())
	cpuResult = cpuWork(0)
	report("WasmOneCPUJob", 1, benchmarkCPUJobs(1))
	report("WasmTwoCPUJobs", 2, benchmarkCPUJobs(2))
	if cpuResult == 0 {
		panic("unexpected CPU benchmark result")
	}
}

func benchmarkLifecycle() time.Duration {
	done := make(chan struct{}, 1)
	start := time.Now()
	for range lifecycleIterations {
		go func() {
			done <- struct{}{}
		}()
		<-done
	}
	return time.Since(start)
}

func benchmarkChannel() time.Duration {
	request := make(chan struct{})
	response := make(chan struct{})
	go func() {
		for range channelIterations {
			<-request
			response <- struct{}{}
		}
	}()

	start := time.Now()
	for range channelIterations {
		request <- struct{}{}
		<-response
	}
	return time.Since(start)
}

func benchmarkCPUJobs(jobs int) time.Duration {
	var wg sync.WaitGroup
	results := make([]uint64, jobs)
	wg.Add(jobs)
	start := time.Now()
	for i := range results {
		go func() {
			results[i] = cpuWork(uint64(i + 1))
			wg.Done()
		}()
	}
	wg.Wait()
	cpuResult = 0
	for _, result := range results {
		cpuResult ^= result
	}
	return time.Since(start)
}

//go:noinline
func cpuWork(value uint64) uint64 {
	for i := uint64(0); i < cpuIterations; i++ {
		value = value*1664525 + 1013904223 + i
	}
	return value
}

func report(name string, iterations int, elapsed time.Duration) {
	println("Benchmark"+name+"-1", iterations, elapsed.Nanoseconds()/int64(iterations), "ns/op")
}
