// MVP demo for "llgo as an ESP-IDF component" on ESP32-S31.
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main
// component; app_main calls the generated C entry, which hands control to
// the llgo fiber scheduler (runtime.RunMain).
package main

import (
	"llgoidf/goapp/espwifi"
	_ "unsafe"
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

func main() {
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

	// ---- WiFi pipeline (M0): C glue link check, one-shot setup, start and
	// STA_START through the event mailbox. Scan (M1) and connect + DHCP
	// (M2) extend this sequence.
	espwifi.PrintVersion()
	println("[go] wifi: setup ...")
	if e := espwifi.Setup(); e != 0 {
		println("[go] wifi setup err =", int(e), espwifi.ErrName(e))
		println("[go] wifi FAIL setup")
		return
	}
	println("[go] wifi setup err=0")

	if e := espwifi.EventsRegister(); e != 0 {
		println("[go] wifi events register err =", int(e), espwifi.ErrName(e))
		println("[go] wifi FAIL events")
		return
	}
	if e := espwifi.SetSTA("myssid", "mypassword"); e != 0 {
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
	} else {
		println("[go] wifi: no STA_START in 3s, last event kind =", int(ev.Kind))
		println("[go] wifi FAIL sta_start")
		return
	}

	println("[go] done")
}
