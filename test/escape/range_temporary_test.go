package escape

import (
	"runtime"
	"strings"
	"testing"
)

//go:noinline
func mapIterationSum(m map[int]int) int {
	sum := 0
	for k, v := range m {
		sum += k + v
	}
	return sum
}

//go:noinline
func stringIterationSum(s string) int {
	sum := 0
	for _, r := range s {
		sum += int(r)
	}
	return sum
}

func TestRangeTemporaryRestartAndGC(t *testing.T) {
	// The inner range executes again in one invocation. A nil/empty map must
	// not reuse fields left by the preceding iterator.
	maps := []map[int]int{{1: 10, 2: 20}, nil, {}, {3: 30}}
	want := []int{33, 0, 0, 33}
	for i, m := range maps {
		sum := 0
		for k, v := range m {
			runtime.GC()
			sum += k + v
		}
		if sum != want[i] {
			t.Fatalf("range %d: got %d, want %d", i, sum, want[i])
		}
	}
	texts := []string{strings.Repeat("a", 65) + "世\xff", "", "z"}
	for i, s := range texts {
		sum := 0
		for offset, r := range s {
			if offset%16 == 0 {
				runtime.GC()
			}
			sum += int(r)
		}
		if expected := []int{65*97 + 19990 + 65533, 0, 122}[i]; sum != expected {
			t.Fatalf("string range %d: got %d, want %d", i, sum, expected)
		}
	}
	count := 0
	for range maps[0] {
		for range maps[0] {
			runtime.GC()
			count++
		}
	}
	if count != 4 {
		t.Fatalf("nested ranges share iterator storage: got %d iterations", count)
	}
}

func TestRangeTemporaryMutationAndBreak(t *testing.T) {
	m := map[int]int{1: 10, 2: 20, 3: 30}
	count := 0
	for range m {
		clear(m)
		runtime.GC()
		count++
	}
	if count != 1 {
		t.Fatalf("cleared map yielded %d entries, want 1", count)
	}
	for i := 0; i < 3; i++ {
		for range map[int]int{i: i} {
			runtime.GC()
			break
		}
	}
}

//go:noinline
func rangeWithFullStackBudget(m map[int]int, s string, padding [65536]byte) int {
	runtime.KeepAlive(padding)
	sum := int(padding[0])
	for k, v := range m {
		runtime.GC()
		sum += k + v
	}
	for _, r := range s {
		runtime.GC()
		sum += int(r)
	}
	return sum
}

func TestRangeStackBudgetPreservesOwnership(t *testing.T) {
	var padding [65536]byte
	padding[0] = 1
	if got := rangeWithFullStackBudget(map[int]int{1: 41}, "z", padding); got != 165 {
		t.Fatalf("range with full stack budget: got %d, want 165", got)
	}
}
