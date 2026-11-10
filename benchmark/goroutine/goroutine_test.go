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

type workerBatch struct {
	count int
	done  chan struct{}
}

type reusableWorker struct {
	batches chan workerBatch
	stop    chan struct{}
	stopped chan struct{}
}

func newReusableWorker() *reusableWorker {
	w := &reusableWorker{
		batches: make(chan workerBatch),
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go func() {
		defer close(w.stopped)
		for {
			select {
			case batch := <-w.batches:
				for range batch.count {
					benchmarkTask()
				}
				close(batch.done)
			case <-w.stop:
				return
			}
		}
	}()
	return w
}

func (w *reusableWorker) run(count int) {
	done := make(chan struct{})
	w.batches <- workerBatch{count: count, done: done}
	<-done
}

func (w *reusableWorker) close() {
	close(w.stop)
	<-w.stopped
}

func BenchmarkGoroutineLifecycle(b *testing.B) {
	for _, count := range []int{1, 16, 128, 1024} {
		count := count
		b.Run("current/newproc/"+strconv.Itoa(count)+"/"+goroutineImplementation, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
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
		})

		b.Run("reused-worker/dispatch/"+strconv.Itoa(count)+"/"+goroutineImplementation, func(b *testing.B) {
			worker := newReusableWorker()
			b.Cleanup(worker.close)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				worker.run(count)
			}
		})
	}
}
