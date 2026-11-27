//go:build js && wasm

package wasmtest

import (
	"runtime"
	"syscall/js"
	"testing"
	"time"
)

// Run with LLGo and official Go to enforce the same callback contract.
func TestJSCallbackNested(t *testing.T) {
	inner := js.FuncOf(func(this js.Value, args []js.Value) any {
		return args[0].Int() + 1
	})
	defer inner.Release()
	outer := js.FuncOf(func(this js.Value, args []js.Value) any {
		return inner.Invoke(args[0])
	})
	defer outer.Release()
	if got := outer.Invoke(41).Int(); got != 42 {
		t.Fatalf("callback result = %d", got)
	}
}

func TestJSCallbackExternalResult(t *testing.T) {
	obj := js.Global().Get("Object").New()
	callback := js.FuncOf(func(js.Value, []js.Value) any { return 42 })
	defer callback.Release()
	schedule := js.Global().Get("Function").New("cb", "obj", "setTimeout(() => { obj.result = cb(); }, 0)")
	schedule.Invoke(callback, obj)
	deadline := time.Now().Add(time.Second)
	for obj.Get("result").IsUndefined() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := obj.Get("result"); got.Type() != js.TypeNumber || got.Int() != 42 {
		t.Fatalf("external callback returned %v", got)
	}
}

func TestJSCallbackGoroutineSwitch(t *testing.T) {
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		ch := make(chan int)
		go func() { ch <- 42 }()
		return <-ch
	})
	defer callback.Release()
	obj := js.ValueOf(map[string]any{"calls": 0})
	invoke := js.Global().Get("Function").New("cb", "obj", "obj.calls++; const result = cb(); obj.after = result; return result")
	outer := js.FuncOf(func(js.Value, []js.Value) any {
		return invoke.Invoke(callback, obj)
	})
	defer outer.Release()
	for i := 1; i <= 20; i++ {
		if got := outer.Invoke(); got.Type() != js.TypeNumber || got.Int() != 42 {
			t.Fatalf("blocking nested callback returned %v", got)
		}
		if got := obj.Get("calls").Int(); got != i {
			t.Fatalf("JavaScript invocation replayed: calls = %d, want %d", got, i)
		}
		if got := obj.Get("after"); got.Type() != js.TypeNumber || got.Int() != 42 {
			t.Fatalf("JavaScript resumed before the callback completed: %v", got)
		}
		// Exercise the enclosing Fiber again after returning through both JS
		// boundaries: its continuation must no longer point to the callback.
		runtime.Gosched()
	}
}

func TestJSCallbackTimer(t *testing.T) {
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		time.Sleep(time.Millisecond)
		return 42
	})
	defer callback.Release()
	if got := callback.Invoke(); got.Type() != js.TypeNumber || got.Int() != 42 {
		t.Fatalf("sleeping callback returned %v", got)
	}
}

func TestJSCallbackDrainsRunnableWork(t *testing.T) {
	obj := js.ValueOf(map[string]any{"done": false})
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		go func() { obj.Set("done", true) }()
		return nil
	})
	defer callback.Release()
	invoke := js.Global().Get("Function").New("cb", "obj", "cb(); return obj.done")
	if !invoke.Invoke(callback, obj).Bool() {
		t.Fatal("callback returned to JS before runnable work completed")
	}
}

func TestJSCallbackPanic(t *testing.T) {
	// Recover inside the callback. Recovering across the JS boundary leaves
	// an incomplete runtime event even in the official Go implementation.
	callback := js.FuncOf(func(js.Value, []js.Value) (result any) {
		defer func() {
			if got := recover(); got != "callback-panic" {
				t.Errorf("callback panic = %v", got)
			}
			result = 42
		}()
		panic("callback-panic")
	})
	defer callback.Release()
	if got := callback.Invoke().Int(); got != 42 {
		t.Fatalf("recovered callback result = %d", got)
	}
	next := js.FuncOf(func(js.Value, []js.Value) any { return 42 })
	defer next.Release()
	if got := next.Invoke().Int(); got != 42 {
		t.Fatalf("callback after recovery = %d", got)
	}
	runtime.Gosched()
}

func TestJSCallbackReleaseWhileRunning(t *testing.T) {
	var callback js.Func
	callback = js.FuncOf(func(js.Value, []js.Value) any {
		callback.Release()
		// Releasing the last registered function must not remove the event
		// poll hook before this invocation has finished using the scheduler.
		ch := make(chan int)
		go func() { ch <- 42 }()
		return <-ch
	})
	if got := callback.Invoke().Int(); got != 42 {
		t.Fatalf("self-released callback result = %d", got)
	}
	next := js.FuncOf(func(js.Value, []js.Value) any { return 43 })
	defer next.Release()
	if got := next.Invoke().Int(); got != 43 {
		t.Fatalf("callback after re-registration = %d", got)
	}
}

func TestJSCallbackGCAndSuspension(t *testing.T) {
	callback := js.FuncOf(func(this js.Value, args []js.Value) any {
		data := make([]byte, 32<<10)
		data[0], data[len(data)-1] = 19, 23
		value := args[0]
		done := make(chan struct{})
		go func() {
			runtime.GC()
			close(done)
		}()
		<-done
		if !this.Equal(value) || value.Get("marker").Int() != 42 ||
			int(data[0])+int(data[len(data)-1]) != 42 {
			t.Fatal("callback roots did not survive GC and suspension")
		}
		return value
	})
	defer callback.Release()
	object := js.ValueOf(map[string]any{"marker": 42})
	object.Set("callback", callback)
	for i := 0; i < 10; i++ {
		if result := object.Call("callback", object); !result.Equal(object) {
			t.Fatal("callback result lost object identity")
		}
		runtime.Gosched()
	}
}
