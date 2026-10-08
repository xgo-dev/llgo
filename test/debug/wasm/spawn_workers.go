//go:build llgo.wasm.workers

package main

import _ "unsafe"

//go:linkname spawnDebugG github.com/xgo-dev/llgo/runtime/internal/runtime.SpawnIndependentWasmG
func spawnDebugG(func())
