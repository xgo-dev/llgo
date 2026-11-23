// Package espwifi binds llgo-compiled Go code to the ESP-IDF esp_wifi stack
// through the thin C glue in idfproj/main/wifi_glue.c. The Go<->C boundary
// is integer-and-pointer only (no float ABI mixing, see host/make-artifact.sh
// step 4), every C call returns an esp_err code (0 = ESP_OK), and events
// travel through a C-side mailbox that Go polls at vTaskDelay pacing instead
// of callbacks: esp_event handlers run on the event-loop task, which must
// never enter the single-task llgo fiber scheduler.
package espwifi

// Event kinds delivered by the C mailbox (see on_event in wifi_glue.c).
const (
	EvNone            = 0
	EvStaStart        = 1
	EvStaConnected    = 2
	EvStaDisconnected = 3
	EvScanDone        = 4
	EvGotIP           = 5
	EvLostIP          = 6
	EvOverflow        = -1 // the mailbox overflowed; reported once by Poll
)

// errUnsupported is what the host-build stub (wifi_stub.go) returns: it lies
// between the real esp_err ranges so it can never be mistaken for one.
const errUnsupported = 0x7F00

// Event is one mailbox slot: kind, aux (e.g. the STA_DISCONNECTED reason
// code) and the raw network-order IP for EvGotIP.
type Event struct {
	Kind int32
	Aux  int32
	IP   uint32
}

// ApInfo is the flat scan-record form marshalled by wifi_glue.c; the IDF
// wifi_ap_record_t (bitfields, nested structs) never crosses into Go.
type ApInfo struct {
	Bssid    [6]uint8
	Ssid     [33]uint8 // NUL-terminated
	Channel  uint8
	Rssi     int8
	AuthMode int32
}

// SsidString returns the AP's SSID with the trailing NUL stripped.
func (a *ApInfo) SsidString() string {
	for i, c := range a.Ssid {
		if c == 0 {
			return string(a.Ssid[:i])
		}
	}
	return string(a.Ssid[:])
}

// Setup performs the one-shot WiFi bring-up: NVS init (erase + retry on the
// fresh-flash error codes), netif init, default event loop, default STA
// netif and esp_wifi_init. WIFI_INIT_CONFIG_DEFAULT() stays in C.
func Setup() int32 { return setup() }

// EventsRegister hooks the glue's event handler into WIFI_EVENT (any id)
// and IP_EVENT_STA_GOT_IP.
func EventsRegister() int32 { return eventsRegister() }

// PrintVersion prints the IDF version and the current free heap from the C
// side; the string never crosses into Go (integer-only boundary).
func PrintVersion() int32 { return printVersion() }

// Start calls esp_wifi_start; the asynchronous WIFI_EVENT_STA_START shows up
// in the mailbox.
func Start() int32 { return start() }

// ScanStart calls esp_wifi_scan_start(NULL, blocking). A blocking scan
// freezes the llgo scheduler for the scan duration (well under a second on
// emu); the WIFI_EVENT_SCAN_DONE event still lands in the mailbox.
func ScanStart(blocking bool) int32 {
	b := int32(0)
	if blocking {
		b = 1
	}
	return scanStart(b)
}

// ScanFetch takes the scan result list into a fresh slice (at most max
// entries; the C glue holds at most 20) and also frees the driver's scan
// memory. The second result is an esp_err code.
func ScanFetch(max int32) ([]ApInfo, int32) {
	if max > 20 {
		max = 20
	}
	buf := make([]ApInfo, max)
	var n int32
	if e := scanFetch(&buf[0], max, &n); e != 0 {
		return nil, e
	}
	return buf[:n], 0
}

// SetSTA configures the station interface: mode STA, credentials, RAM
// storage. The authmode threshold follows the password length (>=8 chars
// means WPA2-PSK, otherwise open). Call it before Start.
func SetSTA(ssid, pass string) int32 {
	var ssidBuf [32]byte
	copy(ssidBuf[:], ssid)
	var passBuf [64]byte
	copy(passBuf[:], pass)
	return setSTA(&ssidBuf[0], int32(len(ssid)), &passBuf[0], int32(len(pass)))
}

// Poll pops one mailbox slot into ev: 1 = event delivered, 0 = empty. If the
// queue ever overflowed, a single EvOverflow slot is reported first.
func Poll(ev *Event) int32 { return poll(&ev.Kind, &ev.Aux, &ev.IP) }

// WaitEvent polls the mailbox until one of the wanted kinds arrives or
// timeoutTicks elapse. Each retry sleeps vTaskDelay(10) = 100ms; a busy poll
// without the delay would starve the idle tasks and trip the watchdog.
// Unmatched events are consumed and the last one is returned on timeout as a
// best-effort diagnostic.
func WaitEvent(want []int32, timeoutTicks uint32) (Event, bool) {
	var last Event
	for t := uint32(0); t < timeoutTicks; t += 10 {
		var ev Event
		if Poll(&ev) == 1 {
			last = ev
			for _, w := range want {
				if ev.Kind == w {
					return ev, true
				}
			}
		}
		vTaskDelay(10)
	}
	return last, false
}

// ErrName maps the esp_err codes this pipeline can hit to their IDF names.
func ErrName(code int32) string {
	switch code {
	case 0:
		return "ESP_OK"
	case -1:
		return "ESP_FAIL"
	case 0x101:
		return "ESP_ERR_NO_MEM"
	case 0x102:
		return "ESP_ERR_INVALID_ARG"
	case 0x103:
		return "ESP_ERR_INVALID_STATE"
	case 0x107:
		return "ESP_ERR_TIMEOUT"
	case 0x110D:
		return "ESP_ERR_NVS_NO_FREE_PAGES"
	case 0x1110:
		return "ESP_ERR_NVS_NEW_VERSION_FOUND"
	case 0x3001:
		return "ESP_ERR_WIFI_NOT_INIT"
	case 0x3002:
		return "ESP_ERR_WIFI_NOT_STARTED"
	case 0x3005:
		return "ESP_ERR_WIFI_MODE"
	case 0x3006:
		return "ESP_ERR_WIFI_STATE"
	case 0x3007:
		return "ESP_ERR_WIFI_CONN"
	case 0x300A:
		return "ESP_ERR_WIFI_SSID"
	case 0x300B:
		return "ESP_ERR_WIFI_PASSWORD"
	case 0x300C:
		return "ESP_ERR_WIFI_TIMEOUT"
	case 0x3015:
		return "ESP_ERR_WIFI_NOT_ASSOC"
	}
	return "esp_err 0x" + hex6(uint32(code))
}

// hex6 formats the low 24 bits as fixed-width hex (esp_err codes fit).
func hex6(v uint32) string {
	const digits = "0123456789ABCDEF"
	var buf [6]byte
	for i := 0; i < 6; i++ {
		buf[5-i] = digits[(v>>(uint(i)*4))&0xF]
	}
	return string(buf[:])
}
