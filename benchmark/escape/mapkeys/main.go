package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

type key struct{ A, B uint64 }
type largeKey [33]uint64

//go:noinline
func lookup(m map[key]int, k key) int { return m[k] }

//go:noinline
func update(m map[key]int, k key, v int) int { m[k] = v; return v }

//go:noinline
func remove(m map[key]int, k key) int { delete(m, k); return len(m) }

//go:noinline
func indirect(m map[largeKey]int, k largeKey) int { return m[k] }

//go:noinline
func scalar(m map[uint64]int, k uint64) int { return m[k] }

//go:noinline
func mapRange(m map[key]int) int {
	sum := 0
	for k, v := range m {
		sum += int(k.A+k.B) + v
	}
	return sum
}

//go:noinline
func stringRange(s string) int {
	sum := 0
	for _, r := range s {
		sum += int(r)
	}
	return sum
}

type result struct {
	Name     string
	Calls    int
	Bytes    uint64
	NSPerOp  float64
	Checksum int
}

func measure(name string, operation func(int) int, want int) result {
	const calls = 100000
	for i := 0; i < 2000; i++ {
		operation(i)
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	sum := 0
	for i := 0; i < calls; i++ {
		sum += operation(i)
	}
	elapsed := time.Since(started)
	runtime.ReadMemStats(&after)
	if sum != want {
		panic(fmt.Sprintf("%s checksum %d, want %d", name, sum, want))
	}
	return result{name, calls, after.TotalAlloc - before.TotalAlloc, float64(elapsed.Nanoseconds()) / calls, sum}
}

func main() {
	const calls = 100000
	k := key{1, 2}
	m := map[key]int{k: 42}
	big := largeKey{0: 1, 32: 2}
	large := map[largeKey]int{big: 42}
	fast := map[uint64]int{7: 42}
	ranged := map[key]int{k: 39}
	sum := 0
	for i := 0; i < calls; i++ {
		sum += (i & 127) + 1
	}
	results := []result{
		measure("Struct key lookup", func(int) int { return lookup(m, k) }, 42*calls),
		measure("Struct key update", func(i int) int { return update(m, k, (i&127)+1) }, sum),
		measure("Struct key delete (missing)", func(i int) int { return remove(m, key{uint64(i) + 10, 2}) }, calls),
		measure("Indirect key lookup", func(int) int { return indirect(large, big) }, 42*calls),
		measure("Map range", func(int) int { return mapRange(ranged) }, 42*calls),
		measure("String range", func(int) int { return stringRange("a世z") }, 20209*calls),
		measure("Scalar key control", func(int) int { return scalar(fast, 7) }, 42*calls),
	}
	if len(m) != 1 || m[k] != ((calls-1)&127)+1 {
		panic("map contents corrupted")
	}
	if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
		panic(err)
	}
}
