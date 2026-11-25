// Soft-AP probe for "llgo as an ESP-IDF component" on ESP32-S31.
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main
// component; app_main calls the generated C entry, which hands control to
// the llgo fiber scheduler (runtime.RunMain).
//
// The probe drives the whole AP-mode surface of pkg/esp/wifi and nothing else:
// configure the soft AP, bring it up, read its address back, watch a client
// associate and get a DHCP lease, serve it one TCP session, deauthenticate it,
// stop, and switch to APSTA to prove the mode setters and the interface
// configs do not undo each other. Every wait is bounded; every failure prints
// a FAIL line and returns, so a broken step never looks like a hang.
package main

import (
	_ "unsafe" // the pull //go:linkname declaration below needs it

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/net"
	"github.com/xgo-dev/llgo/pkg/esp/wifi"
)

// Probe constants. The SSID/passphrase/channel must match the acceptance
// script's esp-emu flags: the emulator joins the firmware's own soft AP with
// --wifi-fw-ap-password, and it neither joins a hidden SSID nor a WPA3-only
// one.
const (
	apSSID    = "llgo-ap"
	apPass    = "llgoappass"
	apChannel = 6
	apMaxSTA  = 2
	apPort    = 8000

	// apExpectIP is the address IDF's default soft-AP netif carries; the
	// acceptance run asserts it, so a silently different netif setup shows.
	apExpectIP = "192.168.4.1"

	// acceptBudgetMs bounds how long the probe waits for the host client.
	// The client is driven by host/run-acceptance.sh and usually connects
	// within a few seconds of the AP coming up.
	acceptBudgetMs = 90000

	// quantum bounds the diagnostic waits below: 100 ms, the poller's own
	// cadence.
	quantum = 100
)

//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

// heapMark prints the IDF heap watermarks at one pipeline stage. The AP-mode
// RAM ledger is the difference between these lines: Setup sizes for a station,
// the AP netif (with its DHCP server) appears in SetAP, and the driver's AP
// buffers in Start.
func heapMark(stage string) {
	println("[go] ap heap "+stage,
		"internal free =", int(wifi.HeapInternalFree()),
		"min =", int(wifi.HeapMin()),
		"| total free =", int(wifi.HeapFree()),
		"| psram =", int(wifi.HeapSpiramFree()))
}

// errText formats an esp error as "<code> <name>", the shape the other esp
// probes print.
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

// dec32 formats v in decimal: println has no number form without a separating
// space, and the golden lines pin the exact shapes.
func dec32(v int) string {
	neg := v < 0
	x := uint32(v)
	if neg {
		x = -uint32(x)
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

// waitKind waits up to timeoutMs for one event kind, walking past (and
// discarding) the kinds it is not waiting for. It returns the last event it
// saw either way, so a caller can report what did arrive.
//
// The accounting is the package's own: Subscription.Next(quantum) parks the
// task for at least one quantum when nothing arrives, so one loop step is at
// least one quantum of wall clock. A caller that gets an event back early just
// spends the remaining budget on the next step.
func waitKind(sub *wifi.Subscription, kind int32, timeoutMs int32) (wifi.Event, bool) {
	var last wifi.Event
	for waited := int32(0); waited < timeoutMs; waited += quantum {
		ev, err := sub.Next(quantum)
		if err != nil {
			continue
		}
		if ev.Kind == kind {
			return ev, true
		}
		last = ev
	}
	return last, false
}

// drain discards everything currently pending in sub and reports how many
// events it dropped. Called before a phase that must only be satisfied by a
// fresh event (a stale AP_STOP would otherwise pass for Stop's own).
func drain(sub *wifi.Subscription) int {
	n := 0
	for {
		if _, err := sub.Next(0); err != nil {
			return n
		}
		n++
	}
}

// settle lets the C-side ring and the poller catch up (a few quanta) before a
// drain, so "everything pending" is really everything.
func settle() {
	vTaskDelay(20) // 200 ms
}

func produce(ch chan int, v int) { ch <- v }

func main() {
	defer wifi.Finish() // flush C stdio on every exit path
	println("[go] hello from llgo on FreeRTOS/ESP32-S31 (soft AP probe)")

	// Scheduler sanity before the radio work: one goroutine + channel
	// roundtrip proves the fiber scheduler is alive.
	ch := make(chan int)
	go produce(ch, 42)
	if got := <-ch; got == 42 {
		println("[go] goroutine channel roundtrip ok")
	} else {
		println("[go] goroutine channel roundtrip got", got)
		println("[go] ap FAIL scheduler")
		return
	}

	heapMark("baseline")
	wifi.PrintVersion()

	println("[go] wifi: setup ...")
	if err := wifi.Setup(); err != nil {
		println("[go] wifi setup err =", errText(err))
		println("[go] ap FAIL setup")
		return
	}
	println("[go] wifi setup err=0")
	if err := wifi.EventsRegister(); err != nil {
		println("[go] wifi events register err =", errText(err))
		println("[go] ap FAIL events")
		return
	}

	// Arm the station auto-reconnect machine before any radio work. Nothing
	// in the AP path may wake it: it reacts to EvStaDisconnected (the
	// *station* losing its AP), never to a soft-AP client leaving, so it has
	// to stay idle through association, DHCP, deauth and stop.
	wifi.SetAutoReconnect(true)
	println("[go] wifi reconn armed, phase =", wifi.ReconnectState().String())
	heapMark("init")

	// Nothing configured the AP yet, so no AP netif exists: the three AP
	// queries must say so (ESP_ERR_WIFI_MODE) rather than answer with a
	// zeroed address or a slice nobody filled.
	if _, err := wifi.GetAPIP(); err != nil {
		println("[go] ap pre-config get_ap_ip err =", errText(err))
		if esperr.Code(err) != 0x3005 {
			println("[go] ap FAIL pre_config_ip")
			return
		}
	} else {
		println("[go] ap: GetAPIP answered before SetAP")
		println("[go] ap FAIL pre_config_ip")
		return
	}
	if _, err := wifi.APStations(); err != nil {
		println("[go] ap pre-config stations err =", errText(err))
		if esperr.Code(err) != 0x3005 {
			println("[go] ap FAIL pre_config_stations")
			return
		}
	} else {
		println("[go] ap: APStations answered before SetAP")
		println("[go] ap FAIL pre_config_stations")
		return
	}

	// ---- Configure the soft AP -------------------------------------------
	cfg := wifi.APConfig{
		SSID:     apSSID,
		Password: apPass,
		Channel:  apChannel,
		MaxSTA:   apMaxSTA,
		AuthMode: wifi.AuthWPA2PSK,
	}
	if err := wifi.SetAP(cfg); err != nil {
		println("[go] wifi set_ap err =", errText(err))
		println("[go] ap FAIL set_ap")
		return
	}
	println("[go] wifi set_ap err=0")
	heapMark("ap-config")

	// Read the config back from the driver: this is the evidence that the
	// channel and the auth mode took effect, not merely that a call returned
	// zero.
	back, err := wifi.GetAPConfig()
	if err != nil {
		println("[go] wifi get_ap_config err =", errText(err))
		println("[go] ap FAIL get_config")
		return
	}
	hidden := 0
	if back.Hidden {
		hidden = 1
	}
	println("[go] ap config ssid="+back.SSID,
		"ch =", int(back.Channel), "maxsta =", back.MaxSTA,
		"auth =", int(back.AuthMode), "hidden =", hidden)
	if back.SSID != apSSID || int(back.Channel) != apChannel ||
		back.AuthMode != wifi.AuthWPA2PSK || back.MaxSTA != apMaxSTA || hidden != 0 {
		println("[go] ap FAIL config_readback")
		return
	}
	println("[go] ap config readback ok")

	// ---- Start and watch it come up --------------------------------------
	apEvents := wifi.Subscribe(wifi.EvAPStart, wifi.EvAPStop,
		wifi.EvAPStaConnected, wifi.EvAPStaDisconnected, wifi.EvAPStaIPAssigned)
	// The station kinds are watched too, to prove they stay silent while the
	// firmware serves as an AP - except EvStaStart, which the APSTA restart
	// must deliver once (the station interface coming up).
	staEvents := wifi.Subscribe(wifi.EvStaStart, wifi.EvStaConnected,
		wifi.EvStaDisconnected, wifi.EvGotIP)

	if err := wifi.Start(); err != nil {
		println("[go] wifi start err =", errText(err))
		println("[go] ap FAIL start")
		return
	}
	if _, ok := waitKind(apEvents, wifi.EvAPStart, 5000); !ok {
		println("[go] ap: no AP_START in 5s")
		println("[go] ap FAIL ap_start")
		return
	}
	println("[go] ap START")
	heapMark("ap-start")

	ip, err := wifi.GetAPIP()
	if err != nil {
		println("[go] wifi get_ap_ip err =", errText(err))
		println("[go] ap FAIL get_ap_ip")
		return
	}
	println("[go] ap ip = " + wifi.IPString(ip))
	if wifi.IPString(ip) != apExpectIP {
		println("[go] ap FAIL ap_ip")
		return
	}
	println("[go] ap ip PASS")

	// Before a client is up the list must be a valid empty slice: "nobody is
	// associated" is reported, not an error.
	sts, err := wifi.APStations()
	if err != nil {
		println("[go] wifi ap_stations err =", errText(err))
		println("[go] ap FAIL stations")
		return
	}
	// Informational: the emulator joins as soon as it sees the beacons, so this
	// count may be 0 or 1 depending on timing. What matters is that the call
	// answers with a slice instead of an error.
	println("[go] ap stations on start n =", len(sts))

	// ---- A client associates and the AP's DHCP server leases it ----------
	staEv, ok := waitKind(apEvents, wifi.EvAPStaConnected, 30000)
	if !ok {
		println("[go] ap: no client associated in 30s")
		println("[go] ap FAIL no_client")
		return
	}
	println("[go] ap STA_CONNECTED aid =", int(staEv.AID), "mac =", stationMAC(staEv.MAC))
	clientMAC := staEv.MAC

	ipEv, ok := waitKind(apEvents, wifi.EvAPStaIPAssigned, 15000)
	if !ok {
		println("[go] ap: no AP_STAIPASSIGNED in 15s")
		println("[go] ap FAIL no_lease")
		return
	}
	println("[go] ap STA_IPASSIGNED ip = " + wifi.IPString(ipEv.IP))
	if ipEv.IP == 0 {
		println("[go] ap FAIL lease_ip")
		return
	}

	sts, err = wifi.APStations()
	if err != nil {
		println("[go] wifi ap_stations err =", errText(err))
		println("[go] ap FAIL stations")
		return
	}
	println("[go] ap stations n =", len(sts))
	for i := range sts {
		println("[go] ap station mac =", sts[i].MACString(),
			"rssi =", int(sts[i].RSSI), "ip =", wifi.IPString(sts[i].IP))
	}
	if len(sts) == 0 || sts[0].IP == 0 {
		println("[go] ap FAIL station_list")
		return
	}
	println("[go] ap station list PASS")
	println("[go] ap ready")

	// ---- One TCP session over the soft AP --------------------------------
	// The listener binds the wildcard address, so it answers on the AP netif's
	// 192.168.4.1. The host reaches it through esp-emu's hostfwd rule, which
	// rides the emulator's station link into this AP.
	lis, err := net.Listen(apPort)
	if err != nil {
		println("[go] ap tcp listen err =", errText(err))
		println("[go] ap FAIL tcp_listen")
		return
	}
	defer lis.Close()
	println("[go] ap tcp listen port =", apPort)

	served := false
	for waited := int32(0); waited < acceptBudgetMs && !served; waited += 2000 {
		if err := lis.SetDeadline(2000); err != nil {
			println("[go] ap tcp deadline err =", errText(err))
			break
		}
		conn, err := lis.Accept()
		if err != nil {
			continue
		}
		if err := conn.SetDeadline(10000); err != nil {
			conn.Close()
			continue
		}
		req := make([]byte, 64)
		n, rerr := conn.Read(req)
		if n > 0 {
			println("[go] ap tcp request = " + trimEOL(string(req[:n])))
		}
		reply := "llgo-ap-ok\n"
		if _, werr := conn.Write([]byte(reply)); werr != nil {
			println("[go] ap tcp write err =", errText(werr))
			conn.Close()
			continue
		}
		conn.Close()
		if rerr == nil && n > 0 {
			served = true
			println("[go] ap tcp reply sent")
		}
	}
	if !served {
		println("[go] ap: no client session within the budget")
		println("[go] ap FAIL tcp")
		return
	}
	println("[go] ap TCP PASS")
	println("[go] ap PASS")
	heapMark("tcp")

	// ---- Deauthenticate the client ---------------------------------------
	// DeauthStation resolves the AID from the association events; the
	// departure must show up as EvAPStaDisconnected with the same MAC, and it
	// must leave the station reconnect machine idle.
	settle()
	drain(apEvents)
	if err := wifi.DeauthStation(clientMAC); err != nil {
		println("[go] ap deauth err =", errText(err))
		println("[go] ap deauth skipped")
	} else if ev, ok := waitKind(apEvents, wifi.EvAPStaDisconnected, 5000); ok {
		println("[go] ap STA_DISCONNECTED mac =", stationMAC(ev.MAC), "reason =", int(ev.Aux))
		println("[go] ap deauth PASS")
		// Immediately after the departure the list must be empty: the "nobody
		// is associated" answer, an empty slice and not an error.
		after, aerr := wifi.APStations()
		if aerr == nil && len(after) == 0 {
			println("[go] ap empty station list PASS")
		} else {
			println("[go] ap stations after deauth n =", len(after), "err =", errText(aerr))
		}
		// A MAC nobody associated with: the driver answers (NOT_FOUND), the
		// glue does not guess an AID, and nothing is deauthenticated.
		if uerr := wifi.DeauthStation([6]uint8{0x02, 0, 0, 0, 0, 1}); uerr != nil {
			println("[go] ap deauth unknown mac err =", errText(uerr))
			println("[go] ap deauth unknown mac PASS")
		} else {
			println("[go] ap: DeauthStation accepted an unknown MAC")
			println("[go] ap deauth unknown mac FAIL")
		}
	} else {
		println("[go] ap: no STA_DISCONNECTED in 5s after deauth")
		println("[go] ap deauth FAIL")
	}

	// ---- Stop and restart in APSTA mode ----------------------------------
	settle()
	drain(apEvents)
	if err := wifi.Stop(); err != nil {
		println("[go] wifi stop err =", errText(err))
		println("[go] ap FAIL stop")
		return
	}
	if _, ok := waitKind(apEvents, wifi.EvAPStop, 5000); !ok {
		println("[go] ap: no AP_STOP in 5s")
		println("[go] ap FAIL ap_stop")
		return
	}
	println("[go] ap STOP")
	heapMark("ap-stop")

	// APSTA: pick the mode explicitly, then configure the station interface.
	// Neither call may disturb the AP config set above - that is the contract
	// SetAP/SetSTA document, and the readbacks below are its evidence. The
	// station is pointed at an SSID nobody serves: it is configured, never
	// connected, which is what this probe can verify without a second network.
	if err := wifi.SetModeAPSTA(); err != nil {
		println("[go] wifi set_mode_apsta err =", errText(err))
		println("[go] ap FAIL apsta_mode")
		return
	}
	println("[go] ap mode APSTA")
	if err := wifi.SetSTA("llgo-ap-nosuchnet", "nosuchpassword"); err != nil {
		println("[go] wifi set_sta err =", errText(err))
		println("[go] ap FAIL apsta_set_sta")
		return
	}
	println("[go] ap sta configured (no connect attempted)")
	back, err = wifi.GetAPConfig()
	if err != nil {
		println("[go] wifi get_ap_config err =", errText(err))
		println("[go] ap FAIL apsta_config")
		return
	}
	if back.SSID != apSSID || int(back.Channel) != apChannel || back.AuthMode != wifi.AuthWPA2PSK {
		println("[go] ap: AP config changed under APSTA: ssid="+back.SSID,
			"ch =", int(back.Channel), "auth =", int(back.AuthMode))
		println("[go] ap FAIL apsta_config")
		return
	}
	if startErr := wifi.Start(); startErr != nil {
		println("[go] wifi start err =", errText(startErr))
		println("[go] ap FAIL apsta_start")
		return
	}
	if _, ok := waitKind(apEvents, wifi.EvAPStart, 5000); !ok {
		println("[go] ap: no AP_START in 5s after the APSTA restart")
		println("[go] ap FAIL apsta_start")
		return
	}
	ip, err = wifi.GetAPIP()
	if err != nil || wifi.IPString(ip) != apExpectIP {
		println("[go] ap: AP address after APSTA =", wifi.IPString(ip), "err =", errText(err))
		println("[go] ap FAIL apsta_ip")
		return
	}
	println("[go] ap APSTA restart ok, ap ip = " + wifi.IPString(ip))
	if _, ok := waitKind(staEvents, wifi.EvStaStart, 3000); ok {
		println("[go] ap APSTA station interface started")
	} else {
		println("[go] ap: no STA_START in 3s after the APSTA restart")
		println("[go] ap FAIL apsta_sta_start")
		return
	}
	heapMark("apsta")

	// ---- The station half must have stayed silent -------------------------
	settle()
	staSeen := drain(staEvents)
	phase := wifi.ReconnectState().String()
	wifi.SetAutoReconnect(false)
	println("[go] ap reconn: phase =", phase, "sta link events =", staSeen)
	if phase != "idle" || staSeen != 0 {
		println("[go] ap FAIL sta_quiet")
	} else {
		println("[go] ap sta quiet PASS")
	}

	settle()
	drain(apEvents)
	if err := wifi.Stop(); err != nil {
		println("[go] wifi stop err =", errText(err))
		println("[go] ap FAIL final_stop")
		return
	}
	if _, ok := waitKind(apEvents, wifi.EvAPStop, 5000); !ok {
		println("[go] ap: no AP_STOP in 5s after the final Stop")
		println("[go] ap FAIL final_stop")
		return
	}
	println("[go] ap final stop ok")
	heapMark("done")

	// Mode conflict, as a checked claim rather than documentation prose: an
	// explicitly chosen STA mode has no AP interface, so the driver refuses the
	// AP configuration. The code it reports is the one the package documents.
	if err := wifi.SetModeSTA(); err != nil {
		println("[go] ap mode-conflict set_mode_sta err =", errText(err))
		println("[go] ap FAIL mode_conflict")
		return
	}
	if err := wifi.SetAP(cfg); err != nil {
		println("[go] ap mode-conflict set_ap err =", errText(err))
		println("[go] ap mode-conflict PASS")
	} else {
		println("[go] ap: SetAP accepted under an explicit STA mode")
		println("[go] ap FAIL mode_conflict")
		return
	}
	println("[go] ap done")
}

// stationMAC renders a MAC the way the event and station records carry it:
// aa:bb:cc:dd:ee:ff.
func stationMAC(mac [6]uint8) string {
	return wifi.Station{MAC: mac}.MACString()
}

// trimEOL drops the trailing CR/LF a client sends with its request line, so
// the logged line stays one line.
func trimEOL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
