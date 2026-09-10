//go:build js && wasm

package syscall

import "syscall/js"

// fsCall prefers Node's fs.*Sync APIs so file I/O does not park on a js.FuncOf
// callback. Official Go waits on a buffered channel after fs.write; that
// nested park aborts LLGo's Emscripten Fiber/Asyncify scheduler. Methods
// without a Sync counterpart keep the original callback-and-channel path.
func fsCall(name string, args ...any) (val js.Value, err error) {
	syncName := name + "Sync"
	if !jsFS.Get(syncName).IsUndefined() {
		defer recoverErr(&err)
		return jsFS.Call(syncName, args...), nil
	}
	return fsCallAsync(name, args...)
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
