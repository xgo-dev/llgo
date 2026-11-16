//go:build goexperiment.simd && (amd64 || arm64)

package simd_test

import (
	"strings"
	"testing"
)

type interleaveInteger interface {
	int8 | uint8 | int16 | uint16 | int32 | uint32 | int64 | uint64
}

func testInterleave[T interleaveInteger](t *testing.T, n, group int, op string, apply func([]T, []T, []T)) {
	t.Helper()
	x, y, out := make([]T, n), make([]T, n), make([]T, n)
	for sample := 0; sample < 4; sample++ {
		for i := range x {
			v := uint64(i + 1)
			if sample == 1 {
				v = ^v
			}
			if sample == 2 {
				v = uint64(1) << uint(i%64)
			}
			if sample == 3 {
				v = 0x123456789abcdef0 + uint64(i)*0x1111111111111111
			}
			x[i], y[i] = T(v), T(v^0x5aa55aa55aa55aa5)
		}
		apply(x, y, out)
		var want []T
		for base := 0; base < n; base += group {
			a, b := x[base:base+group], y[base:base+group]
			switch op {
			case "ConcatEven", "ConcatOdd":
				start := 0
				if op == "ConcatOdd" {
					start = 1
				}
				for _, src := range [][]T{a, b} {
					for i := start; i < group; i += 2 {
						want = append(want, src[i])
					}
				}
			case "InterleaveEven", "InterleaveOdd":
				start := 0
				if op == "InterleaveOdd" {
					start = 1
				}
				for i := start; i < group; i += 2 {
					want = append(want, a[i], b[i])
				}
			default:
				start := 0
				if strings.Contains(op, "Hi") {
					start = group / 2
				}
				for i := start; i < start+group/2; i++ {
					want = append(want, a[i], b[i])
				}
			}
		}
		for i, v := range out {
			if v != want[i] {
				t.Fatalf("sample %d lane %d: got %v want %v", sample, i, out, want)
			}
		}
	}
}
