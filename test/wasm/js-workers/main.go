//go:build js && wasm && llgo.wasm.workers

package main

import (
	"math"
	"runtime"
	"syscall/js"
	"time"
	_ "unsafe"

	"github.com/xgo-dev/llgo/runtime/wasmworkers"
)

//go:linkname workerID github.com/xgo-dev/llgo/runtime/internal/runtime.SchedulerProcID
func workerID() int

//go:linkname callMainJSWorker github.com/xgo-dev/llgo/runtime/internal/runtime.CallMainJSWorker
func callMainJSWorker(func()) bool

//go:linkname hasMainJSWorkerExecutor github.com/xgo-dev/llgo/runtime/internal/runtime.HasMainJSWorkerExecutorForTesting
func hasMainJSWorkerExecutor() bool

//go:linkname queueTransition github.com/xgo-dev/llgo/runtime/internal/runtime.ExerciseWasmRunqueueTransitionForTesting
func queueTransition(func())

func main() {
	testLocalAllocations()
	testPortablePrimitives()
	println("js workers: queue transitions")
	queueTransition(func() {})
	queueTransition(runtime.GC)
	println("js workers: finalizers")
	testFinalizerContention()
	println("js workers: shared values")
	testSharedValues()
	println("js workers: callbacks")
	testCallbacks()
	println("js workers: panic and Goexit")
	testPanicAndGoexit()
	testExecutorLifetime()
	println("wasm js workers ok")
}

func testFinalizerContention() {
	done := make(chan int, 8)
	for range 8 {
		wasmworkers.GoIndependent(func() {
			for i := range 32 {
				object := new([64]byte)
				finalizer := func(*[64]byte) { panic("canceled finalizer ran") }
				runtime.SetFinalizer(object, finalizer)
				// Replacing and canceling an entry take the collector lock while
				// holding the registry lock. Other workers must still reach STW.
				runtime.SetFinalizer(object, finalizer)
				if i%4 == 0 {
					runtime.GC()
				}
				runtime.SetFinalizer(object, nil)
				runtime.KeepAlive(object)
			}
			done <- 1
		})
	}
	for range 8 {
		await(done)
	}
	runtime.GC()
}

func testLocalAllocations() {
	js.Global()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range 1000 {
		if js.Undefined().Type() != js.TypeUndefined || js.Global().IsUndefined() {
			panic("local JavaScript primitives changed")
		}
	}
	runtime.ReadMemStats(&after)
	if after.TotalAlloc != before.TotalAlloc {
		panic("local JavaScript calls allocated forwarding state")
	}
}

func testPortablePrimitives() {
	// Portable conversions must not create a worker-zero executor. Inspect
	// this G's executor directly: background timer/finalizer Gs can start here.
	time.Sleep(time.Millisecond)
	done := make(chan int)
	owners := make(map[int]bool)
	for range 8 {
		waitForGoroutines(2)
		wasmworkers.GoIndependent(func() {
			inputs := [6]any{nil, true, false, js.Undefined(), js.Null(), js.Func{Value: js.Null()}}
			types := [6]js.Type{js.TypeNull, js.TypeBoolean, js.TypeBoolean, js.TypeUndefined, js.TypeNull, js.TypeNull}
			if hasMainJSWorkerExecutor() {
				panic("new caller already has a JavaScript executor")
			}
			for i := range 1000 {
				index := i % len(inputs)
				if js.ValueOf(inputs[index]).Type() != types[index] || (js.Value{}).Type() != js.TypeUndefined {
					panic("portable JavaScript primitive changed")
				}
			}
			if hasMainJSWorkerExecutor() {
				panic("portable JavaScript primitives created an executor")
			}
			done <- workerID()
		})
		owners[<-done] = true
	}
	if len(owners) < 2 {
		panic("portable primitives did not exercise remote workers")
	}
}

func testSharedValues() {
	global := js.Global()
	object := js.ValueOf(map[string]any{"value": 42})
	identity := js.Global().Get("Function").New("value", "return value")
	arrayConstructor := js.Global().Get("Uint8Array")
	progress := make(chan int, 8)
	for range 8 {
		wasmworkers.GoIndependent(func() {
			origin := workerID()
			if !js.Global().Equal(global) {
				panic("Go workers use different JavaScript globals")
			}
			for i := range 32 {
				alias := identity.Invoke(object)
				if !alias.Equal(object) || alias.Get("value").Int() != 42 {
					panic("cross-worker call lost object identity")
				}
				private := js.ValueOf(map[string]any{"index": i, "object": alias})
				if i%4 == 0 {
					runtime.GC()
				}
				if !private.Get("object").Equal(object) || private.Get("index").Int() != i {
					panic("worker-private JavaScript value lost its GC root")
				}
				private.Set("text", "worker")
				if private.Get("text").String() != "worker" {
					panic("cross-worker property assignment failed")
				}
				private.Delete("text")
				if !private.Get("text").IsUndefined() {
					panic("cross-worker property deletion failed")
				}
				bytes := arrayConstructor.New(3)
				if !bytes.InstanceOf(arrayConstructor) || bytes.Length() != 3 {
					panic("cross-worker constructor changed its realm")
				}
				if js.CopyBytesToJS(bytes, []byte{1, 2, 3}) != 3 {
					panic("cross-worker byte upload failed")
				}
				bytes.SetIndex(1, 42)
				var copied [3]byte
				if js.CopyBytesToGo(copied[:], bytes) != 3 || copied != [3]byte{1, 42, 3} {
					panic("cross-worker byte download failed")
				}
				if !js.ValueOf(math.NaN()).IsNaN() || js.ValueOf(math.NaN()).Truthy() ||
					js.ValueOf(0).Truthy() || !js.ValueOf(true).Bool() || !js.ValueOf(nil).IsNull() {
					panic("cross-worker primitive semantics changed")
				}
				if workerID() != origin {
					panic("JavaScript proxy migrated the caller's C stack")
				}
				progress <- origin
			}
		})
	}
	owners := make(map[int]bool)
	// Each round reports progress; its JS operations and private GC roots must
	// complete within the deadline even when the full concurrent batch is slow.
	for range 8 * 32 {
		owners[await(progress)] = true
	}
	if len(owners) < 2 {
		panic("shared values were not used by distinct Go workers")
	}
}

func testCallbacks() {
	println("js workers: yielding callbacks")
	testCallbackYield()
	println("js workers: synchronous and external callbacks")
	progress := make(chan int, 8*3)
	gcReady, gcRelease := make(chan int, 8), make(chan struct{})
	object := js.ValueOf(map[string]any{"value": 42})
	callback := js.FuncOf(func(this js.Value, args []js.Value) any {
		if workerID() != 0 || !this.Equal(object) || !args[0].Equal(object) {
			panic("JavaScript callback lost its realm or arguments")
		}
		result := make(chan int, 1)
		wasmworkers.GoIndependent(func() {
			gcReady <- 1
			<-gcRelease
			value := object.Get("value").Int()
			progress <- 1
			result <- value
		})
		return await(result) + args[1].Int()
	})
	defer callback.Release()
	invoke := js.Global().Get("Function").New("cb", "object", "return cb.call(object, object, 7)")
	for range 8 {
		wasmworkers.GoIndependent(func() {
			if invoke.Invoke(callback, object).Int() != 49 {
				panic("nested callback did not return synchronously")
			}
			progress <- 1
			event := make(chan int, 1)
			cb := js.FuncOf(func(js.Value, []js.Value) any {
				event <- workerID()
				return nil
			})
			js.Global().Call("setTimeout", cb, 0)
			if await(event) != 0 {
				panic("external callback ran outside the shared realm")
			}
			cb.Release()
			progress <- 1
		})
	}
	// Collect while all eight synchronous callbacks and their child Gs are
	// parked, so every callback's JS roots are tested by the same collection.
	// Eight serial collections would charge their cumulative wall time to
	// each callback's otherwise independent 30-second result deadline.
	for range 8 {
		await(gcReady)
	}
	runtime.GC()
	close(gcRelease)
	// Bound each completed child JS call, synchronous return and external
	// event, rather than charging all concurrent GC/callback work to one timer.
	for range 8 * 3 {
		await(progress)
	}
}

func testCallbackYield() {
	wake := make(chan struct{})
	ready := make(chan struct{})
	// This G predates the callback and does not inherit its callback scope.
	go func() { <-wake; close(ready) }()
	unrelated := js.FuncOf(func(js.Value, []js.Value) any {
		close(wake)
		for {
			select {
			case <-ready:
				return true
			default:
				runtime.Gosched()
			}
		}
	})
	if !unrelated.Invoke().Bool() {
		panic("yielding callback starved unrelated Go work")
	}
	unrelated.Release()

	inner := js.FuncOf(func(_ js.Value, args []js.Value) any {
		// Register and release a callback while already inside a native JS
		// callback stack, exercising both registry locks during nested work.
		nested := js.FuncOf(func(_ js.Value, args []js.Value) any { return args[0] })
		defer nested.Release()
		runtime.Gosched()
		return nested.Invoke(args[0]).Int()
	})
	defer inner.Release()
	callback := js.FuncOf(func(_ js.Value, args []js.Value) any {
		// An inner callback must not overwrite the outer handler's yield state.
		value := inner.Invoke(args[0]).Int()
		runtime.Gosched()
		ready := make(chan struct{})
		go func() { close(ready) }()
		for {
			select {
			case <-ready:
				return value
			default:
				runtime.Gosched()
			}
		}
	})
	defer callback.Release()
	done := make(chan int, 128)
	for i := range 128 {
		wasmworkers.GoIndependent(func() {
			if callback.Invoke(i).Int() != i {
				panic("yielding callback lost its synchronous result")
			}
			done <- i
		})
	}
	for range 128 {
		await(done)
	}
}

func testPanicAndGoexit() {
	expected := js.ValueOf(map[string]any{"message": "cross-worker exception"})
	throw := js.Global().Get("Function").New("error", "throw error")
	done := make(chan int, 8)
	for range 8 {
		wasmworkers.GoIndependent(func() {
			defer func() {
				if expected.Get("message").String() != "cross-worker exception" {
					panic("Goexit defer could not use JavaScript")
				}
				done <- workerID()
			}()
			func() {
				defer func() {
					failure, ok := recover().(js.Error)
					if !ok || !failure.Value.Equal(expected) {
						panic("JavaScript exception lost identity crossing workers")
					}
				}()
				throw.Invoke(expected)
				panic("JavaScript throw returned")
			}()
			if expected.Get("message").String() != "cross-worker exception" {
				panic("JS executor did not survive a recovered panic")
			}
			func() {
				defer func() {
					if _, ok := recover().(*runtime.PanicNilError); !ok {
						panic("forwarded panic(nil) became Goexit")
					}
				}()
				if !callMainJSWorker(func() { panic(nil) }) {
					panic(nil)
				}
			}()
			if !callMainJSWorker(runtime.Goexit) {
				runtime.Goexit()
			}
			panic("proxy resumed a goroutine after Goexit")
		})
	}
	owners := make(map[int]bool)
	for range 8 {
		owners[await(done)] = true
	}
	if len(owners) < 2 {
		panic("Goexit did not exercise a remote worker")
	}
}

func testExecutorLifetime() {
	// Earlier callback/executor Gs may still be retiring after their last
	// channel send. Start with only main and the runtime timer G remaining.
	waitForGoroutines(2)
	baseline := runtime.NumGoroutine()
	done := make(chan int, 32)
	for range 32 {
		wasmworkers.GoIndependent(func() {
			if js.Global().Get("Object").Type() != js.TypeFunction {
				panic("short-lived caller could not use JavaScript")
			}
			done <- 1
		})
	}
	for range 32 {
		await(done)
	}
	waitForGoroutines(baseline)
}

func waitForGoroutines(limit int) {
	deadline := time.Now().Add(30 * time.Second)
	for runtime.NumGoroutine() > limit {
		if time.Now().After(deadline) {
			panic("JavaScript executor leaked after its caller exited")
		}
		runtime.Gosched()
	}
}

func await(ch <-chan int) int {
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case value := <-ch:
		return value
	case <-timer.C:
		panic("cross-worker JavaScript operation timed out")
	}
}
