//go:build llgo && wasm && !(wasip1 && llgo.wasi_threads) && !llgo.wasm.workers

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
// scheduler. The hook dispatches queued external events on ordinary Gs and
// coordinates return from synchronous JS events after runnable Go work drains.
func RegisterWasmCallbackPoll(poll func()) {
	wasmCallbackPollHook = poll
}

// RegisterWasmTimerHooks connects the Go-derived timer heap on first use.
// Programs without timers can drop the timer implementation at link time.
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
		pendingPCLN := pollWasmPCLN()
		if gp := popWasmRunq(); gp != nil {
			return gp
		}
		if pendingPCLN {
			// Asyncify waits on the system stack so the JS promise can
			// settle even before the standard-library timers are initialized.
			wasmevent.Wait(wasmPCLNPollInterval)
			continue
		}
		var wait uint64
		var active bool
		if wasmTimerWaitHook != nil {
			wait, active = wasmTimerWaitHook()
		}
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
