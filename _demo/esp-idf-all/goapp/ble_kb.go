package main

import (
	"time"

	"github.com/xgo-dev/llgo/pkg/esp/ble"
)

// BLE keyboard host.
const (
	kbRoundMs     = 10000 // one scan round
	kbConnectMs   = 15000
	kbSecPollMs   = 500 // one security-state poll
	kbSecWaitN    = 16  // polls before giving up on encryption (~8 s)
	appearanceKbd = 0x03C1
)

func bleInit() {
	if err := ble.Init(); err != nil {
		println("[go] ble FAIL init err =", errText(err))
		return
	}
	println("[go] ble init ok backend =", ble.Backend(),
		"(shared Bluedroid stack - keyboard pairing ready)")
	heapLedger("ble-init")
}

// kbScanStep services one scan round: the first advertising keyboard gets
// connected (state 2), the round's expiry returns to state 0.
func kbScanStep(sub *ble.Subscription) int {
	for {
		ev, err := sub.Next(200)
		if err != nil {
			wifiPump()
			continue
		}
		switch ev.Kind {
		case int32(ble.EvAdv):
			adv := ev.AdvData()
			app := advAppearance(adv)
			hid := advHasUUID16(adv, 0x1812)
			name := ble.AdvName(adv)
			if !hid && (app < appearanceKbd || app > appearanceKbd+3) && !nameHasKeyboard(name) {
				continue
			}
			println(tagl()+"KEYBOARD-FOUND name =", dash(name),
				"addr =", ble.AddrString(ev.Addr),
				"addrtype =", int(ev.AddrType), "rssi =", int(ev.Rssi),
				"hid =", bool32(hid), "app =", hex16(app))
			if serr := ble.StopScan(); serr != nil {
				println(tagl()+"scan stop err =", errText(serr))
			}
			// Swap to an adv-free subscription: a fresh, empty channel
			// that the link-setup events cannot race with (see sub).
			bleSub.Close()
			bleSub = ble.Subscribe(ble.EvScanDone, ble.EvConnect,
				ble.EvDisconnect, ble.EvError, ble.EvMtu, ble.EvNotify)
			// Connect immediately: this keyboard rotates its address on
			// every advertising burst, so the discovered address is only
			// alive for the current burst. Any delay here (e.g. draining
			// stale scan events) makes the controller target a dead
			// address and the link fails with HCI 0x13. The connect wait
			// below is time-based and skips scan events, so no drain is
			// needed.
			op, cerr := ble.Connect(ev.Peer(), ble.ConnectOptions{TimeoutMs: kbConnectMs})
			if cerr != nil {
				println(tagl()+"connect err =", errText(cerr))
				return 0
			}
			kbOp = op
			return 2
		case int32(ble.EvScanDone):
			return 0
		case int32(ble.EvError):
			println(tagl()+"scan err =", errText(ble.ErrStatus(ev.Status)))
		}
		wifiPump()
	}
}

// kbT0 is the tick at which the link came up, the zero for the kbStep
// stage offsets.
var kbT0 uint32

// kbOp is the connect attempt in flight (state 2); kbScanStep stores it,
// kbConnectStep (via the main loop) matches it against the connect event.
var kbOp ble.OpID

// kbBondConnectMs bounds a bonded reconnect attempt: long enough for a link
// the peer accepts immediately, short enough that a stale bond does not stall
// the round-robin before the scan retry.
const kbBondConnectMs = 6000

// bleSub is the package-level keyboard subscription. It is re-subscribed
// without EvAdv once the keyboard is found: the event pump drops events
// when a subscriber's channel is full, and a live scan flood keeps the
// channel full - a connect event arriving into that channel would be
// dropped unseen. A fresh channel without the advertising kind cannot
// fill up: nothing the link setup produces is that fast.
var bleSub *ble.Subscription

// kbConnectStep waits for the attempt's completion: the connection id moves
// the machine to the pairing state, anything else back to scanning.
func kbConnectStep(bleSub *ble.Subscription, op ble.OpID) (int, ble.ConnID) {
	// Time-based deadline: backlog events return from Next instantly, so a
	// count-based loop can exhaust in under a second without ever seeing
	// the connect event. Bluedroid connects may also exceed 8 s.
	connectStart := time.Now()
	connectDeadline := connectStart.Add(20 * time.Second)
	for time.Now().Before(connectDeadline) {
		ev, err := bleSub.Next(500)
		if err != nil {
			wifiPump()
			continue
		}
		if ev.Kind == int32(ble.EvConnect) && ev.OpID == op {
			if ev.Status != 0 {
				println(tagl()+"connect status =", ble.ErrName(ev.Status))
				return 0, 0
			}
			conn := ev.ConnID
			kbT0 = nowMs()
			println(tagl()+"connect ok conn =", dec32(uint32(conn)))
			return 3, conn
		}
	}
	println(tagl()+"connect timeout waited_ms =",
		int(time.Since(connectStart).Milliseconds()))
	return 0, 0
}

// kbServe runs pairing, the GATT walk and the report window for one
// keyboard, then returns the machine to scanning. Every failure path drops
// the connection and restarts the scan rounds.
func kbServe(bleSub *ble.Subscription, conn ble.ConnID) (int, ble.ConnID) {
	// Start pairing right away: HID characteristics refuse unencrypted
	// access. The PASSKEY prints on the C side for the operator to type.
	rc := secInitiate(uint32(conn))
	println(tagl()+"security initiate rc =", int(rc), "(PASSKEY prints above when the keyboard asks)")
	sec := int32(-1)
	for i := 0; i < kbSecWaitN; i++ {
		if s := connSec(uint32(conn)); s >= 0 && s&1 == 1 {
			sec = s
			break
		}
		yieldMs(kbSecPollMs)
		wifiPump()
	}
	if sec < 0 {
		println(tagl() + "sec timeout (no encryption in ~8 s - continuing, reads may be refused)")
	} else {
		kbStep("enc ok")
		println(tagl()+"ENC enc =", bool32(sec&1 == 1), "auth =", bool32(sec&2 == 2),
			"bonded =", bool32(sec&4 == 4))
	}

	if !kbWalk(bleSub, conn) {
		return kbDrop(bleSub, conn)
	}
	// The report window: every HID payload prints as raw hex until the link
	// drops. Wifi events are pumped between waits so the AP keeps moving.
	kbStep("REPORTS-WINDOW open")
	n := 0
	for {
		ev, err := bleSub.Next(300)
		if err != nil {
			wifiPump()
			continue
		}
		if ev.Kind == int32(ble.EvDisconnect) && ev.ConnID == conn {
			println(tagl()+"disconnect reason =", ble.DisconnectReasonName(ev.Status),
				"after", n, "reports")
			return 0, 0
		}
		if ev.Kind != int32(ble.EvNotify) || ev.ConnID != conn {
			wifiPump()
			continue
		}
		n++
		println(tagl()+"report n =", n, "handle =", dec32(uint32(ev.AttHandle)),
			"=>", hidDecode(ev.DataBytes()))
	}
}

// kbDrop disconnects and returns to the scan state.
func kbDrop(bleSub *ble.Subscription, conn ble.ConnID) (int, ble.ConnID) {
	if derr := ble.Disconnect(conn); derr == nil {
		for i := 0; i < 10; i++ {
			ev, err := bleSub.Next(500)
			if err == nil && ev.Kind == int32(ble.EvDisconnect) && ev.ConnID == conn {
				break
			}
		}
	}
	return 0, 0
}

// kbStep prints a stage line prefixed with the milliseconds elapsed since
// the link came up (kbT0), so each phase's setup cost is comparable across
// runs without decoding absolute tick stamps.
func kbStep(label string) {
	println(tagl()+"+", int((nowMs()-kbT0)&0x7FFFFFFF), "ms", label)
}

func kbWalk(bleSub *ble.Subscription, conn ble.ConnID) bool {
	svcs, err := ble.Services(conn)
	if err != nil {
		println(tagl()+"services err =", errText(err))
		return false
	}
	// One summary line per phase: the walk runs a serial BTC round-trip per
	// read/subscribe, and per-item logging made the discovery look like a
	// flood of unrelated noise.
	hidSvc, hidFound := ble.Service{}, false
	names := make([]string, 0, len(svcs))
	for _, s := range svcs {
		u := s.UUID.String()
		names = append(names, u)
		if u == "1812" {
			hidSvc, hidFound = s, true
		}
	}
	kbStep("services discovered")
	println(tagl()+"services count =", len(svcs), ":", joinStrings(names, " "))
	if !hidFound {
		println(tagl() + "FAIL no hid service (not a keyboard?)")
		return false
	}

	chrs, err := ble.Characteristics(conn, hidSvc.UUID)
	if err != nil {
		println(tagl()+"chrs err =", errText(err))
		return false
	}
	notifyN, subs := 0, 0
	for i := range chrs {
		if chrs[i].CanNotify() {
			notifyN++
		}
	}
	kbStep("hid characteristics discovered")
	println(tagl()+"hid chrs =", len(chrs), "notify =", notifyN)

	for i := range chrs {
		c := chrs[i]
		u := c.UUID.String()
		// No reads: protocol-mode and hid-info are descriptive, and a
		// report characteristic reads back empty by design - every read
		// here costs ~1.5-2.6 s of ATT round-trips and buys nothing.
		// Subscribe the first notifying input report only: every key press
		// of this keyboard arrives on that one characteristic, and each
		// further CCCD write is a full ATT round-trip (~1.7 s on this link)
		// that buys no additional keys.
		if c.CanNotify() && u == "2a4d" && subs == 0 {
			if _, serr := ble.SubscribeChar(conn, c); serr != nil {
				println(tagl()+"subscribe", chrName(u), "handle =",
					dec32(uint32(c.ValueHandle)), "err =", errText(serr))
			} else {
				subs++
			}
		}
		wifiPump()
	}
	kbStep("subscriptions done")
	println(tagl()+"subscribed =", subs, "(first input report;", notifyN, "notify chars exist)")
	return true
}

// nameHasKeyboard reports whether the advertised name mentions a keyboard
// (case-insensitive), without dragging the strings package into the image.
func nameHasKeyboard(name string) bool {
	const probe = "keyb"
	if len(name) < len(probe) {
		return false
	}
	for i := 0; i+len(probe) <= len(name); i++ {
		match := true
		for j := 0; j < len(probe); j++ {
			c := name[i+j]
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}
			if c != probe[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// advHasUUID16 reports whether the payload's 16-bit service UUID lists
// (incomplete 0x02, complete 0x03, solicitation 0x14) contain uuid.
func advHasUUID16(adv []uint8, uuid uint16) bool {
	for i := 0; i < len(adv); {
		l := int(adv[i])
		if l == 0 || i+1+l > len(adv) {
			return false
		}
		t := adv[i+1]
		if t == 0x02 || t == 0x03 || t == 0x14 {
			for j := i + 2; j+1 < i+1+l; j += 2 {
				if uint16(adv[j])|uint16(adv[j+1])<<8 == uuid {
					return true
				}
			}
		}
		i += 1 + l
	}
	return false
}

// advAppearance returns the payload's appearance value (AD type 0x19), 0
// when absent.
func advAppearance(adv []uint8) uint16 {
	for i := 0; i < len(adv); {
		l := int(adv[i])
		if l == 0 || i+1+l > len(adv) {
			return 0
		}
		if adv[i+1] == 0x19 && l >= 3 {
			return uint16(adv[i+2]) | uint16(adv[i+3])<<8
		}
		i += 1 + l
	}
	return 0
}

// chrName names the HID-profile characteristics by their SIG UUIDs.
func chrName(u string) string {
	switch u {
	case "2a22":
		return "boot-kb-input"
	case "2a32":
		return "boot-kb-output"
	case "2a33":
		return "boot-mouse-input"
	case "2a4a":
		return "hid-info"
	case "2a4b":
		return "report-map"
	case "2a4c":
		return "hid-control-point"
	case "2a4d":
		return "report"
	case "2a4e":
		return "protocol-mode"
	}
	return "-"
}

// hexBytes renders a payload as lowercase hex pairs, space separated.
// hidKeyNames maps a HID Usage ID (boot keyboard) to its printable name.
// Keys outside the mapped range print as <hex> so nothing is swallowed.
var hidKeyNames = map[byte]string{
	0x04: "a", 0x05: "b", 0x06: "c", 0x07: "d", 0x08: "e", 0x09: "f",
	0x0a: "g", 0x0b: "h", 0x0c: "i", 0x0d: "j", 0x0e: "k", 0x0f: "l",
	0x10: "m", 0x11: "n", 0x12: "o", 0x13: "p", 0x14: "q", 0x15: "r",
	0x16: "s", 0x17: "t", 0x18: "u", 0x19: "v", 0x1a: "w", 0x1b: "x",
	0x1c: "y", 0x1d: "z",
	0x1e: "1", 0x1f: "2", 0x20: "3", 0x21: "4", 0x22: "5",
	0x23: "6", 0x24: "7", 0x25: "8", 0x26: "9", 0x27: "0",
	0x28: "Enter", 0x29: "Esc", 0x2a: "Backspace", 0x2b: "Tab", 0x2c: "Space",
	0x2d: "-", 0x2e: "=", 0x2f: "[", 0x30: "]", 0x31: "backslash", 0x33: ";",
	0x34: "'", 0x35: "grave", 0x36: ",", 0x37: ".", 0x38: "/",
	0x39: "CapsLock", 0x4f: "Right", 0x50: "Left", 0x51: "Down", 0x52: "Up",
}

// hidDecode renders a boot-keyboard input report as readable presses, e.g.
// "[LShift] a" while typing uppercase A and "release" on the key-up report.
func hidDecode(b []byte) string {
	if len(b) < 3 {
		return "short(" + hexBytes(b) + ")"
	}
	mods := b[0]
	var sb []byte
	for _, m := range []struct {
		mask byte
		name string
	}{
		{0x01, "[LCtrl] "}, {0x02, "[LShift] "}, {0x04, "[LAlt] "}, {0x08, "[LGUI] "},
		{0x10, "[RCtrl] "}, {0x20, "[RShift] "}, {0x40, "[RAlt] "}, {0x80, "[RGUI] "},
	} {
		if mods&m.mask != 0 {
			sb = append(sb, m.name...)
		}
	}
	keys := 0
	for _, k := range b[2:] {
		if k == 0 {
			continue
		}
		if name, ok := hidKeyNames[k]; ok {
			sb = append(sb, name...)
		} else {
			sb = append(sb, '<')
			sb = append(sb, hexDigits[k>>4], hexDigits[k&0xF])
			sb = append(sb, '>')
		}
		sb = append(sb, ' ')
		keys++
	}
	if keys == 0 {
		if len(sb) == 0 {
			return "release"
		}
		return string(sb) + "release"
	}
	return string(sb[:len(sb)-1])
}
