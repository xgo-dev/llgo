package gotest

import "testing"

type aggregateReturnPair struct {
	Value uint64
	Flag  bool
}

//go:noinline
func aggregateReturnSnapshot(src *aggregateReturnPair, first bool) aggregateReturnPair {
	saved := *src
	*src = aggregateReturnPair{99, !saved.Flag}
	if first {
		return saved
	}
	saved.Value++
	return saved
}

//go:noinline
func aggregateReturnIntegers(x uint64, y uint32) (uint64, uint32) {
	if x < uint64(y) {
		return x + 1, y
	}
	return x - 1, y + 1
}

func TestAggregateReturnSnapshots(t *testing.T) {
	for _, first := range []bool{false, true} {
		for _, flag := range []bool{false, true} {
			for _, value := range []uint64{0, 31, 1 << 32, 1<<63 + 3, ^uint64(0)} {
				src := aggregateReturnPair{value, flag}
				want := src
				if !first {
					want.Value++
				}
				if got := aggregateReturnSnapshot(&src, first); got != want {
					t.Fatalf("snapshot(%d, %v, %v) = %v, want %v", value, flag, first, got, want)
				}
				if src != (aggregateReturnPair{99, !flag}) {
					t.Fatalf("source mutation lost: %v", src)
				}
			}
		}
	}
	for _, x := range []uint64{0, 31, 1 << 32, 1<<63 + 3, ^uint64(0)} {
		for _, y := range []uint32{0, 31, 1<<31 + 3, ^uint32(0)} {
			gotX, gotY := aggregateReturnIntegers(x, y)
			wantX, wantY := x+1, y
			if x >= uint64(y) {
				wantX, wantY = x-1, y+1
			}
			if gotX != wantX || gotY != wantY {
				t.Fatalf("integer pair(%d, %d) = (%d, %d), want (%d, %d)", x, y, gotX, gotY, wantX, wantY)
			}
		}
	}
}
