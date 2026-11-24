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
//	err := wifi.Setup()                  // NVS + netif + event loop + esp_wifi_init
//	wifi.EventsRegister()
//	wifi.SetSTA(ssid, pass)
//	wifi.Start()
//	staStart := wifi.Subscribe(wifi.EvStaStart)
//	ev, err := staStart.Next(3000)
//	aps, err := wifi.Scan(wifi.ScanOptions{}) // subscribe + non-blocking start + fetch
//	for i := range aps { println(aps[i].SsidString()) }
//
// Every driver result comes back as the family's shared error value:
// nil = ESP_OK, and the error carries the raw esp_err code (inspectable;
// wifi.ErrName names the codes this pipeline can hit with their IDF
// names).
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
//
// # Subscribers: one poller, many consumers
//
// The C ring is a single-consumer mailbox, but applications usually have
// more than one goroutine interested in events. Subscribe lazily starts the
// package's one poller goroutine and fans every polled event out to all
// matching subscriptions; Subscription.Next is a bounded wait with the same
// semantics as WaitEvent. The poller keeps the WaitEvent discipline that
// the llgo scheduler requires on this target: every wait is a bounded
// quantum (vTaskDelay(10) = 100 ms, which yields the FreeRTOS task and feeds
// the idle task and watchdog) plus a runtime.Gosched hand-off, so no
// goroutine ever holds the single-task scheduler across an unbounded or
// long C-level wait. Direct Poll/WaitEvent use and Subscribe are mutually
// exclusive: whoever touched the mailbox first owns it, and the loser
// silently steals (or loses) events. Because the poller alternates with
// other goroutines at Go-level scheduling points, a goroutine parked on a
// channel receive that nobody will serve can starve while the poller keeps
// its quanta - the same single-consumer-era caveat WaitEvent always had;
// wait with Next (bounded) instead.
//
// # Scan
//
// Scan strings the scan state machine together - subscribe to EvScanDone,
// start the scan non-blocking, wait, fetch - and ScanFetch reports an
// over-limit request instead of silently clamping it.
//
// # Auto-reconnect
//
// SetAutoReconnect starts an optional state machine that reconnects the
// station after an unexpected disconnection (1 s exponential backoff,
// capped at 30 s, reset by GOT_IP). A Disconnect the program itself called
// never triggers it. The machine adds no synthetic event kinds -
// EvStaDisconnected / EvStaConnected / EvGotIP keep telling the whole story
// through the regular fan-out - and ReconnectState names its phase (idle /
// backoff / connecting). See reconnect.go for the full contract.
package wifi

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// errOf wraps a raw esp_err code as the error value this package returns
// (nil for ESP_OK), named through ErrName so a printed error reads like
// the pre-error ErrName output.
func errOf(code int32) error {
	if code == 0 {
		return nil
	}
	return &esperr.Error{Code: code, Name: ErrName(code)}
}

// ErrTimeout reports a bounded wifi wait that found nothing:
// ESP_ERR_TIMEOUT (0x107) - Scan with no SCAN_DONE event inside
// ScanOptions.TimeoutMs, or a Subscription.Next that saw no event inside
// its bound. (The esp_err family keeps its own code for the condition;
// the transports' ErrTimeout carries the lwip ETIMEDOUT value instead.)
var ErrTimeout = &esperr.Error{Code: 0x107, Name: ErrName(0x107)}

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

// EventKind identifies one mailbox event kind; it aliases int32 so the Ev*
// constants below pass directly as Subscribe arguments.
type EventKind = int32

// Event is one mailbox slot: kind, aux (e.g. the STA_DISCONNECTED reason
// code) and the raw network-order IP for EvGotIP.
type Event struct {
	Kind int32
	Aux  int32
	IP   uint32
}

// Pacing and buffer bounds of the subscriber model.
const (
	// pollQuantumMs is the pacing quantum shared by the event poller and
	// every bounded wait in this package: one vTaskDelay(10) = 100 ms at
	// the IDF's default 100 Hz tick, the same cadence WaitEvent always had.
	pollQuantumMs = 100

	// tickMs is the assumed FreeRTOS tick period (configTICK_RATE_HZ = 100).
	tickMs = 10

	// subQueueLen is the per-subscription event buffer. The poller drops
	// events for a subscriber whose buffer is full instead of blocking the
	// fan-out for everyone; a subscriber keeps up by polling Next.
	subQueueLen = 8

	// MaxScanResults is the number of scan records the C glue holds (and
	// therefore the most ScanFetch can ever return).
	MaxScanResults = 20

	// DefaultScanTimeoutMs bounds Scan's wait for the scan-done event when
	// ScanOptions.TimeoutMs is not set. A scan that has not finished within
	// it is not going to.
	DefaultScanTimeoutMs = 10000
)

// Subscription receives a copy of every mailbox event whose kind it asked
// for (EvOverflow is broadcast to all subscribers). Create one with
// Subscribe, consume it with Next, retire it with Close. A Subscription is
// single-consumer: exactly one goroutine calls Next on it.
type Subscription struct {
	kinds []EventKind
	ch    chan Event
}

// subscribers is the poller's fan-out list. The llgo scheduler is
// cooperative (all goroutines multiplex one FreeRTOS task and only switch
// at blocking calls), so plain slices are race-free here as long as no
// function yields between reading a slice and acting on it - which
// Subscribe, Close and dispatch all honor.
var subscribers []*Subscription

// pollerStarted gates the lazy start of the single poller goroutine.
var pollerStarted bool

// Subscribe starts the package's single poller goroutine (once, lazily) and
// returns a subscription that will receive a copy of every subsequent
// mailbox event matching one of kinds. An empty kinds list matches every
// kind. Events already sitting in the mailbox are not retroactively
// delivered; subscribe before the operation whose events you need.
//
// After the first Subscribe the poller owns the C mailbox: Poll and
// WaitEvent (both deprecated) must not be called anywhere in the program -
// each side would silently steal the other's events.
func Subscribe(kinds ...EventKind) *Subscription {
	s := &Subscription{kinds: kinds, ch: make(chan Event, subQueueLen)}
	subscribers = append(subscribers, s)
	if !pollerStarted {
		pollerStarted = true
		go pollEvents()
	}
	return s
}

// Close unsubscribes: the poller stops delivering to s, and events already
// buffered stay readable through Next. Closing twice is harmless. A
// subscription that is never closed keeps its buffered events alive until
// the program ends.
func (s *Subscription) Close() {
	for i, e := range subscribers {
		if e == s {
			last := len(subscribers) - 1
			subscribers[i] = subscribers[last]
			subscribers[last] = nil
			subscribers = subscribers[:last]
			return
		}
	}
}

// Next waits up to timeoutMs for the subscription's next event. It returns
// (event, nil) when one arrived and (zero Event, ErrTimeout) when the bound
// passed with nothing (a non-positive timeout makes a single non-blocking
// check, reported the same way on a miss). Like WaitEvent, the wait is
// bounded and paced by the poller's pollQuantumMs quanta: a timeout may
// overshoot by one quantum and sub-quantum bounds are not honored. Each
// idle iteration yields with runtime.Gosched so the poller (and any other
// runnable goroutine) keeps its scheduling slice - no C-level call is held
// across the wait, which would freeze the whole llgo scheduler.
func (s *Subscription) Next(timeoutMs int32) (Event, error) {
	for ms := int32(0); ms < timeoutMs; ms += pollQuantumMs {
		select {
		case ev := <-s.ch:
			return ev, nil
		default:
		}
		gosched()
	}
	select {
	case ev := <-s.ch:
		return ev, nil
	default:
		return Event{}, ErrTimeout
	}
}

// pollEvents is the package's single mailbox consumer: drain whatever the
// C ring holds, fan each event out to the subscribers, hand the rest of the
// quantum to other goroutines (Gosched), then rest the CPU for one quantum
// (vTaskDelay - a real FreeRTOS yield that also feeds the idle task and the
// watchdog). The loop never holds a C-level wait across a scheduling point:
// vTaskDelay bounds its own freeze to pollQuantumMs and Gosched hands the
// scheduler away, so other goroutines regain control at least once per
// quantum - the same bounded-wait discipline WaitEvent has always used.
func pollEvents() {
	for {
		for {
			var ev Event
			if Poll(&ev) != 1 {
				break
			}
			dispatch(ev)
		}
		gosched()
		vTaskDelay(pollQuantumMs / tickMs)
	}
}

// dispatch hands ev to every subscriber whose filter matches, without ever
// blocking: a subscriber that has not drained its buffer in time loses the
// event (documented at subQueueLen). Iterating the subscribers slice
// directly is safe under the cooperative scheduler: Subscribe/Close only
// run when the poller is parked in a blocking call, never mid-loop.
func dispatch(ev Event) {
	for _, s := range subscribers {
		if s.delivers(ev.Kind) {
			select {
			case s.ch <- ev:
			default:
			}
		}
	}
}

// delivers reports whether events of kind go to s. An empty filter matches
// every kind; the mailbox's global overflow report matches all of them.
func (s *Subscription) delivers(kind EventKind) bool {
	if kind == EvOverflow {
		return true
	}
	if len(s.kinds) == 0 {
		return true
	}
	for _, k := range s.kinds {
		if k == kind {
			return true
		}
	}
	return false
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
func Setup() error { return errOf(setup()) }

// EventsRegister hooks the glue's event handler into WIFI_EVENT (any id)
// and IP_EVENT (any id, GOT_IP and LOST_IP are dispatched by id in the
// handler).
func EventsRegister() error { return errOf(eventsRegister()) }

// PrintVersion prints the IDF version and the current free heap from the C
// side; the string never crosses into Go (integer-only boundary).
func PrintVersion() error { return errOf(printVersion()) }

// SetModeSTA switches the driver to station mode without a target SSID.
func SetModeSTA() error { return errOf(setModeSTA()) }

// Start calls esp_wifi_start; the asynchronous WIFI_EVENT_STA_START shows up
// in the mailbox.
func Start() error { return errOf(start()) }

// ScanStart calls esp_wifi_scan_start(NULL, blocking). A blocking scan
// freezes the llgo scheduler for the scan duration (well under a second on
// emu); the WIFI_EVENT_SCAN_DONE event still lands in the mailbox.
func ScanStart(blocking bool) error {
	b := int32(0)
	if blocking {
		b = 1
	}
	return errOf(scanStart(b))
}

// ScanFetch takes the scan result list into a fresh slice (at most max
// entries; the C glue holds at most MaxScanResults) and also frees the
// driver's scan memory. A max of 0 or less returns an empty slice and polls
// nothing. Asking for more than MaxScanResults is reported rather than
// clamped away: the fetch still runs at MaxScanResults, and the error is
// ESP_ERR_INVALID_ARG (0x102) alongside the fetched prefix (both results
// are set - the slice is usable, the error says it was clamped).
func ScanFetch(max int32) ([]ApInfo, error) {
	if max <= 0 {
		return []ApInfo{}, nil
	}
	over := int32(0)
	if max > MaxScanResults {
		max = MaxScanResults
		over = 0x102 // ESP_ERR_INVALID_ARG: the glue cannot hold more
	}
	buf := make([]ApInfo, max)
	var n int32
	if e := scanFetch(&buf[0], max, &n); e != 0 {
		return nil, errOf(e)
	}
	if over != 0 {
		return buf[:n], errOf(over)
	}
	return buf[:n], nil
}

// ScanResult is one scan record - the flat ApInfo wire form under the name
// that describes what a Scan call returns.
type ScanResult = ApInfo

// ScanOptions bounds a Scan call.
type ScanOptions struct {
	// Max caps the fetched records. The C glue holds at most
	// MaxScanResults; 0 or less means MaxScanResults.
	Max int32

	// TimeoutMs bounds the wait for the scan-done event. 0 or less means
	// DefaultScanTimeoutMs.
	TimeoutMs int32
}

// Scan runs one scan and returns its records, stringing the state machine
// together that the raw API leaves to the caller: subscribe to EvScanDone,
// ScanStart(false) (the scheduler stays live, unlike a blocking scan), wait
// for SCAN_DONE, ScanFetch. Errors: ScanStart's when the scan could not
// start, ErrTimeout (ESP_ERR_TIMEOUT, 0x107) when no SCAN_DONE arrived
// within the bound, and ScanFetch's otherwise (including the over-limit
// report from above). Events that race the scan for kinds nobody
// subscribes to are dropped; keep your own subscriptions for those. The
// scan-done subscription is closed on return.
func Scan(opts ScanOptions) ([]ScanResult, error) {
	max := opts.Max
	if max <= 0 {
		max = MaxScanResults
	}
	timeoutMs := opts.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = DefaultScanTimeoutMs
	}
	sub := Subscribe(EvScanDone)
	defer sub.Close()
	if err := ScanStart(false); err != nil {
		return nil, err
	}
	if ev, err := sub.Next(timeoutMs); err != nil || ev.Kind != EvScanDone {
		return nil, ErrTimeout // no SCAN_DONE within the bound
	}
	return ScanFetch(max)
}

// SetSTA configures the station interface: mode STA, credentials, RAM
// storage. Credentials are bounded by the wifi_config_t field sizes: an
// SSID longer than 32 or a password longer than 64 bytes returns
// ESP_ERR_INVALID_ARG (0x102) before anything crosses into C. The authmode
// threshold follows the password length (>=8 chars means WPA2-PSK,
// otherwise open). Call it before Start.
func SetSTA(ssid, pass string) error {
	if len(ssid) > 32 || len(pass) > 64 {
		return errOf(0x102) // ESP_ERR_INVALID_ARG: does not fit wifi_config_t
	}
	var ssidBuf [32]byte
	copy(ssidBuf[:], ssid)
	var passBuf [64]byte
	copy(passBuf[:], pass)
	return errOf(setSTA(&ssidBuf[0], int32(len(ssid)), &passBuf[0], int32(len(pass))))
}

// Connect calls esp_wifi_connect: a single attempt. Association or
// handshake failures arrive as EvStaDisconnected with a reason code; the
// application retries (see the loop in main.go). While the auto-reconnect
// state machine is enabled it owns this call - see SetAutoReconnect.
func Connect() error { return errOf(connect()) }

// Disconnect calls esp_wifi_disconnect. While the auto-reconnect state
// machine is enabled (SetAutoReconnect(true)), a program-initiated
// disconnect counts as intentional: the flag is armed before the driver call
// (the event cannot outrun it) and the matching EvStaDisconnected is
// consumed without scheduling a retry. Only an unexpected disconnection -
// an EvStaDisconnected the program did not ask for - starts the
// backoff-and-retry sequence. If the driver call fails, no event will
// arrive, so the flag is dropped again.
func Disconnect() error {
	if autoReconnect {
		userDisconnect = true
	}
	err := errOf(disconnect())
	if err != nil && autoReconnect {
		userDisconnect = false
	}
	return err
}

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
//
// Deprecated: Poll makes its caller the mailbox's single consumer, which
// collides with the subscriber poller once any Subscribe has happened - the
// two sides silently steal each other's events. Use Subscribe and
// Subscription.Next; keep direct Poll only in programs that never Subscribe.
func Poll(ev *Event) int32 { return poll(&ev.Kind, &ev.Aux, &ev.IP) }

// waitEvent is the shared body of WaitEvent and WaitEventMs: poll the
// mailbox until one of the wanted kinds arrives or timeoutMs elapses. Each
// retry sleeps vTaskDelay(10) = 100ms; a busy poll without the delay would
// starve the idle tasks and trip the watchdog. Unmatched events are
// consumed and the last one is returned on timeout as a best-effort
// diagnostic.
func waitEvent(want []int32, timeoutMs int32) (Event, bool) {
	var last Event
	for ms := int32(0); ms < timeoutMs; ms += pollQuantumMs {
		var ev Event
		if Poll(&ev) == 1 {
			last = ev
			for _, w := range want {
				if ev.Kind == w {
					return ev, true
				}
			}
		}
		vTaskDelay(pollQuantumMs / tickMs)
	}
	return last, false
}

// WaitEvent polls the mailbox until one of the wanted kinds arrives or
// timeoutTicks elapse; the unit is the scheduler tick (10 ms at the IDF's
// default 100 Hz), so bounds that are not multiples of 10 overshoot to the
// next 100 ms quantum.
//
// Deprecated: WaitEvent blocks the scheduler in vTaskDelay and makes its
// caller the mailbox's single consumer - mutually exclusive with Subscribe
// (the two sides silently steal each other's events), and only one
// goroutine may call it. Use Subscribe and Subscription.Next; keep
// WaitEvent only in programs that never Subscribe.
func WaitEvent(want []int32, timeoutTicks uint32) (Event, bool) {
	return waitEvent(want, int32(timeoutTicks)*tickMs)
}

// WaitEventMs is WaitEvent with the bound in milliseconds instead of
// scheduler ticks: Next(3000)-style bounds line up directly.
//
// Deprecated: the same single-consumer caveat as WaitEvent applies - mutually
// exclusive with Subscribe. Use Subscribe and Subscription.Next.
func WaitEventMs(want []int32, timeoutMs int32) (Event, bool) {
	return waitEvent(want, timeoutMs)
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
