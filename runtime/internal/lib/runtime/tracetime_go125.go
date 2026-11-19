//go:build go1.25
// +build go1.25

package runtime

import (
	_ "unsafe"
)

const (
	// osHasLowResClock indicates that timestamps produced by nanotime on the platform have a
	// low resolution, typically on the order of 1 ms or more.
	osHasLowResClock = false
)

const traceTimeDiv = 64

// traceClockUnitsPerSecond estimates the number of trace clock units per
// second that elapse.
//
//go:linkname traceClockUnitsPerSecond runtime/trace.runtime_traceClockUnitsPerSecond
func traceClockUnitsPerSecond() uint64 {
	if osHasLowResClock {
		// We're using cputicks as our clock, so we need a real estimate.
		//return uint64(ticksPerSecond() / traceTimeDiv)
		panic("TODO: ticksPerSecond for windows")
	}
	// Our clock is nanotime, so it's just the constant time division.
	// (trace clock units / nanoseconds) * (1e9 nanoseconds / 1 second)
	return uint64(1.0 / float64(traceTimeDiv) * 1e9)
}
