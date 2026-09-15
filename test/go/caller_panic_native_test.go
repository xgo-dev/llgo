//go:build !wasm

package gotest

import "testing"

// Wasm guests cannot create the subprocesses required by these checks. The
// Wasm acceptance host launches the fatal paths using retained test artifacts.
func TestCallerPanicTraceback(t *testing.T) {
	testCallerPanicTraceback(t)
}

func TestCallerRepanicTraceback(t *testing.T) {
	testCallerRepanicTraceback(t)
}
