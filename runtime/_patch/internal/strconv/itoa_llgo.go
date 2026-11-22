// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

//go:build go1.27

// The decimal helpers this patch replaces have their current shape only in
// Go 1.27's src/internal/strconv; older toolchains keep the upstream
// implementations verbatim.

package strconv

//llgo:skip formatBase10 AppendUint FormatUint FormatInt

// LLGo patch: on 32-bit targets (RV32 baremetal among them) a uint64
// division compiles to the __udivdi3 compiler-rt software routine, which
// costs roughly 10x more cycles than the hardware 32-bit divider, and the
// generic slice helpers behind `append(dst, digits...)` plus the helper
// call chain add several runtime calls per conversion. The decimal entry
// points below therefore route values that fit in 32 bits to a uint32
// implementation built on the hardware divider, writing digits directly
// into the destination slice.
//
// For u < 2**32 the uint32 quotient and remainder of any division are
// mathematically identical to the uint64 ones, and the digit pair tables
// and digit counts are unchanged, so the emitted bytes are bit for bit
// identical to upstream on every platform. Values that need the full
// 64-bit range keep the upstream implementation.

// formatBase10 formats the decimal representation of u into a.
// The caller is responsible for ensuring that a is big enough to hold u.
// If a is too big, leading zeros will be filled in as needed.
func formatBase10(a []byte, u uint64) {
	nd := len(a)
	for nd >= 8 {
		// Format last 8 digits (4 pairs).
		var x3210 uint32
		if u < 1<<32 {
			w := uint32(u)
			x3210 = w % 1e8
			u = uint64(w / 1e8)
		} else {
			x3210 = uint32(u % 1e8)
			u /= 1e8
		}
		x32, x10 := x3210/1e4, x3210%1e4
		x1, x0 := (x10/100)*2, (x10%100)*2
		x3, x2 := (x32/100)*2, (x32%100)*2
		a[nd-1], a[nd-2] = smalls[x0+1], smalls[x0]
		a[nd-3], a[nd-4] = smalls[x1+1], smalls[x1]
		a[nd-5], a[nd-6] = smalls[x2+1], smalls[x2]
		a[nd-7], a[nd-8] = smalls[x3+1], smalls[x3]
		nd -= 8
	}

	x := uint32(u)
	if nd >= 4 {
		// Format last 4 digits (2 pairs).
		x10 := x % 1e4
		x /= 1e4
		x1, x0 := (x10/100)*2, (x10%100)*2
		a[nd-1], a[nd-2] = smalls[x0+1], smalls[x0]
		a[nd-3], a[nd-4] = smalls[x1+1], smalls[x1]
		nd -= 4
	}
	if nd >= 2 {
		// Format last 2 digits.
		x0 := (x % 1e2) * 2
		x /= 1e2
		a[nd-1], a[nd-2] = smalls[x0+1], smalls[x0]
		nd -= 2
	}
	if nd > 0 {
		// Format final digit.
		a[0] = byte('0' + x)
	}
}

// formatUint32 writes the decimal digits of u, which must fit in 32 bits,
// into the tail of a and returns the number of digits written. The caller
// is responsible for ensuring that len(a) is at least the number of decimal
// digits of u (10 covers every uint32); only the trailing written bytes are
// touched.
func formatUint32(a []byte, u uint32) int {
	p := len(a)
	for u >= 100 {
		x := (u % 100) * 2
		u /= 100
		p -= 2
		a[p] = smalls[x]
		a[p+1] = smalls[x+1]
	}
	// u < 100: one or two digits left.
	if u >= 10 {
		x := u * 2
		p -= 2
		a[p] = smalls[x]
		a[p+1] = smalls[x+1]
	} else {
		p--
		a[p] = byte('0' + u)
	}
	return len(a) - p
}

// numDigits32 returns the number of decimal digits of u. numDigits requires
// an argument of at least 1; u == 0 yields 1.
func numDigits32(u uint32) int {
	// Balanced compare chain: at most 4 comparisons.
	if u >= 1e4 {
		if u >= 1e7 {
			if u >= 1e9 {
				return 10
			}
			if u >= 1e8 {
				return 9
			}
			return 8
		}
		if u >= 1e6 {
			return 7
		}
		if u >= 1e5 {
			return 6
		}
		return 5
	}
	if u >= 1e2 {
		if u >= 1e3 {
			return 4
		}
		return 3
	}
	if u >= 1e1 {
		return 2
	}
	return 1
}

func AppendUint(dst []byte, i uint64, base int) []byte {
	if base == 10 {
		if i < 1<<32 {
			u := uint32(i)
			// Digit count via balanced compare chain, inlined so the
			// compiler can fold the bounds checks of the write loop.
			var nd int
			if u >= 1e4 {
				if u >= 1e7 {
					if u >= 1e9 {
						nd = 10
					} else if u >= 1e8 {
						nd = 9
					} else {
						nd = 8
					}
				} else if u >= 1e6 {
					nd = 7
				} else if u >= 1e5 {
					nd = 6
				} else {
					nd = 5
				}
			} else if u >= 1e2 {
				if u >= 1e3 {
					nd = 4
				} else {
					nd = 3
				}
			} else if u >= 1e1 {
				nd = 2
			} else {
				nd = 1
			}
			m := len(dst)
			if cap(dst)-m >= nd {
				// Write the digits into the tail of dst directly.
				dst = dst[:m+nd]
				p := m + nd
				for u >= 100 {
					x := (u % 100) * 2
					u /= 100
					p -= 2
					dst[p] = smalls[x]
					dst[p+1] = smalls[x+1]
				}
				if u >= 10 {
					x := u * 2
					p -= 2
					dst[p] = smalls[x]
					dst[p+1] = smalls[x+1]
				} else {
					p--
					dst[p] = byte('0' + u)
				}
				return dst
			}
			// Not enough capacity: go through the generic append path.
			var a [10]byte
			p := len(a) - nd
			formatUint32(a[len(a)-nd:], u)
			return append(dst, a[p:]...)
		}
		var a [24]byte
		nd := numDigits(i)
		formatBase10(a[:nd], i)
		return append(dst, a[:nd]...)
	}
	dst, _ = formatBits(dst, i, base, false, true)
	return dst
}

func FormatUint(i uint64, base int) string {
	if base == 10 {
		if i < nSmalls {
			return small(int(i))
		}
		if i < 1<<32 {
			var a [10]byte
			nd := numDigits32(uint32(i))
			formatUint32(a[len(a)-nd:], uint32(i))
			return string(a[len(a)-nd:])
		}
		var a [24]byte
		nd := numDigits(i)
		formatBase10(a[:nd], i)
		return string(a[:nd])
	}
	_, s := formatBits(nil, i, base, false, false)
	return s
}

func FormatInt(i int64, base int) string {
	if base == 10 {
		if 0 <= i && i < nSmalls {
			return small(int(i))
		}
		if i < 0 && i > -(1<<32) {
			// Negative magnitude fits in 32 bits (sign compare only, so
			// MinInt64 cannot reach this branch): "-" plus uint32 digits.
			return "-" + FormatUint(uint64(uint32(-i)), 10)
		}
		if 0 <= i && i < 1<<32 {
			var a [10]byte
			nd := numDigits32(uint32(i))
			formatUint32(a[len(a)-nd:], uint32(i))
			return string(a[len(a)-nd:])
		}
		var a [24]byte
		u := uint64(i)
		if i < 0 {
			u = -u
		}
		nd := numDigits(u)
		formatBase10(a[1:1+nd], u)
		if i < 0 {
			a[0] = '-'
			return string(a[:1+nd])
		}
		return string(a[1 : 1+nd])
	}
	_, s := formatBits(nil, uint64(i), base, i < 0, false)
	return s
}
