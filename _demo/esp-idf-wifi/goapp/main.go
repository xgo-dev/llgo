// MVP demo for "llgo as an ESP-IDF component" on ESP32-S31.
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main
// component; app_main calls the generated C entry, which hands control to
// the llgo fiber scheduler (runtime.RunMain).
package main

import (
	"github.com/xgo-dev/llgo/pkg/esp/wifi"
	_ "unsafe" // for the //go:linkname declarations
)

func produce(ch chan int, v int) {
	ch <- v
}

func main() {
	defer wifi.Finish() // flush C stdio on every exit path
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
		heapMark("after net")
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
		heapMark("after tls")
	} else {
		println("[go] tls skipped (probeTLS switch off; every tls path is deadline-bounded)")
	}

	// WebSocket over the net/tls transports: crypto smoke + three sessions
	// (plain, wss permissive, wss strict) against the host wssrv, all
	// strictly sequential on this goroutine.
	if probeWS {
		wsProbe()
		heapMark("after ws")
	} else {
		println("[go] ws skipped (probeWS switch off; every ws path is deadline-bounded)")
	}

	println("[go] wifi done")

	println("[go] done")
}
