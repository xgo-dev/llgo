// LITTEST darwin/arm64 linux/amd64
//
// Fiber-swap FP-integrity fixture: two goroutines each hold float values
// across cooperative yields and verify neither fiber's FP state bled into
// the other's. Two independent assertions ride every round:
//
//   - fpregs: on even rounds the _wrap/fpregs.c probe plants distinct bit
//     patterns in fs0-fs11 + fcsr (F-extension targets only) before the
//     yield and checks them after the swap; llgoFiberSwap must carry them
//     across the other fiber's whole run. The round holds no live FP value
//     of its own, so the probe's deliberate f-register clobber is safe
//     (see the worker comment).
//   - locals: on odd rounds mix(vals, k) is computed before the yield and
//     recomputed after the identical swap; if the compiler keeps the running
//     product in an f register across the fiberSwap call boundary, a swap
//     without FP save would corrupt it. (Caller-saved fa/ft values cannot
//     be probed this way, by design: the psABI gives them no cross-call
//     lifetime, and the yield points are ordinary external calls.)
//
// The workers pair through two unbuffered channels in asymmetric roles: the
// lead fills and sends first, the trailer receives first, then both follow
// the same fill → want → send → receive → verify shape. Each worker does
// exactly one send and one receive per round, so the hand-off counts balance
// across rounds and every send meets its partner already parked in the
// matching receive — no crossed-send deadlock, one cooperative swap per
// hand-off. The trailer fills right after taking the hand-off, so while the
// lead is parked its register pattern is not only swapped out but also
// overwritten by the trailer's fill: the lead's post-swap check can only
// pass through the saved frame on its own stack, never through register
// residue.
//
// Values use power-of-two-friendly bases so float32 arithmetic stays exact
// and the equality checks never depend on rounding. Hosted runs the same
// logic against the OS scheduler; the golden output is identical everywhere.

// CHECK-LABEL: define float @main.mix({{.*}}, float %{{[0-9]+}}){{.*}} {
// CHECK: fmul float
// CHECK: fadd float
// The probe pair stays an external call: the naked C bodies must never be
// inlined or reordered around the yield.
// CHECK: call void @llgo_fp_fill()
// CHECK: call i32 @llgo_fp_check()

package main

import _ "unsafe"

const LLGoFiles = "_wrap/fpregs.c"

//go:linkname fpFill C.llgo_fp_fill
func fpFill()

//go:linkname fpCheck C.llgo_fp_check
func fpCheck() int32

func mix(vals []float32, k float32) float32 {
	s := float32(0)
	for _, v := range vals {
		s = s*k + v
	}
	return s
}

const rounds = 12

// The lead opens each round with fill+send; the trailer opens with the
// matching receive, then fills. Rounds alternate: even rounds run the
// register probe alone, odd rounds run the live-local assertion alone.
// Keeping the two apart is load-bearing: the probe's deliberate f-register
// clobber must meet no live FP value (the compiler may legitimately keep a
// mix() product in fs across the fill call), and the live-local round must
// not refill the registers it wants to compare across the swap.
func worker(trailer bool, mine, theirs, done chan int, base float32) {
	vals := []float32{base, base + 1.5, base*2 - 0.5, base*0.5 + 1}
	for round := 0; round < rounds; round++ {
		if trailer {
			<-theirs
		}
		if round%2 == 0 {
			// Register-probe round: no FP value of this goroutine is live
			// across the fill, so the deliberate clobber is safe, and the
			// parked pattern can only come back through the swap frame
			// (the partner overwrites the registers while we are parked).
			fpFill()
			mine <- round
			if !trailer {
				<-theirs
			}
			if fpCheck() == 0 {
				println("FAIL fpregs")
			}
		} else {
			// Live-local round: the product crosses the swap; recomputing
			// must reproduce it bit-exactly.
			k := 1.5 + float32(round)*0.25
			want := mix(vals, k)
			mine <- round
			if !trailer {
				<-theirs
			}
			if mix(vals, k) != want {
				println("FAIL locals")
			}
			// Evolve so every round compares a fresh value set.
			vals[3] += 0.125
		}
	}
	done <- 1
}

func main() {
	done := make(chan int)
	a := make(chan int)
	b := make(chan int)
	go worker(false, a, b, done, 1.5)
	go worker(true, b, a, done, 8.25)
	<-done
	<-done
	println("fpuyield ok")
}
