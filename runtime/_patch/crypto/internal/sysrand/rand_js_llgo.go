//go:build llgo && js && wasm

//llgo:skip getRandomValues

package sysrand

import _ "unsafe"

// LLGo adapts the official Go host import to the selected JavaScript provider.
// Keep the standard-library helper and replace only its private host boundary.
//
//go:linkname getRandomValues runtime.getRandomData
func getRandomValues(p []byte)
