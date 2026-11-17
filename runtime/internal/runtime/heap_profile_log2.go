// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

//go:build !nogc && !baremetal

package runtime

// heapProfileLog2 implements a fast approximation to the base 2 log of a
// float64. This is used to compute a geometric distribution for heap
// sampling, without introducing dependencies into package math. This
// uses a very rough approximation using the float64 exponent and the
// first 25 bits of the mantissa. The top 5 bits of the mantissa are
// used to load limits from a table of constants and the rest are used
// to scale linearly between them.
func heapProfileLog2(x float64) float64 {
	const fastlogScaleBits = 20
	const fastlogScaleRatio = 1.0 / (1 << fastlogScaleBits)

	xBits := heapProfileFloat64bits(x)
	// Extract the exponent from the IEEE float64, and index a constant
	// table with the first 10 bits from the mantissa.
	xExp := int64((xBits>>52)&0x7FF) - 1023
	xManIndex := (xBits >> (52 - heapProfileLogBits)) % (1 << heapProfileLogBits)
	xManScale := (xBits >> (52 - heapProfileLogBits - fastlogScaleBits)) % (1 << fastlogScaleBits)

	low, high := heapProfileLog2Table[xManIndex], heapProfileLog2Table[xManIndex+1]
	return float64(xExp) + low + (high-low)*float64(xManScale)*fastlogScaleRatio
}
