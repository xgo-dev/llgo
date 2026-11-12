package stringequal

import (
	"runtime"
	"strconv"
	"testing"
)

var stringEqualityBenchmarkResult int

//go:noinline
func benchmarkEqualStrings(a, b string) bool { return a == b }

//go:noinline
func benchmarkEqualBytes(a []byte, b string) bool { return string(a) == b }

//go:noinline
func benchmarkOwnedString(b []byte) string { return string(b) }

func BenchmarkStringEquality(b *testing.B) {
	implementation := "Go"
	if runtime.Compiler == "llgo" {
		implementation = "LLGo"
	}
	for _, n := range []int{8, 128, 1024} {
		for _, kind := range []string{"equal", "first", "last"} {
			x, y := make([]byte, n+1)[1:], make([]byte, n+2)[2:]
			for i := range x {
				x[i], y[i] = byte(i*17+3), byte(i*17+3)
			}
			if kind == "first" {
				y[0] ^= 1
			}
			if kind == "last" {
				y[n-1] ^= 1
			}
			xs, ys := benchmarkOwnedString(x), benchmarkOwnedString(y)
			for _, test := range []struct {
				name string
				call func() bool
			}{
				{"strings", func() bool { return benchmarkEqualStrings(xs, ys) }},
				{"bytes", func() bool { return benchmarkEqualBytes(x, ys) }},
			} {
				b.Run(test.name+"/"+kind+"/"+strconv.Itoa(n)+"/"+implementation, func(b *testing.B) {
					sum := 0
					for i := 0; i < b.N; i++ {
						if test.call() {
							sum++
						}
					}
					b.StopTimer()
					want := 0
					if kind == "equal" {
						want = b.N
					}
					if sum != want {
						b.Fatalf("comparison checksum %d, want %d", sum, want)
					}
					stringEqualityBenchmarkResult = sum
				})
			}
		}
	}
}
