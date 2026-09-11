//go:build llgo && js && wasm

// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at the repository root for license terms.

package runtime

// Adapted from Go's runtime/lock_js.go handleEvent and beforeIdle. An event
// must return on its original G, after its handler and other runnable work
// complete. Only the scheduler primitives differ from the Go implementation.
var wasmJSEvents []*wasmJSEvent

type wasmJSEvent struct {
	gp       *g
	returned bool
}

func HandleWasmEvent(handler func()) {
	e := &wasmJSEvent{gp: getg()}
	wasmJSEvents = append(wasmJSEvents, e)
	handler()
	e.returned = true
	gopark()
	wasmJSEvents[len(wasmJSEvents)-1] = nil
	wasmJSEvents = wasmJSEvents[:len(wasmJSEvents)-1]
}

// PollWasmEvent is called by syscall/js's existing callback poll hook. Keep
// this out of the generic scheduler, so programs without syscall/js pay no
// code-size cost for JS event-stack semantics.
func PollWasmEvent() {
	if currentG != &wasmSched.systemG {
		return
	}
	for len(wasmJSEvents) != 0 && wasmSched.runq.Len() == 0 {
		if wasmPollTimersHook != nil {
			wasmPollTimersHook()
		}
		if wasmSched.runq.Len() != 0 {
			return
		}
		e := wasmJSEvents[len(wasmJSEvents)-1]
		if e.returned {
			goready(e.gp)
			return
		}
		// A synchronous callback owns the JS stack until its handler
		// returns. Like Go's beforeIdle, poll Go timers without yielding to
		// the JS event loop. Waiting for an async JS event here deadlocks.
		if wasmTimerWaitHook != nil {
			if _, active := wasmTimerWaitHook(); active {
				continue
			}
		}
		fatal("all goroutines are asleep - deadlock!")
		return
	}
}
