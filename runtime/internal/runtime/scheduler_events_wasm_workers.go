//go:build llgo && js && wasm && llgo.wasm.workers

// Copyright (c) 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache License 2.0.

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/wasmsync"

type wasmEventHooks struct {
	pollTimers    func()
	timerWait     func() (wait uint64, active bool)
	pollCallbacks func()
}

var wasmSchedulerEventHooks struct {
	lock  wasmsync.Mutex
	hooks wasmEventHooks
}

// RegisterWasmCallbackPoll connects a host callback source to every worker.
// Emscripten values belong to one JavaScript realm, so each physical worker
// must consume and dispatch its own callback queue.
func RegisterWasmCallbackPoll(poll func()) {
	wasmSchedulerEventHooks.lock.Lock(wasmGCAllocatorYield)
	wasmSchedulerEventHooks.hooks.pollCallbacks = poll
	wasmSchedulerEventHooks.lock.Unlock()
}

// RegisterWasmTimerHooks connects the Go-derived timer heap. The timer
// implementation serializes heap access across workers.
func RegisterWasmTimerHooks(poll func(), wait func() (uint64, bool)) {
	wasmSchedulerEventHooks.lock.Lock(wasmGCAllocatorYield)
	wasmSchedulerEventHooks.hooks.pollTimers = poll
	wasmSchedulerEventHooks.hooks.timerWait = wait
	wasmSchedulerEventHooks.lock.Unlock()
}

func loadWasmEventHooks() wasmEventHooks {
	wasmSchedulerEventHooks.lock.Lock(wasmGCAllocatorYield)
	hooks := wasmSchedulerEventHooks.hooks
	wasmSchedulerEventHooks.lock.Unlock()
	return hooks
}

func (hooks wasmEventHooks) pollCallbackEvents(worker *wasmWorker) {
	if hooks.pollCallbacks == nil || worker == nil || worker.pollingCallback {
		return
	}
	// pollCallbacks starts one G per queued host event. Mark that narrow
	// interval so newprocBackend keeps emval handles in their originating
	// JavaScript realm even when polling happens from a running G's safepoint.
	worker.pollingCallback = true
	hooks.pollCallbacks()
	worker.pollingCallback = false
}

func (hooks wasmEventHooks) pollTimerEvents() {
	if hooks.pollTimers != nil {
		hooks.pollTimers()
	}
}

// WakeWasmScheduler interrupts worker zero after another worker publishes an
// earlier timer or host work.
func WakeWasmScheduler() {
	wakeWasmEventWorker()
}

// WakeWasmCallbackPoll interrupts every physical worker so realm-affine host
// resource releases are observed by the worker that owns them.
func WakeWasmCallbackPoll() {
	wakeAllWasmWorkers()
}
