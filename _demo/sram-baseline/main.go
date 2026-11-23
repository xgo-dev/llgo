// SRAM baseline probe for ESP32-S31 baremetal (llgo -target esp32s31):
// no ESP-IDF, no WiFi. Runs the workload the WiFi demo runs before the
// WiFi stack comes up (strings, slices, goroutine + channel, prints) plus
// scan-record-sized state and allocation rounds, and reports the tinygogc
// accounting (runtime.ReadMemStats) at every node.
//
// Together with _demo/esp-idf-wifi this forms the two halves of the SRAM
// budget: baremetal = static segments + GC window; IDF = the same static
// segments + GC window + the IDF heap pool that the WiFi stack draws from.
package main

import (
	"runtime"
	"unsafe" // unsafe.Pointer + //go:linkname
)

// Heap window and bootstrap stack markers from the target linker script
// (targets/esp32s31.app.elf.ld): the window is [_heapStart, _heapEnd) and
// the linker stack is the NOLOAD .stack section [_stack_end, _stack_top).
//
//go:linkname heapStart C._heapStart
var heapStart [0]byte

//go:linkname heapEnd C._heapEnd
var heapEnd [0]byte

//go:linkname stackTop C._stack_top
var stackTop [0]byte

//go:linkname stackEnd C._stack_end
var stackEnd [0]byte

// gcReserve is the slack the runtime leaves at the window base for libc
// allocation it cannot wrap (tinygogc gc_link.go gcMemoryLayout); the
// allocator starts at _heapStart + 2048.
const gcReserve = 2048

// apLike mirrors the WiFi demo's scan record (ApInfo: 6 + 33 + 1 + 1 + pad
// + 4 = 48 bytes) so the workload's live set matches the real pipeline.
type apLike struct {
	bssid    [6]uint8
	ssid     [33]uint8
	channel  uint8
	rssi     int8
	authMode int32
}

func produce(ch chan int, v int) { ch <- v }

func printMem(tag string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	println("[go] sram mem "+tag,
		"alloc =", int(m.Alloc), "total =", int(m.TotalAlloc),
		"sys =", int(m.Sys))
	println("[go] sram mem "+tag,
		"inuse =", int(m.HeapInuse), "idle =", int(m.HeapIdle),
		"stackinuse =", int(m.StackInuse), "stacksys =", int(m.StackSys),
		"gcsys =", int(m.GCSys), "numgc =", int(m.NumGC))
}

func main() {
	println("[go] sram: baremetal memory baseline (no IDF, no WiFi)")
	println("[go] sram: heap_start =", int(uintptr(unsafe.Pointer(&heapStart))),
		"heap_end =", int(uintptr(unsafe.Pointer(&heapEnd))))
	println("[go] sram: gc window =",
		int(uintptr(unsafe.Pointer(&heapEnd))-uintptr(unsafe.Pointer(&heapStart))-gcReserve),
		"stack window =", int(uintptr(unsafe.Pointer(&stackTop))-uintptr(unsafe.Pointer(&stackEnd))))
	printMem("boot")

	// Workload: the same shape as the WiFi demo before the stack comes up.
	ch := make(chan int)
	go produce(ch, 42)
	if got := <-ch; got == 42 {
		println("[go] sram: goroutine channel roundtrip ok")
	}
	nums := make([]int, 16)
	sum := 0
	for i := range nums {
		nums[i] = i * i
		sum += nums[i]
	}
	println("[go] sram: heap alloc sum =", sum)

	// The WiFi demo's own live heap while connected: 20 scan records,
	// ssid/pass strings, the event struct.
	aps := make([]apLike, 20)
	for i := range aps {
		aps[i].ssid[0] = 'h'
		aps[i].channel = uint8(i % 14)
	}
	ssid := "myssid"
	pass := "mypassword"
	println("[go] sram: aps =", len(aps), "ssid =", ssid, "pass len =", len(pass))
	printMem("loaded")

	// Allocation rounds: 8 KiB live per round, released and collected, to
	// see where the steady state settles.
	for round := 1; round <= 5; round++ {
		tmp := make([]byte, 8*1024)
		for i := range tmp {
			tmp[i] = byte(i)
		}
		if tmp[len(tmp)-1] != byte(len(tmp)-1) {
			println("[go] sram: buffer mismatch")
		}
		tmp = nil
		runtime.GC()
		println("[go] sram: round", round)
		printMem("round")
	}
	printMem("steady")
	println("[go] sram: done")
}
