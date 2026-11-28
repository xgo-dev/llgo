package main

import (
	"os"
	"runtime"
	"strconv"
)

func main() {
	if len(os.Args) != 2 {
		panic("expected allocation size")
	}
	size, err := strconv.Atoi(os.Args[1])
	if err != nil || size <= 0 {
		panic("invalid allocation size")
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	value := make([]byte, size)
	value[0], value[size/2], value[size-1] = 0x12, 0x34, 0x56
	runtime.GC()
	if value[0] != 0x12 || value[size/2] != 0x34 || value[size-1] != 0x56 {
		panic("arena allocation did not survive collection")
	}
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		panic("arena allocation was not collected")
	}
	// One appropriately sized arena must suffice; repeatedly adding arenas
	// that are too small must not consume the module's memory limit.
	if after.HeapSys > before.HeapSys+uint64(size)+(1<<20) {
		panic("arena growth exceeded allocation plus metadata allowance")
	}
	runtime.KeepAlive(value)
	println("wasi gc arena boundary ok")
}
