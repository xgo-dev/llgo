// Differential test for 64-bit integer division and remainder on every
// operand shape the riscv32 runtime dispatches on: dividend and divisor both
// inside 32 bits (one hardware divider), divisor below 2**16 (exact
// base-2**16 digits), divisor in [2**16, 2**32) (normalized Knuth digits), and
// full 64-bit divisors (Knuth Algorithm D), each in signed and unsigned form.
//
// The reference below divides bit by bit with shifts, compares and
// subtracts, so it never reaches the __divdi3/__moddi3/__udivdi3/__umoddi3
// libcalls that native `/` and `%` compile to on this target: a mismatch means
// the runtime's division is wrong, not that both sides share a bug. Hosted and
// baremetal runs must print the same lines, which is what expect.txt pins.
package main

// Two's-complement bounds. They are variables, not constants, so the signed
// values can be reinterpreted as uint64 operands below.
var (
	maxU64 = uint64(1<<64 - 1)
	maxI64 = uint64(1<<63 - 1)
	minI64 = int64(-1 << 63)
)

var (
	digest uint64 // order-sensitive fold of every checked quotient/remainder
	pairs  int
	fails  int
)

// reserve turns the result of a comparison into output, so a wrong result
// cannot be optimized away or merely counted.
func reserve(q, r uint64) {
	digest = (digest^q)*1099511628211 + r
}

// udivmodRef divides n by d bit by bit, restoring a partial remainder of at
// most 64 bits. Requires d != 0.
func udivmodRef(n, d uint64) (q, r uint64) {
	if d > 1<<63 {
		// A divisor above 2**63 leaves a quotient of at most 1, and the
		// restoring loop below would overflow its 64-bit remainder.
		if n >= d {
			return 1, n - d
		}
		return 0, n
	}
	for i := 63; i >= 0; i-- {
		r <<= 1
		r |= (n >> uint(i)) & 1
		if r >= d {
			r -= d
			q |= uint64(1) << uint(i)
		}
	}
	return q, r
}

// sdivmodRef is Go's signed contract on top of udivmodRef: the quotient
// truncates toward zero, the remainder takes the dividend's sign, and
// minInt64 / -1 wraps to minInt64.
func sdivmodRef(n, d int64) (q, r int64) {
	un, ud := uint64(n), uint64(d)
	if n < 0 {
		un = -un
	}
	if d < 0 {
		ud = -ud
	}
	uq, ur := udivmodRef(un, ud)
	q = int64(uq)
	if (n < 0) != (d < 0) {
		q = -q
	}
	r = int64(ur)
	if n < 0 {
		r = -r
	}
	return q, r
}

func checkU(n, d uint64) {
	wq, wr := udivmodRef(n, d)
	gq, gr := n/d, n%d
	pairs++
	reserve(gq, gr)
	if gq != wq || gr != wr {
		fails++
		println("MISMATCH uint64", n, "/", d, "got", gq, gr, "want", wq, wr)
	}
}

func checkI(n, d int64) {
	wq, wr := sdivmodRef(n, d)
	gq, gr := n/d, n%d
	pairs++
	reserve(uint64(gq), uint64(gr))
	if gq != wq || gr != wr {
		fails++
		println("MISMATCH int64", n, "/", d, "got", gq, gr, "want", wq, wr)
	}
}

// check runs one operand pair through both signednesses unless the divisor is
// zero (a Go division by zero panics by design; the runtime guard for that is
// covered by arith-divrem).
func check(n, d uint64) {
	if d == 0 {
		return
	}
	checkU(n, d)
	checkI(int64(n), int64(d))
}

func main() {
	// Boundaries: zero quotient, unit quotient, 2**32 and 2**16 crossing,
	// max values, and the minInt64/-1 overflow pair.
	nums := [...]uint64{
		0, 1, 2, 3, 7, 9, 10,
		1<<16 - 1, 1 << 16, 1<<16 + 1,
		1<<32 - 1, 1 << 32, 1<<32 + 1,
		123456789, 1234567890123456789,
		1<<62 - 1, 1 << 62, 1<<63 - 1, 1 << 63, 1<<63 + 1,
		maxU64, maxU64 - 1,
		uint64(minI64), uint64(minI64) + 1, maxI64,
	}
	dens := [...]uint64{
		0, 1, 2, 3, 7, 10, 100, 1000, 10000,
		1<<16 - 1, 1 << 16, 1<<16 + 1,
		1<<24 + 1, 1000000007, 998244353,
		1<<31 - 1, 1 << 31, 1<<32 - 1, 1 << 32, 1<<32 + 1,
		maxI64, 1 << 63, maxU64 - 1, maxU64,
	}
	for _, n := range nums {
		for _, d := range dens {
			check(n, d)
		}
	}

	// Random vectors from a fixed-seed xorshift so the run is reproducible
	// and the golden output is meaningful.
	x := uint64(0x243f6a8885a308d3)
	next := func() uint64 {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		return x
	}
	for i := 0; i < 512; i++ {
		n := next()
		// Half the divisors are masked to a single base-2**16 digit, the
		// shape the exact short-division path serves.
		d := next()
		if i&1 == 0 {
			d &= 0xFFFF
		}
		check(n, d)
	}
	// Divisors next to powers of two, where the Knuth estimate is most
	// likely to need its correction step.
	for sh := 0; sh < 64; sh++ {
		base := uint64(1) << uint(sh)
		n := next()
		check(n, base)
		if base > 1 {
			check(n, base-1)
		}
		if base < maxU64 {
			check(n, base+1)
		}
		check(maxU64-base, base)
	}

	if fails != 0 {
		println("divmod64 FAILED pairs=", pairs, "fails=", fails)
		return
	}
	// Printed as a full 64-bit value: int is 64 bits hosted and 32 bits on
	// the baremetal targets, so narrowing here would print differently.
	println("divmod64 ok pairs=", pairs, "digest=", digest)
}
