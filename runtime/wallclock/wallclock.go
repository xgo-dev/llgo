// Package wallclock exposes the llgo runtime's settable wall clock for
// programs that need real calendar time on RTC-less hardware — log
// timestamps, TLS certificate validation and the like.
//
// The baremetal RV32 fiber targets (e.g. esp32s31) have no RTC: their wall
// clock starts at the Unix epoch and advances at the cycle clock's rate.
// SetWallTime moves the wall clock's epoch anchor so subsequent time.Now
// readings carry the given instant. The move follows the stdlib time
// contract: the wall segment may jump forward or backward freely, while the
// monotonic segment (time.Since, timers, time.Sleep) keeps reading the
// untouched cycle clock and never goes backwards.
//
// Targets without a usable clock are documented no-ops: hosted and Wasm
// targets already read the real OS clock, and the runtime deliberately does
// not re-anchor a live host clock; nogc and xtensa baremetal targets have no
// clock at all. See runtime/internal/lib/runtime/time_*.go for the
// per-target policy.
//
// Under the upstream Go toolchain (go vet, gopls, export-data builds) the
// package compiles as an ordinary declaration; the link below resolves only
// when llgo links the program.
package wallclock

import _ "unsafe" // for go:linkname

//go:linkname setWallTime runtime.SetWallTime
func setWallTime(now int64)

// SetWallTime sets the wall clock to the given Unix time in nanoseconds.
// Later time.Now calls read the wall segment from that instant onward.
// time.Since and every monotonic consumer are unaffected.
func SetWallTime(now int64) {
	setWallTime(now)
}
