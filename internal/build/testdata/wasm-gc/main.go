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
	testSuspendedGRoots()
	testRecoveredRootChain()
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

type suspendedRoots struct {
	p     *payload
	slice []byte
	text  string
	any   any
	fn    func() uint64
	array [2]*payload
}

//go:noinline
func suspendedRootWorker(ready chan<- struct{}, resume <-chan struct{}, done chan<- struct{}) {
	closurePayload := &payload{value: 0x11223344}
	bytes := []byte{'w', 'a', 's', 'm'}
	roots := suspendedRoots{
		p:     &payload{value: 0x55667788},
		slice: []byte{1, 2, 3, 4},
		text:  string(bytes),
		any:   &payload{value: 0x99aabbcc},
		fn:    func() uint64 { return closurePayload.value },
		array: [2]*payload{{value: 0xddeeff00}, {value: 0x10203040}},
	}
	ready <- struct{}{}
	<-resume

	if roots.p.value != 0x55667788 ||
		len(roots.slice) != 4 || roots.slice[0] != 1 || roots.slice[3] != 4 ||
		roots.text != "wasm" ||
		roots.any.(*payload).value != 0x99aabbcc ||
		roots.fn() != 0x11223344 ||
		roots.array[0].value != 0xddeeff00 || roots.array[1].value != 0x10203040 {
		panic("suspended goroutine roots were not retained")
	}
	done <- struct{}{}
}

func testSuspendedGRoots() {
	ready := make(chan struct{})
	resume := make(chan struct{})
	done := make(chan struct{})
	go suspendedRootWorker(ready, resume, done)
	<-ready

	runtime.GC()
	for i := 0; i < 4096; i++ {
		garbage = &payload{value: uint64(i)}
	}
	garbage = nil
	close(resume)
	<-done
}

//go:noinline
func usePayload(*payload) {}

//go:noinline
func panicWithRoot(value *payload) {
	usePayload(value)
	panic("root-chain unwind")
}

//go:noinline
func clobberStack(depth int, value uint64) uint64 {
	var words [32]uint64
	for i := range words {
		words[i] = value + uint64(i)
	}
	if depth != 0 {
		return words[depth%len(words)] + clobberStack(depth-1, value+1)
	}
	return words[0]
}

func testRecoveredRootChain() {
	live := &payload{value: 0xabcdef01}
	func() {
		defer func() {
			if recover() == nil {
				panic("panic was not recovered")
			}
		}()
		panicWithRoot(live)
	}()

	_ = clobberStack(32, 1)
	runtime.GC()
	if live.value != 0xabcdef01 {
		panic("root chain was not restored after recover")
	}
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
