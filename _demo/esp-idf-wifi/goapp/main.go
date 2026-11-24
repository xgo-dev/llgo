// MVP demo for "llgo as an ESP-IDF component" on ESP32-S31.
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main
// component; app_main calls the generated C entry, which hands control to
// the llgo fiber scheduler (runtime.RunMain).
package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/gpio"
	"github.com/xgo-dev/llgo/pkg/esp/i2c"
	"github.com/xgo-dev/llgo/pkg/esp/i2s"
	"github.com/xgo-dev/llgo/pkg/esp/net"
	"github.com/xgo-dev/llgo/pkg/esp/tls"
	"github.com/xgo-dev/llgo/pkg/esp/wifi"
	"github.com/xgo-dev/llgo/pkg/esp/ws"
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
		"free =", int(wifi.HeapFree()), "min =", int(wifi.HeapMin()),
		"| internal =", int(wifi.HeapInternalFree()),
		"| psram =", int(wifi.HeapSpiramFree()))
}

// heapStats reports the collector's own view of the Go heap: usable data
// bytes, reserved bytes including block-state bytes, the number of segments,
// and the bytes taken from PSRAM (runtime: tinygogc/gc_segment_idf.go).
//
//go:linkname heapStats github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc.HeapStats
func heapStats() (usable, reserved uintptr, segments int, external uintptr)

// gcCollect runs a full collection now. It is tinygogc.GC, the function behind
// gcCollect() on baremetal. Kept as a direct linkname: importing the whole
// runtime package for one function would pull its finalizer/reflect machinery
// into the image (the libffi build gap that once forced this workaround is
// fixed, but the image-size reason stands).
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

// dec32 formats v in decimal (negatives with a leading minus). println has
// no number form without a separating space, and the udp probe's golden
// lines pin the exact "n=128" shape, so the probe concatenates this in.
func dec32(v int) string {
	neg := v < 0
	x := uint32(v)
	if neg {
		x = -uint32(v)
	}
	if x == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for x > 0 {
		i--
		buf[i] = byte('0' + x%10)
		x /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// errCode extracts the esp pipeline code an error carries, 0 for nil - the
// same number the pre-error API printed ("err = 0").
func errCode(err error) int32 {
	if err == nil {
		return 0
	}
	if e, ok := err.(*esperr.Error); ok {
		return e.Code
	}
	return -1
}

// errText formats an esp error as "<decimal code> <name>", exactly the two
// tokens the pre-error probes printed side by side via int(e) and
// tls.ErrName(e) ("32258 TLS_ERR_CERT_VERIFY"), so the acceptance-golden
// diagnostic lines stay byte-identical. Errors without a name (the
// peripheral esp_err wrappers) render as the bare code; a non-esp error
// degrades to its own Error() text.
func errText(err error) string {
	if e, ok := err.(*esperr.Error); ok {
		if e.Name != "" {
			return dec32(int(e.Code)) + " " + e.Name
		}
		return dec32(int(e.Code))
	}
	if err == nil {
		return "0"
	}
	return err.Error()
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

// ---- Peripheral probes (pkg/esp/gpio, pkg/esp/i2c, pkg/esp/i2s).
//
// Each probe prints one [go] periph line per call and must terminate on its
// own: every driver path used here has a driver-side or emulator-side
// timeout, so a missing peripheral model (esp-emu fidelity is only proven
// for WiFi and PSRAM) surfaces as an esp_err code, not a hang. If a probe
// ever does hang the emulator or the board, flip its switch below to false
// - the remaining probes stay independent.
const (
	probeGPIO   = true
	probeI2C    = true
	probeI2S    = true
	probeNet    = true
	probeUDP    = true
	probeTLS    = true
	probeWS     = true
	probeReconn = true
)

// level01 maps a GetLevel bool back to IDF's 0/1 for the probe's golden
// output line ("high = 1 low = 0 toggled = 1").
func level01(b bool) int {
	if b {
		return 1
	}
	return 0
}

func gpioProbe() {
	p := gpio.Pin(4)
	if err := p.SetDirection(gpio.Output, gpio.PullNone); err != nil {
		println("[go] periph gpio set_direction err =", errCode(err))
		return
	}
	if err := p.SetLevel(true); err != nil {
		println("[go] periph gpio set_level(true) err =", errCode(err))
		return
	}
	high := p.GetLevel()
	if err := p.SetLevel(false); err != nil {
		println("[go] periph gpio set_level(false) err =", errCode(err))
		return
	}
	low := p.GetLevel()
	if err := p.Toggle(); err != nil {
		println("[go] periph gpio toggle err =", errCode(err))
		return
	}
	toggled := p.GetLevel()
	println("[go] periph gpio roundtrip high =", level01(high),
		"low =", level01(low), "toggled =", level01(toggled))
	if high && !low && toggled {
		println("[go] periph gpio PASS")
	} else {
		println("[go] periph gpio FAIL (roundtrip mismatch)")
	}
}

func i2cProbe() {
	bus, err := i2c.NewBus(8, 9)
	if err != nil {
		println("[go] periph i2c new_bus err =", errCode(err))
		return
	}
	println("[go] periph i2c bus install ok")
	// Address 0x50 with nothing attached. On hardware the transfer fails
	// with an explicit error (NACK/timeout) within the 1 s driver timeout -
	// never a hang and never a crash. The emulator does not model ACK/NACK,
	// so there the same read reports err = 0 and a 0xFF byte; real transfers
	// need real hardware.
	dev, err := bus.Device(0x50, 100000)
	if err != nil {
		println("[go] periph i2c add_device err =", errCode(err))
		bus.Close()
		return
	}
	var reg [1]byte
	err = dev.Read(reg[:])
	println("[go] periph i2c read err =", errCode(err), "byte =", int(reg[0]))
	if err := dev.Close(); err != nil {
		println("[go] periph i2c rm_device err =", errCode(err))
	}
	if err := bus.Close(); err != nil {
		println("[go] periph i2c del_bus err =", errCode(err))
		return
	}
	println("[go] periph i2c done")
}

func i2sProbe() {
	ch, err := i2s.NewChannel(true)
	if err != nil {
		println("[go] periph i2s alloc err =", errCode(err))
		return
	}
	println("[go] periph i2s channel alloc ok")
	// TX channel: bclk GPIO4, ws GPIO5, dout GPIO6, no mclk and no din.
	if err := ch.ConfigureStd(i2s.StdConfig{
		MCLK: -1, BCLK: 4, WS: 5, DOUT: 6, DIN: -1,
		Bits: i2s.Bits16, Stereo: true, SampleRate: 44100,
	}); err != nil {
		println("[go] periph i2s init_std err =", errCode(err))
		ch.Close()
		return
	}
	println("[go] periph i2s std init ok")
	if err := ch.Enable(); err != nil {
		println("[go] periph i2s enable err =", errCode(err))
		ch.Close()
		return
	}
	// One DMA-scale buffer of 16-bit stereo silence (256 frames = 1024 bytes).
	silence := make([]byte, 1024)
	n, werr := ch.Write(silence)
	println("[go] periph i2s write n =", n, "err =", errCode(werr))
	if err := ch.Disable(); err != nil {
		println("[go] periph i2s disable err =", errCode(err))
	}
	if err := ch.Close(); err != nil {
		println("[go] periph i2s del err =", errCode(err))
		return
	}
	println("[go] periph i2s done")
}

func periphProbe() {
	println("[go] periph: start")
	if probeGPIO {
		gpioProbe()
	} else {
		println("[go] periph gpio skipped (known to hang this target)")
	}
	if probeI2C {
		i2cProbe()
	} else {
		println("[go] periph i2c skipped (known to hang this target)")
	}
	if probeI2S {
		i2sProbe()
	} else {
		println("[go] periph i2s skipped (known to hang this target)")
	}
}

// compiledCAPEM reads the acceptance CA (or the negative-test CA) that
// idfproj/host/gencert.sh generated into main/tls_ca.h / tls_ca_wrong.h,
// which demo_glue.c exposes. An empty result means "not compiled in": the
// probe reports SKIP for the VERIFY_REQUIRED phase, exactly like the
// scan-only wifi run. The buffer lives on the Go heap (a 2 KiB stack buffer
// would violate the fiber-stack discipline).
//
//go:linkname demoTLSCA C.llgo_demo_tls_ca
func demoTLSCA(buf *byte, cap int32, outLen *int32)

//go:linkname demoTLSCAWrong C.llgo_demo_tls_ca_wrong
func demoTLSCAWrong(buf *byte, cap int32, outLen *int32)

func compiledCAPEM(buf *byte, outLen *int32, wrong bool) []byte {
	if wrong {
		demoTLSCAWrong(buf, int32(2048), outLen)
	} else {
		demoTLSCA(buf, int32(2048), outLen)
	}
	if *outLen <= 0 {
		return nil
	}
	return unsafe.Slice(buf, *outLen)
}

// tlsRound exercises one live TLS session: write the 256-byte pattern, read
// the echo back in short reads, compare, and close with a graceful
// close_notify (whose host-side evidence is the tlssrv log line "peer sent
// TLS close_notify"). Reports the outcome as a [go] tls line; returns
// whether the round verified.
func tlsRound(c *tls.TLSConn, tag string) bool {
	const patternLen = 256
	var msg [patternLen]byte
	s := uint32(0x12345678)
	for i := range msg {
		s = s*1664525 + 1013904223
		msg[i] = byte(s >> 24)
	}
	if err := c.SetDeadline(5000); err != nil {
		println("[go] tls", tag, "set_deadline err =", errText(err))
		return false
	}
	if w, werr := c.Write(msg[:]); werr != nil || w != patternLen {
		println("[go] tls", tag, "write n =", w, "want", patternLen)
		return false
	}
	var got [patternLen]byte
	total := 0
	for total < patternLen {
		n, rerr := c.Read(got[total:])
		if n > 0 {
			total += n
			continue
		}
		if rerr != nil && rerr != esperr.ErrEOF {
			println("[go] tls", tag, "read err =", errText(rerr), "at", total)
		} else {
			println("[go] tls", tag, "read peer closed at", total)
		}
		break
	}
	mismatches := 0
	for i := 0; i < total; i++ {
		if got[i] != msg[i] {
			mismatches++
		}
	}
	if err := c.Close(); err != nil {
		println("[go] tls", tag, "close report =", errText(err))
	}
	if total == patternLen && mismatches == 0 {
		println("[go] tls", tag, "echo", total, "/", patternLen, "bytes verified")
		return true
	}
	println("[go] tls", tag, "echo got", total, "/", patternLen,
		"bytes, mismatches =", mismatches)
	return false
}

// tlsProbe exercises pkg/esp/tls end to end over the emulator's NAT against
// idfproj/host/tlssrv (TLS echo on 127.0.0.1:9443, forwarded as
// 192.168.4.1:9443). Phases:
//
//  1. VERIFY_NONE dial + 256-byte echo + close_notify - the deterministic
//     core ("[go] tls none ...");
//  2. VERIFY_REQUIRED dial with the compiled-in CA, same exchange
//     ("[go] tls required ..."); SKIPped when no CA was compiled in (run
//     idfproj/host/gencert.sh), like the scan-only wifi run;
//  3. negative test: a dial with a CA that signed nothing must be rejected
//     with CertVerifyFailed ("[go] tls bad-ca ...");
//  4. the Config+Dial probes the deprecated TLSDial never covered: correct
//     ServerName + CA passes ("[go] tls sni ok"), wrong ServerName and
//     wrong CA are rejected ("[go] tls sni reject-host / reject-ca"), and
//     InsecureSkipVerify still completes ("[go] tls skip ok") - the four
//     SNI acceptance lines; each passing dial is proven usable by a quiet
//     echo round plus the exported ConnectionState.
//
// "[go] tls PASS" fires only when phase 1 passed AND phase 2 ran and passed
// AND phase 4's four probes went the right direction
// - that is the emu acceptance gate (--exit-on). Without a compiled-in CA
// phase 2 is SKIPped and the probe ends at "[go] tls done" - PASS is never
// printed, so a gate run in that shape cannot exit green by accident (it
// runs to the emu's 150 s timeout instead); use --exit-on "[go] tls done"
// for the run that reports the SKIP, same convention as the scan-only wifi
// run. A FAIL path ends the probe without the done marker, so it ends at
// the timeout too. Every wait
// is bounded (20 ms quanta under deadlines), so a missing peer surfaces as
// an error code, never a hang; probeTLS is the contingency switch.
func tlsProbe() {
	println("[go] tls: start")
	heapMark("tls-baseline")

	t0 := rdcycle()
	c, err := tls.TLSDial(net.IPv4(192, 168, 4, 1), 9443, 15000, nil)
	if err != nil {
		println("[go] tls none dial err =", errText(err))
		println("[go] tls FAIL dial-none")
		return
	}
	println("[go] tls none dial ok in", int((rdcycle()-t0)/240000), "ms")
	// Live-connection mark: the buffers + handshake state are held right
	// now (this is the "does the default MFL hit the internal-heap red
	// line" number). tlsRound closes before the next mark, so the free
	// column between marks is post-close - only the live marks show the
	// held footprint.
	heapMark("tls-none-live")
	if !tlsRound(c, "none") {
		println("[go] tls FAIL none")
		return
	}
	heapMark("tls-none")

	caBuf := make([]byte, 2048)
	wrongBuf := make([]byte, 2048)
	var caLen, wrongLen int32
	ca := compiledCAPEM(&caBuf[0], &caLen, false)
	caWrong := compiledCAPEM(&wrongBuf[0], &wrongLen, true)
	if ca == nil {
		println("[go] tls required SKIP: no CA compiled in (run idfproj/host/gencert.sh)")
		println("[go] tls done")
		return
	}
	t0 = rdcycle()
	c, err = tls.TLSDial(net.IPv4(192, 168, 4, 1), 9443, 15000, ca)
	if err != nil {
		println("[go] tls required dial err =", errText(err))
		println("[go] tls FAIL dial-required")
		return
	}
	println("[go] tls required dial ok in", int((rdcycle()-t0)/240000), "ms")
	heapMark("tls-required-live")
	if !tlsRound(c, "required") {
		println("[go] tls FAIL required")
		return
	}
	heapMark("tls-required")

	// Negative test: a CA that signed nothing must not authenticate the
	// peer. The dial fails inside the handshake with CertVerifyFailed;
	// any other outcome (success, or a different error) is a FAIL.
	if caWrong == nil {
		println("[go] tls bad-ca SKIP: no wrong CA compiled in")
	} else {
		c, err = tls.TLSDial(net.IPv4(192, 168, 4, 1), 9443, 15000, caWrong)
		if err == tls.ErrCertVerifyFailed {
			println("[go] tls bad-ca correctly rejected (", errText(err), ")")
		} else {
			println("[go] tls bad-ca dial err =", errText(err))
			println("[go] tls FAIL bad-ca (want CertVerifyFailed)")
			return
		}
		heapMark("tls-badca")
	}

	// SNI/identity probes through the Config+Dial path - the checks the
	// deprecated TLSDial never did. The TCP destination stays the NAT alias
	// the whole run dials; the certificate gencert.sh made names the
	// service behind it (SAN DNS:localhost, IP:127.0.0.1), so the verified
	// name travels in Config.ServerName, not in the address. Four probes,
	// one line each: the right name passes, the wrong name and the wrong CA
	// are rejected inside the handshake, and the explicit skip shape still
	// completes. Any other outcome prints a FAIL line and stops before
	// PASS, so the emu acceptance gate covers these four directions too.
	natPeer := "192.168.4.1:9443"

	// Correct name + correct CA + a version floor: a fully verified
	// session. ConnectionState is read while the session is live (State on
	// a closed conn legitimately reports nothing), then the quiet echo
	// round proves the session usable and closes it.
	c, err = tls.Dial(natPeer, 15000, &tls.Config{
		ServerName: "localhost",
		RootCAs:    ca,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		println("[go] tls sni dial err =", errText(err))
		println("[go] tls FAIL sni-ok")
		return
	}
	if !tlsStateVerified(c) {
		c.Close()
		println("[go] tls FAIL sni-ok (ConnectionState)")
		return
	}
	if !tlsRoundQuiet(c) {
		println("[go] tls FAIL sni-ok (echo)")
		return
	}
	println("[go] tls sni ok")

	// Wrong name, right CA: the handshake must refuse the peer - this is
	// the MITM shape TLSDial would have accepted (any cert of this CA).
	c, err = tls.Dial(natPeer, 15000, &tls.Config{ServerName: "nope.llgo.test", RootCAs: ca})
	if err == tls.ErrCertVerifyFailed {
		println("[go] tls sni reject-host (", errText(err), ")")
	} else {
		println("[go] tls sni dial err =", errText(err))
		println("[go] tls FAIL reject-host (want CertVerifyFailed)")
		return
	}

	// Right name, wrong CA: the chain cannot be built to the anchor that
	// signed nothing.
	c, err = tls.Dial(natPeer, 15000, &tls.Config{ServerName: "localhost", RootCAs: caWrong})
	if err == tls.ErrCertVerifyFailed {
		println("[go] tls sni reject-ca (", errText(err), ")")
	} else {
		println("[go] tls sni dial err =", errText(err))
		println("[go] tls FAIL reject-ca (want CertVerifyFailed)")
		return
	}

	// Explicit skip: no identity checking, session still fully usable
	// (same live-State + quiet-round order as the sni-ok probe).
	c, err = tls.Dial(natPeer, 15000, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		println("[go] tls skip dial err =", errText(err))
		println("[go] tls FAIL skip-ok")
		return
	}
	if !tlsStateVerified(c) {
		c.Close()
		println("[go] tls FAIL skip-ok (ConnectionState)")
		return
	}
	if !tlsRoundQuiet(c) {
		println("[go] tls FAIL skip-ok (echo)")
		return
	}
	println("[go] tls skip ok")

	println("[go] tls PASS")
	println("[go] tls done")
}

// tlsRoundQuiet is tlsRound without the per-step prints (the sni probes
// below report exactly one line each): the 256-byte pattern out, the echo
// back in short reads, byte-compare, graceful close. Closes the conn on
// every path; returns whether the round verified.
func tlsRoundQuiet(c *tls.TLSConn) bool {
	const patternLen = 256
	var msg [patternLen]byte
	s := uint32(0x9e3779b9)
	for i := range msg {
		s = s*1664525 + 1013904223
		msg[i] = byte(s >> 24)
	}
	defer c.Close()
	if err := c.SetDeadline(5000); err != nil {
		return false
	}
	if w, werr := c.Write(msg[:]); werr != nil || w != patternLen {
		return false
	}
	var got [patternLen]byte
	total := 0
	for total < patternLen {
		n, _ := c.Read(got[total:])
		if n > 0 {
			total += n
			continue
		}
		return false // peer closed early (0, io.EOF) or a real error
	}
	for i := 0; i < total; i++ {
		if got[i] != msg[i] {
			return false
		}
	}
	return true
}

// tlsStateVerified checks ConnectionState's shape on a live session: the
// negotiated version and cipher suite strings are present and the peer
// certificate exported with content. No prints - the sni probes keep one
// line per probe; a FAIL line only fires when the probe overall fails.
func tlsStateVerified(c *tls.TLSConn) bool {
	st := c.State()
	return st.Version != "" && st.CipherSuite != "" &&
		len(st.PeerCerts) == 1 && len(st.PeerCerts[0]) > 0
}

// wsRound exercises one live WebSocket session: send the 256-byte pattern
// as one binary frame, read the echo back (skipping control events),
// compare, ping and wait for the pong, then close with the RFC's normal
// closure code. Reports the outcome as [go] ws lines; returns whether the
// round verified. The host-side evidence for the close handshake is the
// wssrv log line "close handshake (close code 1000)".
func wsRound(w *ws.WSConn, tag string, seed uint32) bool {
	const patternLen = 256
	var msg [patternLen]byte
	s := seed
	for i := range msg {
		s = s*1664525 + 1013904223
		msg[i] = byte(s >> 24)
	}
	if err := w.SetDeadline(5000); err != nil {
		println("[go] ws", tag, "set_deadline err =", errText(err))
		return false
	}
	if r, werr := w.WriteMessage(msg[:], ws.OpBinary); werr != nil || r != patternLen {
		println("[go] ws", tag, "write r =", r, "want", patternLen)
		w.Close(1002)
		return false
	}
	var got [patternLen]byte
	total := 0
	for total < patternLen {
		n, rerr := w.ReadMessage(got[total:])
		if n > 0 {
			total += n
			continue
		}
		if rerr != nil && rerr != esperr.ErrEOF {
			println("[go] ws", tag, "read err =", errText(rerr), "at", total)
		} else {
			println("[go] ws", tag, "control/zero-length event during echo read")
		}
		break
	}
	mismatches := 0
	for i := 0; i < total; i++ {
		if got[i] != msg[i] {
			mismatches++
		}
	}
	ok := total == patternLen && mismatches == 0
	if ok {
		if w.LastOpcode() != ws.OpBinary {
			println("[go] ws", tag, "echo opcode =", int(w.LastOpcode()), "want binary")
			ok = false
		}
	}
	if ok {
		println("[go] ws", tag, "echo", total, "/", patternLen, "bytes verified")
	} else {
		println("[go] ws", tag, "echo got", total, "/", patternLen,
			"bytes, mismatches =", mismatches)
		w.Close(1002)
		return false
	}

	// Ping/pong: the pong is consumed by the read path and reported as a
	// (0, nil) event with PongSeen() true - pump until it shows up or the
	// deadline (2 s) cuts the wait.
	if err := w.Ping(); err != nil {
		println("[go] ws", tag, "ping err =", errText(err))
		w.Close(1002)
		return false
	}
	w.SetDeadline(2000)
	sawPong := false
	var scratch [8]byte
	for i := 0; i < 100 && !sawPong; i++ {
		if _, rerr := w.ReadMessage(scratch[:]); rerr != nil && rerr != ws.ErrTimeout {
			println("[go] ws", tag, "pong wait err =", errText(rerr))
			w.Close(1002)
			return false
		}
		sawPong = w.PongSeen()
	}
	if sawPong {
		println("[go] ws", tag, "pong ok")
	} else {
		println("[go] ws", tag, "FAIL pong")
		w.Close(1002)
		return false
	}

	// Closing handshake: the deadline bounds the wait for the peer's echo
	// (nil = graceful); wssrv logs the code as the host-side evidence.
	w.SetDeadline(2000)
	if err := w.Close(1000); err != nil {
		println("[go] ws", tag, "close report =", errText(err))
		return false
	}
	println("[go] ws", tag, "close ok (peer code", int(w.CloseCode()), ")")
	return true
}

// wsSession dials the transport, handshakes /ws, runs one round, and
// prints the live/post-close heap marks named ws-<tag>-live / ws-<tag>.
func wsSession(tag, hostport string, port uint16, caPEM []byte) bool {
	t0 := rdcycle()
	var c ws.Conn
	var derr error
	if caPEM == nil && port == 9000 {
		c, derr = net.Dial(net.IPv4(192, 168, 4, 1), port, 10000)
	} else {
		c, derr = tls.TLSDial(net.IPv4(192, 168, 4, 1), port, 15000, caPEM)
	}
	if derr != nil {
		println("[go] ws", tag, "dial err =", errText(derr))
		return false
	}
	w, err := ws.HandshakeURL(c, hostport, "/ws", 5000)
	if err != nil {
		println("[go] ws", tag, "handshake err =", errText(err))
		return false
	}
	println("[go] ws", tag, "handshake ok in", int((rdcycle()-t0)/240000), "ms")
	heapMark("ws-" + tag + "-live")
	ok := wsRound(w, tag, uint32(0x12345678)^(uint32(port)<<16))
	heapMark("ws-" + tag)
	return ok
}

// wsProbe exercises pkg/esp/ws end to end over the emulator's NAT against
// idfproj/host/wssrv (raw echo or WebSocket on 127.0.0.1:9000, TLS echo or
// wss on 127.0.0.1:9443, sniffed per connection). Phases, strictly
// sequential on this goroutine (a quantum wait is a C-level select that
// gives the scheduler no yield point):
//
//  0. crypto smoke: sha1("abc") (FIPS 180-1), the RFC 6455 section 1.3
//     accept-key example and RFC 4648 base64 vectors against the in-package
//     SHA-1/base64 (the package stays stdlib-free for image size - see the
//     ws package doc);
//  1. ws://192.168.4.1:9000/ws (plain TCP transport);
//  2. wss://192.168.4.1:9443/ws with TLSDial VERIFY_NONE;
//  3. wss://192.168.4.1:9443/ws with the compiled-in CA (VERIFY_REQUIRED);
//     SKIPped when no CA was compiled in (run idfproj/host/gencert.sh),
//     like the tls probe's required phase.
//
// Each of phases 1-3 runs handshake + binary echo + ping/pong + close.
// "[go] ws PASS" fires only when phase 1 AND 2 passed AND phase 3 ran and
// passed - that is the emu acceptance gate (--exit-on). Without a
// compiled-in CA phase 3 is SKIPped and the probe ends at "[go] ws done" -
// PASS is never printed, so a gate run in that shape cannot exit green by
// accident (it runs to the emu's 150 s timeout instead); use --exit-on
// "[go] ws done" for the run that reports the SKIP. A FAIL path ends the
// probe without the done marker, so it ends at the timeout too. Every wait
// is deadline-bounded (net/tls quanta), so a missing peer surfaces as an
// error code, never a hang; probeWS is the contingency switch.
func wsProbe() {
	println("[go] ws: start")
	heapMark("ws-baseline")

	// Phase 0: handshake-crypto known values.
	wantSum := [20]byte{
		0xa9, 0x99, 0x3e, 0x36, 0x47, 0x06, 0x81, 0x6a, 0xba, 0x3e,
		0x25, 0x71, 0x78, 0x50, 0xc2, 0x6c, 0x9c, 0xd0, 0xd8, 0x9d,
	}
	if got := ws.SHA1Sum([]byte("abc")); got != wantSum {
		println("[go] ws sha1 abc MISMATCH")
		println("[go] ws FAIL crypto-smoke")
		return
	}
	println("[go] ws sha1 abc ok")
	if ws.AcceptKey([]byte("dGhlIHNhbXBsZSBub25jZQ==")) != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		println("[go] ws accept-key MISMATCH (RFC 6455 example)")
		println("[go] ws FAIL crypto-smoke")
		return
	}
	println("[go] ws accept-key ok (RFC 6455 example)")
	enc := ws.Base64StdEncode([]byte("foobar"))
	dec, derr := ws.Base64StdDecode("Zm9vYg==")
	if enc != "Zm9vYmFy" || derr != nil || len(dec) != 4 || string(dec) != "foob" {
		println("[go] ws base64 MISMATCH", enc)
		println("[go] ws FAIL crypto-smoke")
		return
	}
	println("[go] ws base64 vectors ok (RFC 4648)")

	// Seed the handshake-key / mask-key PRNG with the cycle counter (the
	// best entropy a lab bench has; see the ws package doc's randomness
	// note for what a deployment would use).
	ws.Seed(uint64(rdcycle()) | 1)

	plainOK := wsSession("plain", "192.168.4.1:9000", 9000, nil)
	if !plainOK {
		println("[go] ws FAIL plain")
		return
	}
	noneOK := wsSession("wss-none", "192.168.4.1:9443", 9443, nil)
	if !noneOK {
		println("[go] ws FAIL wss-none")
		return
	}

	caBuf := make([]byte, 2048)
	var caLen int32
	ca := compiledCAPEM(&caBuf[0], &caLen, false)
	if ca == nil {
		println("[go] ws wss-required SKIP: no CA compiled in (run idfproj/host/gencert.sh)")
		println("[go] ws done")
		return
	}
	reqOK := wsSession("wss-required", "192.168.4.1:9443", 9443, ca)
	if !reqOK {
		println("[go] ws FAIL wss-required")
		return
	}
	println("[go] ws PASS")
	println("[go] ws done")
}

// netProbe exercises pkg/esp/net end to end over the emulator's NAT. After
// the association above the station owns 192.168.4.2 and the emulator
// aliases its gateway address 192.168.4.1 to the host machine, forwarding
// each guest connection to the same port on the host's loopback - which is
// where idfproj/host/echosrv listens (go run ./idfproj/host/echosrv).
//
// The 256-byte pattern echo through that NAT is the PASS gate ([go] net
// PASS). Resolve is best-effort: a failed lookup reports SKIP. The loopback
// self-connect is best-effort up to the handshake - a failed Listen or Dial
// reports SKIP - but once Dial has succeeded into the listener's backlog, a
// failed Accept or a broken round-trip reports FAIL: with the loopback path
// already proven good, that would be a bug in the socket stack, not a
// missing environment feature. Every wait goes through the bounded-quantum
// model, so a missing peer surfaces as an errno, never a hang. Like the
// peripheral probes above, this one must terminate on its own; probeNet is
// the contingency switch.
func netProbe() {
	println("[go] net: start")
	heapMark("net-baseline")

	// 256 bytes of LCG pseudo-random pattern: an echo cannot pass by
	// accident, and the generator needs no imports.
	const patternLen = 256
	var msg [patternLen]byte
	s := uint32(0x9E3779B9)
	for i := range msg {
		s = s*1664525 + 1013904223
		msg[i] = byte(s >> 24)
	}

	t0 := rdcycle()
	// 192.168.4.1 is the esp-emu NAT gateway alias for the host (the
	// firmware itself is 192.168.4.2); on real hardware this would be the AP
	// side's address.
	c, err := net.Dial(net.IPv4(192, 168, 4, 1), 9000, 10000)
	if err != nil {
		println("[go] net dial err =", errText(err))
		println("[go] net FAIL dial")
		return
	}
	println("[go] net dial ok in", int((rdcycle()-t0)/240000), "ms")

	if err := c.SetDeadline(5000); err != nil {
		println("[go] net set_deadline err =", errText(err))
		println("[go] net FAIL deadline")
		c.Close()
		return
	}
	if w, werr := c.Write(msg[:]); werr != nil || w != patternLen {
		println("[go] net write n =", w, "want", patternLen)
		println("[go] net FAIL write")
		c.Close()
		return
	}
	var got [patternLen]byte
	total := 0
	for total < patternLen {
		n, rerr := c.Read(got[total:])
		if n > 0 {
			total += n
			continue
		}
		if rerr != nil && rerr != esperr.ErrEOF {
			println("[go] net read err =", errText(rerr), "at", total)
		} else {
			println("[go] net read peer closed at", total)
		}
		break
	}
	mismatches := 0
	for i := 0; i < total; i++ {
		if got[i] != msg[i] {
			mismatches++
		}
	}
	c.Close()
	if total == patternLen && mismatches == 0 {
		println("[go] net echo", total, "/", patternLen, "bytes verified")
		println("[go] net PASS")
	} else {
		println("[go] net echo got", total, "/", patternLen,
			"bytes, mismatches =", mismatches)
		println("[go] net FAIL echo")
		return
	}
	heapMark("net-echo")

	// Best-effort: name resolution through the emulator's DNS proxy. The
	// dotted-quad path is instant; this checks the localhost lookup.
	t0 = rdcycle()
	ip, rerr := net.Resolve("localhost")
	if rerr != nil {
		println("[go] net resolve localhost err =", errText(rerr), "- SKIP")
	} else {
		println("[go] net resolve localhost =", wifi.IPString(ip),
			"in", int((rdcycle()-t0)/240000), "ms")
	}

	// Best-effort: loopback Listen/Accept self-connect (127.0.0.1 works in
	// the emulator because IDF enables LWIP_NETIF_LOOPBACK). Kept strictly
	// sequential on one goroutine: a quantum wait is a C-level select that
	// gives the llgo scheduler no yield point, so a concurrent shape cannot
	// interleave. Measured under esp-emu (dial goroutine + accept main,
	// 3 s windows each): the accept main ran its whole 3026 ms window alone
	// and timed out, and the dial goroutine - starved the whole time - only
	// ran once the accept side reached a channel join, completing its
	// connect in 0 ms. Sequential also matches the protocol: the handshake
	// completes into the listener's backlog before Accept, so the concurrent
	// form would add nothing but a guaranteed 3 s serialization stall.
	ln, lerr := net.Listen(9001)
	if lerr != nil {
		println("[go] net listen 9001 err =", errText(lerr), "- SKIP")
		return
	}
	cc, derr := net.Dial(net.IPv4(127, 0, 0, 1), 9001, 3000)
	if derr != nil {
		println("[go] net loopback SKIP: dial err =", errText(derr))
		ln.Close()
		return
	}
	ln.SetDeadline(3000)
	lc, aerr := ln.Accept()
	if aerr != nil {
		println("[go] net loopback FAIL: accept err =", errText(aerr))
		cc.Close()
		ln.Close()
		return
	}
	cc.SetDeadline(3000)
	lc.SetDeadline(3000)
	var one, atAccept, back [1]byte
	one[0] = 0x5A
	ok := false
	if w, werr := cc.Write(one[:]); werr == nil {
		ok = w == 1
	}
	if ok {
		n, rerr := lc.Read(atAccept[:])
		ok = n == 1 && rerr == nil && atAccept[0] == 0x5A
	}
	if ok {
		w, werr := lc.Write(atAccept[:])
		ok = werr == nil && w == 1
	}
	if ok {
		n, rerr := cc.Read(back[:])
		ok = n == 1 && rerr == nil && back[0] == 0x5A
	}
	cc.Close()
	lc.Close()
	ln.Close()
	if ok {
		println("[go] net loopback self-connect PASS")
	} else {
		println("[go] net loopback FAIL")
	}
	heapMark("net-loopback")
}

// udpProbe exercises pkg/esp/net's UDP datagram sockets (UDPConn) end to
// end over the same NAT path netProbe just proved, plus the guest loopback.
// Three deadline-bounded round-trips, strictly sequential on this
// goroutine (a quantum wait is a C-level select that gives the llgo
// scheduler no yield point):
//
//  1. unconnected form against the host peer: WriteTo + ReadFrom to
//     192.168.4.1:9002 - forwarded by the NAT to idfproj/host/udpsrv,
//     which echoes each datagram back to its source;
//  2. connected form against the same peer: DialUDP + Write + Read (one
//     Read per datagram: a datagram socket has no partial reads, so the
//     echo arrives as a single piece);
//  3. loopback self-conversation: ListenUDP on 127.0.0.1:9003 (127.0.0.1
//     works in the emulator because IDF enables LWIP_NETIF_LOOPBACK), a
//     connected DialUDP into it, and the bound socket's ReadFrom/WriteTo
//     answering - the UDP analog of netProbe's loopback self-connect.
//
// "[go] net udp PASS" is the probe's gate; every failure prints a
// "[go] net udp FAIL ..." line and returns without PASS. Every wait is
// deadline-bounded (20 ms quanta under explicit deadlines), so a missing
// peer surfaces as an errno, never a hang; probeUDP is the contingency
// switch, same convention as the other probes.
func udpProbe() {
	println("[go] net udp: start")
	const (
		hostLen     = 128 // phase 1: unconnected WriteTo/ReadFrom datagram
		peerLen     = 256 // phase 2: connected Write/Read datagram
		lbLen       = 64  // phase 3: loopback datagram
		udpPort     = 9002
		lbUdpPort   = 9003
		udpDeadline = 5000
	)
	// lcg fills n bytes with the same pseudo-random generator netProbe
	// uses; an echo cannot pass by accident, and it needs no imports.
	lcg := func(seed uint32, n int32) []byte {
		buf := make([]byte, n)
		s := seed
		for i := range buf {
			s = s*1664525 + 1013904223
			buf[i] = byte(s >> 24)
		}
		return buf
	}
	mism := func(a, b []byte) int {
		m := 0
		for i := range a {
			if a[i] != b[i] {
				m++
			}
		}
		return m
	}
	deadline := func(c *net.UDPConn) bool {
		if err := c.SetDeadline(udpDeadline); err != nil {
			println("[go] net udp set_deadline err =", errText(err))
			return false
		}
		return true
	}

	// Phase 1: unconnected WriteTo/ReadFrom to the host peer. The NAT
	// rewrites the echo's source back to 192.168.4.1:9002, so ReadFrom's
	// sender address is deterministic and gets verified against it.
	host := net.Addr{IP: net.IPv4(192, 168, 4, 1), Port: udpPort}
	c, err := net.ListenUDP(nil)
	if err != nil {
		println("[go] net udp listen err =", errText(err))
		println("[go] net udp FAIL listen")
		return
	}
	if !deadline(c) {
		println("[go] net udp FAIL listen-deadline")
		c.Close()
		return
	}
	msg := lcg(0x51ED270B, hostLen)
	if w, werr := c.WriteTo(msg, host); werr != nil || w != hostLen {
		println("[go] net udp write_to n =", w, "err =", errText(werr))
		println("[go] net udp FAIL write-to")
		c.Close()
		return
	}
	var got [hostLen]byte
	n, from, rerr := c.ReadFrom(got[:])
	if rerr != nil || n != hostLen || mism(msg, got[:]) != 0 {
		println("[go] net udp read_from n =", n, "err =", errText(rerr),
			"peer =", from.String(), "mismatches =", mism(msg, got[:]))
		println("[go] net udp FAIL read-from")
		c.Close()
		return
	}
	if from != host {
		println("[go] net udp read_from peer =", from.String(), "want", host.String())
		println("[go] net udp FAIL read-from-peer")
		c.Close()
		return
	}
	c.Close()
	println("[go] net udp peer " + from.String())
	println("[go] net udp round-trip ok n=" + dec32(n))

	// Phase 2: connected DialUDP Write/Read to the same peer.
	cc, derr := net.DialUDP(nil, &host)
	if derr != nil {
		println("[go] net udp dial err =", errText(derr))
		println("[go] net udp FAIL dial")
		return
	}
	if !deadline(cc) {
		println("[go] net udp FAIL dial-deadline")
		cc.Close()
		return
	}
	msg2 := lcg(0x2545F491, peerLen)
	if w, werr := cc.Write(msg2); werr != nil || w != peerLen {
		println("[go] net udp write n =", w, "want", peerLen)
		println("[go] net udp FAIL write")
		cc.Close()
		return
	}
	var got2 [peerLen]byte
	n2, rerr2 := cc.Read(got2[:])
	if rerr2 != nil || n2 != peerLen || mism(msg2, got2[:]) != 0 {
		println("[go] net udp read n =", n2, "err =", errText(rerr2),
			"mismatches =", mism(msg2, got2[:]))
		println("[go] net udp FAIL read")
		cc.Close()
		return
	}
	cc.Close()
	println("[go] net udp connected round-trip ok n=" + dec32(n2))

	// Phase 3: loopback self-conversation on 127.0.0.1. UDP has no
	// accept: the bound socket ReadFrom's the datagram, sees the client's
	// ephemeral port in the sender address, and WriteTo's the echo back.
	lb := net.Addr{IP: net.IPv4(127, 0, 0, 1), Port: lbUdpPort}
	srv, serr := net.ListenUDP(&lb)
	if serr != nil {
		println("[go] net udp listen loopback err =", errText(serr))
		println("[go] net udp FAIL listen-loopback")
		return
	}
	cli, cerr := net.DialUDP(nil, &lb)
	if cerr != nil {
		println("[go] net udp dial loopback err =", errText(cerr))
		println("[go] net udp FAIL dial-loopback")
		srv.Close()
		return
	}
	if !deadline(srv) || !deadline(cli) {
		println("[go] net udp FAIL loopback-deadline")
		cli.Close()
		srv.Close()
		return
	}
	msg3 := lcg(0x0B6675F1, lbLen)
	if w, werr := cli.Write(msg3); werr != nil || w != lbLen {
		println("[go] net udp loopback write n =", w, "want", lbLen)
		println("[go] net udp FAIL loopback-write")
		cli.Close()
		srv.Close()
		return
	}
	var atSrv, back [lbLen]byte
	n3, from3, rerr3 := srv.ReadFrom(atSrv[:])
	if rerr3 != nil || n3 != lbLen || mism(msg3, atSrv[:]) != 0 {
		println("[go] net udp loopback read_from n =", n3, "err =", errText(rerr3),
			"mismatches =", mism(msg3, atSrv[:]))
		println("[go] net udp FAIL loopback-read-from")
		cli.Close()
		srv.Close()
		return
	}
	if w, werr := srv.WriteTo(atSrv[:n3], from3); werr != nil || w != n3 {
		println("[go] net udp loopback write_to n =", w, "err =", errText(werr))
		println("[go] net udp FAIL loopback-echo")
		cli.Close()
		srv.Close()
		return
	}
	n4, rerr4 := cli.Read(back[:])
	cli.Close()
	srv.Close()
	if rerr4 != nil || n4 != lbLen || mism(msg3, back[:]) != 0 {
		println("[go] net udp loopback read n =", n4, "err =", errText(rerr4),
			"mismatches =", mism(msg3, back[:]))
		println("[go] net udp FAIL loopback-read")
		return
	}
	println("[go] net udp loopback round-trip ok n=" + dec32(n4))
	println("[go] net udp PASS")
}

// reconnProbe exercises pkg/esp/wifi's auto-reconnect state machine end to
// end. An emulator cannot make its soft AP drop the station on its own, so
// the probe drives every transition itself; the one IDF signal it cannot
// fake (a router vanishing while the program idles) is reproduced by a
// failed association - connecting to an SSID nobody serves delivers the
// same unexpected EvStaDisconnected a lost router would, and from there the
// machine's behavior is identical. Phases, strictly sequential on this
// goroutine:
//
//  1. SetAutoReconnect(true) + a user Disconnect: the DISCONNECTED arrives
//     (plain IDF event) but the machine must stay idle - nothing reconnects
//     in the quiet window that follows;
//  2. a manual Connect still associates while the machine is enabled, and
//     its GOT_IP resets the backoff;
//	3. a failed association (llgo-nosuchap SSID): the station is switched
//	   over through an intentional Disconnect first (the driver refuses a
//	   new target while associated - "disconnect before connecting to new
//	   ap"), whose event the machine consumes without scheduling; then the
//	   association fails and the machine - nobody else, this code makes no
//	   Connect call in the window - drives the retries; a second
//	   DISCONNECTED inside the window is its fingerprint;
//  4. restoring the real credentials: the machine's next attempt associates
//     and its GOT_IP ends the episode (backoff reset);
//  5. SetAutoReconnect(false): a user disconnect now stays down, proving
//     the off switch;
//  6. a final manual Connect restores the association for the probes that
//     follow.
//
// "[go] wifi reconn: PASS" fires only when every phase held. Every wait is
// a bounded Next window, so a machine that never acts surfaces as a FAIL
// line, never a hang. On real hardware the router-power-cycle variant of
// phase 3 (AP gone mid-association) remains a pending verification item -
// the emulator cannot inject it.
func reconnProbe(ssid, pass string) {
	println("[go] wifi reconn: start")
	ev := wifi.Subscribe(wifi.EvStaDisconnected, wifi.EvStaConnected, wifi.EvGotIP)
	defer ev.Close()

	// waitKind consumes events until one of kind arrives or timeoutMs
	// passes; intermediate events of other kinds are dropped.
	waitKind := func(kind int32, timeoutMs int32) (wifi.Event, bool) {
		for ms := int32(0); ms < timeoutMs; ms += 100 {
			if e, err := ev.Next(100); err == nil {
				if e.Kind == kind {
					return e, true
				}
			}
		}
		return wifi.Event{}, false
	}
	// quiet reports whether no event of interest arrives within ms.
	quiet := func(ms int32) bool {
		for left := int32(0); left < ms; left += 100 {
			if _, err := ev.Next(100); err == nil {
				return false
			}
		}
		return true
	}

	// Phase 1: the machine is enabled, then the program disconnects on
	// purpose - the disconnect event must not schedule a retry.
	wifi.SetAutoReconnect(true)
	println("[go] wifi reconn: enable phase=" + wifi.ReconnectState().String())
	if wifi.Disconnect() != nil {
		println("[go] wifi reconn: FAIL user-disconnect")
		return
	}
	if _, ok := waitKind(wifi.EvStaDisconnected, 3000); !ok {
		println("[go] wifi reconn: FAIL user-disconnect-event")
		return
	}
	if !quiet(5000) || wifi.ReconnectState() != wifi.ReconnIdle {
		println("[go] wifi reconn: FAIL suppress")
		return
	}
	println("[go] wifi reconn: user disconnect suppressed")

	// Phase 2: a manual reconnect still works with the machine enabled.
	if wifi.Connect() != nil {
		println("[go] wifi reconn: FAIL manual-connect")
		return
	}
	if _, ok := waitKind(wifi.EvGotIP, 15000); !ok || wifi.ReconnectState() != wifi.ReconnIdle {
		println("[go] wifi reconn: FAIL manual-reconnect")
		return
	}
	println("[go] wifi reconn: manual reconnect ok")

	// Phase 3: unexpected disconnection via a failed association. The
	// driver rejects a new target while associated ("disconnect before
	// connecting to new ap"), so the switch goes through an intentional
	// Disconnect first - its event arms the flag and the machine consumes
	// it without scheduling. From there the machine owns esp_wifi_connect.
	if wifi.Disconnect() != nil {
		println("[go] wifi reconn: FAIL phase3-disconnect")
		return
	}
	if _, ok := waitKind(wifi.EvStaDisconnected, 3000); !ok {
		println("[go] wifi reconn: FAIL phase3-disconnect-event")
		return
	}
	if wifi.SetSTA("llgo-nosuchap-xyz", "nosuchpassword") != nil {
		println("[go] wifi reconn: FAIL set-sta-bad")
		return
	}
	if wifi.Connect() != nil {
		println("[go] wifi reconn: FAIL bad-connect")
		return
	}
	if _, ok := waitKind(wifi.EvStaDisconnected, 10000); !ok {
		println("[go] wifi reconn: FAIL unexpected-disconnect")
		return
	}
	// No Connect call from this code until phase 4: the next
	// DISCONNECTED can only come from a machine-driven retry.
	if _, ok := waitKind(wifi.EvStaDisconnected, 12000); !ok {
		println("[go] wifi reconn: FAIL no-auto-retry")
		return
	}
	println("[go] wifi reconn: auto retries observed")

	// Phase 4: restore the real credentials; the machine's next attempt
	// (still on its own backoff schedule) recovers and ends the episode.
	if wifi.SetSTA(ssid, pass) != nil {
		println("[go] wifi reconn: FAIL set-sta-good")
		return
	}
	got, ok := waitKind(wifi.EvGotIP, 20000)
	if !ok || got.IP == 0 {
		println("[go] wifi reconn: FAIL auto-recover")
		return
	}
	println("[go] wifi reconn: auto recovered ip=" + wifi.IPString(got.IP))

	// Phase 5: switch the machine off; a user disconnect stays down.
	wifi.SetAutoReconnect(false)
	if wifi.Disconnect() != nil {
		println("[go] wifi reconn: FAIL disable-disconnect")
		return
	}
	if _, ok := waitKind(wifi.EvStaDisconnected, 3000); !ok {
		println("[go] wifi reconn: FAIL disable-disconnect-event")
		return
	}
	if !quiet(4000) || wifi.ReconnectState() != wifi.ReconnIdle {
		println("[go] wifi reconn: FAIL disable-suppress")
		return
	}
	println("[go] wifi reconn: disable stays down")

	// Phase 6: restore the association the net/tls/ws probes expect.
	if wifi.Connect() != nil {
		println("[go] wifi reconn: FAIL restore-connect")
		return
	}
	got, ok = waitKind(wifi.EvGotIP, 15000)
	if !ok || got.IP == 0 {
		println("[go] wifi reconn: FAIL restore-ip")
		return
	}
	println("[go] wifi reconn: restored ip=" + wifi.IPString(got.IP))
	println("[go] wifi reconn: PASS")
}

func main() {
	defer wifi.Finish() // flush C stdio on every exit path
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
	wifi.PrintVersion()
	println("[go] wifi: setup ...")
	if err := wifi.Setup(); err != nil {
		println("[go] wifi setup err =", errText(err))
		println("[go] wifi FAIL setup")
		return
	}
	println("[go] wifi setup err=0")
	heapMark("init")

	if err := wifi.EventsRegister(); err != nil {
		println("[go] wifi events register err =", errText(err))
		println("[go] wifi FAIL events")
		return
	}
	ssid, pass := wifi.DefaultSTA() // compiled in via idfproj/main/wifi_creds.h
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
		if err := wifi.SetModeSTA(); err != nil {
			println("[go] wifi set_mode err =", errText(err))
			println("[go] wifi FAIL set_mode")
			return
		}
	} else if err := wifi.SetSTA(ssid, pass); err != nil {
		println("[go] wifi set_sta err =", errText(err))
		println("[go] wifi FAIL set_sta")
		return
	}
	if err := wifi.Start(); err != nil {
		println("[go] wifi start err =", errText(err))
		println("[go] wifi FAIL start")
		return
	}
	if _, err := wifi.Subscribe(wifi.EvStaStart).Next(3000); err == nil {
		println("[go] wifi STA_START")
		heapMark("start")
	} else {
		println("[go] wifi: no STA_START in 3s")
		println("[go] wifi FAIL sta_start")
		return
	}

	// Scan (M1): emu's built-in soft AP must show up in the list. The Scan
	// wrapper subscribes to SCAN_DONE, starts the scan non-blocking, waits
	// and fetches; the scheduler stays live throughout.
	aps, err := wifi.Scan(wifi.ScanOptions{})
	if err != nil {
		println("[go] wifi scan err =", errText(err))
		println("[go] wifi FAIL scan")
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
	periphProbe()
	if noCreds {
		println("[go] wifi: scan-only run complete, no association attempted")
		return
	}

	// Connect + DHCP (M2): esp_wifi_connect attempts once; a failed
	// association/handshake arrives as EvStaDisconnected with a reason code.
	connEvents := wifi.Subscribe(wifi.EvStaConnected, wifi.EvStaDisconnected)
	connected := false
	for attempt := 1; attempt <= 3 && !connected; attempt++ {
		if err := wifi.Connect(); err != nil {
			println("[go] wifi connect err =", errText(err))
			println("[go] wifi FAIL connect")
			return
		}
		ev, nerr := connEvents.Next(3000)
		if nerr == nil && ev.Kind == wifi.EvStaConnected {
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
	gotIP := wifi.Subscribe(wifi.EvGotIP)
	ev, _ := gotIP.Next(6000)
	ip := ev.IP
	if ip == 0 {
		ip = wifi.GetIP()
	}
	if ip == 0 {
		println("[go] wifi FAIL no_ip")
		return
	}
	println("[go] wifi GOT_IP ip=" + wifi.IPString(ip))
	println("[go] wifi PASS ip=" + wifi.IPString(ip))
	heapMark("dhcp")

	// Subscriber fan-out: two EvGotIP subscriptions must each receive their
	// own copy of the same mailbox event, which a station disconnect +
	// reconnect regenerates (the soft AP re-runs DHCP). subDisc only absorbs
	// the disconnect event so the reconnect starts from a settled state.
	subDisc := wifi.Subscribe(wifi.EvStaDisconnected)
	subA := wifi.Subscribe(wifi.EvGotIP)
	subB := wifi.Subscribe(wifi.EvGotIP)
	if err := wifi.Disconnect(); err != nil {
		println("[go] wifi sub fanout FAIL disconnect err =", errText(err))
	} else {
		subDisc.Next(3000)
		if err := wifi.Connect(); err != nil {
			println("[go] wifi sub fanout FAIL connect err =", errText(err))
		} else {
			evA, errA := subA.Next(15000)
			evB, errB := subB.Next(15000)
			if errA == nil && errB == nil && evA.Kind == wifi.EvGotIP && evB.Kind == wifi.EvGotIP &&
				evA.IP != 0 && evA.IP == evB.IP {
				println("[go] wifi sub fanout ok ip=" + wifi.IPString(evA.IP))
			} else {
				println("[go] wifi sub fanout FAIL gotip sub1 kind =", int(evA.Kind),
					"sub2 kind =", int(evB.Kind))
			}
		}
	}

	// Steady state: the WiFi stack keeps module-specific buffers parked
	// while connected; give it 10 seconds and look again.
	vTaskDelay(1000)
	heapMark("steady+10s")

	// TCP over lwip: NAT dial + echo is the PASS gate; resolve and the
	// loopback self-connect are best-effort extras.
	if probeNet {
		netProbe()
	} else {
		println("[go] net skipped (probeNet switch off; every net path is deadline-bounded)")
	}

	// UDP over lwip: the datagram analog right after the TCP probe - NAT
	// echo in both the unconnected and the connected form, plus the
	// loopback self-conversation. Same bounded-wait discipline.
	if probeUDP {
		udpProbe()
	} else {
		println("[go] net udp skipped (probeUDP switch off; every net path is deadline-bounded)")
	}

	// WiFi auto-reconnect: cycles the association itself - user disconnect
	// (suppressed), manual reconnect, a failed association the machine must
	// retry on its own, recovery, then the off switch - and restores the
	// association before the transport probes below. An emulator cannot
	// make its soft AP drop the station on its own, so a driven failed
	// association stands in for the router disappearing; see the probe's
	// comment and README for the hardware item that stays unverified.
	if probeReconn {
		reconnProbe(ssid, pass)
	} else {
		println("[go] wifi reconn skipped (probeReconn switch off)")
	}

	// TLS on top of lwip: both authmodes plus the wrong-CA negative test
	// against the host TLS peer. tlsProbe needs no extra scheduling care
	// beyond what net already documents (all waits quantum-bounded).
	if probeTLS {
		tlsProbe()
	} else {
		println("[go] tls skipped (probeTLS switch off; every tls path is deadline-bounded)")
	}

	// WebSocket over the net/tls transports: crypto smoke + three sessions
	// (plain, wss permissive, wss strict) against the host wssrv, all
	// strictly sequential on this goroutine.
	if probeWS {
		wsProbe()
	} else {
		println("[go] ws skipped (probeWS switch off; every ws path is deadline-bounded)")
	}

	println("[go] wifi done")

	println("[go] done")
}
