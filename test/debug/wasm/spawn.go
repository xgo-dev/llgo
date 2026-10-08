//go:build !llgo.wasm.workers

package main

func spawnDebugG(fn func()) { go fn() }
