//go:build js && wasm && !llgo.wasm.emscripten

package main

import (
	"errors"
	"os"
	"syscall"
	"syscall/js"
)

func init() {
	// Node and custom hosts supply their own filesystem. Exercise the Go
	// browser fallback only when it is active.
	if js.Global().Get("fs").Get("constants").Get("O_WRONLY").Int() != -1 {
		return
	}
	const message = "wasm browser filesystem fallback ok\n"
	if n, err := os.Stdout.WriteString(message); err != nil || n != len(message) {
		panic("browser console write failed")
	}
	if err := os.Stdout.Sync(); err != nil {
		panic("browser console sync failed")
	}
	if _, err := os.Open("/unsupported"); !errors.Is(err, syscall.ENOSYS) {
		panic("browser filesystem did not report ENOSYS")
	}
}
