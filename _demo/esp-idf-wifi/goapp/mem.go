package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/pkg/esp/wifi"
)

// heapMark prints the IDF heap watermarks at one pipeline stage.
func heapMark(stage string) {
	println("[go] wifi heap "+stage,
		"free =", int(wifi.HeapFree()), "min =", int(wifi.HeapMin()),
		"| internal =", int(wifi.HeapInternalFree()),
		"| psram =", int(wifi.HeapSpiramFree()))
}

// gcExtAnchor is a Go global, hence an address in internal SRAM.
var gcExtAnchor byte

// benchSink keeps the read loops of memBench from being optimized away.
var benchSink int

// gcHeapMark prints both allocators' ledgers at one pipeline stage: the IDF
// pools and the collector's accounting, from Go and from C.
func gcHeapMark(stage string) {
	heapMark(stage)
	usable, reserved, segments, external := heapStats()
	println("[go] gc heap "+stage,
		"usable =", int(usable), "reserved =", int(reserved), "segments =", segments,
		"| psram bytes =", int(external), "(c side =", int(gcExtBytes), ")")
}

// psramProbe keeps a working set far larger than the static GC heap alive and
// re-reads every byte of it, to show the heap growing into PSRAM segments.
// Byte j of block i holds (i*7+j)&0xff, so a block read back from the wrong
// place cannot verify; a collection runs after every allocation. It reports
// whether the external heap was used.
func psramProbe() bool {
	const (
		blockSize     = 64 << 10
		blockCount    = 96         // 6 MiB in total
		minSpiramFree = 256 << 10  // below this the collector keeps the static heap only
		wantChecksum  = 0x16ca9dc5 // FNV-1a over the pattern, computed on the host
	)
	if free := wifi.HeapSpiramFree(); free < minSpiramFree {
		println("[go] psram: external heap unavailable (spiram free =", int(free),
			"), the Go heap stays in its static window")
		return false
	}
	if gcExtHeapCap == 0 {
		println("[go] psram: external heap disabled (llgo_gc_ext_heap_cap = 0), the Go heap stays in its static window")
		return false
	}
	if gcExtHeapCap == ^uintptr(0) {
		println("[go] psram: budget = 75% of the free spiram (auto)")
	} else {
		println("[go] psram: budget =", int(gcExtHeapCap), "bytes (llgo_gc_ext_heap_cap)")
	}
	gcHeapMark("before psram probe")
	dirtyChunks := dirtySpiram()

	blocks := make([][]byte, blockCount)
	var allocCycles, fillCycles, gcCycles, maxGcCycles uint64
	dirtyBytes := 0
	for i := range blocks {
		start := rdcycle()
		buf := make([]byte, blockSize)
		allocated := rdcycle()
		// New blocks must be zero whatever heap_caps_malloc handed out.
		for _, b := range buf {
			if b != 0 {
				dirtyBytes++
			}
		}
		writing := rdcycle()
		for j := range buf {
			buf[j] = byte(i*7 + j)
		}
		filled := rdcycle()
		gcCollect()
		collected := rdcycle()
		allocCycles += uint64(allocated - start)
		fillCycles += uint64(filled - writing)
		gc := uint64(collected - filled)
		gcCycles += gc
		if gc > maxGcCycles {
			maxGcCycles = gc
		}
		blocks[i] = buf
	}

	verifyStart := rdcycle()
	checksum := uint32(2166136261) // FNV-1a offset basis
	mismatches := 0
	for i, buf := range blocks {
		for j := range buf {
			b := buf[j]
			if b != byte(i*7+j) {
				mismatches++
			}
			checksum = (checksum ^ uint32(b)) * 16777619
		}
	}
	verifyCycles := uint64(rdcycle() - verifyStart)

	live := 0
	for _, buf := range blocks {
		live += len(buf)
	}
	println("[go] psram: live =", live, "bytes in", len(blocks), "blocks",
		"| checksum =", hex8(checksum), "mismatches =", mismatches)
	println("[go] psram: fresh blocks hold", dirtyBytes, "non-zero bytes; psram dirtied beforehand in",
		len(dirtyChunks), "x 64 KiB chunks")
	println("[go] psram: alloc cycles/block =", int(allocCycles/blockCount),
		"| fill cycles/block =", int(fillCycles/blockCount),
		"| verify cycles/block =", int(verifyCycles/blockCount),
		"| gc cycles avg/max =", int(gcCycles/blockCount), int(maxGcCycles))
	gcHeapMark("after psram probe")
	if mismatches != 0 || checksum != wantChecksum {
		println("[go] psram: FAILED, want checksum", hex8(wantChecksum))
	} else {
		println("[go] psram: 6 MiB live and verified across collections")
	}
	return true
}

// dirtySpiram fills as much PSRAM as heap_caps_malloc hands out with a
// non-zero pattern and releases it, so that the segments the collector takes
// afterwards are not zero by accident: Go requires new allocations to be zeroed.
func dirtySpiram() (chunks []unsafe.Pointer) {
	const chunk = 64 << 10
	for {
		ptr := heapCapsMalloc(chunk, 1<<10) // MALLOC_CAP_SPIRAM
		if ptr == nil {
			break
		}
		buffer := (*[chunk]byte)(ptr)
		for j := range buffer {
			buffer[j] = 0xa5
		}
		chunks = append(chunks, ptr)
	}
	for _, ptr := range chunks {
		heapCapsFree(ptr)
	}
	return chunks
}

// heapAddressTops samples the top byte of the PSRAM and of the internal-SRAM
// address ranges: a C allocation restricted to SPIRAM for the former, a Go
// global for the latter. The collector gives no placement API, so memBench
// tells the two apart by that byte.
func heapAddressTops() (psram, internal byte) {
	ptr := heapCapsMalloc(64, 1<<10) // MALLOC_CAP_SPIRAM
	if ptr == nil {
		return 0, 0
	}
	defer heapCapsFree(ptr)
	return byte(uintptr(ptr) >> 24), byte(uintptr(unsafe.Pointer(&gcExtAnchor)) >> 24)
}

// fiberStackReport starts n goroutines while the static window is full and
// reports where their stacks live: fiber stacks are allocated from the same
// heap, so by then they come out of a PSRAM segment.
func fiberStackReport(psramTop byte) {
	const n = 8
	tops := make(chan byte, n)
	for i := 0; i < n; i++ {
		go func() {
			var marker uint64
			tops <- byte(uintptr(unsafe.Pointer(&marker)) >> 24)
		}()
	}
	onPsram := 0
	for i := 0; i < n; i++ {
		if <-tops == psramTop {
			onPsram++
		}
	}
	println("[go] psram:", onPsram, "of", n, "goroutines started with the static window full ran on a psram stack")
}

// memBench times allocating, writing and reading back count blocks of 16 KiB
// that are all live at once, plus the collection while they are live. The
// read pass starts after the whole working set is written, so little of it is
// still in the cache and a PSRAM block is really read from PSRAM.
func memBench(stage string, count int) {
	const blockSize = 16 << 10
	psramTop, _ := heapAddressTops()
	blocks := make([][]byte, count)
	var internal, psram int
	var allocInternal, allocPsram, writeInternal, writePsram, readInternal, readPsram uint64
	var firstInternal, firstPsram uintptr
	for i := range blocks {
		allocated := rdcycle()
		buf := make([]byte, blockSize)
		start := rdcycle()
		for j := range buf {
			buf[j] = byte(i + j)
		}
		cycles := uint64(rdcycle() - start)
		first := uintptr(unsafe.Pointer(&buf[0]))
		if byte(first>>24) == psramTop {
			psram++
			allocPsram += uint64(start - allocated)
			writePsram += cycles
			if firstPsram == 0 {
				firstPsram = first
			}
		} else {
			internal++
			allocInternal += uint64(start - allocated)
			writeInternal += cycles
			if firstInternal == 0 {
				firstInternal = first
			}
		}
		blocks[i] = buf
	}
	for _, buf := range blocks {
		start := rdcycle()
		sum := 0
		for _, b := range buf {
			sum += int(b)
		}
		cycles := uint64(rdcycle() - start)
		benchSink += sum
		if byte(uintptr(unsafe.Pointer(&buf[0]))>>24) == psramTop {
			readPsram += cycles
		} else {
			readInternal += cycles
		}
	}
	start := rdcycle()
	gcCollect()
	gcCycles := uint64(rdcycle() - start)

	print("[go] bench "+stage+": ", count, "x 16 KiB")
	print(" | internal blocks =", internal)
	if internal != 0 {
		print(" at", hex8(uint32(firstInternal)),
			"(alloc", int(allocInternal/uint64(internal)),
			"write", int(writeInternal/uint64(internal)),
			"read", int(readInternal/uint64(internal)), "cycles)")
	}
	print(" | psram blocks =", psram)
	if psram != 0 {
		print(" at", hex8(uint32(firstPsram)),
			"(alloc", int(allocPsram/uint64(psram)),
			"write", int(writePsram/uint64(psram)),
			"read", int(readPsram/uint64(psram)), "cycles)")
	}
	println(" | gc cycles =", int(gcCycles))
}

// wordReadBench sums 16 KiB word by word, once in internal SRAM and once in
// PSRAM, both taken from the IDF heap directly so that the location does not
// depend on the collector. Word access is what copies and the collector use,
// and it is fast enough to be memory-bound on hardware; byte loops are not.
func wordReadBench() {
	const words = 4 << 10           // 16 KiB, small enough for the internal heap too
	internal, psram := 1<<11, 1<<10 // MALLOC_CAP_INTERNAL, MALLOC_CAP_SPIRAM
	for _, region := range []struct {
		name string
		caps uint32
	}{{"internal", uint32(internal)}, {"psram", uint32(psram)}} {
		ptr := heapCapsMalloc(4*words, region.caps)
		if ptr == nil {
			println("[go] bench word read " + region.name + ": no memory for the buffer")
			continue
		}
		start := rdcycle()
		sum := uint32(0)
		buffer := (*[words]uint32)(ptr)
		for i := range buffer {
			sum += buffer[i]
		}
		cycles := rdcycle() - start
		heapCapsFree(ptr)
		benchSink += int(sum)
		println("[go] bench word read "+region.name+" (16 KiB at "+hex8(uint32(uintptr(ptr)))+"):",
			int(cycles), "cycles,", int(cycles/words), "per word")
	}
}
