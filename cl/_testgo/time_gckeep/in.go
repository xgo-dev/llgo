// Negative GC experiment for the RV32 baremetal fiber scheduler's callback
// timer list: the payload object is armed on a pending AfterFunc and nothing
// else points at it — its constructor frame is dead by the time the
// collections run, and a running goroutine's stack is only scanned above its
// live stack pointer, so the sole path keeping it alive is the package-level
// timer table in the globals region (baremetalTimers → record → closure →
// payload). Three explicit collections with allocation churn run while the
// timer is pending; a collector that missed that root chain would hand the
// payload's memory to the churn allocations and the callback would print
// garbage or fault. ReadMemStats proves the collections really ran, so a
// green run cannot be explained by "the GC never executed".

package main

import (
	"runtime"
	"time"
)

// keep is deliberately heap-resident: the struct escapes into the closure,
// the slice's backing array is a make allocation, and the string is built at
// run time. The tag bytes may live in rodata, but they are only reachable
// through the struct — a collected struct turns every read of them into
// garbage.
type keep struct {
	a, b int
	tag  string
	data []int
}

// tagWith keeps the string construction out of constant folding.
func tagWith(suffix string) string {
	return "he" + suffix
}

// armKeep owns the payload exclusively: its frame dies at return, leaving the
// pending timer's record as the payload's only root.
func armKeep(done chan bool) *time.Timer {
	p := &keep{a: 40, b: 2, tag: tagWith("ap"), data: make([]int, 4)}
	for i := range p.data {
		p.data[i] = i * i
	}
	return time.AfterFunc(50*time.Millisecond, func() {
		ok := p.a+p.b == 42 && p.tag == "heap" && len(p.data) == 4 &&
			p.data[0] == 0 && p.data[1] == 1 && p.data[2] == 4 && p.data[3] == 9
		println("payload", p.a+p.b, p.tag, len(p.data))
		done <- ok
	})
}

func main() {
	done := make(chan bool, 1)
	armKeep(done)

	// Churn plus explicit collections while the timer is pending. The 50 ms
	// deadline gives the three rounds orders of magnitude more cycles than
	// they need on every backend, and under esp-emu the cycle advance is
	// instruction-deterministic, so the rounds completing before the callback
	// is a reproduced property of the run, not a wall-clock race.
	for i := 0; i < 3; i++ {
		sink := make([]int, 512)
		for j := range sink {
			sink[j] = j
		}
		_ = sink
		runtime.GC()
	}

	ok := <-done
	println("kept", ok)

	// The negative-control: assert the three collections actually ran, so
	// the survival above was tested rather than vacuous.
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	println("gc ran", stats.NumGC >= 3)
	println("done")
}
