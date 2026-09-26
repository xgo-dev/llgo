// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

//go:build js && wasm
// +build js,wasm

package js

import (
	"sync"

	"github.com/xgo-dev/llgo/runtime/internal/clite"
	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// Emscripten callback IDs and emval queues belong to one JavaScript worker.
//
//llgointernal:tls
var (
	funcsMu         sync.Mutex
	funcs           map[uint32]func(Value, []Value) any
	nextFuncID      uint32
	activeCallbacks uint32
)

var callbackPoll struct {
	sync.Mutex
	registered bool
}

func ensureCallbackPoll() {
	callbackPoll.Lock()
	if !callbackPoll.registered {
		llruntime.RegisterWasmCallbackPoll(pollCallbacks)
		callbackPoll.registered = true
	}
	callbackPoll.Unlock()
}

// Func is a wrapped Go function to be called by JavaScript.
type Func struct {
	Value // the JavaScript function that invokes the Go function
	id    uint32
}

// FuncOf returns a function to be used by JavaScript.
//
// The Go function fn is called with JavaScript's "this" and arguments, and
// its result is returned synchronously. Nested calls run on the calling G;
// external events resume the scheduler and run on a new G. A callback may
// block on Go goroutines or timers, but must not wait for another asynchronous
// JS event: its caller still owns the JS event loop, as in the Go runtime.
//
// Func.Release must be called to free up resources when the function will not be invoked any more.
func FuncOf(fn func(this Value, args []Value) any) Func {
	ensureEmvalGlobals()
	funcsMu.Lock()
	if funcs == nil {
		funcs = make(map[uint32]func(Value, []Value) any)
		nextFuncID = 1
		// The installed JavaScript closure retains its queue, so install it
		// once per worker even if all callbacks are later released.
		emval_install_invoke()
	}
	ensureCallbackPoll()
	id := nextFuncID
	nextFuncID++
	funcs[id] = fn
	funcsMu.Unlock()
	var buf [20]byte
	sid := string(itoa(buf[:], uint64(id)))
	// A modularized Emscripten build intentionally has no global Module.
	// Capture the exported bridge from this module instance so callbacks work
	// in Node, browsers, and workers without leaking a process-global module.
	invoke := emval_get_module_property(c.Str("_llgo_invoke"))
	factory := functionConstructor.New(ValueOf("invoke"), ValueOf(`
		return function() {
			const event = { id:`+sid+`, this: this, args: arguments };
			return invoke(event);
		};
	`))
	wrap := factory.Invoke(invoke)
	return Func{
		id:    id,
		Value: wrap,
	}
}

func itoa(buf []byte, val uint64) []byte {
	i := len(buf) - 1
	for val >= 10 {
		buf[i] = byte(val%10 + '0')
		i--
		val /= 10
	}
	buf[i] = byte(val + '0')
	return buf[i:]
}

// Release frees up resources allocated for the function.
// The function must not be invoked after calling Release.
// It is allowed to call Release while the function is still running.
func (c Func) Release() {
	funcsMu.Lock()
	delete(funcs, c.id)
	stopCallbackPollLocked()
	funcsMu.Unlock()
}

func stopCallbackPollLocked() {
	if keepWasmCallbackPoll || len(funcs) != 0 || activeCallbacks != 0 || emval_has_pending_invoke() {
		return
	}
	callbackPoll.Lock()
	if callbackPoll.registered {
		llruntime.RegisterWasmCallbackPoll(nil)
		callbackPoll.registered = false
	}
	callbackPoll.Unlock()
}

func retainCallback() {
	funcsMu.Lock()
	activeCallbacks++
	funcsMu.Unlock()
}

func dispatchSynchronousCallback(handle c.Ulong) {
	// Bounded workers keep their callback poll installed. An external JS
	// callback can enter on the scheduler's system fiber without a G, so it
	// must not acquire funcsMu before HandleWasmEvent starts its handler G.
	if !keepWasmCallbackPoll {
		retainCallback()
	}
	runCallback(uintptr(handle), llruntime.SchedulerProcID())
}

func runCallback(handle uintptr, owner int) {
	llruntime.HandleWasmEvent(func() { dispatchCallback(handle, owner) })
	if !keepWasmCallbackPoll {
		funcsMu.Lock()
		activeCallbacks--
		stopCallbackPollLocked()
		funcsMu.Unlock()
	}
}

func dispatchCallback(handle uintptr, owner int) {
	if current := llruntime.SchedulerProcID(); current != owner {
		panic("syscall/js: callback dispatched outside its JavaScript realm")
	}
	llruntime.MarkCurrentJSRealm()
	defer cEmvalDecref(handle)
	cb := Value{ref: ref(handle)}
	id := uint32(cb.Get("id").Int())
	funcsMu.Lock()
	f, ok := funcs[id]
	funcsMu.Unlock()
	if !ok {
		Global().Get("console").Call("error", "call to released function")
		return
	}

	// Call the js.Func with arguments
	this := cb.Get("this")
	argsObj := cb.Get("args")
	args := make([]Value, argsObj.Length())
	for i := range args {
		args[i] = argsObj.Index(i)
	}
	result := f(this, args)

	// Return the result to js
	cb.Set("result", result)
}

func pollCallbacks() {
	pollEmvalReleases()
	// The host sets a byte in wasm memory when it enqueues the first event, so
	// an idle scheduler does not cross the wasm/JavaScript boundary merely to
	// inspect an empty JavaScript array.
	for emval_has_pending_invoke() {
		handle := emval_take_pending_invoke()
		if handle == 0 {
			break
		}
		// Only single-worker builds can remove the poll hook while a
		// callback is pending. Bounded workers keep it installed, and polling
		// may run without a G on the scheduler's system fiber.
		if !keepWasmCallbackPoll {
			retainCallback()
		}
		owner := llruntime.SchedulerProcID()
		go runCallback(handle, owner)
	}
	llruntime.PollWasmEvent()
}
