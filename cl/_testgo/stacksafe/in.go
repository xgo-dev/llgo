// Stack-guard safety fixture for the RV32 baremetal fiber scheduler:
// recursions that stay well inside the fiber stack bounds must complete
// normally with the swap-point overflow guard active, on the 16 KiB main
// fiber and on a 2 KiB goroutine fiber alike. runtime.Gosched at every level
// drives each frame through the guarded swap, and the accumulated result is
// checked so the recursion is real work, not dead code. The overflow
// counterpart lives in ./_testgo/stackover.
//
// rec/recNext recurse through each other with //go:noinline so the optimizer
// cannot flatten the descent into a loop (see stackover for the rationale).
// Stack budget: each level is two 16-byte noinline frames = 32 B, so 40
// levels are 40×32 B = 1280 B of frames, plus the ~80 B transient
// Gosched→goschedBackend→fiberSwap chain at the deepest yield: ≈1360 B
// against the 2 KiB goroutine stack's 2048−68 = 1980 B guarded span.
//
// Fiber-scheduler emu fixture: hosted stacks grow instead of fataling, and
// the xtensa ESP32 keeps the pthread scheduler, so the hosted and esp32
// suites skip it; C3 runs it through the shared emu auto-walk, S31 through
// the explicit golden in cl/embed_stackguard_test.go.
package main

import "runtime"

//go:noinline
func rec(n int) int {
	runtime.Gosched()
	return recNext(n)
}

//go:noinline
func recNext(n int) int {
	if n <= 0 {
		return 0
	}
	return rec(n-1) + 1
}

func main() {
	if got := rec(40); got != 40 {
		println("main bad", got)
		return
	}
	println("main ok")
	done := make(chan bool)
	go func() {
		if got := rec(40); got != 40 {
			println("worker bad", got)
			done <- false
			return
		}
		println("worker ok")
		done <- true
	}()
	ok := <-done
	println("joined", ok)
}
