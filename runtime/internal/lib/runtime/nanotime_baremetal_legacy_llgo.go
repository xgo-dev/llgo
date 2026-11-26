//go:build baremetal && !go1.23

package runtime

// runtimeNano backs the legacy baremetal timer hooks under a pre-go1.23
// standard library. That combination has no monotonic clock domain: the
// fiber scheduler and the wall anchor it feeds do not exist, so time.Now
// keeps reporting the zero instant and timer deadlines collapse onto the
// same base (see time_baremetal_legacy_llgo.go). The cycle clock is
// deliberately not used here: without the anchor advancing time.Now's wall
// segment in lockstep, a cycle-based monotonic reading would desynchronize
// the two domains of every Time value. This mirrors the degenerate stub
// nanotime_baremetal_llgo.go selects once go1.23 turns on.
func runtimeNano() int64 { return 0 }

// The standard library also links directly to runtime.nanotime.
func nanotime() int64 { return runtimeNano() }
