//go:build js && wasm

package wasmtest

import (
	"os"
	"reflect"
	"syscall/js"
	"testing"
	"time"
)

func TestReflectCallCanSleep(t *testing.T) {
	delayed := func() {
		time.Sleep(time.Millisecond)
		println("woke")
	}
	reflect.ValueOf(delayed).Call(nil)
}

func TestReflectCallComplex128(t *testing.T) {
	add := func(v complex128) complex128 {
		return v + complex(1, 2)
	}
	got := reflect.ValueOf(add).Call([]reflect.Value{reflect.ValueOf(complex(3, 4))})
	if len(got) != 1 || got[0].Complex() != complex(4, 6) {
		t.Fatalf("Call(add) = %v, want [(4+6i)]", got)
	}
}

func TestReflectMakeFuncComplex64(t *testing.T) {
	fn := reflect.MakeFunc(reflect.TypeOf((func(complex64) complex64)(nil)), func(args []reflect.Value) []reflect.Value {
		v := args[0].Complex()
		return []reflect.Value{reflect.ValueOf(complex64(v + complex(1, 2)))}
	})
	got := fn.Call([]reflect.Value{reflect.ValueOf(complex64(complex(3, 4)))})
	if len(got) != 1 || got[0].Complex() != complex(4, 6) {
		t.Fatalf("MakeFunc(complex64) = %v, want [(4+6i)]", got)
	}
}

func TestJSCallMapsThrownError(t *testing.T) {
	fs := js.Global().Get("fs")
	statSync := fs.Get("statSync")
	if statSync.IsUndefined() {
		t.Fatal("fs.statSync is required to test thrown JS exception mapping")
	}

	defer func() {
		got := recover()
		if got == nil {
			t.Fatal("js.Value.Call did not panic")
		}
		jsErr, ok := got.(js.Error)
		if !ok {
			t.Fatalf("panic = %T %v, want js.Error", got, got)
		}
		if code := jsErr.Get("code").String(); code != "ENOENT" {
			t.Fatalf("js.Error code = %q, want ENOENT", code)
		}
	}()
	fs.Call("statSync", "/llgo-pr2539-definitely-does-not-exist")
	t.Fatal("js.Value.Call returned")
}

func TestFSCallMapsSyncException(t *testing.T) {
	fs := js.Global().Get("fs")
	if fs.Get("statSync").IsUndefined() {
		t.Fatal("fs.statSync is required to test sync exception mapping")
	}
	_, err := os.Stat("/llgo-pr2539-definitely-does-not-exist")
	if err == nil {
		t.Fatal("os.Stat of a missing path succeeded")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("os.Stat missing path = %v, want os.IsNotExist", err)
	}
}

func TestFSCallAsyncFallback(t *testing.T) {
	fs := js.Global().Get("fs")
	syncWrite := fs.Get("writeSync")
	if syncWrite.IsUndefined() {
		t.Fatal("fs.writeSync is required to test the async fallback")
	}
	fs.Set("writeSync", js.Undefined())
	t.Cleanup(func() { fs.Set("writeSync", syncWrite) })

	path := "llgo-fs-async-fallback.txt"
	const want = "hello-async\n"
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatalf("WriteFile without writeSync: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ReadFile = %q, want %q", got, want)
	}
}

func TestFSCallKeepsReadAsync(t *testing.T) {
	testFSCallUsesAsyncPath(t, "read", func() {
		buf := make([]byte, 1)
		_, _ = os.Stdin.Read(buf)
	})
}

func TestFSCallKeepsFsyncAsync(t *testing.T) {
	testFSCallUsesAsyncPath(t, "fsync", func() {
		_ = os.Stdout.Sync()
	})
}

func testFSCallUsesAsyncPath(t *testing.T, name string, fn func()) {
	t.Helper()
	fs := js.Global().Get("fs")
	orig := fs.Get(name)
	origSync := fs.Get(name + "Sync")
	if orig.IsUndefined() {
		t.Fatalf("fs.%s is required to test the async path", name)
	}

	called := make(chan struct{}, 1)
	fs.Set(name, js.FuncOf(func(this js.Value, args []js.Value) any {
		select {
		case called <- struct{}{}:
		default:
		}
		if len(args) == 0 {
			return nil
		}
		cb := args[len(args)-1]
		cb.Invoke(js.Null(), 0)
		return nil
	}))
	t.Cleanup(func() {
		fs.Set(name, orig)
		fs.Set(name+"Sync", origSync)
	})
	if !origSync.IsUndefined() {
		fs.Set(name+"Sync", js.Undefined())
	}

	fn()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatalf("fs.%s used a sync path instead of the callback API", name)
	}
}

func TestJSValueZeroIsUndefined(t *testing.T) {
	var value js.Value
	if !value.IsUndefined() || value.Type() != js.TypeUndefined {
		t.Fatalf("zero js.Value = undefined %v, type %v", value.IsUndefined(), value.Type())
	}
	if !value.Equal(js.Undefined()) {
		t.Fatal("zero js.Value does not equal js.Undefined()")
	}
}

func TestHostCallbackWakesScheduler(t *testing.T) {
	done := make(chan struct{}, 1)
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		done <- struct{}{}
		return nil
	})
	defer callback.Release()

	js.Global().Call("setTimeout", callback, 0)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("JavaScript callback did not wake the scheduler")
	}
}

func TestJSFuncDispatchesAfterGoCall(t *testing.T) {
	done := make(chan int, 1)
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		done <- 7
		return 7
	})
	defer callback.Release()

	got := callback.Invoke()
	if !got.IsUndefined() {
		t.Fatalf("Invoke() = %v, want undefined for async FuncOf", got)
	}
	select {
	case n := <-done:
		if n != 7 {
			t.Fatalf("callback result = %d, want 7", n)
		}
	case <-time.After(time.Second):
		t.Fatal("js.FuncOf callback did not run after Invoke returned")
	}
}

func TestJSFuncCompletesBufferedChannelAfterCallReturns(t *testing.T) {
	c := make(chan int, 1)
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		c <- 99
		return nil
	})
	defer callback.Release()

	obj := js.Global().Get("Object").New()
	obj.Set("write", callback)
	obj.Call("write")
	select {
	case got := <-c:
		if got != 99 {
			t.Fatalf("got %d, want 99", got)
		}
	case <-time.After(time.Second):
		t.Fatal("js.FuncOf callback did not send after Call returned")
	}
}

func TestJSFuncCanParkFromGoCall(t *testing.T) {
	done := make(chan int, 1)
	cb := js.FuncOf(func(js.Value, []js.Value) any {
		time.Sleep(time.Millisecond)
		done <- 42
		return nil
	})
	defer cb.Release()
	cb.Invoke()
	select {
	case got := <-done:
		if got != 42 {
			t.Fatalf("parked callback result = %d, want 42", got)
		}
	case <-time.After(time.Second):
		t.Fatal("parked js.FuncOf callback did not resume")
	}
}

func TestJSFuncReinstallDuringCallback(t *testing.T) {
	installed := make(chan struct{})
	result := make(chan int, 1)
	var old, fresh js.Func
	old = js.FuncOf(func(js.Value, []js.Value) any {
		old.Release()
		fresh = js.FuncOf(func(js.Value, []js.Value) any {
			result <- 42
			return 42
		})
		close(installed)
		return nil
	})
	old.Invoke()
	select {
	case <-installed:
	case <-time.After(time.Second):
		t.Fatal("js.FuncOf callback did not install a replacement function")
	}
	defer fresh.Release()
	fresh.Invoke()
	select {
	case got := <-result:
		if got != 42 {
			t.Fatalf("fresh callback result = %d, want 42", got)
		}
	case <-time.After(time.Second):
		t.Fatal("replacement js.FuncOf callback did not run")
	}
}

func TestHostCallbackCanBlock(t *testing.T) {
	done := make(chan int, 1)
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		value := make(chan int)
		go func() {
			value <- 42
		}()
		got := <-value
		done <- got
		return got
	})
	defer callback.Release()

	js.Global().Call("setTimeout", callback, 0)
	select {
	case got := <-done:
		if got != 42 {
			t.Fatalf("got %d, want 42", got)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked JavaScript callback prevented another goroutine from running")
	}
}
