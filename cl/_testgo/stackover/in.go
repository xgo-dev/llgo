// Stack-overflow fixture for the RV32 baremetal fiber scheduler: an
// unbounded recursion on a 2 KiB goroutine fiber must end in the runtime's
// deterministic "fatal error: goroutine stack overflow" once the swap-point
// guard sees sp below the fiber stack floor, instead of silently smashing
// the adjacent heap blocks. runtime.Gosched at every level makes the guarded
// swap the detection checkpoint, keeping the pre-fatal overshoot to a single
// frame. The safety counterpart lives in ./_testgo/stacksafe.
//
// The warmup recursion runs first at a safe depth on the 16 KiB main fiber:
// its printed line separates "the guard fires at genuine exhaustion" from
// "the guard misfires at the first yield" - without it a broken guard and a
// correct one would produce the same two-line golden. Under a tiny explicit
// -goroutine-stack-size the warmup still passes on the main fiber while the
// goroutine fatals, which is the expected small-stack behavior.
//
// Fiber-scheduler emu fixture (see stacksafe): hosted and esp32 suites skip
// it, and it expects a fatal line, which the shared emu firmware runner
// rejects — the C3 auto-walk therefore ignores it and both S31 and C3 run it
// through the fatal-tolerant runner in cl/embed_stackguard_test.go.
package main

import "runtime"

// rec/recNext recurse through each other, both //go:noinline: a plain
// self-recursive helper whose result nobody observes collapses into a
// Gosched self-loop at -Oz (the optimizer drops the never-descending
// recursion), which would quietly defeat this fixture. Mutual recursion
// through noinline frames cannot be flattened, so every level really grows
// the stack.
//
//go:noinline
func rec(n int) int {
	runtime.Gosched()
	return recNext(n)
}

//go:noinline
func recNext(n int) int {
	return rec(n+1) + 1
}

// warm/warmNext are the bounded counterparts used for the safe-depth warmup;
// same mutual-noinline shape so the optimizer cannot flatten them either.
//
//go:noinline
func warm(n int) int {
	runtime.Gosched()
	return warmNext(n)
}

//go:noinline
func warmNext(n int) int {
	if n <= 0 {
		return 0
	}
	return warm(n-1) + 1
}

func main() {
	println("start")
	if got := warm(10); got != 10 {
		println("warmup bad", got)
		return
	}
	println("warmup ok")
	done := make(chan bool)
	go func() {
		rec(1)
		done <- true
	}()
	<-done
	println("unreachable")
}
