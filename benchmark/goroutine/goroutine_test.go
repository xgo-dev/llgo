/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package goroutine

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

var benchmarkSink atomic.Int64

func benchmarkTask() {
	benchmarkSink.Add(1)
}

func runGoroutineBatch(count int) {
	var wg sync.WaitGroup
	wg.Add(count)
	for range count {
		go func() {
			benchmarkTask()
			wg.Done()
		}()
	}
	wg.Wait()
}

type reusableWorker struct {
	batches chan int
	done    chan struct{}
	stop    chan struct{}
	stopped chan struct{}
}

func newReusableWorker() *reusableWorker {
	w := &reusableWorker{
		batches: make(chan int),
		done:    make(chan struct{}),
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go func() {
		defer close(w.stopped)
		for {
			select {
			case count := <-w.batches:
				for range count {
					benchmarkTask()
				}
				w.done <- struct{}{}
			case <-w.stop:
				return
			}
		}
	}()
	return w
}

func (w *reusableWorker) run(count int) {
	w.batches <- count
	<-w.done
}

func (w *reusableWorker) close() {
	close(w.stop)
	<-w.stopped
}

func TestReusableWorkerBatches(t *testing.T) {
	worker := newReusableWorker()
	t.Cleanup(worker.close)
	for _, count := range []int{0, 1, 16, 128, 1024, 1, 0} {
		before := benchmarkSink.Load()
		worker.run(count)
		if got := benchmarkSink.Load() - before; got != int64(count) {
			t.Fatalf("batch of %d tasks completed %d tasks", count, got)
		}
	}
}

func BenchmarkGoroutineLifecycle(b *testing.B) {
	for _, count := range []int{1, 16, 128, 1024} {
		b.Run("create-and-wait/goroutines="+strconv.Itoa(count)+"/"+goroutineImplementation, func(b *testing.B) {
			runGoroutineBatch(count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				runGoroutineBatch(count)
			}
		})

		b.Run("serial-dispatch/tasks="+strconv.Itoa(count)+"/"+goroutineImplementation, func(b *testing.B) {
			worker := newReusableWorker()
			b.Cleanup(worker.close)
			worker.run(count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				worker.run(count)
			}
		})
	}
}
