package main

import (
	"encoding/json"
	"os"
	"runtime"
	"time"
)

type state struct{ Words [14]uint64 }

//go:noinline
func byValue(s state) state {
	s.Words[0] += s.Words[13]
	s.Words[7] ^= s.Words[0]
	return s
}

//go:noinline
func modify(s *state) {
	s.Words[0] += s.Words[13]
	s.Words[7] ^= s.Words[0]
}

//go:noinline
func borrowed(s state) state {
	modify(&s)
	return s
}

//go:noinline
func run(fn func(state) state, count int) uint64 {
	var s state
	s.Words[0], s.Words[7], s.Words[13] = 17, 23, 3
	for i := 0; i < count; i++ {
		s = fn(s)
	}
	return s.Words[0] ^ s.Words[7] ^ s.Words[13]
}

func main() {
	const count = 100000
	expected := uint64(17)
	other := uint64(23)
	for i := 0; i < count; i++ {
		expected += 3
		other ^= expected
	}
	expected ^= other ^ 3
	type row struct {
		Case     string `json:"case"`
		Sample   int    `json:"sample"`
		Bytes    uint64 `json:"bytes"`
		Ns       int64  `json:"ns"`
		Checksum uint64 `json:"checksum"`
	}
	var rows []row
	for _, c := range []struct {
		name string
		fn   func(state) state
	}{{"value", byValue}, {"borrow", borrowed}} {
		if got := run(c.fn, count); got != expected {
			panic("warmup checksum")
		}
		for sample := 0; sample < 7; sample++ {
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			start := time.Now()
			got := run(c.fn, count)
			elapsed := time.Since(start).Nanoseconds()
			runtime.ReadMemStats(&after)
			if got != expected {
				panic("checksum")
			}
			rows = append(rows, row{c.name, sample, after.TotalAlloc - before.TotalAlloc, elapsed, got})
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Compiler string `json:"compiler"`
		Count    int    `json:"count"`
		Rows     []row  `json:"rows"`
	}{runtime.Compiler, count, rows}); err != nil {
		panic(err)
	}
}
