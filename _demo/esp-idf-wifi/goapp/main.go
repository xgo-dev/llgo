// MVP demo for "llgo as an ESP-IDF component" on ESP32-S31.
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main
// component; app_main calls the generated C entry, which hands control to
// the llgo fiber scheduler (runtime.RunMain).
package main

import (
	"unsafe"

	"llgoidf/goapp/espwifi"
)

// FreeRTOS kernel bindings. llgo emits an external reference to the C symbol
// and the ESP-IDF link environment resolves it against FreeRTOS.
//
//go:linkname xTaskGetTickCount C.xTaskGetTickCount
func xTaskGetTickCount() uint32

//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

func produce(ch chan int, v int) {
	ch <- v
}

// llgoFixStreams is the assembly helper merged into llgo_out.o
// (host/llgo_syms.S): it repoints clite's stdout/stderr/stdin handles at the
// ESP-IDF picolibc streams, because clite's baremetal init fopen fails under
// IDF's VFS. Must run after package init (clite's Fopen assignment) and
// before the first print.
//
//go:linkname fixStreams C.llgoFixStreams
func fixStreams()

// heapMark prints the IDF heap watermarks at one pipeline stage.
func heapMark(stage string) {
	println("[go] wifi heap "+stage,
		"free =", int(espwifi.HeapFree()), "min =", int(espwifi.HeapMin()),
		"| internal =", int(espwifi.HeapInternalFree()),
		"| psram =", int(espwifi.HeapSpiramFree()))
}

// heapStats reports the collector's own view of the Go heap: usable data
// bytes, reserved bytes including block-state bytes, the number of segments,
// and the bytes taken from PSRAM (runtime: tinygogc/gc_segment_idf.go).
//
//go:linkname heapStats github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc.HeapStats
func heapStats() (usable, reserved uintptr, segments int, external uintptr)

// gcCollect runs a full collection now. It is tinygogc.GC, the function behind
// gcCollect() on baremetal. The demo links it directly because importing the
// runtime package drags libffi into the build (runtime/internal/lib/runtime/
// mfinal_ffi.go), and its C sources do not compile for esp32s31-idf.
//
//go:linkname gcCollect github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc.GC
func gcCollect()

// gcExtBytes mirrors the C symbol the collector writes after each PSRAM
// segment, so that the Go side and the C side of the ledger can be compared.
//
//go:linkname gcExtBytes C.llgo_gc_ext_bytes
var gcExtBytes uintptr

// gcExtHeapCap mirrors the budget the C side compiled in (app_main.c).
//
//go:linkname gcExtHeapCap C.llgo_gc_ext_heap_cap
var gcExtHeapCap uintptr

//go:linkname heapCapsMalloc C.heap_caps_malloc
func heapCapsMalloc(size, caps uint32) unsafe.Pointer

//go:linkname heapCapsFree C.heap_caps_free
func heapCapsFree(ptr unsafe.Pointer)

//go:linkname rdcycle C.llgo_demo_rdcycle
func rdcycle() uint32

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

func hex8(v uint32) string {
	const digits = "0123456789abcdef"
	var s [8]byte
	for i := 7; i >= 0; i-- {
		s[i] = digits[v&15]
		v >>= 4
	}
	return string(s[:])
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
	if free := espwifi.HeapSpiramFree(); free < minSpiramFree {
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

func main() {
	defer espwifi.Finish() // flush C stdio on every exit path
	fixStreams()
	println("[go] hello from llgo on FreeRTOS/ESP32-S31")

	// Goroutine + channel: exercises the fiber scheduler multiplexing Gs
	// inside the FreeRTOS main task.
	ch := make(chan int)
	go produce(ch, 42)
	if got := <-ch; got == 42 {
		println("[go] goroutine channel roundtrip ok")
	} else {
		println("[go] goroutine channel roundtrip got", got)
	}

	// Heap allocation: exercises tinygogc inside the IDF DRAM window.
	nums := make([]int, 16)
	sum := 0
	for i := range nums {
		nums[i] = i * i
		sum += nums[i]
	}
	println("[go] heap alloc sum =", sum)

	// C binding: FreeRTOS tick must advance across a vTaskDelay(100).
	t0 := xTaskGetTickCount()
	println("[go] tick before vTaskDelay =", int(t0))
	vTaskDelay(100)
	t1 := xTaskGetTickCount()
	delta := int(t1) - int(t0)
	println("[go] tick after vTaskDelay =", int(t1), "delta =", delta)
	if delta >= 90 && delta <= 130 {
		println("[go] vTaskDelay(100) advanced ~100 ticks: PASS")
	} else {
		println("[go] unexpected tick delta:", delta)
	}

	// ---- WiFi pipeline. Heap marks bracket every stage on the IDF heap
	// pool; the llgo GC heap is a separate static 256 KiB window and never
	// appears in these numbers.
	heapMark("baseline")
	espwifi.PrintVersion()
	println("[go] wifi: setup ...")
	if e := espwifi.Setup(); e != 0 {
		println("[go] wifi setup err =", int(e), espwifi.ErrName(e))
		println("[go] wifi FAIL setup")
		return
	}
	println("[go] wifi setup err=0")
	heapMark("init")

	if e := espwifi.EventsRegister(); e != 0 {
		println("[go] wifi events register err =", int(e), espwifi.ErrName(e))
		println("[go] wifi FAIL events")
		return
	}
	ssid, pass := espwifi.DefaultSTA() // compiled in via idfproj/main/wifi_creds.h
	// An empty SSID means a scan-only run: the driver comes up and scans, but
	// the station never associates, so no AP and no network traffic is
	// involved. The memory ledger only needs the driver running.
	noCreds := ssid == ""
	if noCreds {
		println("[go] wifi: scan-only, no credentials compiled in")
	}
	// Station mode is what makes STA_START fire; the driver's default is soft
	// AP. A scan-only run sets the mode alone, because configuring an empty
	// SSID is rejected (ESP_ERR_WIFI_SSID).
	if noCreds {
		if e := espwifi.SetModeSTA(); e != 0 {
			println("[go] wifi set_mode err =", int(e), espwifi.ErrName(e))
			println("[go] wifi FAIL set_mode")
			return
		}
	} else if e := espwifi.SetSTA(ssid, pass); e != 0 {
		println("[go] wifi set_sta err =", int(e), espwifi.ErrName(e))
		println("[go] wifi FAIL set_sta")
		return
	}
	if e := espwifi.Start(); e != 0 {
		println("[go] wifi start err =", int(e), espwifi.ErrName(e))
		println("[go] wifi FAIL start")
		return
	}
	if ev, ok := espwifi.WaitEvent([]int32{espwifi.EvStaStart}, 300); ok {
		println("[go] wifi STA_START")
		heapMark("start")
	} else {
		println("[go] wifi: no STA_START in 3s, last event kind =", int(ev.Kind))
		println("[go] wifi FAIL sta_start")
		return
	}

	// Scan (M1): emu's built-in soft AP must show up in the list.
	if e := espwifi.ScanStart(true); e != 0 {
		println("[go] wifi scan err =", int(e), espwifi.ErrName(e))
		println("[go] wifi FAIL scan")
		return
	}
	aps, e := espwifi.ScanFetch(20)
	if e != 0 {
		println("[go] wifi scan_fetch err =", int(e), espwifi.ErrName(e))
		println("[go] wifi FAIL scan_fetch")
		return
	}
	println("[go] wifi scan n =", len(aps))
	for i := range aps {
		ap := &aps[i]
		println("[go] wifi ap ssid="+ap.SsidString(),
			"ch =", int(ap.Channel), "rssi =", int(ap.Rssi))
	}
	println("[go] wifi scan done")
	heapMark("scan")

	// Memory pressure on the Go heap: once with a working set that fits the
	// static window, then 6 MiB live (PSRAM segments), then a larger working
	// set again now that the heap spans both kinds of memory.
	memBench("static heap", 4)
	if psramProbe() {
		psramTop, _ := heapAddressTops()
		memBench("psram segments", 12)
		gcHeapMark("after psram bench")
		fiberStackReport(psramTop)
	}
	wordReadBench()
	if noCreds {
		println("[go] wifi: scan-only run complete, no association attempted")
		return
	}

	// Connect + DHCP (M2): esp_wifi_connect attempts once; a failed
	// association/handshake arrives as EvStaDisconnected with a reason code.
	connected := false
	for attempt := 1; attempt <= 3 && !connected; attempt++ {
		if e := espwifi.Connect(); e != 0 {
			println("[go] wifi connect err =", int(e), espwifi.ErrName(e))
			println("[go] wifi FAIL connect")
			return
		}
		ev, ok := espwifi.WaitEvent([]int32{espwifi.EvStaConnected, espwifi.EvStaDisconnected}, 300)
		if ok && ev.Kind == espwifi.EvStaConnected {
			connected = true
			println("[go] wifi STA_CONNECTED")
		} else {
			println("[go] wifi disconnect reason =", int(ev.Aux), "attempt =", attempt)
		}
	}
	if !connected {
		println("[go] wifi FAIL no_ap")
		return
	}

	// DHCP: the GOT_IP event carries the address; GetIP polls esp_netif as
	// the fallback read.
	ev, _ := espwifi.WaitEvent([]int32{espwifi.EvGotIP}, 600)
	ip := ev.IP
	if ip == 0 {
		ip = espwifi.GetIP()
	}
	if ip == 0 {
		println("[go] wifi FAIL no_ip")
		return
	}
	println("[go] wifi GOT_IP ip=" + espwifi.IPString(ip))
	println("[go] wifi PASS ip=" + espwifi.IPString(ip))
	heapMark("dhcp")

	// Steady state: the WiFi stack keeps module-specific buffers parked
	// while connected; give it 10 seconds and look again.
	vTaskDelay(1000)
	heapMark("steady+10s")
	println("[go] wifi done")

	println("[go] done")
}
