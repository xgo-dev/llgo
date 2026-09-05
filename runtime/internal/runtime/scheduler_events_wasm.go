//go:build llgo && wasm && !(wasip1 && llgo.wasi_threads)

// Copyright (c) 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache License 2.0.

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/wasmevent"

var (
	wasmPollTimersHook   func()
	wasmTimerWaitHook    func() (wait uint64, active bool)
	wasmCallbackPollHook func()
)

// RegisterWasmCallbackPoll connects a host callback source to the logical
// scheduler. Host bridges only queue callbacks and wake the host wait; the Go
// callbacks themselves are started as ordinary Gs from this poll point.
func RegisterWasmCallbackPoll(poll func()) {
	wasmCallbackPollHook = poll
}

// RegisterWasmTimerHooks connects the Go-derived timer heap when the standard
// runtime package is linked. Programs that do not use timers keep the nil
// hooks and remain linkable without pulling in timer state.
func RegisterWasmTimerHooks(poll func(), wait func() (uint64, bool)) {
	wasmPollTimersHook = poll
	wasmTimerWaitHook = wait
}

func pollWasmEvents() {
	if wasmCallbackPollHook != nil {
		wasmCallbackPollHook()
	}
	if wasmPollTimersHook != nil {
		wasmPollTimersHook()
	}
}

func popWasmRunq() *g {
	pollWasmEvents()
	return wasmSched.runq.Pop()
}

func waitWasmRunq() *g {
	for {
		if gp := popWasmRunq(); gp != nil {
			return gp
		}
		if wasmTimerWaitHook == nil {
			return nil
		}
		wait, active := wasmTimerWaitHook()
		if !active {
			// A registered Emscripten callback can publish runnable work even
			// when the Go timer heap is empty. Yield for the longest host wait;
			// llgo_wasm_host_wake interrupts it as soon as an event arrives.
			// WASI never registers this JS callback hook and retains immediate
			// deadlock detection in the no-timer case.
			if wasmCallbackPollHook == nil {
				return nil
			}
			wait = ^uint64(0)
		}
		if wait != 0 {
			wasmevent.Wait(wait)
		}
	}
}
