package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sync"
	"testing"
)

type cell struct {
	mu    sync.Mutex
	value uint64
}

//go:noinline
func (c *cell) explicit() uint64 {
	c.mu.Lock()
	c.value++
	value := c.value
	c.mu.Unlock()
	return value
}

//go:noinline
func (c *cell) deferred() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
	return c.value
}
func main() {
	testing.Init()
	if err := flag.Set("test.benchtime", "500ms"); err != nil {
		panic(err)
	}
	for _, name := range []string{"explicit", "deferred"} {
		c := new(cell)
		call := c.explicit
		if name == "deferred" {
			call = c.deferred
		}
		result := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			c.value = 0
			for i := 0; i < b.N; i++ {
				if call() != uint64(i+1) {
					panic("bad result")
				}
			}
		})
		data, err := json.Marshal(map[string]any{"case": name, "nsPerOp": float64(result.T.Nanoseconds()) / float64(result.N), "reportedMallocsPerOp": result.AllocsPerOp(), "reportedBytesPerOp": result.AllocedBytesPerOp(), "iterations": result.N})
		if err != nil {
			panic(err)
		}
		fmt.Println(string(data))
	}
}
