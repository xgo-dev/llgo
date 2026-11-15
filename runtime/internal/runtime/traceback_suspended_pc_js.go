//go:build llgo && js && wasm && llgo.wasm.debugger

// Copyright 2026 The XGo Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package runtime

import (
	"github.com/xgo-dev/llgo/runtime/internal/clite/emscripten"
	"github.com/xgo-dev/llgo/runtime/internal/wasmdebugpc"
)

func captureWasmSuspension(gp *g) {
	// Asyncify replays call entries while restoring a fiber. Such a native
	// stack is incomplete and must not overwrite its original suspension.
	if emscripten.FiberRewinding() != 0 || gp == nil || gp.tracebackThread == nil {
		return
	}
	update := beginWasmCallerUpdate()
	node := (*wasmDebuggerNode)(gp.tracebackThread)
	node.pcCount = wasmdebugpc.Capture(&node.pcs[0], uint32(len(node.pcs)))
	endWasmCallerUpdate(update)
}
