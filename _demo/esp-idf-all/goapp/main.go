// Comprehensive verification firmware for llgo on ESP32-S31: one firmware
// exercising every capability at once, so a single board session answers all
// of them.
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
//
// The firmware brings up, concurrently:
//
//   - soft AP "llgo-ap" (WPA2, channel 6) with a DHCP server and a tiny HTTP
//     server on port 8000 (GET / answers "llgo-ap-ok"); every station is
//     deauthenticated 40 s after it associated, so the whole AP lifecycle
//     (associate -> lease -> serve -> kick -> reconnect) cycles on its own.
//   - an A2DP sink named "llgo-audio": decoded PCM streams to the ES8311 /
//     NS4150B analog path through the pkg/esp/bt glue's I2S pipeline, and the
//     control plane logs connect / codec-config / volume events plus the
//     byte counters' per-tick deltas (~352800 B per 2 s at 44.1 kHz stereo).
//   - a BLE central host on the SAME Bluedroid stack (dual-mode controller):
//     scan rounds every 10 s, connect to the first HID keyboard it recognizes
//     (appearance 0x03C1..0x03C4, HID service UUID 0x1812, or a name
//     containing "keyboard"), run the pairing flow (passkey is printed on
//     this console - type it on the keyboard), then subscribe to every
//     notifying HID characteristic and print each report as raw hex.
//   - the ES8311 codec: brought up over I2C (0x18; the board's CE pin is
//     low, which is Espressif's own address-select scheme, not a disable)
//     with the audited register sequence, then a self-test - 3 s of 440 Hz
//     sine through the speaker, a mic gain sweep with RMS statistics, and a
//     loopback replay of the capture - pausing A2DP for the duration (the
//     I2S clock pins are exclusive), then reconfiguring the codec for A2DP
//     rates and re-enabling the sink. From then on the amp follows the A2DP
//     link: on when a phone streams, off when it leaves.
//
// Every subsystem degrades independently: a failed bring-up prints a FAIL
// line and the rest keeps running, so one broken capability never costs the
// evidence for the others. Heap watermarks print at each bring-up stage and
// every minute - the all-on memory ledger.
package main

import (
	"runtime"

	"github.com/xgo-dev/llgo/pkg/esp/ble"
	"github.com/xgo-dev/llgo/pkg/esp/wifi"
)

func main() {
	println("[go] all: llgo comprehensive verification firmware (WiFi AP + A2DP + BLE keyboard + audio)")
	println("[go] all: sequence - 1) join AP llgo-ap, open http://192.168.4.1:8000/",
		"2) phone connects llgo-audio and plays",
		"3) keyboard in pairing mode (PASSKEY prints here)",
		"(the codec self-test runs first, ~20 s of A2DP pause)")

	// Scheduler sanity before any subsystem work.
	ch := make(chan int)
	go produce(ch, 42)
	if got := <-ch; got != 42 {
		println("[go] all FAIL scheduler roundtrip")
		return
	}
	println("[go] all goroutine roundtrip ok")
	heapLedger("baseline")

	btInit()
	bleInit()
	a2dpInit()
	apInit()

	go a2dpLoop()
	go httpLoop()
	go codecLoop()

	// The keyboard host and the AP station management share the MAIN
	// goroutine: the BLE and esp_wifi control calls run deep C paths (BTC
	// message copies, driver API locks) that need the main fiber's stack -
	// the same stack every verified bt/ble/wifi probe ran its lifecycle on.
	// The background fibers keep to the light drivers their probes proved
	// (lwip poll, bt/ble mailbox polls, i2c/i2s).
	controlLoop()
}

func produce(ch chan int, v int) { ch <- v }

// The control loop's wifi-side state (single consumer: the main goroutine;
// the cooperative scheduler makes the plain globals race-free).
var (
	ctlWifiSub   *wifi.Subscription
	ctlStations  map[[6]uint8]uint32
	ctlBusyUntil uint32 // queries wait until the AP has been quiet this long
	ctlNeedTable bool
)

// controlLoop is the MAIN goroutine's service loop. It owns two event
// streams and the keyboard state machine:
//
//   - the soft AP's station events (association, lease, departure) and the
//     quiet-window jobs over them (station table print, the 40 s kick);
//   - the BLE keyboard host: scan rounds every 10 s, connect on the first
//     recognized keyboard, the pairing flow, the GATT walk, the report
//     window.
//
// Why the MAIN goroutine runs the BLE and esp_wifi control calls: both go
// through deep C paths (BTC message deep copies, the driver's API locks)
// that need the main fiber's stack headroom - the same stack every verified
// bt/ble/wifi probe ran its lifecycle on. The background fibers stay on the
// light drivers their own probes proved (lwip poll, bt/ble mailbox polls,
// i2c/i2s). During a synchronous GATT step (Services/Read/SubscribeChar,
// each internally bounded and Gosched-yielding) wifi events queue in the
// driver's ring and are drained right after.
func controlLoop() {
	ctlWifiSub = wifi.Subscribe(wifi.EvAPStaConnected, wifi.EvAPStaDisconnected,
		wifi.EvAPStaIPAssigned, wifi.EvAPStop)
	defer ctlWifiSub.Close()
	bleSub = ble.Subscribe(ble.EvAdv, ble.EvScanDone, ble.EvConnect,
		ble.EvDisconnect, ble.EvError, ble.EvMtu, ble.EvNotify)
	defer func() {
		if bleSub != nil {
			bleSub.Close()
		}
	}()

	ctlStations = make(map[[6]uint8]uint32)

	// The keyboard machine starts after the codec self-test finished: one
	// less moving part in the boot window. The memory cadence already runs
	// here, so the self-test window shows its own ledger line.
	lastMem := nowMs()
	for !selftestDone {
		lastMem = memWatch(lastMem)
		wifiPump()
		yieldMs(500)
	}

	kbRound := 0
	// -1 bonded reconnect (tried once before the first scan), 0 scan-needed,
	// 1 scanning, 2 connecting, 3 pairing+serve.
	kbState := -1
	kbBondTried := false
	var kbConn ble.ConnID
	for {
		lastMem = memWatch(lastMem)
		// One wifi event or a short idle, then one keyboard step: both
		// streams keep moving.
		ev, err := ctlWifiSub.Next(200)
		if err == nil {
			apHandle(ev)
		}
		switch kbState {
		case -1: // a stored bond connects straight away (every round:
			// a keyboard that reboots mid-session is caught here instead of
			// waiting for its pairing advertisement, which a bonded peer
			// does not send)
			kbBondTried = true
			if bs, berr := ble.BondedPeers(); berr != nil {
				println(tagl()+"bond list err =", errText(berr))
				kbState = 0
			} else if len(bs) == 0 {
				kbState = 0
			} else {
				println(tagl() + "bonded peer stored - connecting without a scan")
				kbOp = 0
				op, cerr := ble.Connect(bs[0], ble.ConnectOptions{TimeoutMs: kbBondConnectMs})
				if cerr != nil {
					println(tagl()+"bonded connect err =", errText(cerr))
					kbState = 0
				} else {
					kbOp = op
					kbState = 2
				}
			}
		case 0: // start a scan round
			kbRound++
			println(tagl()+"round", kbRound, "- scan", kbRoundMs/1000, "s (put a keyboard in pairing mode)")
			op, serr := ble.StartScan(ble.ScanOptions{FilterDuplicates: true, DurationMs: kbRoundMs})
			_ = op
			if serr != nil {
				println(tagl()+"scan err =", errText(serr))
				yieldMs(5000)
				continue
			}
			kbState = 1
		case 1: // scanning: watch the round's events
			kbState = kbScanStep(bleSub)
		case 2: // connecting: the attempt kbOp is in flight
			kbState, kbConn = kbConnectStep(bleSub, kbOp)
		case 3: // pairing + walk + reports for kbConn
			kbState, kbConn = kbServe(bleSub, kbConn)
			if kbState == 0 && kbBondTried {
				kbState = -1 // retry the stored bond before scanning again
			}
		}
	}
}

// yieldMs is this firmware's only wait primitive: a busy Gosched loop with
// a wall-clock bound - the wait shape every verified probe uses (the
// subscription Next loops). Deliberately NOT time.Sleep: the fiber sleeper
// list is unproven under concurrency in this repo, and one run died inside
// the runtime's goroutine status machine with sleeps in flight.
func yieldMs(n int32) {
	deadline := nowMs() + uint32(n)
	for int32(nowMs()-deadline) < 0 {
		runtime.Gosched()
	}
}

func bool32(v bool) uint32 {
	if v {
		return 1
	}
	return 0
}
