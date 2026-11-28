//go:build !nogc

package main

import (
	"runtime"
	"unique"
)

func verifyWorkerGC() {
	// Its persistent cleanup worker must not suppress main-Goexit detection.
	unique.Make("wasi-goexit-worker")
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	runtime.GC()
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		panic("initial Goexit thread prevented worker GC")
	}
}
