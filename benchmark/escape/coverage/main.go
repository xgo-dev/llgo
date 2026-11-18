// Allocation probes shared by LLGo and cmd/compile. Run with -n=100000.
package main

import (
	"flag"
	"fmt"
	"runtime"
	"time"
)

var result int

//go:noinline
func bytesUse(b []byte) int { return int(b[0]) + len(b) }

//go:noinline
func stringUse(s string) int { return int(s[0]) + len(s) }

//go:noinline
func fixedSlice(n int) int {
	b := make([]byte, 16)
	b[0] = byte(n)
	return bytesUse(b)
}

//go:noinline
func dynamicSlice(n, size int) int {
	b := make([]byte, size)
	b[0] = byte(n)
	return bytesUse(b)
}

//go:noinline
func fixedCapacity(n int) int {
	b := make([]byte, n, 16)
	b[0] = 3
	return bytesUse(b)
}

//go:noinline
func toBytes(s string) int {
	b := []byte(s)
	b[0]++
	return bytesUse(b)
}

//go:noinline
func fromBytes(b []byte) int { return stringUse(string(b)) }

//go:noinline
func readOnlyBytes(s string) int { return bytesUse([]byte(s)) }

//go:noinline
func temporaryStringCompare(b []byte, s string) int {
	if string(b) == s {
		return 1
	}
	return 0
}

//go:noinline
func temporaryMapLookup(m map[string]int, b []byte) int { return m[string(b)] }

//go:noinline
func toRunes(s string) int {
	r := []rune(s)
	return int(r[0]) + len(r)
}

//go:noinline
func fromRunes(r []rune) int { return stringUse(string(r)) }

//go:noinline
func concat(a, b string) int { return stringUse(a + b) }

//go:noinline
func smallMap(n int) int {
	m := make(map[int]int, 8)
	m[0] = n
	return m[0]
}

type reader interface{ Read() int }
type value struct{ p *int }

func (v value) Read() int { return *v.p }

//go:noinline
func interfaceCall(n int) int {
	v := value{&n}
	var r reader = v
	return r.Read()
}

func main() {
	n := flag.Int("n", 100000, "calls per probe")
	flag.Parse()
	text := "hello世"
	largeText := string(make([]byte, 128))
	largeBytes := []byte(largeText)
	lookup := map[string]int{largeText: 2}
	bytes := []byte(text)
	runes := []rune(text)
	probes := []struct {
		name string
		call func(int) int
	}{
		{"fixed-slice", fixedSlice},
		{"dynamic-slice-small", func(n int) int { return dynamicSlice(n, 16) }},
		{"dynamic-slice-large", func(n int) int { return dynamicSlice(n, 128) }},
		{"fixed-capacity", func(int) int { return fixedCapacity(12) }},
		{"string-to-bytes", func(int) int { return toBytes(text) }},
		{"readonly-string-bytes", func(int) int { return readOnlyBytes(largeText) }},
		{"temporary-string-compare", func(int) int { return temporaryStringCompare(largeBytes, largeText) }},
		{"temporary-map-lookup", func(int) int { return temporaryMapLookup(lookup, largeBytes) }},
		{"bytes-to-string", func(int) int { return fromBytes(bytes) }},
		{"string-to-runes", func(int) int { return toRunes(text) }},
		{"runes-to-string", func(int) int { return fromRunes(runes) }},
		{"string-concat", func(int) int { return concat(text, text) }},
		{"small-map", smallMap},
		{"known-interface", interfaceCall},
	}
	fmt.Println("probe,calls,allocated_bytes,ns_per_call,checksum")
	for _, p := range probes {
		for i := range 100 {
			result = p.call(i)
		}
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		start := time.Now()
		sum := 0
		for i := range *n {
			sum += p.call(i)
		}
		elapsed := time.Since(start)
		runtime.ReadMemStats(&after)
		result = sum
		fmt.Printf("%s,%d,%d,%.2f,%d\n", p.name, *n, after.TotalAlloc-before.TotalAlloc, float64(elapsed.Nanoseconds())/float64(*n), result)
	}
}
