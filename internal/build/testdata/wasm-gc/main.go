package main

import "runtime"

type payload struct {
	value uint64
}

var (
	globalRoot *payload
	garbage    *payload
	liveChunks [][]byte
)

func main() {
	if testAlignedAlloc() == 0 {
		panic("aligned allocation failed")
	}
	testRoots()
	testReclamation()
	testHeapGrowth()
	println("wasm gc ok")
}

func testRoots() {
	globalRoot = &payload{value: 0x12345678}
	runtime.GC()
	if globalRoot.value != 0x12345678 {
		panic("global root was not retained")
	}
	globalRoot = nil
}

//go:noinline
func allocateGarbage() {
	for i := 0; i < 1024; i++ {
		garbage = &payload{value: uint64(i)}
	}
	garbage = nil
}

func testReclamation() {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	allocateGarbage()
	runtime.GC()

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if after.TotalAlloc <= before.TotalAlloc || after.Mallocs <= before.Mallocs {
		panic("allocation statistics did not advance")
	}
	if after.Frees <= before.Frees {
		panic("unreachable objects were not reclaimed")
	}
}

func testHeapGrowth() {
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	const chunkSize = 1 << 20
	chunkCount := int(before.HeapSys/chunkSize) + 8
	if chunkCount > 128 {
		panic("initial heap is too large for the bounded growth test")
	}
	liveChunks = make([][]byte, 0, chunkCount)
	for i := 0; i < chunkCount; i++ {
		chunk := make([]byte, chunkSize)
		chunk[0] = byte(i + 1)
		chunk[len(chunk)-1] = byte(i + 2)
		liveChunks = append(liveChunks, chunk)
	}

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if after.HeapSys <= before.HeapSys {
		panic("heap did not grow")
	}
	runtime.GC()
	for i, chunk := range liveChunks {
		if chunk[0] != byte(i+1) || chunk[len(chunk)-1] != byte(i+2) {
			panic("live object was corrupted during heap growth")
		}
	}
	liveChunks = nil
}
