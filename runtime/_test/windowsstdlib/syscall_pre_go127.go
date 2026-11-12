//go:build !go1.27

package main

// Older selected SDKs retain their original 42-argument contract and fixture.
func testDynamicWindowsSyscalls() {}
