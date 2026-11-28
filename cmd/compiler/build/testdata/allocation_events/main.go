package main

import (
	"fmt"
	"runtime"
)

var bytesSink []byte
var pointerSink *node
var before, after runtime.MemStats

type node struct {
	next *node
	data int
}

//go:noinline
func allocate(size int) {
	if size == 0 {
		pointerSink = &node{next: pointerSink, data: 42}
	} else {
		bytesSink = make([]byte, size)
	}
}

func main() {
	// Run without testing's background goroutines, whose allocations would
	// contribute to the same process-wide counter during the measurement.
	for _, size := range []int{1, 1024, 65536, 0} {
		for i := 0; i < 100; i++ {
			allocate(size)
		}
		runtime.ReadMemStats(&before)
		runtime.ReadMemStats(&after)
		runtime.ReadMemStats(&before)
		const count = 1000
		for i := 0; i < count; i++ {
			allocate(size)
		}
		runtime.ReadMemStats(&after)
		if got := after.Mallocs - before.Mallocs; got != count {
			panic(fmt.Sprintf("size=%d: %d allocations report %d events", size, count, got))
		}
	}
	fmt.Println("allocation events: OK")
}
