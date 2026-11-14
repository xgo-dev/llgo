//go:build llgo && js && wasm && llgo.wasm.debugger

// Copyright 2026 The XGo Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0.

// Package wasmdebugpc captures browser Wasm call sites before Asyncify unwinds.
package wasmdebugpc

import _ "unsafe"

const LLGoFiles = "_wrap/capture.c"

//go:linkname Capture C.llgo_wasm_debug_capture_pcs
func Capture(pcs *uint64, capacity uint32) uint32
