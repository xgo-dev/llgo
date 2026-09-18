//go:build go1.26 && !wasm
// +build go1.26,!wasm

package test

import "testing"

func TestBuiltinPrintGo126FloatFormat(t *testing.T) {
	got := runBuiltinPrintProbe(t)
	want := builtinPrintWant()
	if got != want {
		t.Fatalf("builtin print output mismatch:\n got %q\nwant %q", got, want)
	}
}
