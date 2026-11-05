package bits_test

import (
	"math/bits"
	"testing"
)

// Rotate one bit at a time, independently of the production shift formula.
func rotateOneBitOracle(value uint64, count, width int) uint64 {
	mask := ^uint64(0) >> uint(64-width)
	value &= mask
	count %= width
	if count < 0 {
		count += width
	}
	for range count {
		value = ((value << 1) | (value >> uint(width-1))) & mask
	}
	return value
}

//go:noinline
func invokeRotate[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](fn func(T, int) T, value T, count int) T {
	return fn(value, count)
}

func TestRotateLeftSignedCountsAndFunctionValues(t *testing.T) {
	counts := []int{-int(^uint(0)>>1) - 1, int(^uint(0) >> 1)}
	for count := -129; count <= 129; count++ {
		counts = append(counts, count)
	}
	values := []uint64{0, 1, ^uint64(0), 1 << 63, 0x123456789abcdef0}
	seed := uint64(0x123456789abcdef0)
	for range 32 {
		seed = seed*6364136223846793005 + 1442695040888963407
		values = append(values, seed)
	}
	for _, value := range values {
		for _, count := range counts {
			want8 := uint8(rotateOneBitOracle(value, count, 8))
			want16 := uint16(rotateOneBitOracle(value, count, 16))
			want32 := uint32(rotateOneBitOracle(value, count, 32))
			want64 := rotateOneBitOracle(value, count, 64)
			wantWord := uint(rotateOneBitOracle(value, count, bits.UintSize))
			if bits.RotateLeft8(uint8(value), count) != want8 ||
				bits.RotateLeft16(uint16(value), count) != want16 ||
				bits.RotateLeft32(uint32(value), count) != want32 ||
				bits.RotateLeft64(value, count) != want64 ||
				bits.RotateLeft(uint(value), count) != wantWord {
				t.Fatalf("direct rotate: value=%x count=%d", value, count)
			}
			if invokeRotate(bits.RotateLeft8, uint8(value), count) != want8 ||
				invokeRotate(bits.RotateLeft16, uint16(value), count) != want16 ||
				invokeRotate(bits.RotateLeft32, uint32(value), count) != want32 ||
				invokeRotate(bits.RotateLeft64, value, count) != want64 ||
				invokeRotate(bits.RotateLeft, uint(value), count) != wantWord {
				t.Fatalf("function value rotate: value=%x count=%d", value, count)
			}
		}
	}
}

func TestRotateLeftCallEvaluation(t *testing.T) {
	trace := 0
	value := func() uint32 { trace = trace*10 + 1; return 0x12345678 }
	count := func() int { trace = trace*10 + 2; return -int(^uint(0) >> 1) }
	want := uint32(rotateOneBitOracle(0x12345678, -int(^uint(0)>>1), 32))
	if got := bits.RotateLeft32(value(), count()); got != want || trace != 12 {
		t.Fatalf("direct arguments: result=%x trace=%d", got, trace)
	}
	trace = 0
	func() {
		defer bits.RotateLeft32(value(), count())
		if trace != 12 {
			t.Fatal("defer arguments evaluated late or out of order", trace)
		}
	}()
	trace = 0
	go bits.RotateLeft32(value(), count())
	if trace != 12 {
		t.Fatal("go arguments evaluated late or out of order", trace)
	}
	trace = 0
	result := make(chan uint32, 1)
	go func(x uint32, k int) { result <- bits.RotateLeft32(x, k) }(value(), count())
	if got := <-result; got != want || trace != 12 {
		t.Fatalf("goroutine rotate: result=%x trace=%d", got, trace)
	}
}
