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

// heapMark prints the IDF heap watermarks at one pipeline stage.
func heapMark(stage string) {
	println("[go] wifi heap "+stage,
		"free =", int(espwifi.HeapFree()), "min =", int(espwifi.HeapMin()))
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
	if ssid == "" {
		println("[go] wifi: no credentials compiled in (idfproj/main/wifi_creds.h)")
		println("[go] wifi FAIL no_creds")
		return
	}
	if e := espwifi.SetSTA(ssid, pass); e != 0 {
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
		println("[go] wifi ap ssid=" + ap.SsidString(),
			"ch =", int(ap.Channel), "rssi =", int(ap.Rssi))
	}
	println("[go] wifi scan done")
	heapMark("scan")

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
