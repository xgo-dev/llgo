// Package wifi binds llgo-compiled Go code to the ESP-IDF esp_wifi stack.
//
// # Layout
//
// Every package under pkg/esp follows the same template (glue lives in a
// subdirectory because the go tool rejects C sources inside a Go package):
//
//	wifi.go          public API, no build tag
//	wifi_idf.go      //go:build llgo.idf - bodyless //go:linkname externs
//	wifi_stub.go     //go:build !llgo.idf - host-build stubs (go vet/gopls)
//	wifi_glue.c      glue/wifi_glue.c - ESP-IDF C glue template, compiled
//	                 only under ESP_PLATFORM
//
// # Usage
//
// On the device (llgo build -target esp32s31-idf) the package talks to IDF
// through the glue; on any other build the stubs return errUnsupported, so
// `go build` / `go vet` / gopls all stay clean:
//
//	e := wifi.Setup()                    // NVS + netif + event loop + esp_wifi_init
//	wifi.EventsRegister()
//	wifi.SetModeSTA()                    // or wifi.SetSTA(ssid, pass)
//	wifi.Start()
//	ev, ok := wifi.WaitEvent([]int32{wifi.EvStaStart}, 300)
//	wifi.ScanStart(true)
//	aps, e := wifi.ScanFetch(20)
//	for i := range aps { println(aps[i].SsidString()) }
//
// Every C call returns the raw esp_err code (0 = ESP_OK); wifi.ErrName maps
// the codes this pipeline can hit to their IDF names.
//
// # Glue integration
//
// The C glue is a template: copy glue/wifi_glue.c into your project's main
// component (WHOLE_ARCHIVE required - see pkg/esp/README.md for both
// integration recipes and the boundary rules the glue must obey).
//
// # Events: mailbox, not callbacks
//
// esp_event handlers run on the event-loop task, which must never enter the
// single-task llgo fiber scheduler. The glue therefore only copies events
// into a C-side ring; Go polls it (Poll) paced by vTaskDelay(10) = 100 ms -
// the wait helper is WaitEvent. Blocking driver calls (ScanStart) are safe:
// they block on a driver semaphore, which yields the CPU.
package wifi

// Event kinds delivered by the C mailbox (see on_event in wifi_glue.c).
const (
	// EvNone is the zero-value sentinel, never delivered by Poll: a zero
	// Event{} and a timed-out WaitEvent both carry this kind.
	EvNone            = 0
	EvStaStart        = 1
	EvStaConnected    = 2
	EvStaDisconnected = 3
	EvScanDone        = 4
	EvGotIP           = 5
	EvLostIP          = 6
	EvOverflow        = -1 // the mailbox overflowed; reported once by Poll
)

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
// and IP_EVENT (any id, GOT_IP and LOST_IP are dispatched by id in the
// handler).
func EventsRegister() int32 { return eventsRegister() }

// PrintVersion prints the IDF version and the current free heap from the C
// side; the string never crosses into Go (integer-only boundary).
func PrintVersion() int32 { return printVersion() }

// SetModeSTA switches the driver to station mode without a target SSID.
func SetModeSTA() int32 { return setModeSTA() }

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
// memory. A max of 0 or less returns an empty slice and polls nothing. The
// second result is an esp_err code.
func ScanFetch(max int32) ([]ApInfo, int32) {
	if max <= 0 {
		return []ApInfo{}, 0
	}
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
// storage. Credentials are bounded by the wifi_config_t field sizes: an
// SSID longer than 32 or a password longer than 64 bytes returns
// ESP_ERR_INVALID_ARG (0x102) before anything crosses into C. The authmode
// threshold follows the password length (>=8 chars means WPA2-PSK,
// otherwise open). Call it before Start.
func SetSTA(ssid, pass string) int32 {
	if len(ssid) > 32 || len(pass) > 64 {
		return 0x102 // ESP_ERR_INVALID_ARG: does not fit wifi_config_t
	}
	var ssidBuf [32]byte
	copy(ssidBuf[:], ssid)
	var passBuf [64]byte
	copy(passBuf[:], pass)
	return setSTA(&ssidBuf[0], int32(len(ssid)), &passBuf[0], int32(len(pass)))
}

// Connect calls esp_wifi_connect: a single attempt. Association or
// handshake failures arrive as EvStaDisconnected with a reason code; the
// application retries (see the loop in main.go).
func Connect() int32 { return connect() }

// Disconnect calls esp_wifi_disconnect.
func Disconnect() int32 { return disconnect() }

// GetIP returns the station IP in network byte order (0 = nothing
// acquired yet), reading esp_netif directly.
func GetIP() uint32 { return getIP() }

// DefaultSTA returns the credentials compiled into the firmware via
// idfproj/main/wifi_creds.h (gitignored; see wifi_creds.h.example). Both
// strings are empty when the firmware was built without that file.
func DefaultSTA() (string, string) {
	var ssid [32]byte
	var pass [64]byte
	var sl, pl int32
	defaultSTA(&ssid[0], int32(len(ssid)), &sl, &pass[0], int32(len(pass)), &pl)
	return string(ssid[:sl]), string(pass[:pl])
}

// Finish flushes the C-side stdio buffers. Defer it at the top of main:
// once the firmware parks the main task, nothing will ever flush pending
// UART output again.
func Finish() { finish() }

// HeapFree returns the IDF heap pool's current free bytes
// (esp_get_free_heap_size). It does not include the llgo GC heap, which is
// a static 256 KiB window outside the IDF allocator.
func HeapFree() uint32 { return heapFree() }

// HeapMin returns the lowest point the IDF heap pool has ever reached
// (esp_get_minimum_free_heap_size) - the watermark that catches transient
// WiFi allocation bursts.
func HeapMin() uint32 { return heapMin() }

// HeapInternalFree returns free bytes in the internal-SRAM pool.
func HeapInternalFree() uint32 { return heapInternalFree() }

// HeapSpiramFree returns free bytes in the external PSRAM pool.
func HeapSpiramFree() uint32 { return heapSpiramFree() }

// IPString formats the address returned by GetIP / the GotIP event as a
// dotted quad. The glue returns esp_netif's ip.addr: the four network-order
// octets live in memory, and the little-endian RV32 load hands them to Go
// as [b0, b1, b2, b3] in the low-to-high bytes of the uint32.
func IPString(ip uint32) string {
	const digits = "0123456789"
	b := [4]byte{uint8(ip), uint8(ip >> 8), uint8(ip >> 16), uint8(ip >> 24)}
	s := ""
	for i, v := range b {
		if i > 0 {
			s += "."
		}
		if v >= 100 {
			s += string(digits[v/100])
			s += string(digits[v/10%10])
			s += string(digits[v%10])
		} else if v >= 10 {
			s += string(digits[v/10])
			s += string(digits[v%10])
		} else {
			s += string(digits[v])
		}
	}
	return s
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
