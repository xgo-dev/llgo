//go:build llgo && !wasip1

package main

import "github.com/xgo-dev/llgo/test/buildcache/dep1"

func verifyRecoverCache() {
	var recovered any
	func() {
		defer dep1.Recover(&recovered)
		panic("cached recover")
	}()
	if recovered != "cached recover" {
		panic("dependency recover failed after cache lookup")
	}
}
