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
// # Access point
//
// The same stack serves the soft AP: SetAP describes the network
// (SSID/passphrase/channel/hidden/station limit/auth mode), Start brings it
// up, and the AP's own DHCP server leases 192.168.4.x addresses to clients
// with no Go code in the path:
//
//	err := wifi.SetAP(wifi.APConfig{SSID: "llgo-ap", Password: "llgoappass",
//		Channel: 6, MaxSTA: 2, AuthMode: wifi.AuthWPA2PSK})
//	wifi.Start()
//	ap := wifi.Subscribe(wifi.EvAPStart)
//	ap.Next(3000)
//	ip, err := wifi.GetAPIP()        // 192.168.4.1
//	sts, err := wifi.APStations()    // MAC + RSSI + leased IP per client
//
// SetModeAPSTA runs both interfaces at once; SetAP and SetSTA then
// configure them without undoing each other (see SetAP for the ordering
// rule). Stop ends an AP episode without tearing the driver down.
//
// The AP side of the mailbox adds five event kinds (EvAPStart, EvAPStop,
// EvAPStaConnected, EvAPStaDisconnected, EvAPStaIPAssigned). The three that
// are about a client carry its MAC in the Event itself, the two that come from
// an association or a departure carry its AID as well, and the lease event
// carries the address. The station-side semantics are untouched:
// EvStaDisconnected still means the *station* lost its AP, so the
// auto-reconnect machine never reacts to a soft-AP client leaving.
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

// ErrTimeout reports a bounded wifi wait that found nothing - a
// Subscription.Next, or a Scan's wait for the SCAN_DONE event, whose bound
// passed with no event. It is the package's semantic sentinel - match it
// with errors.Is - and it keeps ESP_ERR_TIMEOUT (0x107) on Error.Code and
// Error.Name as the diagnostic annotation, so code-style checks against the
// esp_err value keep working. The wait call sites wrap it with esperr.Wrap
// to name the operation that timed out. (The esp_err family keeps its own
// code for the condition; the transports' ErrTimeout carries the lwip
// ETIMEDOUT value instead.)
var ErrTimeout = newSentinel("esp/wifi: timeout", 0x107, "ESP_ERR_TIMEOUT")

// ErrGlueABI reports a glue/binding contract mismatch: the wifi_glue.c a
// firmware links was written for a different version of this package. The C
// symbols keep their names across versions, so a stale copy would not fail the
// link - it would silently answer with zeros for the fields it does not know
// (the soft-AP client's MAC and AID, for instance). Setup compares the glue's
// version with glueABIVersion and refuses to run on a mismatch, so the failure
// arrives as one named error instead of as empty data minutes later. It is the
// package's semantic sentinel - match it with errors.Is - and it keeps the
// glue's 0x7F01 stub-segment code on Error.Code as the diagnostic annotation.
var ErrGlueABI = newSentinel("esp/wifi: glue ABI mismatch", 0x7F01, "LLGO_ERR_GLUE_ABI")

// newSentinel declares a semantic sentinel that keeps its historical
// pipeline code and ErrName name on Error.Code/Error.Name as the annotation
// (esperr.New is text-only; the driver still reports the code, and the
// demos' errText helpers read the fields off the bare value).
func newSentinel(text string, code int32, name string) *esperr.Error {
	e, _ := esperr.New(text).(*esperr.Error)
	e.Code = code
	e.Name = name
	return e
}

// errOf turns one glue result into the error this package returns (nil for
// ESP_OK): every code forwards as a bare esperr.Coded value, named through
// ErrName so a printed error reads like the driver's own name for the code.
func errOf(code int32) error {
	if code == 0 {
		return nil
	}
	return esperr.Coded(code, ErrName(code))
}

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

	// Access-point kinds. They share the mailbox and the fan-out with the
	// station kinds above and never alias them: a client associating with
	// the soft AP is EvAPStaConnected, not EvStaConnected, so the station
	// half of a program (and the auto-reconnect machine) sees exactly what
	// it saw before.
	EvAPStart           = 7  // the soft AP came up (WIFI_EVENT_AP_START)
	EvAPStop            = 8  // the soft AP stopped (WIFI_EVENT_AP_STOP)
	EvAPStaConnected    = 9  // a client associated: MAC + AID
	EvAPStaDisconnected = 10 // a client left: MAC + AID, reason code in Aux
	EvAPStaIPAssigned   = 11 // the AP's DHCP server leased it an IP: MAC + IP

	EvOverflow = -1 // the mailbox overflowed; reported once by Poll
)

// EventKind identifies one mailbox event kind; it aliases int32 so the Ev*
// constants below pass directly as Subscribe arguments.
type EventKind = int32

// Event is one mailbox slot.
type Event struct {
	Kind int32
	// Aux is the driver's reason code: the STA_DISCONNECTED reason for
	// EvStaDisconnected, the AP_STADISCONNECTED reason for
	// EvAPStaDisconnected, 0 elsewhere.
	Aux int32
	// IP is the raw network-order address: the station's for EvGotIP, the
	// client's leased address for EvAPStaIPAssigned (0 elsewhere). Format
	// it with IPString.
	IP uint32
	// MAC is the soft AP client the event is about (EV_AP_STA*); all zero
	// for the station kinds. It rides in the event rather than in a
	// follow-up query because a client can associate and leave between any
	// two queries - APStations reports who is here *now*, the event
	// reports who came or went.
	MAC [6]uint8
	// AID is the association id the soft AP assigned the client
	// (EV_AP_STA*), 0 elsewhere.
	AID uint16
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
// check, reported the same way on a miss).
//
// The bound is wall clock, one pollQuantumMs quantum at a time: every idle
// iteration parks the FreeRTOS task in vTaskDelay (which is also what feeds
// the idle task and the watchdog) instead of spinning on Gosched. A timeout
// can therefore overshoot by up to a quantum - because the sleep is a whole
// quantum, sub-quantum bounds are not honored - and undershoot by up to a
// tick's phase (the sleep lands on the next tick boundary, so a quantum is
// 90..100 ms in practice). An event that arrives during a sleep is seen on
// the next pass, so delivery latency is up to one quantum plus the poller's
// own pass. No C-level call is held across a scheduling point, so other
// goroutines keep running.
func (s *Subscription) Next(timeoutMs int32) (Event, error) {
	for ms := int32(0); ms < timeoutMs; ms += pollQuantumMs {
		select {
		case ev := <-s.ch:
			return ev, nil
		default:
		}
		gosched()
		vTaskDelay(pollQuantumMs / tickMs)
	}
 select {
	case ev := <-s.ch:
		return ev, nil
	default:
		return Event{}, esperr.Wrap("wifi: wait event", ErrTimeout)
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

// glueABIVersion is the contract version this package speaks; the C glue
// reports the version it implements (llgo_wifi_glue_abi). It changes whenever a
// Go-facing signature or field set does.
const glueABIVersion = 2

// Setup performs the one-shot WiFi bring-up: NVS init (erase + retry on the
// fresh-flash error codes), netif init, default event loop, default STA
// netif and esp_wifi_init. WIFI_INIT_CONFIG_DEFAULT() stays in C. The first
// thing it checks is that the glue speaks this package's contract - a mismatch
// is ErrGlueABI and nothing else runs.
func Setup() error {
	if v := glueABI(); v != glueABIVersion {
		return esperr.Wrap("wifi: setup", ErrGlueABI)
	}
	return errOf(setup())
}

// EventsRegister hooks the glue's event handler into WIFI_EVENT (any id)
// and IP_EVENT (any id, GOT_IP and LOST_IP are dispatched by id in the
// handler).
func EventsRegister() error { return errOf(eventsRegister()) }

// PrintVersion prints the IDF version and the current free heap from the C
// side; the string never crosses into Go (integer-only boundary).
func PrintVersion() error { return errOf(printVersion()) }

// Driver modes handed to esp_wifi_set_mode. The numbers are IDF's
// wifi_mode_t values, passed through unchanged.
const (
	modeSTA   = 1
	modeAP    = 2
	modeAPSTA = 3
)

// SetModeSTA switches the driver to station mode without a target SSID.
// A mode chosen here is explicit: SetSTA and SetAP no longer change it (see
// SetAP for the APSTA ordering rule).
func SetModeSTA() error { return errOf(setMode(modeSTA)) }

// SetModeAP switches the driver to soft-AP mode without a configuration
// (SetAP alone already selects AP mode; this pins it when both interfaces
// are configured).
func SetModeAP() error { return errOf(setMode(modeAP)) }

// SetModeAPSTA switches the driver to AP + station mode: the soft AP serves
// clients while the station associates elsewhere. Configure the two
// interfaces with SetAP and SetSTA; neither changes an explicitly chosen
// mode.
func SetModeAPSTA() error { return errOf(setMode(modeAPSTA)) }

// Start calls esp_wifi_start for whatever mode is configured. The
// asynchronous WIFI_EVENT_STA_START or WIFI_EVENT_AP_START shows up in the
// mailbox (EvStaStart / EvAPStart); an APSTA start delivers both.
func Start() error { return errOf(start()) }

// Stop calls esp_wifi_stop. For an AP that ends beaconing: the pending
// WIFI_EVENT_AP_STOP arrives as EvAPStop and the driver releases its
// AP-mode buffers. The AP netif and its static address survive (GetAPIP
// still answers), and there is no Deinit counterpart - the package keeps
// its one-shot Setup model, so a program that wants the radio again calls
// Start after reconfiguring.
func Stop() error { return errOf(stop()) }

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
		return nil, esperr.Wrap("wifi: scan", ErrTimeout) // no SCAN_DONE within the bound
	}
	return ScanFetch(max)
}

// SetSTA configures the station interface: mode STA, credentials, RAM
// storage. Credentials are bounded by the wifi_config_t field sizes: an
// SSID longer than 32 or a password longer than 64 bytes returns
// ESP_ERR_INVALID_ARG (0x102) before anything crosses into C. The authmode
// threshold follows the password length (>=8 chars means WPA2-PSK,
// otherwise open). Call it before Start.
//
// Mode: if no mode was chosen explicitly this selects STA (or keeps APSTA
// when SetAP already configured the AP); after SetModeSTA/SetModeAP/SetModeAPSTA
// it configures the station interface and leaves the mode alone.
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

// AuthMode is the security mode of the soft AP. The numbers are IDF's
// wifi_auth_mode_t values verbatim, so a mode this package does not name still
// reads back correctly from GetAPConfig; SetAP accepts only the four below and
// reports the rest (WEP, WPA, enterprise, WAPI, OWE) as ESP_ERR_INVALID_ARG
// before the call.
type AuthMode int32

const (
	// AuthOpen serves an open network: APConfig.Password must be empty.
	AuthOpen AuthMode = 0
	// AuthWPA2PSK is WPA2-PSK (AES/CCMP): the mode every client of this
	// generation joins, esp-emu's own station included.
	AuthWPA2PSK AuthMode = 3
	// AuthWPA3PSK is WPA3-SAE, which requires protected management
	// frames - clients without PMF cannot join.
	AuthWPA3PSK AuthMode = 6
	// AuthWPA2WPA3PSK is the WPA2/WPA3 transition mode: PSK and SAE are
	// both offered and PMF is capable rather than required.
	AuthWPA2WPA3PSK AuthMode = 7
)

const (
	// DefaultMaxSTA is the station limit SetAP applies when
	// APConfig.MaxSTA is 0 - the IDF soft AP's own default (4).
	DefaultMaxSTA = 4

	// MaxAPStations is the package's upper bound for APConfig.MaxSTA and
	// for the length of an APStations result: IDF's ESP_WIFI_MAX_CONN_NUM
	// on this package's target chip (15 on the S31). A chip with a smaller
	// limit - the C2 has 4 - is handled by the glue: it rejects a larger
	// MaxSTA with ESP_ERR_INVALID_ARG and APStations sizes itself from
	// llgo_wifi_max_sta, so it never returns more records than the driver
	// can hold.
	MaxAPStations = 15
)

// APConfig is the soft-AP configuration SetAP applies. Where a field has a
// driver default, the zero value selects it: an SSID alone is a working open
// network, and SSID + Password + AuthWPA2PSK a working WPA2 one.
type APConfig struct {
	// SSID is the network name: 1..32 bytes. Empty is
	// ESP_ERR_WIFI_SSID (0x300A), longer than 32 ESP_ERR_INVALID_ARG
	// (0x102) - both before anything crosses into C.
	SSID string

	// Password is the WPA2 passphrase: empty for AuthOpen, otherwise
	// 8..63 bytes (WPA2 requires 8; 63 is the driver's soft-AP maximum,
	// one byte less than the station-side buffer). A length that does not
	// fit is ESP_ERR_INVALID_ARG (0x102, the same "does not fit" code
	// SetSTA uses); a password of usable length that the auth mode cannot
	// use - or a mode that needs one and has none - is ESP_ERR_WIFI_PASSWORD
	// (0x300B).
	Password string

	// Channel is the 2.4 GHz channel, 1..14. 0 means 1: the package never
	// leaves channel selection to the driver, so a firmware's channel is
	// the same in every run.
	Channel uint8

	// Hidden stops beaconing the SSID. Clients that already know the name
	// still join; a scanner shows nothing.
	Hidden bool

	// MaxSTA caps associated stations, 1..MaxAPStations. 0 means
	// DefaultMaxSTA. The driver reserves nothing per slot up front
	// (measured: 2 and 4 cost the same at Start), so the limit neither
	// saves nor costs memory until clients attach; it is the admission
	// control.
	MaxSTA int

	// AuthMode is the security mode. A password with AuthOpen is rejected
	// (0x300B) instead of being silently ignored.
	AuthMode AuthMode
}

// SetAP configures the soft AP: name, passphrase, channel, hidden flag,
// station limit and auth mode, hands the config to the driver for the AP
// interface (WIFI_IF_AP) and - unless a mode was already chosen explicitly -
// switches the driver to AP mode, the same one-call shape as SetSTA.
//
// # Ordering with the mode setters
//
// SetModeSTA, SetModeAP and SetModeAPSTA choose the driver mode explicitly,
// and from then on neither SetSTA nor SetAP changes it again. An APSTA
// program therefore configures both interfaces without one call undoing the
// other:
//
//	wifi.SetModeAPSTA()
//	wifi.SetAP(apCfg)       // AP interface, mode stays APSTA
//	wifi.SetSTA(ssid, pass) // station interface, mode stays APSTA
//	wifi.Start()
//
// Without an explicit mode, SetAP means AP and SetSTA means STA (the
// pre-AP default), so the two single-interface recipes keep working
// unchanged.
//
// # What it creates
//
// The AP netif - the interface the soft-AP DHCP server runs on, holding the
// default address 192.168.4.1 - is created here, lazily and once. That is
// why most of AP mode's fixed RAM cost shows up in heap marks taken after
// this call, and the rest after Start; Setup itself stays station-sized.
//
// Call it before Start. Errors: the validation codes above, then the
// driver's own (ESP_ERR_WIFI_MODE 0x3005 when the explicitly chosen mode has
// no AP interface, ESP_ERR_WIFI_PASSWORD, ESP_ERR_WIFI_SSID, ...).
func SetAP(cfg APConfig) error {
	if len(cfg.SSID) > 32 {
		return errOf(0x102) // ESP_ERR_INVALID_ARG: does not fit wifi_ap_config_t
	}
	if len(cfg.SSID) == 0 {
		return errOf(0x300A) // ESP_ERR_WIFI_SSID: the driver rejects an empty SSID
	}
	if len(cfg.Password) > 63 {
		return errOf(0x102) // ESP_ERR_INVALID_ARG: the AP passphrase is at most 63
	}
	if n := len(cfg.Password); n != 0 && n < 8 {
		return errOf(0x300B) // ESP_ERR_WIFI_PASSWORD: WPA2 needs 8 or more
	}
	switch cfg.AuthMode {
	case AuthOpen:
		if len(cfg.Password) != 0 {
			return errOf(0x300B) // ESP_ERR_WIFI_PASSWORD: a password needs an auth mode
		}
	case AuthWPA2PSK, AuthWPA3PSK, AuthWPA2WPA3PSK:
		if len(cfg.Password) < 8 {
			return errOf(0x300B) // ESP_ERR_WIFI_PASSWORD
		}
	default:
		return errOf(0x102) // ESP_ERR_INVALID_ARG: no soft AP serves this mode
	}
	if cfg.Channel > 14 {
		return errOf(0x102) // ESP_ERR_INVALID_ARG: 2.4 GHz only
	}
	if cfg.MaxSTA < 0 || cfg.MaxSTA > MaxAPStations {
		return errOf(0x102) // ESP_ERR_INVALID_ARG
	}
	channel := cfg.Channel
	if channel == 0 {
		channel = 1
	}
	maxSTA := cfg.MaxSTA
	if maxSTA == 0 {
		maxSTA = DefaultMaxSTA
	}
	hidden := int32(0)
	if cfg.Hidden {
		hidden = 1
	}
	var ssidBuf [32]byte
	copy(ssidBuf[:], cfg.SSID)
	var passBuf [64]byte
	copy(passBuf[:], cfg.Password)
	return errOf(setAP(&ssidBuf[0], int32(len(cfg.SSID)),
		&passBuf[0], int32(len(cfg.Password)),
		int32(channel), hidden, int32(maxSTA), int32(cfg.AuthMode)))
}

// GetAPConfig reads the soft-AP configuration back out of the driver
// (esp_wifi_get_config on WIFI_IF_AP) - the answer to "did the channel and
// the auth mode actually take effect", as opposed to "did the call return
// zero". Password is never marshalled back and stays empty.
//
// Errors: the driver's. ESP_ERR_WIFI_MODE (0x3005) when the chosen mode has
// no AP interface, or when SetAP never ran (no AP netif exists).
func GetAPConfig() (APConfig, error) {
	var ssid [32]byte
	var ssidLen, channel, hidden, maxSTA, authMode int32
	if e := getAPConfig(&ssid[0], int32(len(ssid)), &ssidLen,
		&channel, &hidden, &maxSTA, &authMode); e != 0 {
		return APConfig{}, errOf(e)
	}
	return APConfig{
		SSID:     string(ssid[:ssidLen]),
		Channel:  uint8(channel),
		Hidden:   hidden != 0,
		MaxSTA:   int(maxSTA),
		AuthMode: AuthMode(authMode),
	}, nil
}

// GetAPIP returns the soft AP's own address in network byte order -
// 192.168.4.1 with the IDF defaults - taken from the AP netif. The address
// is static and assigned when SetAP creates the netif, so it is available
// before Start; format it with IPString.
//
// Errors: ESP_ERR_WIFI_MODE (0x3005) when no AP netif exists (SetAP never
// ran) or the netif read fails.
func GetAPIP() (uint32, error) {
	var ip uint32
	if e := getAPIP(&ip); e != 0 {
		return 0, errOf(e)
	}
	return ip, nil
}

// Station is one client associated with the soft AP: its MAC, its average
// RSSI, and the IPv4 address the AP's DHCP server leased it (0 until a lease
// exists). The driver's station-list API reports exactly these three; the
// association id exists only in the association event, which carries it in
// Event.AID.
type Station struct {
	MAC  [6]uint8
	RSSI int8
	IP   uint32
}

// MACString formats the station's MAC as aa:bb:cc:dd:ee:ff.
func (s Station) MACString() string {
	const digits = "0123456789abcdef"
	buf := make([]byte, 0, 17)
	for i, b := range s.MAC {
		if i > 0 {
			buf = append(buf, ':')
		}
		buf = append(buf, digits[b>>4], digits[b&0xF])
	}
	return string(buf)
}

// APStations lists the clients currently associated with the soft AP:
// esp_wifi_ap_get_sta_list for MAC and RSSI, plus the address the AP's DHCP
// server leased to each (read from the DHCP server's own lease records, or
// from the ARP cache for a client that has not renewed). Nothing in this
// package implements DHCP - the IDF soft AP's built-in server hands out the
// 192.168.4.x leases and this call reads them back.
//
// An empty (non-nil) slice means nobody is associated; the slice holds at
// most MaxAPStations records. Errors: the driver's, e.g. ESP_ERR_WIFI_MODE
// (0x3005) while the mode has no AP interface.
func APStations() ([]Station, error) {
	max := int(maxSTA()) // the chip's ESP_WIFI_MAX_CONN_NUM
	if max <= 0 || max > MaxAPStations {
		max = MaxAPStations
	}
	buf := make([]Station, max)
	var n int32
	if e := apStaList(&buf[0], int32(max), &n); e != 0 {
		return nil, errOf(e)
	}
	return buf[:n], nil
}

// DeauthStation deauthenticates the client with this MAC. esp_wifi_deauth_sta
// takes an association id and the station list carries none, so the glue asks
// the driver for it (esp_wifi_ap_get_sta_aid) and passes it on - there is no
// bookkeeping on this side, and no AID is ever guessed.
//
// The driver's own codes come back unchanged: ESP_ERR_NOT_FOUND (0x105) when
// no station with this MAC is associated, ESP_ERR_WIFI_MODE (0x3005) while the
// mode has no AP interface. The departure itself arrives in the mailbox as
// EvAPStaDisconnected with the MAC and the reason code, like any client that
// left on its own. There is no deauthenticate-everyone form (the driver's
// aid=0 wildcard stays unused): loop over APStations.
func DeauthStation(mac [6]uint8) error {
	return errOf(deauthSta(&mac[0]))
}

// PowerSaveMode is the Wi-Fi modem power-save mode - IDF's wifi_ps_type_t
// verbatim (esp_wifi_types_generic.h). Modem sleep is the station's side of
// the 802.11 power-save protocol: between the beacons it must wake for, the
// driver switches RF, PHY and baseband off, and the interval between those
// wake-ups is the knob that trades idle current against downlink latency.
// The modes, with the behavior the IDF docs state (esp_wifi.h's attention
// note, api-guides/wifi-driver/wifi-performance-and-power-save.rst,
// api-guides/low-power-mode/low-power-mode-wifi.rst):
//
//	PowerSaveNone      Modem sleep off: the radio stays on, downlink data is
//	                   received the moment it arrives, and the idle current is
//	                   the full radio-on figure (ballpark ~100 mA on the
//	                   radio-class boards; empirical, measure on the target
//	                   board - the emulator does not model RF power). The mode
//	                   for latency-bound links.
//	PowerSaveMinModem  Wake for every DTIM beacon. Broadcast data follows
//	                   the DTIM, so nothing is lost, but downlink receive
//	                   latency can reach one DTIM cycle (AP-determined; a
//	                   short DTIM weakens the saving - IDF docs). Idle current
//	                   drops to roughly a fifth of PowerSaveNone (ballpark
//	                   ~20-30 mA; empirical, board-dependent). This is the
//	                   driver's default after esp_wifi_init (esp_wifi.h).
//	PowerSaveMaxModem  Wake every listen_interval beacons (the
//	                   wifi_sta_config_t field, set before Connect). The
//	                   longest interval saves the most, but DTIMs are missed,
//	                   so buffered broadcast data is lost and the receive
//	                   latency grows to the listen interval (IDF docs).
//	                   Verify against the real AP on hardware.
//
// The esp32s31 sleep-current tables are still TODO in the IDF docs, so the mA
// figures above are radio-class ballparks, not S31 measurements; the
// qualitative behavior (what wakes the station, what gets lost, how far the
// latency reaches) is quoted from the docs.
//
// Modem sleep is a property of the station association: it engages when the
// station connects and stops when it disconnects (IDF docs), and the driver
// itself accepts esp_wifi_set_ps in any mode without effect there (verified
// against the net80211 entry point, which validates only the 0..2 range), so
// this package refuses a mode without a station interface (NULL, AP) in the
// glue before the driver can swallow the call - see SetPowerSave.
type PowerSaveMode int32

const (
	// PowerSaveNone disables modem sleep: radio always on, lowest receive
	// latency, highest idle current (WIFI_PS_NONE).
	PowerSaveNone PowerSaveMode = 0
	// PowerSaveMinModem wakes the station for every DTIM beacon: no
	// broadcast loss, receive latency up to one DTIM cycle, the driver's
	// default after esp_wifi_init (WIFI_PS_MIN_MODEM).
	PowerSaveMinModem PowerSaveMode = 1
	// PowerSaveMaxModem wakes the station every listen_interval beacons
	// (wifi_sta_config_t): the most saving, buffered broadcast data may be
	// lost, latency up to the listen interval (WIFI_PS_MAX_MODEM).
	PowerSaveMaxModem PowerSaveMode = 2
)

// String names the mode with IDF's WIFI_PS_* spelling ("NONE",
// "MIN_MODEM", "MAX_MODEM"), so probe logs and errors print the driver's
// own name for the mode; values outside the enum render INVALID.
func (m PowerSaveMode) String() string {
	switch m {
	case PowerSaveNone:
		return "NONE"
	case PowerSaveMinModem:
		return "MIN_MODEM"
	case PowerSaveMaxModem:
		return "MAX_MODEM"
	default:
		return "INVALID"
	}
}

// SetPowerSave selects the modem power-save mode for the station interface.
// It works on a started or running driver (any time after Setup): the IDF
// docs recommend calling it right after esp_wifi_init, and the setting
// engages when the station associates. Errors: ESP_ERR_INVALID_ARG (0x102)
// for a value outside the three modes, ESP_ERR_WIFI_MODE (0x3005) when the
// driver has no station interface - mode NULL (before Setup, or after it
// before anything picks a mode) or AP; the soft AP never sleeps, while an
// APSTA mode passes and the setting applies to its station half.
func SetPowerSave(mode PowerSaveMode) error { return errOf(setPowerSave(int32(mode))) }

// GetPowerSave reads the mode back out of the driver (esp_wifi_get_ps) -
// the answer to "which strategy is the station actually running", as
// opposed to "did the call return zero". Until SetPowerSave changes it the
// answer is PowerSaveMinModem. Errors: ESP_ERR_WIFI_MODE (0x3005) when the
// driver has no station interface (mode NULL or AP).
func GetPowerSave() (PowerSaveMode, error) {
	var ps int32
	if e := getPowerSave(&ps); e != 0 {
		return 0, errOf(e)
	}
	return PowerSaveMode(ps), nil
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
func Poll(ev *Event) int32 {
	return poll(&ev.Kind, &ev.Aux, &ev.IP, &ev.MAC[0], &ev.AID)
}

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
	case 0x105:
		return "ESP_ERR_NOT_FOUND"
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
	case 0x3003:
		return "ESP_ERR_WIFI_NOT_STOPPED"
	case 0x3004:
		return "ESP_ERR_WIFI_IF"
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
	case 0x7F01:
		return "LLGO_ERR_GLUE_ABI"
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
