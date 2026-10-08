//go:build js && wasm && llgo.wasm.workers

package wasmtest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"syscall/js"
	"testing"
	"time"
	_ "unsafe"

	"github.com/xgo-dev/llgo/runtime/_test/workerlocality"
	"github.com/xgo-dev/llgo/runtime/wasmworkers"
)

//go:linkname schedulerProcID github.com/xgo-dev/llgo/runtime/internal/runtime.SchedulerProcID
func schedulerProcID() int

type workerCallbackResult struct {
	origin   int
	callback int
	err      error
}

type workerFinalizerBarrier struct {
	padding [128]uintptr
}

func TestWorkerSharedJSValues(t *testing.T) {
	object := js.ValueOf(map[string]any{"value": 42})
	owner := schedulerProcID()
	result := make(chan workerCallbackResult, 1)
	go func() {
		// Neither descendant needs to stay on the originating Go worker to
		// use a JavaScript value passed through the closure.
		go func() {
			worker := schedulerProcID()
			if value := object.Get("value").Int(); value != 42 {
				result <- workerCallbackResult{origin: owner, callback: worker, err: fmt.Errorf("JavaScript value = %d on worker %d, want 42", value, worker)}
				return
			}
			result <- workerCallbackResult{origin: owner, callback: worker}
		}()
	}()
	if got := <-result; got.err != nil {
		t.Fatal(got.err)
	}
}

func TestWorkerTLSRoots(t *testing.T) {
	result := make(chan int, 1)
	go func() {
		workerlocality.InstallTLSValue(42)
		workerlocality.InstallTLSFunc(43)
		runtime.GC()
		result <- workerlocality.LoadTLSValue() + workerlocality.CallTLSFunc()
	}()
	if got := <-result; got != 85 {
		t.Fatalf("worker TLS values = %d, want 85", got)
	}
}

func TestWorkerEmvalFinalizersStayInRealm(t *testing.T) {
	const goroutines = 8
	created := make(chan workerCallbackResult, goroutines)
	for range goroutines {
		wasmworkers.GoIndependent(func() {
			owner := schedulerProcID()
			var err error
			for range 8 {
				if got := js.ValueOf("temporary").String(); got != "temporary" {
					err = fmt.Errorf("temporary JavaScript value = %q", got)
					break
				}
			}
			created <- workerCallbackResult{origin: owner, err: err}
		})
	}
	owners := make(map[int]bool)
	for range goroutines {
		result := <-created
		if result.err != nil {
			t.Fatal(result.err)
		}
		owners[result.origin] = true
	}
	if len(owners) < 2 {
		t.Fatalf("JavaScript values were used on %d worker, want at least 2", len(owners))
	}

	finalizersDone := make(chan struct{})
	installWorkerFinalizerBarrier(finalizersDone)
	for range 24 {
		clobberWorkerStack(16, 1)
		runtime.GC()
		select {
		case <-finalizersDone:
			goto finalizersComplete
		default:
			time.Sleep(time.Millisecond)
		}
	}
	t.Fatal("JavaScript value finalizers did not make progress")

finalizersComplete:

	checked := make(chan workerCallbackResult, goroutines)
	for range goroutines {
		wasmworkers.GoIndependent(func() {
			owner := schedulerProcID()
			object := js.Global().Get("Object").New()
			object.Set("owner", owner)
			got := object.Get("owner").Int()
			var err error
			if got != owner {
				err = fmt.Errorf("JavaScript object owner = %d, want %d", got, owner)
			}
			checked <- workerCallbackResult{origin: owner, err: err}
		})
	}
	owners = make(map[int]bool)
	for range goroutines {
		result := <-checked
		if result.err != nil {
			t.Fatal(result.err)
		}
		owners[result.origin] = true
	}
	if len(owners) < 2 {
		t.Fatalf("JavaScript values were checked on %d worker, want at least 2", len(owners))
	}
}

func TestRetiredWorkerFiberReleasesFinalizer(t *testing.T) {
	// A finalizable allocation can itself be kept alive by an unrelated
	// conservative root. Check independent retired fibers before concluding
	// that the fiber storage remains rooted.
	for range 3 {
		if retiredWorkerFiberReleasesFinalizer(t) {
			return
		}
	}
	t.Fatal("finalizers stayed reachable after their worker fibers exited")
}

//go:noinline
func retiredWorkerFiberReleasesFinalizer(t *testing.T) bool {
	finalized := make(chan struct{})
	installed := make(chan struct{})
	go func() {
		installWorkerFinalizerBarrier(finalized)
		close(installed)
	}()
	<-installed
	// The notification precedes the goroutine's return. Run a later task on
	// each worker so its scheduler has retired the finished fiber before GC.
	settled := make(chan int, 2)
	for range 2 {
		wasmworkers.GoIndependent(func() { settled <- schedulerProcID() })
	}
	owners := map[int]bool{}
	for range 2 {
		owners[<-settled] = true
	}
	if len(owners) != 2 {
		t.Fatalf("fiber retirement reached %d workers, want 2", len(owners))
	}

	// The G has exited, but a conservative reference in its retired fiber
	// storage must not keep the finalizable object alive indefinitely.
	for range 24 {
		clobberWorkerStack(16, 1)
		runtime.GC()
		select {
		case <-finalized:
			return true
		default:
			time.Sleep(time.Millisecond)
		}
	}
	return false
}

//go:noinline
func installWorkerFinalizerBarrier(done chan<- struct{}) {
	barrier := &workerFinalizerBarrier{}
	runtime.SetFinalizer(barrier, func(*workerFinalizerBarrier) { close(done) })
}

//go:noinline
func clobberWorkerStack(depth int, value uintptr) uintptr {
	var words [32]uintptr
	for i := range words {
		words[i] = value + uintptr(i)
	}
	if depth != 0 {
		return words[depth%len(words)] + clobberWorkerStack(depth-1, value+1)
	}
	return words[0]
}

func TestConcurrentWorkerOutput(t *testing.T) {
	const writers = 16
	results := make(chan workerCallbackResult, writers)
	for range writers {
		wasmworkers.GoIndependent(func() {
			origin := schedulerProcID()
			_, err := fmt.Fprint(os.Stdout, ".")
			results <- workerCallbackResult{origin: origin, err: err}
		})
	}

	workers := make(map[int]bool)
	for range writers {
		result := <-results
		if result.err != nil {
			t.Fatal(result.err)
		}
		workers[result.origin] = true
	}
	if len(workers) < 2 {
		t.Fatalf("output ran on %d scheduler worker, want at least 2", len(workers))
	}
	fmt.Fprintln(os.Stdout)
}

func TestWorkerJSFilesystemHandles(t *testing.T) {
	const writers = 8
	dir := t.TempDir()
	results := make(chan workerCallbackResult, writers)
	for i := range writers {
		path := filepath.Join(dir, fmt.Sprintf("worker-%d.txt", i))
		wasmworkers.GoIndependent(func() {
			origin := schedulerProcID()
			fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_WRONLY|syscall.O_TRUNC, 0o600)
			if err == nil {
				var n int
				n, err = syscall.Write(fd, []byte("worker"))
				if err == nil && n != len("worker") {
					err = fmt.Errorf("write %s: wrote %d bytes", path, n)
				}
				if closeErr := syscall.Close(fd); err == nil {
					err = closeErr
				}
			}
			results <- workerCallbackResult{origin: origin, err: err}
		})
	}
	workers := make(map[int]bool)
	for range writers {
		result := <-results
		if result.err != nil {
			t.Fatal(result.err)
		}
		workers[result.origin] = true
	}
	if len(workers) < 2 {
		t.Fatalf("file operations ran on %d worker, want at least 2", len(workers))
	}
	for i := range writers {
		path := filepath.Join(dir, fmt.Sprintf("worker-%d.txt", i))
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "worker" {
			t.Fatalf("read %s = %q, %v", path, data, err)
		}
	}
}

func TestWorkerIndependentFromJSCallback(t *testing.T) {
	const tasks = 16
	results := make(chan workerCallbackResult, tasks)
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		origin := schedulerProcID()
		for range tasks {
			wasmworkers.GoIndependent(func() {
				results <- workerCallbackResult{origin: origin, callback: schedulerProcID()}
			})
		}
		return nil
	})
	defer callback.Release()
	js.Global().Call("setTimeout", callback, 0)
	remote := false
	timeout := time.After(30 * time.Second)
	for range tasks {
		select {
		case result := <-results:
			remote = remote || result.callback != result.origin
		case <-timeout:
			t.Fatal("independent work from JavaScript callback did not complete")
		}
	}
	if !remote {
		t.Fatal("independent work from JavaScript callback stayed on its origin worker")
	}
}

func TestWorkerHostCallbackRealms(t *testing.T) {
	const callbacks = 16
	ready := make(chan struct{}, callbacks)
	start := make(chan struct{})
	results := make(chan workerCallbackResult, callbacks)
	for range callbacks {
		wasmworkers.GoIndependent(func() {
			origin := schedulerProcID()
			done := make(chan int, 1)
			callback := js.FuncOf(func(js.Value, []js.Value) any {
				done <- schedulerProcID()
				return nil
			})
			defer callback.Release()
			ready <- struct{}{}
			<-start
			if current := schedulerProcID(); current != origin {
				results <- workerCallbackResult{origin: origin, err: fmt.Errorf("goroutine changed worker before JavaScript callback: got %d, want %d", current, origin)}
				return
			}
			js.Global().Call("setTimeout", callback, 0)
			// The outer timeout covers the complete callback batch. Do not create
			// one uncancellable time.After timer per callback: successful selects
			// leave those timers queued and repeated tests can overwhelm the host
			// event loop several seconds later.
			callbackWorker := <-done
			results <- workerCallbackResult{origin: origin, callback: callbackWorker}
		})
	}
	for range callbacks {
		<-ready
	}
	// Every callback is now retained only by its suspended goroutine and the
	// physical worker's syscall/js registry. Collect before JavaScript receives
	// any of them so the test covers both root paths without starving the host
	// event loop behind one collection per callback.
	runtime.GC()
	close(start)

	workers := make(map[int]bool)
	timeout := time.After(30 * time.Second)
	for range callbacks {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatal(result.err)
			}
			if result.callback != 0 {
				t.Fatalf("callback worker = %d, want shared JS worker 0 (caller %d)", result.callback, result.origin)
			}
			workers[result.origin] = true
		case <-timeout:
			t.Fatal("worker callbacks did not make progress")
		}
	}
	if callbacks > 1 && len(workers) < 2 {
		t.Fatalf("callbacks ran on %d scheduler worker, want at least 2", len(workers))
	}
}
