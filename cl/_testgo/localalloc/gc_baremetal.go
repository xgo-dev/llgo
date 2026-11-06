//go:build baremetal

package main

// Baremetal targets use nogc; keep the allocation and lifetime checks without
// importing the hosted runtime package.
func collectLocalAlloc() {}
