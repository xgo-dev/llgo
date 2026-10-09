//go:build llgo && js && wasm && llgo.wasm.workers

package runtime

type wasmJSCall struct {
	fn       func()
	done     chan struct{}
	failure  any
	returned bool
}

func HasMainJSWorkerExecutorForTesting() bool {
	gp := getg()
	return gp != nil && gp.context.platform.jsCalls != nil
}

// CallMainJSWorker runs fn on worker zero when the caller is on another
// worker. It parks only the calling G, leaving both worker schedulers free
// to process nested calls, callbacks, and GC. False means fn was not called.
// Each calling G has its own executor so a JS callback can wait for another
// G that calls back into this realm. Reusing its fiber avoids allocating a
// native and Asyncify stack for every property access.
func CallMainJSWorker(fn func()) bool {
	worker := currentWasmWorker()
	if worker == nil || worker.index == 0 {
		return false
	}
	gp := getg()
	if gp.context.platform.jsCalls == nil {
		calls := make(chan *wasmJSCall)
		gp.context.platform.jsCalls = calls
		spawnMainJSCall(func() {
			for call := range calls {
				runMainJSCall(call)
			}
		})
	}
	call := &wasmJSCall{fn: fn, done: make(chan struct{})}
	gp.context.platform.jsCalls <- call
	<-call.done
	if call.failure != nil {
		panic(call.failure)
	}
	if !call.returned {
		// Goexit also terminates the executor. Detach it before unwinding the
		// caller: a caller's defer may use JS and needs a fresh executor.
		calls := gp.context.platform.jsCalls
		gp.context.platform.jsCalls = nil
		close(calls)
		Goexit()
	}
	return true
}

func runMainJSCall(call *wasmJSCall) {
	defer func() {
		call.failure = recover()
		call.fn = nil
		close(call.done)
	}()
	call.fn()
	call.returned = true
}

func spawnMainJSCall(fn func()) {
	gp := getg()
	previous := gp.context.platform.spawnWorker
	gp.context.platform.spawnWorker = &wasmMultiSched.workers[0]
	defer func() { gp.context.platform.spawnWorker = previous }()
	go fn()
}
