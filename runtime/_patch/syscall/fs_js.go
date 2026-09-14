//go:build js && wasm

package syscall

import "syscall/js"

// fsCall prefers Node's fs.*Sync APIs so file I/O that completes immediately
// (write, stat, open, ...) does not park on a js.FuncOf callback. Official Go
// waits on a buffered channel after fs.write; that nested park aborts LLGo's
// Emscripten Fiber/Asyncify scheduler.
//
// Existence of a Sync counterpart is not enough. read can block indefinitely
// on stdin, pipes, or TTYs, and fsync can stall on slow storage. Using those
// Sync methods freezes the single-threaded Go/JS scheduler, so timers,
// goroutines, and JS callbacks cannot run until the call returns. Those
// methods keep the original callback-and-channel path even when *Sync exists.
func fsCall(name string, args ...any) (val js.Value, err error) {
	if !fsCallMustStayAsync(name) {
		syncName := name + "Sync"
		if !jsFS.Get(syncName).IsUndefined() {
			defer recoverErr(&err)
			return jsFS.Call(syncName, args...), nil
		}
	}
	return fsCallAsync(name, args...)
}

func fsCallMustStayAsync(name string) bool {
	switch name {
	case "read", "fsync":
		return true
	default:
		return false
	}
}

func fsCallAsync(name string, args ...any) (js.Value, error) {
	type callResult struct {
		val js.Value
		err error
	}

	c := make(chan callResult, 1)
	f := js.FuncOf(func(this js.Value, args []js.Value) any {
		var res callResult

		if len(args) >= 1 { // on Node.js 8, fs.utimes calls the callback without any arguments
			if jsErr := args[0]; !jsErr.IsNull() {
				res.err = mapJSError(jsErr)
			}
		}

		res.val = js.Undefined()
		if len(args) >= 2 {
			res.val = args[1]
		}

		c <- res
		return nil
	})
	defer f.Release()
	jsFS.Call(name, append(args, f)...)
	res := <-c
	return res.val, res.err
}
