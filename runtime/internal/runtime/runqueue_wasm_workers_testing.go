//go:build llgo && js && wasm && llgo.wasm.workers

package runtime

import (
	"github.com/xgo-dev/llgo/runtime/internal/pollbudget"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
	"github.com/xgo-dev/llgo/runtime/internal/wasmworkers"
)

// ExerciseWasmRunqueueTransitionForTesting holds this worker's queue from a
// different worker while Gosched transitions the caller to runnable. An
// exhausted budget must not reenter Gosched, and a concurrent GC must stop
// the caller in place until it has actually been inserted into the queue.
func ExerciseWasmRunqueueTransitionForTesting(collect func()) {
	gp := getg()
	worker := currentWasmWorker()
	var held uint32
	previous := gp.context.platform.spawnWorker
	gp.context.platform.spawnWorker = worker
	go func() {}() // leave runnable work to trigger the preemptive slow path
	gp.context.platform.spawnWorker = &wasmMultiSched.workers[1]
	go func() {
		worker.lock.Lock(nil)
		atomic.Store(&held, uint32(1))
		wasmworkers.Wake(&held)
		// Give the caller time to reach its contended enqueue before GC.
		wasmworkers.Wait(&held, 1, 50_000_000)
		collect()
		worker.lock.Unlock()
	}()
	gp.context.platform.spawnWorker = previous
	for atomic.Load(&held) == 0 {
		wasmworkers.Wait(&held, 0, 1_000_000)
	}
	budget := worker.safepointBudget
	worker.safepointBudget = pollbudget.New(1)
	goschedBackend()
	worker.safepointBudget = budget
}
