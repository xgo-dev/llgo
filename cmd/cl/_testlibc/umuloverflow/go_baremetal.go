//go:build baremetal

package main

// Baremetal has no thread backend for go calls. The shared test still checks
// every unsigned width, function values, and deferred argument evaluation.
func checkGoCall() {}
