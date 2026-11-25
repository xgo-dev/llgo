package main

import (
	espnet "github.com/xgo-dev/llgo/pkg/esp/net"
	"github.com/xgo-dev/llgo/pkg/esp/wifi"
)

// Soft AP. apKickMs is how long a station may stay associated before the
// firmware deauthenticates it - the acceptance script's kick step, running
// on its own. A kicked phone simply rejoins for another window.
const (
	apSSID    = "llgo-ap"
	apPass    = "llgoappass"
	apChannel = 6
	apMaxSTA  = 4
	apPort    = 8000

	apExpectIP = "192.168.4.1"
	apKickMs   = 40000
	apQuietMs  = 5000 // the esp_wifi query gate: 5 s without an AP event first
	httpRespOK = "llgo-ap-ok"
)

func apInit() {
	if err := wifi.Setup(); err != nil {
		println("[go] ap FAIL setup err =", errText(err))
		return
	}
	if err := wifi.EventsRegister(); err != nil {
		println("[go] ap FAIL events err =", errText(err))
		return
	}
	// The AP firmware never connects out as a station; the auto-reconnect
	// machine would only react to station events that cannot happen here.
	wifi.SetAutoReconnect(false)

	cfg := wifi.APConfig{
		SSID:     apSSID,
		Password: apPass,
		Channel:  apChannel,
		MaxSTA:   apMaxSTA,
		AuthMode: wifi.AuthWPA2PSK,
	}
	if err := wifi.SetAP(cfg); err != nil {
		println("[go] ap FAIL set_ap err =", errText(err))
		return
	}
	back, err := wifi.GetAPConfig()
	if err != nil || back.SSID != apSSID || int(back.Channel) != apChannel {
		println("[go] ap FAIL config_readback err =", errText(err))
		return
	}
	println("[go] ap config ok ssid =", back.SSID, "ch =", int(back.Channel),
		"maxsta =", back.MaxSTA, "auth =", int(back.AuthMode))

	apEvents := wifi.Subscribe(wifi.EvAPStart, wifi.EvAPStop)
	if err := wifi.Start(); err != nil {
		println("[go] ap FAIL start err =", errText(err))
		return
	}
	if _, ok := waitWifi(apEvents, wifi.EvAPStart, 5000); !ok {
		println("[go] ap FAIL ap_start timeout")
		return
	}
	ip, err := wifi.GetAPIP()
	if err != nil || wifi.IPString(ip) != apExpectIP {
		println("[go] ap FAIL ap_ip", wifi.IPString(ip), errText(err))
		return
	}
	println("[go] ap START ip =", wifi.IPString(ip))
	heapLedger("ap-start")
}

// apHandle services one soft-AP event (association, lease, departure).
func apHandle(ev wifi.Event) {
	ctlBusyUntil = nowMs() + apQuietMs // an event burst just happened
	switch ev.Kind {
	case wifi.EvAPStaConnected:
		ctlStations[ev.MAC] = nowMs()
		ctlNeedTable = true
		println("[go] ap STA_CONNECTED mac =", wifi.Station{MAC: ev.MAC}.MACString(),
			"aid =", int(ev.AID))
	case wifi.EvAPStaIPAssigned:
		ctlNeedTable = true
		println("[go] ap LEASE ip =", wifi.IPString(ev.IP),
			"mac =", wifi.Station{MAC: ev.MAC}.MACString())
	case wifi.EvAPStaDisconnected:
		delete(ctlStations, ev.MAC)
		ctlNeedTable = true
		println("[go] ap STA_DISCONNECTED mac =", wifi.Station{MAC: ev.MAC}.MACString(),
			"reason =", int(ev.Aux))
	case wifi.EvAPStop:
		println("[go] ap STOP (unexpected in this firmware)")
	}
}

// wifiPump drains pending wifi events without blocking and runs the
// quiet-window jobs (station table, the 40 s kick) when the driver's API
// lock has had its rest.
func wifiPump() {
	for {
		ev, err := ctlWifiSub.Next(0)
		if err != nil {
			break
		}
		apHandle(ev)
	}
	now := nowMs()
	if int32(ctlBusyUntil-now) >= 0 {
		return
	}
	if ctlNeedTable {
		ctlNeedTable = false
		printStations()
	}
	// Kick stations past their window - in the same quiet gate, and without
	// any query afterwards (the station leaves the map here; its leave event
	// only re-arms the quiet gate). A kicked phone rejoins for another
	// window.
	for mac, t0 := range ctlStations {
		if now-t0 >= apKickMs {
			ctlNeedTable = false // the kick's own leave event opens no query
			if err := wifi.DeauthStation(mac); err != nil {
				println("[go] ap deauth err =", errText(err),
					"mac =", wifi.Station{MAC: mac}.MACString())
				continue
			}
			println("[go] ap DEAUTH mac =", wifi.Station{MAC: mac}.MACString(),
				"(40 s window elapsed - rejoin for another)")
			delete(ctlStations, mac)
		}
	}
}

func printStations() {
	sts, err := wifi.APStations()
	if err != nil {
		println("[go] ap stations err =", errText(err))
		return
	}
	println("[go] ap stations n =", len(sts))
	for i := range sts {
		println("[go] ap station mac =", sts[i].MACString(),
			"rssi =", int(sts[i].RSSI), "ip =", wifi.IPString(sts[i].IP))
	}
}

// httpLoop answers GET / with "llgo-ap-ok" (and anything else with 404),
// one session at a time, forever.
func httpLoop() {
	lis, err := espnet.Listen(apPort)
	if err != nil {
		println("[go] http FAIL listen err =", errText(err))
		return
	}
	defer lis.Close()
	println("[go] http listening port =", apPort)
	for {
		// Bounded accept + a yield window between attempts: the net package's
		// accept loop never Goscheds (its poll parks the single llgo FreeRTOS
		// task and the retry runs immediately after the wake), so an idle
		// listener left in an unbounded wait would starve every other
		// goroutine of the scheduler - measured on hardware as a total
		// firmware silence. One 1 s attempt, then 200 ms of parked sleep the
		// scheduler can spend on the other fibers.
		if err := lis.SetDeadline(1000); err != nil {
			println("[go] http FAIL deadline err =", errText(err))
			return
		}
		conn, err := lis.Accept()
		if err != nil {
			yieldMs(200)
			continue
		}
		conn.SetDeadline(10000)
		req := make([]byte, 128)
		n, rerr := conn.Read(req)
		line := trimEOL(string(req[:n]))
		resp := "HTTP/1.1 404 not found\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\nno such path\n"
		mark := "404"
		if len(line) >= 5 && line[0:3] == "GET" && httpRequestPath(line) == "/" {
			resp = "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 11\r\nConnection: close\r\n\r\n" + httpRespOK + "\n"
			mark = "200 " + httpRespOK
		}
		if _, werr := conn.Write([]byte(resp)); werr != nil {
			println("[go] http write err =", errText(werr))
			conn.Close()
			continue
		}
		conn.Close()
		served := "0"
		if rerr == nil && n > 0 {
			served = "1"
		}
		println("[go] http", line, "->", mark, "read =", served)
	}
}

// httpRequestPath extracts the path of a "GET /path HTTP/1.x" request line.
func httpRequestPath(line string) string {
	// "GET " prefix is checked by the caller; find the space before HTTP/.
	for i := 4; i < len(line); i++ {
		if line[i] == ' ' {
			return line[4:i]
		}
	}
	return line[4:]
}

func waitWifi(sub *wifi.Subscription, kind int32, timeoutMs int32) (wifi.Event, bool) {
	deadline := nowMs() + uint32(timeoutMs)
	for {
		ev, err := sub.Next(200)
		if err == nil && ev.Kind == kind {
			return ev, true
		}
		if int32(nowMs()-deadline) >= 0 {
			return wifi.Event{}, false
		}
	}
}
