//go:build llgo && js && wasm && !llgo.wasm.emscripten

// Copyright (c) 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache License 2.0.

// Package wasmjs adapts syscall/js host operations to LLGo's physical ABI.
// All public Go API behavior remains in the selected GOROOT's syscall/js.
package wasmjs

const LLGoFiles = "_wrap/host.c"
